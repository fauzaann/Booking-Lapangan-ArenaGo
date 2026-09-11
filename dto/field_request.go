package dto

import (
	"strings"

	"Booking-Lapangan/models"
)

// CreateFieldRequest adalah payload pembuatan lapangan (admin).
type CreateFieldRequest struct {
	Name         string            `json:"name" validate:"required,min=3,max=120"`
	Description  string            `json:"description" validate:"max=1000"`
	Type         string            `json:"type" validate:"required,oneof=PADEL"`
	Location     string            `json:"location" validate:"required,min=3,max=200"`
	PricePerHour float64           `json:"price_per_hour" validate:"required,gt=0"`
	ImageURL     string            `json:"image_url" validate:"omitempty,url,max=500"`
	Status       string            `json:"status" validate:"omitempty,oneof=ACTIVE INACTIVE MAINTENANCE"`
	Facilities   []string          `json:"facilities" validate:"omitempty,dive,max=50"`
	Schedules    []ScheduleRequest `json:"schedules" validate:"omitempty,dive"`
}

// UpdateFieldRequest adalah payload perubahan lapangan (admin).
// Field bertipe pointer agar perubahan parsial dapat dibedakan dari nilai kosong.
type UpdateFieldRequest struct {
	Name         *string   `json:"name" validate:"omitempty,min=3,max=120"`
	Description  *string   `json:"description" validate:"omitempty,max=1000"`
	Type         *string   `json:"type" validate:"omitempty,oneof=PADEL"`
	Location     *string   `json:"location" validate:"omitempty,min=3,max=200"`
	PricePerHour *float64  `json:"price_per_hour" validate:"omitempty,gt=0"`
	ImageURL     *string   `json:"image_url" validate:"omitempty,url,max=500"`
	Status       *string   `json:"status" validate:"omitempty,oneof=ACTIVE INACTIVE MAINTENANCE"`
	Facilities   *[]string `json:"facilities" validate:"omitempty,dive,max=50"`
}

// FieldFilterQuery adalah query string pencarian lapangan.
type FieldFilterQuery struct {
	PaginationQuery
	Type     string   `form:"type"`
	Status   string   `form:"status"`
	Location string   `form:"location"`
	Search   string   `form:"search"`
	MinPrice *float64 `form:"min_price"`
	MaxPrice *float64 `form:"max_price"`
}

// ScheduleRequest adalah payload jam operasional per hari.
type ScheduleRequest struct {
	Day       string `json:"day" validate:"required,oneof=MONDAY TUESDAY WEDNESDAY THURSDAY FRIDAY SATURDAY SUNDAY"`
	OpenTime  string `json:"open_time" validate:"required,clock"`
	CloseTime string `json:"close_time" validate:"required,clock"`
	IsClosed  bool   `json:"is_closed"`
}

// UpsertScheduleRequest adalah payload pengaturan jadwal sebuah lapangan.
type UpsertScheduleRequest struct {
	Schedules []ScheduleRequest `json:"schedules" validate:"required,min=1,dive"`
}

// ScheduleResponse adalah representasi jadwal untuk client.
type ScheduleResponse struct {
	ID        uint   `json:"id"`
	FieldID   uint   `json:"field_id"`
	Day       string `json:"day"`
	OpenTime  string `json:"open_time"`
	CloseTime string `json:"close_time"`
	IsClosed  bool   `json:"is_closed"`
}

// FieldResponse adalah representasi lapangan untuk client.
type FieldResponse struct {
	ID           uint               `json:"id"`
	Name         string             `json:"name"`
	Description  string             `json:"description"`
	Type         string             `json:"type"`
	Location     string             `json:"location"`
	PricePerHour float64            `json:"price_per_hour"`
	ImageURL     string             `json:"image_url,omitempty"`
	Facilities   []string           `json:"facilities"`
	Status       string             `json:"status"`
	Schedules    []ScheduleResponse `json:"schedules,omitempty"`
}

// SlotResponse adalah satu slot per jam pada endpoint availability.
type SlotResponse struct {
	StartTime string  `json:"start_time"`
	EndTime   string  `json:"end_time"`
	Price     float64 `json:"price"`
	Available bool    `json:"available"`
	Reason    string  `json:"reason,omitempty"`
}

// AvailabilityResponse adalah ketersediaan lapangan pada satu tanggal.
type AvailabilityResponse struct {
	FieldID      uint           `json:"field_id"`
	FieldName    string         `json:"field_name"`
	Date         string         `json:"date"`
	Day          string         `json:"day"`
	IsOpen       bool           `json:"is_open"`
	OpenTime     string         `json:"open_time,omitempty"`
	CloseTime    string         `json:"close_time,omitempty"`
	PricePerHour float64        `json:"price_per_hour"`
	Slots        []SlotResponse `json:"slots"`
}

// EncodeFacilities menggabungkan daftar fasilitas menjadi satu kolom teks.
func EncodeFacilities(facilities []string) string {
	cleaned := make([]string, 0, len(facilities))
	for _, item := range facilities {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}
	return strings.Join(cleaned, ",")
}

// DecodeFacilities memecah kolom teks fasilitas menjadi slice.
func DecodeFacilities(value string) []string {
	if strings.TrimSpace(value) == "" {
		return []string{}
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// NewScheduleResponse memetakan model jadwal menjadi response.
func NewScheduleResponse(schedule models.Schedule) ScheduleResponse {
	return ScheduleResponse{
		ID:        schedule.ID,
		FieldID:   schedule.FieldID,
		Day:       string(schedule.Day),
		OpenTime:  schedule.OpenTime,
		CloseTime: schedule.CloseTime,
		IsClosed:  schedule.IsClosed,
	}
}

// NewScheduleResponses memetakan daftar jadwal menjadi daftar response.
func NewScheduleResponses(schedules []models.Schedule) []ScheduleResponse {
	result := make([]ScheduleResponse, 0, len(schedules))
	for _, schedule := range schedules {
		result = append(result, NewScheduleResponse(schedule))
	}
	return result
}

// NewFieldResponse memetakan model lapangan menjadi response.
func NewFieldResponse(field models.Field) FieldResponse {
	response := FieldResponse{
		ID:           field.ID,
		Name:         field.Name,
		Description:  field.Description,
		Type:         string(field.Type),
		Location:     field.Location,
		PricePerHour: field.PricePerHour,
		ImageURL:     field.ImageURL,
		Facilities:   DecodeFacilities(field.Facilities),
		Status:       string(field.Status),
	}
	if len(field.Schedules) > 0 {
		response.Schedules = NewScheduleResponses(field.Schedules)
	}
	return response
}

// NewFieldResponses memetakan daftar lapangan menjadi daftar response.
func NewFieldResponses(fields []models.Field) []FieldResponse {
	result := make([]FieldResponse, 0, len(fields))
	for _, field := range fields {
		result = append(result, NewFieldResponse(field))
	}
	return result
}
