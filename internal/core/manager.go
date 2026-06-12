package core

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/egorbr4z/panel/internal/config"
	"github.com/egorbr4z/panel/internal/models"
	"gorm.io/gorm"
)

// Manager supervises both proxy cores and keeps their on-disk configs in sync
// with the database. Reconciliation is debounced so a burst of admin edits
// triggers a single regeneration + restart instead of a restart storm.
type Manager struct {
	db  *gorm.DB
	cfg *config.Config

	xray    *Process
	singbox *Process

	xrayCfgPath    string
	singboxCfgPath string
	apiPort        int
	clashAddr      string

	mu           sync.Mutex
	debounce     *time.Timer
	lastXrayHash string
	lastSingHash string
}

// NewManager constructs the manager and its (not yet started) core processes.
func NewManager(db *gorm.DB, cfg *config.Config) *Manager {
	coresDir := filepath.Join(cfg.DataDir, "cores")
	_ = os.MkdirAll(coresDir, 0o750)

	xrayCfg := filepath.Join(coresDir, "xray.json")
	singCfg := filepath.Join(coresDir, "sing-box.json")

	return &Manager{
		db:             db,
		cfg:            cfg,
		xrayCfgPath:    xrayCfg,
		singboxCfgPath: singCfg,
		apiPort:        10085,
		clashAddr:      "127.0.0.1:9090",
		xray:           NewProcess("xray", cfg.XrayBin, "run", "-c", xrayCfg),
		singbox:        NewProcess("sing-box", cfg.SingboxBin, "run", "-c", singCfg),
	}
}

// CoreStatus is the externally visible status of a single core.
type CoreStatus struct {
	Name     string `json:"name"`
	State    string `json:"state"`
	Restarts int64  `json:"restarts"`
	Inbounds int    `json:"inbounds"`
}

// Start performs the initial reconciliation, launching whichever cores have
// inbounds defined.
func (m *Manager) Start() error {
	return m.Reconcile()
}

// Stop tears down both cores (used on graceful shutdown).
func (m *Manager) Stop() {
	m.xray.Stop()
	m.singbox.Stop()
}

// ReconcileSoon schedules a debounced reconciliation. Call after any change to
// inbounds or users.
func (m *Manager) ReconcileSoon() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.debounce != nil {
		m.debounce.Stop()
	}
	m.debounce = time.AfterFunc(500*time.Millisecond, func() {
		if err := m.Reconcile(); err != nil {
			m.xray.logs.Write([]byte("reconcile error: " + err.Error()))
		}
	})
}

// Reconcile regenerates both core configs from the DB and restarts a core only
// if its config changed (or if it needs to start/stop).
func (m *Manager) Reconcile() error {
	xraySpecs, singSpecs, err := m.loadSpecs()
	if err != nil {
		return err
	}

	// --- Xray ---
	if len(xraySpecs) == 0 {
		m.xray.Stop()
		m.lastXrayHash = ""
	} else {
		data, err := GenerateXrayConfig(xraySpecs, m.apiPort)
		if err != nil {
			return fmt.Errorf("generate xray config: %w", err)
		}
		if err := m.applyConfig(m.xray, m.xrayCfgPath, data, &m.lastXrayHash); err != nil {
			return err
		}
	}

	// --- sing-box ---
	if len(singSpecs) == 0 {
		m.singbox.Stop()
		m.lastSingHash = ""
	} else {
		data, err := GenerateSingboxConfig(singSpecs, m.clashAddr)
		if err != nil {
			return fmt.Errorf("generate sing-box config: %w", err)
		}
		if err := m.applyConfig(m.singbox, m.singboxCfgPath, data, &m.lastSingHash); err != nil {
			return err
		}
	}
	return nil
}

// applyConfig writes the config if it changed and (re)starts the process.
func (m *Manager) applyConfig(p *Process, path string, data []byte, lastHash *string) error {
	sum := sha256.Sum256(data)
	hash := hex.EncodeToString(sum[:])

	if hash == *lastHash && p.State() == StateRunning {
		return nil // nothing changed and already running
	}
	if err := os.WriteFile(path, data, 0o640); err != nil {
		return fmt.Errorf("write config %s: %w", path, err)
	}
	*lastHash = hash

	if p.State() == StateRunning {
		return p.Restart(context.Background())
	}
	return p.Start()
}

// loadSpecs reads enabled inbounds and their active users, grouped by core.
func (m *Manager) loadSpecs() (xray, sing []InboundSpec, err error) {
	var inbounds []models.Inbound
	if err = m.db.Where("enabled = ?", true).Order("sort_order asc").Find(&inbounds).Error; err != nil {
		return nil, nil, err
	}

	for _, ib := range inbounds {
		var users []models.User
		if err = m.db.
			Joins("JOIN user_inbounds ui ON ui.user_id = users.id").
			Where("ui.inbound_id = ? AND users.status = ?", ib.ID, models.StatusActive).
			Find(&users).Error; err != nil {
			return nil, nil, err
		}
		spec := InboundSpec{Inbound: ib, Users: users}
		switch ib.Core {
		case models.CoreXray:
			xray = append(xray, spec)
		case models.CoreSingbox:
			sing = append(sing, spec)
		}
	}
	return xray, sing, nil
}

// Status returns the live status of both cores.
func (m *Manager) Status() []CoreStatus {
	xrayN, singN := m.inboundCounts()
	return []CoreStatus{
		{Name: "xray", State: m.xray.State().String(), Restarts: m.xray.Restarts(), Inbounds: xrayN},
		{Name: "sing-box", State: m.singbox.State().String(), Restarts: m.singbox.Restarts(), Inbounds: singN},
	}
}

func (m *Manager) inboundCounts() (xray, sing int) {
	var rows []models.Inbound
	m.db.Where("enabled = ?", true).Find(&rows)
	for _, ib := range rows {
		if ib.Core == models.CoreXray {
			xray++
		} else if ib.Core == models.CoreSingbox {
			sing++
		}
	}
	return
}

// Logs returns recent log lines for the named core.
func (m *Manager) Logs(name string) ([]string, bool) {
	switch name {
	case "xray":
		return m.xray.Logs(), true
	case "sing-box", "singbox":
		return m.singbox.Logs(), true
	}
	return nil, false
}

// Restart restarts the named core.
func (m *Manager) Restart(name string) bool {
	switch name {
	case "xray":
		_ = m.xray.Restart(context.Background())
		return true
	case "sing-box", "singbox":
		_ = m.singbox.Restart(context.Background())
		return true
	}
	return false
}

// ConfigPreview returns the current generated config for the named core.
func (m *Manager) ConfigPreview(name string) ([]byte, bool, error) {
	xraySpecs, singSpecs, err := m.loadSpecs()
	if err != nil {
		return nil, false, err
	}
	switch name {
	case "xray":
		data, err := GenerateXrayConfig(xraySpecs, m.apiPort)
		return data, true, err
	case "sing-box", "singbox":
		data, err := GenerateSingboxConfig(singSpecs, m.clashAddr)
		return data, true, err
	}
	return nil, false, nil
}
