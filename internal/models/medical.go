package models

import "time"

type CareJournal struct {
	ID       uint64    `gorm:"primaryKey" json:"id"`
	OrderID  uint64    `gorm:"not null" json:"order_id"`
	LoggedAt time.Time `gorm:"autoCreateTime" json:"logged_at"`

	// Disimpan sebagai JSON blob di database
	// GORM akan mapping []byte ke tipe kolom JSON/TEXT di MySQL
	VitalsData []byte `gorm:"type:json" json:"-"`

	Notes    string `gorm:"type:text" json:"notes"`
	PhotoURL string `gorm:"size:255" json:"photo_url"`

	// Relasi (Opsional, kalau mau preload Order)
	Order *Order `gorm:"foreignKey:OrderID" json:"order,omitempty"`
}

// Helper Struct untuk decoding JSON Vitals saat di-read API
// Ini tidak masuk ke DB, cuma buat formatting response JSON
type VitalsDetail struct {
	Tensi string `json:"tensi"`
	Suhu  string `json:"suhu"`
	Nadi  string `json:"nadi"`
}
