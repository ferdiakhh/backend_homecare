package models

import (
	"time"

	"gorm.io/gorm"
)

// User merepresentasikan tabel 'users' di database
type User struct {
	ID           uint64         `gorm:"primaryKey" json:"id"`
	RoleID       uint           `gorm:"not null" json:"role_id"`
	FullName     string         `gorm:"size:100;not null" json:"full_name"`
	Email        string         `gorm:"uniqueIndex;size:100;not null" json:"email"`
	PasswordHash string         `gorm:"not null" json:"-"` // json:"-" artinya field ini TIDAK AKAN dikirim balik ke frontend (rahasia)
	Phone        string         `gorm:"column:phone_number;size:20;unique" json:"phone"`
	IsVerified   bool           `gorm:"default:false" json:"is_verified"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`

	// Tambahkan Relasi ini (Has Many)
	Patients []Patient `gorm:"foreignKey:CustomerID" json:"patients,omitempty"`
}

// Struct untuk menangkap Input Register dari user
type RegisterInput struct {
	FullName string `json:"full_name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	RoleID   uint   `json:"role_id" binding:"required"` // 1:Admin, 2:Finance, 3:Mitra, 4:Customer
	Phone    string `json:"phone" binding:"required"`
}

// Struct untuk menangkap Input Login
type LoginInput struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}
