package models

type PartnerProfile struct {
	ID              uint64  `gorm:"primaryKey" json:"id"`
	UserID          uint64  `gorm:"not null" json:"user_id"`
	STRNumber       string  `gorm:"size:100" json:"str_number"`
	ExperienceYears int     `json:"experience_years"`
	VideoIntroURL   string  `gorm:"size:255" json:"video_intro_url"` // Link YouTube/Drive
	BioDescription  string  `gorm:"type:text" json:"bio_description"`
	RatingAvg       float64 `gorm:"default:0" json:"rating_avg"`
	// Ensure enough integer digits for longitudes (up to ±180)
	CurrentLat float64 `gorm:"type:decimal(11,8)" json:"current_lat"`
	CurrentLng float64 `gorm:"type:decimal(11,8)" json:"current_lng"`
	IsActive   bool    `gorm:"default:false" json:"is_active"`
	User       User    `gorm:"foreignKey:UserID" json:"user_data,omitempty"`
}

// Struct inputan dari Mitra saat update profil
type UpdateProfileInput struct {
	STRNumber       string  `json:"str_number" binding:"required"`
	ExperienceYears int     `json:"experience_years" binding:"required"`
	VideoIntroURL   string  `json:"video_intro_url" binding:"required,url"` // Validasi URL
	BioDescription  string  `json:"bio_description"`
	CurrentLat      float64 `json:"current_lat"`
	CurrentLng      float64 `json:"current_lng"`
}
