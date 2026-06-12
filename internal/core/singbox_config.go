package core

import (
	"encoding/json"

	"github.com/egorbr4z/panel/internal/models"
)

// singboxProtocols is the set of inbound types this generator supports. These
// all follow sing-box's users[] model. (WireGuard/AmneziaWG is a server of a
// different shape — peers/keys/IPs — and is handled by a dedicated core.)
var singboxProtocols = map[string]bool{
	"hysteria2":   true,
	"tuic":        true,
	"vless":       true,
	"vmess":       true,
	"trojan":      true,
	"shadowsocks": true,
	"anytls":      true,
	"naive":       true,
}

// SingboxSupports reports whether sing-box can serve the given protocol here.
func SingboxSupports(protocol string) bool { return singboxProtocols[protocol] }

// GenerateSingboxConfig renders a full sing-box config from the given inbounds.
// The Clash API is enabled so the panel can poll traffic statistics.
func GenerateSingboxConfig(specs []InboundSpec, clashAddr string) ([]byte, error) {
	inbounds := make([]any, 0, len(specs))
	for _, spec := range specs {
		if in := buildSingboxInbound(spec); in != nil {
			inbounds = append(inbounds, in)
		}
	}

	cfg := map[string]any{
		"log": map[string]any{"level": "warn", "timestamp": true},
		"experimental": map[string]any{
			"clash_api": map[string]any{"external_controller": clashAddr},
		},
		"inbounds": inbounds,
		"outbounds": []any{
			map[string]any{"type": "direct", "tag": "direct"},
			map[string]any{"type": "block", "tag": "block"},
		},
	}
	return json.MarshalIndent(cfg, "", "  ")
}

// buildSingboxInbound assembles one sing-box inbound, injecting the users list.
func buildSingboxInbound(spec InboundSpec) map[string]any {
	ib := spec.Inbound
	if !singboxProtocols[ib.Protocol] {
		return nil
	}

	in := cloneMap(ib.Settings) // base (tls, obfs, method, congestion control...)
	in["type"] = ib.Protocol
	in["tag"] = ib.Tag
	in["listen"] = defaultStr(ib.Listen, "::")
	in["listen_port"] = ib.Port
	in["users"] = buildSingboxUsers(ib.Protocol, spec.Users)
	return in
}

// buildSingboxUsers maps active users to per-protocol sing-box user entries.
func buildSingboxUsers(protocol string, users []models.User) []any {
	out := make([]any, 0, len(users))
	for _, u := range users {
		if u.Status != models.StatusActive {
			continue
		}
		switch protocol {
		case "hysteria2", "trojan", "anytls", "shadowsocks":
			out = append(out, map[string]any{"name": u.Username, "password": u.Password})
		case "naive":
			out = append(out, map[string]any{"username": u.Username, "password": u.Password})
		case "tuic":
			out = append(out, map[string]any{"name": u.Username, "uuid": u.UUID, "password": u.Password})
		case "vmess":
			out = append(out, map[string]any{"name": u.Username, "uuid": u.UUID})
		case "vless":
			out = append(out, map[string]any{"name": u.Username, "uuid": u.UUID})
		}
	}
	return out
}
