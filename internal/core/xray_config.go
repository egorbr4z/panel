package core

import (
	"encoding/json"

	"github.com/egorbr4z/panel/internal/models"
)

// InboundSpec couples an inbound with the users provisioned on it. The manager
// builds these from the DB and hands them to the per-core generators.
type InboundSpec struct {
	Inbound models.Inbound
	Users   []models.User
}

// GenerateXrayConfig renders a full Xray-core config from the given inbounds.
// The API inbound + stats policy are always included so the panel can poll
// per-user traffic and (later) hot-add users over gRPC.
func GenerateXrayConfig(specs []InboundSpec, apiPort int) ([]byte, error) {
	inbounds := []any{
		// Local API inbound (dokodemo-door) for HandlerService/StatsService.
		map[string]any{
			"tag":      "api",
			"listen":   "127.0.0.1",
			"port":     apiPort,
			"protocol": "dokodemo-door",
			"settings": map[string]any{"address": "127.0.0.1"},
		},
	}

	for _, spec := range specs {
		in := buildXrayInbound(spec)
		if in != nil {
			inbounds = append(inbounds, in)
		}
	}

	cfg := map[string]any{
		"log": map[string]any{"loglevel": "warning"},
		"api": map[string]any{
			"tag":      "api",
			"services": []string{"HandlerService", "StatsService"},
		},
		"stats": map[string]any{},
		"policy": map[string]any{
			"levels": map[string]any{
				"0": map[string]any{"statsUserUplink": true, "statsUserDownlink": true},
			},
			"system": map[string]any{"statsInboundUplink": true, "statsInboundDownlink": true},
		},
		"inbounds": inbounds,
		"outbounds": []any{
			map[string]any{"protocol": "freedom", "tag": "direct"},
			map[string]any{"protocol": "blackhole", "tag": "blocked"},
		},
		"routing": map[string]any{
			"rules": []any{
				map[string]any{"type": "field", "inboundTag": []string{"api"}, "outboundTag": "api"},
			},
		},
	}

	return json.MarshalIndent(cfg, "", "  ")
}

// buildXrayInbound assembles one Xray inbound object, injecting the client list
// built from the assigned users.
func buildXrayInbound(spec InboundSpec) map[string]any {
	ib := spec.Inbound
	settings := cloneMap(ib.Settings)
	settings["clients"] = buildXrayClients(ib.Protocol, ib.Settings, spec.Users)

	// VLESS requires a decryption field.
	if ib.Protocol == "vless" {
		if _, ok := settings["decryption"]; !ok {
			settings["decryption"] = "none"
		}
	}

	in := map[string]any{
		"tag":      ib.Tag,
		"listen":   defaultStr(ib.Listen, "0.0.0.0"),
		"port":     ib.Port,
		"protocol": ib.Protocol,
		"settings": settings,
	}
	if len(ib.Stream) > 0 {
		in["streamSettings"] = ib.Stream
	}
	return in
}

// buildXrayClients maps active users to per-protocol Xray client entries.
func buildXrayClients(protocol string, settings map[string]any, users []models.User) []any {
	flow, _ := settings["flow"].(string)
	method, _ := settings["method"].(string)

	clients := make([]any, 0, len(users))
	for _, u := range users {
		if u.Status != models.StatusActive {
			continue
		}
		switch protocol {
		case "vless":
			c := map[string]any{"id": u.UUID, "email": u.Username}
			if flow != "" {
				c["flow"] = flow
			}
			clients = append(clients, c)
		case "vmess":
			clients = append(clients, map[string]any{"id": u.UUID, "email": u.Username})
		case "trojan":
			clients = append(clients, map[string]any{"password": u.Password, "email": u.Username})
		case "shadowsocks":
			c := map[string]any{"password": u.Password, "email": u.Username}
			if method != "" {
				c["method"] = method
			}
			clients = append(clients, c)
		}
	}
	return clients
}

// cloneMap shallow-copies a settings map and drops the clients key (it is
// rebuilt from users on every generation).
func cloneMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m)+1)
	for k, v := range m {
		if k == "clients" {
			continue
		}
		out[k] = v
	}
	return out
}

func defaultStr(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
