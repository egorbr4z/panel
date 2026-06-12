# vpanel

A lightweight, self-hosted VPN/proxy management panel for a **single Debian 13
server** — think *"3x-ui capabilities + Marzban-style subscriptions +
Remnawave-style polished UI, without the multi-node/squad complexity"*.

Designed to run comfortably on a **1 GB RAM / 1 vCPU** box: one Go binary does
the API, embedded SPA, subscription rendering, core supervision and traffic
accounting — the only other resident processes are the proxy cores themselves.

## Protocols & cores

| Core | Protocols |
|------|-----------|
| **Xray-core** | VLESS (Reality/Vision/XTLS), VMess, Trojan, Shadowsocks |
| **sing-box** | VLESS, VMess, Trojan, Shadowsocks, Hysteria2, TUIC, AnyTLS, NaiveProxy, WireGuard/AmneziaWG |
| *Phase 2* | mieru, MTProto (`9seconds/mtg`) — extension points already in place |

## Stack

- **Backend:** Go (Gin + GORM), single static binary (`CGO_ENABLED=0`, pure-Go SQLite).
- **Frontend:** React + Vite + TypeScript, Tailwind, Framer Motion — built into the binary via `go:embed`.
- **DB:** SQLite (WAL) by default; PostgreSQL optional.
- **Deploy:** Docker Compose + `install.sh`; Caddy reverse proxy with automatic TLS.

## Status

This is an in-progress build. Implemented so far:

- [x] Project skeleton, config, DB layer (SQLite/Postgres), auto-migration
- [x] Admin auth: argon2id hashing, JWT access/refresh, `vpanel admin create` CLI
- [x] REST API + middleware (JWT, roles, rate limit), system stats
- [x] Embedded React SPA: animated login, dashboard, inbounds, users, cores
- [x] Core supervision: process lifecycle (backoff, crash-loop, ring-buffer logs)
- [x] Both cores wired with DB-driven config generation + debounced reconcile:
  - **Xray:** VLESS (Reality/Vision), VMess, Trojan, Shadowsocks
  - **sing-box:** Hysteria2, TUIC, VLESS, VMess, Trojan, Shadowsocks, AnyTLS, NaiveProxy
- [x] Inbound CRUD + Reality keypair generator; User CRUD + provisioning
- [x] Deployment: Dockerfile (bundles cores), docker-compose, Caddy, install.sh
- [ ] Subscription system (base64 / Clash / sing-box)
- [ ] Traffic accounting + quota/expiry enforcement
- [ ] WireGuard / AmneziaWG (dedicated core — needs peer/key/IP model)
- [ ] mieru, MTProto (Phase 2)

See [docs/architecture.md](docs/architecture.md) for the full design and the
phased build plan.

## Development

```bash
# Backend (serves embedded placeholder SPA until you build the frontend)
make build && ./vpanel admin create -u admin -p changeme -role sudo
./vpanel                                 # http://localhost:8080

# Frontend dev server (proxies /api to :8080)
cd frontend && npm install && npm run dev

# Full build (frontend embedded into the binary)
make all
```

## Deploy (Debian 13)

```bash
curl -fsSL https://raw.githubusercontent.com/egorbr4z/panel/main/scripts/install.sh | sudo bash
```

The installer sets up Docker, generates config + the first admin, and brings up
the stack behind Caddy with automatic HTTPS.

## License

TBD.
