package dto

import (
	"Booking-Lapangan/models"
	"strings"
)

// CreateFieldRequest untuk payload pembuatan lapangan(admin)
type CreateFieldRequest struct {
	Name        string            `json:"name" binding:"required"`
	Location    string            `json:"location" binding:"required"`
	Description string            `json:"description"`
	Type        string            `json:"type" binding:"required"`
	Price       float64           `json:"price" binding:"required"`
	Facilities  []string          `json:"facilities" validate:"omitempty,dive,max=50"`
	Status      string            `json:"status" binding:"required"`
	Schedules   []ScheduleRequest `json:"schedules" binding:"omitempty"`
}

// UpdateFieldRequest untuk payload update lapangan(admin)
type UpdateFieldRequest struct {
	Name        *string   `json:"name" binding:"omitempty"`
	Location    *string   `json:"location" binding:"omitempty"`
	Description *string   `json:"description" binding:"omitempty"`
	Type        *string   `json:"type" binding:"omitempty"`
	Price       *float64  `json:"price" binding:"omitempty"`
	Facilities  *[]string `json:"facilities" binding:"omitempty"`
	Status      *string   `json:"status" binding:"omitempty,oneof=ACTIVE INACTIVE MAINTENANCE"`
}

// FieldFilter untuk payload pencarian lapangan(user/admin)
type FieldFilter struct {
	PaginationQuery
	Type     string   `form:"type" binding:"omitempty"`
	Status   string   `form:"status" binding:"omitempty"`
	Location string   `form:"location" binding:"omitempty"`
	Search   string   `form:"search" binding:"omitempty"`
	PriceMax *float64 `form:"price_max" binding:"omitempty"`
	PriceMin *float64 `form:"price_min" binding:"omitempty"`
}

// ScheduleRequest untuk payload pembuatan jadwal
type ScheduleRequest struct {
	Day       string `json:"day" binding:"required,oneof=senin selasa rabu kamis jumat sabtu minggu monday tuesday wednesday thursday friday saturday sunday"`
	OpenTime  string `json:"open_time" binding:"required"`
	CloseTime string `json:"close_time" binding:"required"`
	IsClosed  bool   `json:"is_closed"`
}

// UpsertScheduleRequest untuk payload upsert jadwal
type UpsertScheduleRequest struct {
	Schedules []ScheduleRequest `json:"schedules"`
}

// UpdateScheduleRequest untuk payload update jadwal
type UpdateScheduleRequest struct {
	ScheduleRequest []*ScheduleRequest `json:"schedule" binding:"omitempty"`
}

// ScheduleResponse untuk payload menampilkan jadwal
type ScheduleResponse struct {
	ID        int    `json:"id"`
	FieldID   int    `json:"field_id"`
	Day       string `json:"day"`
	OpenTime  string `json:"open_time"`
	CloseTime string `json:"close_time"`
	IsClosed  bool   `json:"is_closed"`
}

// FieldResponse untuk payload menampilkan lapangan
type FieldResponse struct {
	ID          uint               `json:"id"`
	Name        string             `json:"name"`
	Location    string             `json:"location"`
	Description string             `json:"description"`
	Price       float64            `json:"price"`
	Facilities  []string           `json:"facilities"`
	Status      string             `json:"status"`
	Schedule    []ScheduleResponse `json:"schedule"`
}

// SlotResponse untuk payload menampilkan slot satu jam
type SlotResponse struct {
	StartTime   string  `json:"start_time"`
	EndTime     string  `json:"end_time"`
	Price       float64 `json:"price"`
	IsAvailable bool    `json:"is_available"`
	Reason      string  `json:"reason,omitempty"`
}

// AvailableResponse untuk payload menampilkan lapangan yang tersedia pada satu tanggal
type AvailableResponse struct {
	FieldID   uint           `json:"field_id"`
	FieldName string         `json:"field_name"`
	Date      string         `json:"date"`
	Day       string         `json:"day"`
	Price     float64        `json:"price"`
	IsOpen    bool           `json:"is_open"`
	OpenTime  string         `json:"open_time,omitempty"`
	CloseTime string         `json:"close_time,omitempty"`
	Slots     []SlotResponse `json:"slots"`
}

// EncodeFacilities untuk menggabungkan fasilitas menjadi satu kolom teks
func EncodeFacilities(facilities []string) string {
	cleaned := make([]string, 0, len(facilities))
	for _, v := range facilities {
		if trimmed := strings.TrimSpace(v); trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}
	return strings.Join(cleaned, ",")
}

// DecodeFacilities untuk memecah fasilitas menjadi satu kolom teks
func DecodeFacilities(value string) []string {
	if strings.TrimSpace(value) == "" {
		return []string{}
	}
	values := strings.Split(value, ",")
	cleaned := make([]string, 0, len(values))
	for _, v := range values {
		if trimmed := strings.TrimSpace(v); trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}
	return cleaned
}

// NewScheduleResponse untuk membuat satu schedule baru
func NewScheduleResponse(Day, OpenTime, CloseTime string, IsClosed bool) ScheduleResponse {
	return ScheduleResponse{
		Day:       Day,
		OpenTime:  OpenTime,
		CloseTime: CloseTime,
		IsClosed:  IsClosed,
	}
}

// NewScheduleResponses memetakan daftar jadwal menjadi daftar response.
func NewScheduleResponses(Schedule []models.Schedule) []ScheduleResponse {
	result := make([]ScheduleResponse, 0, len(Schedule))
	for _, schedule := range Schedule {
		result = append(result, NewScheduleResponse(string(schedule.Day), schedule.OpenTime, schedule.CloseTime, !schedule.IsActive))
	}
	return result
}

// NewFieldResponse memetakan model lapangan menjadi response.
func NewFieldResponse(field models.Field, schedule []ScheduleResponse) FieldResponse {
	return FieldResponse{
		ID:          field.ID,
		Name:        field.Name,
		Location:    field.Location,
		Description: field.Description,
		Price:       field.Price,
		Facilities:  DecodeFacilities(field.Facilities),
		Status:      string(field.Status),
		Schedule:    schedule,
	}
}

// NewFieldResponses memetakan daftar lapangan menjadi daftar response.
func NewFieldResponses(fields []models.Field, schedule []ScheduleResponse) []FieldResponse {
	result := make([]FieldResponse, 0, len(fields))
	for _, field := range fields {
		result = append(result, NewFieldResponse(field, schedule))
	}
	return result
}
