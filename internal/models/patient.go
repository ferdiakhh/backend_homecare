package models

import "time"

type Patient struct {
	ID             uint64 `gorm:"primaryKey" json:"id"`
	CustomerID     uint64 `gorm:"not null" json:"customer_id"`
	Name           string `gorm:"size:100;not null" json:"name"`
	DOB            string `gorm:"type:date" json:"dob"` // Format YYYY-MM-DD
	Gender         string `gorm:"type:enum('L','P')" json:"gender"`
	Weight         int    `json:"weight"`
	MedicalHistory string `gorm:"type:text" json:"medical_history"`
	AddressDetail  string `gorm:"type:text" json:"address_detail"`
	// Use enough integer digits so longitudes like 106.8 fit.
	// decimal(M,D) where M = total digits, D = decimals. M-D must be >= 3 for longitudes up to 180.
	Lat       float64   `gorm:"type:decimal(11,8)" json:"lat"`
	Lng       float64   `gorm:"type:decimal(11,8)" json:"lng"`
	CreatedAt time.Time `json:"created_at"`
}

type CreatePatientInput struct {
	Name           string  `json:"name" binding:"required"`
	DOB            string  `json:"dob" binding:"required"`
	Gender         string  `json:"gender" binding:"required,oneof=L P"`
	Weight         int     `json:"weight" binding:"required"`
	MedicalHistory string  `json:"medical_history"`
	AddressDetail  string  `json:"address_detail" binding:"required"`
	Lat            float64 `json:"lat"`
	Lng            float64 `json:"lng"`
}
