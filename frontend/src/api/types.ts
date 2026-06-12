export interface Inbound {
  id: number;
  tag: string;
  core: "xray" | "singbox";
  protocol: string;
  listen: string;
  port: number;
  network: string;
  security: string;
  settings: Record<string, unknown>;
  stream: Record<string, unknown>;
  enabled: boolean;
}

export interface User {
  id: number;
  username: string;
  uuid: string;
  password: string;
  subscription_token: string;
  status: string;
  data_limit: number;
  used_up: number;
  used_down: number;
  expire_at: string | null;
  inbounds?: Inbound[];
}

export interface CoreStatus {
  name: string;
  state: string;
  restarts: number;
  inbounds: number;
}

// Protocols grouped by which core serves them.
export const PROTOCOLS: Record<"xray" | "singbox", string[]> = {
  xray: ["vless", "vmess", "trojan", "shadowsocks"],
  singbox: ["hysteria2", "tuic", "vless", "vmess", "trojan", "shadowsocks", "anytls", "naive"],
};
