package core

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/egorbr4z/panel/internal/models"
)

func activeUser(name string) models.User {
	return models.User{
		Username: name,
		UUID:     "11111111-1111-1111-1111-111111111111",
		Password: "secret-" + name,
		Status:   models.StatusActive,
	}
}

func TestGenerateXrayConfig_VlessReality(t *testing.T) {
	spec := InboundSpec{
		Inbound: models.Inbound{
			Tag:      "vless-reality",
			Core:     models.CoreXray,
			Protocol: "vless",
			Port:     443,
			Settings: map[string]any{
				"flow":       "xtls-rprx-vision",
				"decryption": "none",
			},
			Stream: map[string]any{
				"network":  "tcp",
				"security": "reality",
			},
		},
		Users: []models.User{activeUser("alice")},
	}

	data, err := GenerateXrayConfig([]InboundSpec{spec}, 10085)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	inbounds := cfg["inbounds"].([]any)
	if len(inbounds) != 2 { // api + vless
		t.Fatalf("expected 2 inbounds (api + vless), got %d", len(inbounds))
	}
	// The user must appear as a client with the configured flow.
	if !strings.Contains(string(data), "xtls-rprx-vision") || !strings.Contains(string(data), "alice") {
		t.Fatalf("expected client with flow + email in:\n%s", data)
	}
}

func TestGenerateXrayConfig_SkipsNonActiveUsers(t *testing.T) {
	disabled := activeUser("bob")
	disabled.Status = models.StatusLimited
	spec := InboundSpec{
		Inbound: models.Inbound{Tag: "t", Core: models.CoreXray, Protocol: "trojan", Port: 8443},
		Users:   []models.User{disabled},
	}
	data, _ := GenerateXrayConfig([]InboundSpec{spec}, 10085)
	if strings.Contains(string(data), "bob") {
		t.Fatalf("limited user should not be provisioned:\n%s", data)
	}
}

func TestGenerateSingboxConfig_Hysteria2AndTuic(t *testing.T) {
	specs := []InboundSpec{
		{
			Inbound: models.Inbound{
				Tag: "hy2", Core: models.CoreSingbox, Protocol: "hysteria2", Port: 8443,
				Settings: map[string]any{"tls": map[string]any{"enabled": true}},
			},
			Users: []models.User{activeUser("carol")},
		},
		{
			Inbound: models.Inbound{Tag: "tuic", Core: models.CoreSingbox, Protocol: "tuic", Port: 9443},
			Users:   []models.User{activeUser("dave")},
		},
	}

	data, err := GenerateSingboxConfig(specs, "127.0.0.1:9090")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(cfg["inbounds"].([]any)) != 2 {
		t.Fatalf("expected 2 inbounds")
	}
	for _, want := range []string{"hysteria2", "tuic", "carol", "dave", "clash_api"} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("expected %q in config:\n%s", want, data)
		}
	}
}

func TestSingboxSupports(t *testing.T) {
	for _, p := range []string{"hysteria2", "tuic", "vless", "vmess", "trojan", "shadowsocks", "anytls", "naive"} {
		if !SingboxSupports(p) {
			t.Errorf("expected sing-box to support %q", p)
		}
	}
	if SingboxSupports("mtproto") {
		t.Error("mtproto should not be supported by sing-box generator")
	}
}

func TestRealityKeypairUnique(t *testing.T) {
	a, err := GenerateRealityKeypair()
	if err != nil {
		t.Fatal(err)
	}
	b, _ := GenerateRealityKeypair()
	if a.PrivateKey == b.PrivateKey || a.PublicKey == "" || a.ShortID == "" {
		t.Fatalf("keypair not unique/complete: %+v / %+v", a, b)
	}
}
