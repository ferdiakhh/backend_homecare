package models

import (
	"encoding/json" // <--- JANGAN LUPA IMPORT INI
	"time"
)

type CareJournal struct {
	ID       uint64    `gorm:"primaryKey" json:"id"`
	OrderID  uint64    `gorm:"not null" json:"order_id"`
	LoggedAt time.Time `gorm:"autoCreateTime" json:"logged_at"`

	// PERBAIKAN DI SINI:
	// Ganti []byte menjadi json.RawMessage
	VitalsData json.RawMessage `gorm:"type:json" json:"vitals_data"`

	Notes    string `gorm:"type:text" json:"notes"`
	PhotoURL string `gorm:"size:255" json:"photo_url"`

	// Relasi
	Order *Order `gorm:"foreignKey:OrderID" json:"order,omitempty"`
}

// Helper Struct (Tidak berubah)
type VitalsDetail struct {
	Tensi string `json:"tensi"`
	Suhu  string `json:"suhu"`
	Nadi  string `json:"nadi"`
}
