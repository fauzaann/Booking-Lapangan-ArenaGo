package controllers

import "testing"

func TestIsBookingTopic(t *testing.T) {
	tests := []struct {
		name    string
		message string
		want    bool
	}{
		{name: "booking request", message: "Saya mau booking besok jam 7", want: true},
		{name: "court price", message: "Berapa harga lapangan?", want: true},
		{name: "unrelated topic", message: "Siapa presiden Indonesia?", want: false},
		{name: "coding question", message: "Bagaimana cara membuat aplikasi?", want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := isBookingTopic(test.message); got != test.want {
				t.Fatalf("isBookingTopic(%q) = %v, want %v", test.message, got, test.want)
			}
		})
	}
}
