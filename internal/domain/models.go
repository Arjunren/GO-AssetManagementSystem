package domain

import "time"

type Role string

const (
	RoleAdmin     Role = "ADMIN"
	RoleCustodian Role = "CUSTODIAN"
	RoleEmployee  Role = "EMPLOYEE"
)

type AssetStatus string

const (
	StatusAvailable   AssetStatus = "AVAILABLE"
	StatusAssigned    AssetStatus = "ASSIGNED"
	StatusMaintenance AssetStatus = "MAINTENANCE"
	StatusRetired     AssetStatus = "RETIRED"
)

type EventType string

const (
	EventAssigned    EventType = "ASSIGNED"
	EventReturned    EventType = "RETURNED"
	EventMaintenance EventType = "MAINTENANCE"
	EventRetired     EventType = "RETIRED"
	EventAvailable   EventType = "AVAILABLE"
)

type User struct {
	ID           int64     `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Name         string    `json:"name"`
	Role         Role      `json:"role"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
type Asset struct {
	ID           int64       `json:"id"`
	AssetTag     string      `json:"asset_tag"`
	Name         string      `json:"name"`
	Category     string      `json:"category"`
	SerialNumber string      `json:"serial_number"`
	Status       AssetStatus `json:"status"`
	PurchaseDate *time.Time  `json:"purchase_date"`
	PurchaseCost float64     `json:"purchase_cost"`
	AssignedTo   *int64      `json:"assigned_to"`
	AssigneeName string      `json:"assignee_name,omitempty"`
	CreatedBy    int64       `json:"created_by"`
	UpdatedBy    int64       `json:"updated_by"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
}
type AssetEvent struct {
	ID            int64     `json:"id"`
	AssetID       int64     `json:"asset_id"`
	Type          EventType `json:"type"`
	FromUserID    *int64    `json:"from_user_id"`
	ToUserID      *int64    `json:"to_user_id"`
	Notes         string    `json:"notes"`
	PerformedBy   int64     `json:"performed_by"`
	PerformerName string    `json:"performer_name,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}
type Dashboard struct {
	Total       int64   `json:"total"`
	Available   int64   `json:"available"`
	Assigned    int64   `json:"assigned"`
	Maintenance int64   `json:"maintenance"`
	Retired     int64   `json:"retired"`
	TotalValue  float64 `json:"total_value"`
}
type ListParams struct {
	Page, Limit int
	Search      string
	Status      AssetStatus
	Category    string
	ActorID     int64
	ActorRole   Role
}
type Page struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

func NewPage(p ListParams, t int64) Page {
	n := 0
	if t > 0 {
		n = int((t + int64(p.Limit) - 1) / int64(p.Limit))
	}
	return Page{p.Page, p.Limit, t, n}
}
