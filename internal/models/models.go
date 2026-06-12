// Package models defines the GORM entities that anchor the panel's data model.
// JSON-heavy, protocol-specific settings are stored in TEXT columns via GORM's
// json serializer so the schema stays portable between SQLite and PostgreSQL.
package models

import "time"

// AdminRole enumerates the (deliberately small) admin role set.
type AdminRole string

const (
	RoleSudo  AdminRole = "sudo"  // full control incl. managing admins & settings
	RoleAdmin AdminRole = "admin" // manages own users/inbounds only
)

// Admin is a panel operator account.
type Admin struct {
	ID           int64      `gorm:"primaryKey" json:"id"`
	Username     string     `gorm:"uniqueIndex;size:64;not null" json:"username"`
	PasswordHash string     `gorm:"not null" json:"-"`
	Role         AdminRole  `gorm:"size:16;not null;default:admin" json:"role"`
	IsActive     bool       `gorm:"not null;default:true" json:"is_active"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// UserStatus enumerates the lifecycle states of a VPN client.
type UserStatus string

const (
	StatusActive   UserStatus = "active"
	StatusDisabled UserStatus = "disabled" // manually turned off by an admin
	StatusLimited  UserStatus = "limited"  // hit data limit
	StatusExpired  UserStatus = "expired"  // passed expiry date
)

// ResetStrategy controls how a user's used traffic counters are periodically reset.
type ResetStrategy string

const (
	ResetNone  ResetStrategy = "no_reset"
	ResetDay   ResetStrategy = "day"
	ResetWeek  ResetStrategy = "week"
	ResetMonth ResetStrategy = "month"
)

// User is a VPN client provisioned across one or more inbounds.
type User struct {
	ID                int64         `gorm:"primaryKey" json:"id"`
	Username          string        `gorm:"uniqueIndex;size:64;not null" json:"username"`
	UUID              string        `gorm:"size:64;index" json:"uuid"` // VLESS/VMess
	Password          string        `gorm:"size:128" json:"password"`  // Trojan/Shadowsocks secret
	SubscriptionToken string        `gorm:"uniqueIndex;size:64;not null" json:"subscription_token"`
	Status            UserStatus    `gorm:"size:16;not null;default:active;index" json:"status"`
	DataLimit         int64         `gorm:"not null;default:0" json:"data_limit"` // bytes, 0 = unlimited
	UsedUp            int64         `gorm:"not null;default:0" json:"used_up"`
	UsedDown          int64         `gorm:"not null;default:0" json:"used_down"`
	ResetStrategy     ResetStrategy `gorm:"size:16;not null;default:no_reset" json:"reset_strategy"`
	LastResetAt       *time.Time    `json:"last_reset_at,omitempty"`
	ExpireAt          *time.Time    `gorm:"index" json:"expire_at,omitempty"`
	OnlineAt          *time.Time    `json:"online_at,omitempty"`
	Note              string        `gorm:"size:512" json:"note"`
	AdminID           int64         `gorm:"index" json:"admin_id"`
	CreatedAt         time.Time     `json:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at"`

	Inbounds []Inbound `gorm:"many2many:user_inbounds;" json:"inbounds,omitempty"`
}

// CoreName identifies which proxy core hosts an inbound.
type CoreName string

const (
	CoreXray    CoreName = "xray"
	CoreSingbox CoreName = "singbox"
)

// Inbound is a listening endpoint served by one of the cores.
type Inbound struct {
	ID       int64    `gorm:"primaryKey" json:"id"`
	Tag      string   `gorm:"uniqueIndex;size:64;not null" json:"tag"`
	Core     CoreName `gorm:"size:16;not null;index" json:"core"`
	Protocol string   `gorm:"size:32;not null" json:"protocol"` // vless, vmess, trojan, shadowsocks, hysteria2, tuic, ...
	Listen   string   `gorm:"size:64;not null;default:0.0.0.0" json:"listen"`
	Port     int      `gorm:"not null" json:"port"`
	Network  string   `gorm:"size:16" json:"network"`  // tcp, ws, grpc, http2 ...
	Security string   `gorm:"size:16" json:"security"` // reality, tls, none

	// Protocol- and transport-specific configuration, stored as JSON.
	Settings map[string]any `gorm:"serializer:json" json:"settings"`
	Stream   map[string]any `gorm:"serializer:json" json:"stream"`

	Enabled   bool      `gorm:"not null;default:true;index" json:"enabled"`
	SortOrder int       `gorm:"not null;default:0" json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Hosts []Host `gorm:"constraint:OnDelete:CASCADE;" json:"hosts,omitempty"`
}

// Host is a subscription-facing advertisement of an inbound, letting a single
// listener be presented under multiple SNIs / domains / paths (3x-ui style).
type Host struct {
	ID            int64  `gorm:"primaryKey" json:"id"`
	InboundID     int64  `gorm:"index;not null" json:"inbound_id"`
	Remark        string `gorm:"size:128" json:"remark"`
	Address       string `gorm:"size:255" json:"address"`
	Port          int    `json:"port"`
	SNI           string `gorm:"size:255" json:"sni"`
	HostHeader    string `gorm:"size:255" json:"host_header"`
	Path          string `gorm:"size:255" json:"path"`
	Fingerprint   string `gorm:"size:32" json:"fingerprint"`
	Security      string `gorm:"size:16" json:"security"`
	ALPN          string `gorm:"size:64" json:"alpn"`
	AllowInsecure bool   `json:"allow_insecure"`
	SortOrder     int    `gorm:"not null;default:0" json:"sort_order"`
}

// UserInbound is the join row for the User<->Inbound many-to-many relation.
// Declared explicitly so we can add columns (e.g. per-inbound flow) later.
type UserInbound struct {
	UserID    int64 `gorm:"primaryKey" json:"user_id"`
	InboundID int64 `gorm:"primaryKey" json:"inbound_id"`
}

// TrafficLog stores coarse hourly traffic buckets for charts. Pruned to a
// rolling window by a scheduled job to bound table growth.
type TrafficLog struct {
	ID       int64     `gorm:"primaryKey" json:"id"`
	UserID   *int64    `gorm:"index" json:"user_id,omitempty"` // nil = system total
	BucketTS time.Time `gorm:"index" json:"bucket_ts"`         // truncated to the hour
	Up       int64     `json:"up"`
	Down     int64     `json:"down"`
}

// Setting is a singleton-style key/value store for global configuration.
type Setting struct {
	Key   string `gorm:"primaryKey;size:64" json:"key"`
	Value string `gorm:"type:text" json:"value"` // JSON-encoded value
}

// SubTarget identifies a subscription output format.
type SubTarget string

const (
	TargetV2ray     SubTarget = "v2ray"
	TargetClash     SubTarget = "clash"
	TargetClashMeta SubTarget = "clashmeta"
	TargetSingbox   SubTarget = "singbox"
	TargetCustom    SubTarget = "custom"
)

// SubscriptionTemplate is an admin-overridable Go-template for a sub format.
type SubscriptionTemplate struct {
	ID        int64     `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:64;not null" json:"name"`
	Target    SubTarget `gorm:"size:16;not null" json:"target"`
	Body      string    `gorm:"type:text" json:"body"`
	IsDefault bool      `gorm:"not null;default:false" json:"is_default"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AllModels returns every entity for auto-migration.
func AllModels() []any {
	return []any{
		&Admin{},
		&User{},
		&Inbound{},
		&Host{},
		&UserInbound{},
		&TrafficLog{},
		&Setting{},
		&SubscriptionTemplate{},
	}
}
