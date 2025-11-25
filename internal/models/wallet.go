package models

import "time"

type Wallet struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	UserID    uint64    `gorm:"unique;not null" json:"user_id"`
	Balance   float64   `gorm:"default:0" json:"balance"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relasi ke History Transaksi
	Transactions []WalletTransaction `gorm:"foreignKey:WalletID" json:"transactions,omitempty"`
}

type WalletTransaction struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	WalletID  uint64    `json:"wallet_id"`
	OrderID   *uint64   `json:"order_id,omitempty"` // Bisa null kalau Withdrawal
	Amount    float64   `json:"amount"`
	Type      string    `json:"type"`   // INCOME, WITHDRAWAL
	Status    string    `json:"status"` // PENDING, SUCCESS, FAILED
	CreatedAt time.Time `json:"created_at"`
}
