package models

import "testing"

func TestCalculatePlayingPrice(t *testing.T) {
	tests := []struct {
		name            string
		pricePerHour    float64
		duration        int
		originalPrice   float64
		discountAmount  float64
		discountPercent float64
		packageName     string
		totalPrice      float64
	}{
		{
			name:         "one hour has no discount",
			pricePerHour: 100000, duration: 1,
			originalPrice: 100000, totalPrice: 100000,
		},
		{
			name:         "two hours receives five percent discount",
			pricePerHour: 100000, duration: 2,
			originalPrice: 200000, discountAmount: 10000,
			discountPercent: 5, packageName: "Paket 2 Jam", totalPrice: 190000,
		},
		{
			name:         "three hours receives ten percent discount",
			pricePerHour: 100000, duration: 3,
			originalPrice: 300000, discountAmount: 30000,
			discountPercent: 10, packageName: "Paket 3 Jam+", totalPrice: 270000,
		},
		{
			name:         "longer duration keeps the three hour package",
			pricePerHour: 100000, duration: 4,
			originalPrice: 400000, discountAmount: 40000,
			discountPercent: 10, packageName: "Paket 3 Jam+", totalPrice: 360000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculatePlayingPrice(tt.pricePerHour, tt.duration)
			if got.OriginalPrice != tt.originalPrice ||
				got.DiscountAmount != tt.discountAmount ||
				got.DiscountPercent != tt.discountPercent ||
				got.PackageName != tt.packageName ||
				got.TotalPrice != tt.totalPrice {
				t.Fatalf("CalculatePlayingPrice() = %+v", got)
			}
		})
	}
}
