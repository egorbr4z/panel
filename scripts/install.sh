#!/usr/bin/env bash
# vpanel installer for Debian 13.
# Installs Docker if missing, generates configuration and the first sudo admin,
# then brings up the stack with docker compose.
set -euo pipefail

INSTALL_DIR="/opt/vpanel"
REPO_URL="https://github.com/egorbr4z/panel.git"

red() { printf '\033[31m%s\033[0m\n' "$*"; }
grn() { printf '\033[32m%s\033[0m\n' "$*"; }
inf() { printf '\033[36m%s\033[0m\n' "$*"; }

require_root() {
	[ "$(id -u)" -eq 0 ] || { red "Please run as root (sudo)."; exit 1; }
}

check_os() {
	if [ -r /etc/os-release ]; then
		. /etc/os-release
		[ "${ID:-}" = "debian" ] || red "Warning: this installer targets Debian 13; detected ${ID:-unknown}."
	fi
}

check_ram() {
	local free_mb
	free_mb=$(awk '/MemAvailable/ {print int($2/1024)}' /proc/meminfo)
	if [ "${free_mb:-0}" -lt 700 ]; then
		red "Warning: only ${free_mb}MB RAM available. vpanel + 2 cores want ~500MB."
		ensure_swap
	fi
}

ensure_swap() {
	if ! swapon --show | grep -q .; then
		inf "Creating a 1G swapfile as a safety net..."
		fallocate -l 1G /swapfile 2>/dev/null || dd if=/dev/zero of=/swapfile bs=1M count=1024
		chmod 600 /swapfile && mkswap /swapfile && swapon /swapfile
		grep -q '/swapfile' /etc/fstab || echo '/swapfile none swap sw 0 0' >> /etc/fstab
	fi
}

install_docker() {
	if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
		grn "Docker already installed."
		return
	fi
	inf "Installing Docker Engine + compose plugin..."
	apt-get update -y
	apt-get install -y ca-certificates curl git
	install -m 0755 -d /etc/apt/keyrings
	curl -fsSL https://download.docker.com/linux/debian/gpg -o /etc/apt/keyrings/docker.asc
	chmod a+r /etc/apt/keyrings/docker.asc
	echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] \
https://download.docker.com/linux/debian $(. /etc/os-release && echo "$VERSION_CODENAME") stable" \
		> /etc/apt/sources.list.d/docker.list
	apt-get update -y
	apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
	systemctl enable --now docker
}

fetch_repo() {
	if [ -d "$INSTALL_DIR/.git" ]; then
		inf "Updating existing checkout..."
		git -C "$INSTALL_DIR" pull --ff-only
	else
		inf "Cloning repository..."
		git clone --depth 1 "$REPO_URL" "$INSTALL_DIR"
	fi
}

gen_env() {
	local env_file="$INSTALL_DIR/deploy/.env"
	[ -f "$env_file" ] && { grn "Reusing existing .env"; return; }

	read -rp "Panel domain (e.g. panel.example.com): " DOMAIN
	read -rp "Admin username [admin]: " ADMIN_USER
	ADMIN_USER=${ADMIN_USER:-admin}
	read -rsp "Admin password: " ADMIN_PASS; echo

	JWT_SECRET=$(head -c 32 /dev/urandom | xxd -p -c 256)

	cat > "$env_file" <<EOF
PANEL_DOMAIN=${DOMAIN}
VPANEL_DB=sqlite
VPANEL_JWT_SECRET=${JWT_SECRET}
VPANEL_SUB_BASE_URL=https://${DOMAIN}
EOF
	chmod 600 "$env_file"
	# Stash admin creds for the post-up seeding step.
	echo "${ADMIN_USER}|${ADMIN_PASS}" > "$INSTALL_DIR/deploy/.admin-seed"
	chmod 600 "$INSTALL_DIR/deploy/.admin-seed"
}

bring_up() {
	cd "$INSTALL_DIR/deploy"
	inf "Building and starting the stack (first build may take a few minutes)..."
	docker compose up -d --build

	# Seed the first sudo admin via the vpanel CLI inside the container.
	if [ -f .admin-seed ]; then
		IFS='|' read -r U P < .admin-seed
		docker compose exec -T vpanel vpanel admin create -u "$U" -p "$P" -role sudo || true
		rm -f .admin-seed
	fi
}

uninstall() {
	if [ -d "$INSTALL_DIR/deploy" ]; then
		cd "$INSTALL_DIR/deploy" && docker compose down -v || true
	fi
	read -rp "Remove $INSTALL_DIR entirely? [y/N] " yn
	[ "${yn:-N}" = "y" ] && rm -rf "$INSTALL_DIR"
	grn "Uninstalled."
}

main() {
	require_root
	if [ "${1:-}" = "--uninstall" ]; then uninstall; exit 0; fi

	check_os
	check_ram
	install_docker
	fetch_repo
	gen_env
	bring_up

	local domain
	domain=$(grep PANEL_DOMAIN "$INSTALL_DIR/deploy/.env" | cut -d= -f2)
	grn "Done!"
	inf "Panel:        https://${domain}"
	inf "Subscriptions: https://${domain}/sub/<token>"
}

main "$@"
