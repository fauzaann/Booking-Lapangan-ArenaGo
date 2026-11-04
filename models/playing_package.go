package models

import (
	"math"
	"time"
)

// PlayingPackage defines a duration-based discount tier for bookings.
type PlayingPackage struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	Name            string    `gorm:"type:varchar(100);not null" json:"name"`
	MinimumDuration int       `gorm:"not null" json:"minimum_duration"`
	DiscountPercent float64   `gorm:"type:numeric(5,2);not null" json:"discount_percent"`
	IsActive        bool      `gorm:"not null;default:true;index" json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// PlayingPrice contains the price breakdown for a booking duration.
type PlayingPrice struct {
	OriginalPrice   float64
	DiscountAmount  float64
	DiscountPercent float64
	PackageName     string
	TotalPrice      float64
}

// CalculatePlayingPrice applies the duration-based package discounts.
func CalculatePlayingPrice(pricePerHour float64, duration int) PlayingPrice {
	originalPrice := roundPlayingPrice(pricePerHour * float64(duration))
	price := PlayingPrice{
		OriginalPrice: originalPrice,
		TotalPrice:    originalPrice,
	}

	var packageName string
	var discountPercent float64
	switch {
	case duration >= 3:
		packageName, discountPercent = "Paket 3 Jam+", 10
	case duration == 2:
		packageName, discountPercent = "Paket 2 Jam", 5
	default:
		return price
	}

	price.DiscountPercent = discountPercent
	price.DiscountAmount = roundPlayingPrice(originalPrice * discountPercent / 100)
	price.PackageName = packageName
	price.TotalPrice = roundPlayingPrice(originalPrice - price.DiscountAmount)
	return price
}

func roundPlayingPrice(amount float64) float64 {
	return math.Round(amount*100) / 100
}
