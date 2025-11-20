package models

import "time"

type Order struct {
	ID            uint64    `gorm:"primaryKey" json:"id"`
	OrderNo       string    `gorm:"unique;size:50" json:"order_no"`
	CustomerID    uint64    `json:"customer_id"`
	PartnerID     *uint64   `json:"partner_id"` // Pointer karena bisa NULL
	PatientID     uint64    `json:"patient_id"`
	ServiceID     uint      `json:"service_id"`
	TotalAmount   float64   `json:"total_amount"`
	Status        string    `json:"status"` // PENDING_PAYMENT, PAID, etc
	ScheduleStart time.Time `json:"schedule_start"`
	ScheduleEnd   time.Time `json:"schedule_end"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`

	// Relasi (Preload) biar pas query datanya lengkap
	Service Service `gorm:"foreignKey:ServiceID" json:"service"`
	Patient Patient `gorm:"foreignKey:PatientID" json:"patient"`
	Partner *User   `gorm:"foreignKey:PartnerID" json:"partner,omitempty"` // Ambil nama mitra dr tabel user
}

type CreateOrderInput struct {
	PatientID     uint64    `json:"patient_id" binding:"required"`
	ServiceID     uint      `json:"service_id" binding:"required"`
	ScheduleStart time.Time `json:"schedule_start" binding:"required"` // Format: 2025-11-20T08:00:00Z
	DurationHours int       `json:"duration_hours" binding:"required"` // Berapa jam/shift
}
