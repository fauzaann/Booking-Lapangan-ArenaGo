package controllers_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"Booking-Lapangan/dto"
)

func TestPublicFieldBookingDiscoveryFlow(t *testing.T) {
	response, body := requestAPI(t, http.MethodGet, "/api/v1/fields?page=1&limit=10", nil)
	envelope := requireSuccessResponse(t, response, body, http.StatusOK)

	var fields []dto.FieldResponse
	if err := json.Unmarshal(envelope.Data, &fields); err != nil {
		t.Fatalf("decode fields response: %v", err)
	}
	if len(fields) == 0 {
		t.Skip("no fields are configured; list endpoint is healthy, but detail and availability cannot be exercised")
	}

	var selected *dto.FieldResponse
	for i := range fields {
		if fields[i].ID != 0 && fields[i].Status == "ACTIVE" {
			selected = &fields[i]
			break
		}
	}
	if selected == nil {
		t.Skip("no active field is configured; detail and availability cannot be exercised")
	}
	if selected.PricePerHour <= 0 {
		t.Fatalf("active field %d has invalid hourly price %v", selected.ID, selected.PricePerHour)
	}

	t.Run("field details", func(t *testing.T) {
		response, body := requestAPI(t, http.MethodGet, fieldPath(selected.ID, ""), nil)
		envelope := requireSuccessResponse(t, response, body, http.StatusOK)

		var detail dto.FieldResponse
		if err := json.Unmarshal(envelope.Data, &detail); err != nil {
			t.Fatalf("decode field detail: %v", err)
		}
		if detail.ID != selected.ID || detail.Name == "" || detail.Location == "" {
			t.Fatalf("field detail is incomplete or inconsistent: %+v", detail)
		}
	})

	t.Run("weekly operating hours", func(t *testing.T) {
		response, body := requestAPI(t, http.MethodGet, fieldPath(selected.ID, "/schedules"), nil)
		envelope := requireSuccessResponse(t, response, body, http.StatusOK)

		var schedules []dto.ScheduleResponse
		if err := json.Unmarshal(envelope.Data, &schedules); err != nil {
			t.Fatalf("decode field schedules: %v", err)
		}
		for _, schedule := range schedules {
			if schedule.FieldID != selected.ID {
				t.Fatalf("schedule belongs to field %d, want %d", schedule.FieldID, selected.ID)
			}
			if !schedule.IsClosed && (schedule.OpenTime == "" || schedule.CloseTime == "") {
				t.Fatalf("open schedule has incomplete hours: %+v", schedule)
			}
		}
	})

	t.Run("future availability", func(t *testing.T) {
		date := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
		path := fmt.Sprintf("%s/availability?date=%s", fieldPath(selected.ID, ""), date)
		response, body := requestAPI(t, http.MethodGet, path, nil)
		envelope := requireSuccessResponse(t, response, body, http.StatusOK)

		var availability dto.AvailabilityResponse
		if err := json.Unmarshal(envelope.Data, &availability); err != nil {
			t.Fatalf("decode availability response: %v", err)
		}
		if availability.FieldID != selected.ID || availability.Date != date {
			t.Fatalf("availability does not match the requested field/date: %+v", availability)
		}
		if availability.PricePerHour != selected.PricePerHour {
			t.Fatalf("availability price = %v, field price = %v", availability.PricePerHour, selected.PricePerHour)
		}
		for _, slot := range availability.Slots {
			if slot.StartTime == "" || slot.EndTime == "" || slot.Price <= 0 {
				t.Fatalf("availability contains an incomplete slot: %+v", slot)
			}
		}
	})
}
