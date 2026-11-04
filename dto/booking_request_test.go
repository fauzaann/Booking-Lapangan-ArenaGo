package dto

import (
	"testing"

	"Booking-Lapangan/models"
)

func TestNewBookingResponseIncludesPlayingPriceBreakdown(t *testing.T) {
	booking := models.Booking{
		Duration:        3,
		OriginalPrice:   300000,
		DiscountAmount:  30000,
		DiscountPercent: 10,
		PackageName:     "Paket 3 Jam+",
		TotalPrice:      270000,
	}

	response := NewBookingResponse(booking)
	if response.OriginalPrice != 300000 ||
		response.DiscountAmount != 30000 ||
		response.DiscountPercent != 10 ||
		response.PackageName != "Paket 3 Jam+" ||
		response.TotalPrice != 270000 {
		t.Fatalf("unexpected price breakdown: %+v", response)
	}
}

func TestNewBookingResponseUsesTotalAsOriginalForLegacyBooking(t *testing.T) {
	response := NewBookingResponse(models.Booking{TotalPrice: 200000})
	if response.OriginalPrice != 200000 {
		t.Fatalf("OriginalPrice = %v, want 200000", response.OriginalPrice)
	}
}
