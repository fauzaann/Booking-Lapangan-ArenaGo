package dto

import "strings"

//CreateFieldRequest untuk payload pembuatan lapangan(admin)
type CreateFieldRequest struct {
	Name        string `json:"name" binding:"required"`
	Location    string `json:"location" binding:"required"`
	Description string `json:"description"`
	Price       int    `json:"price" binding:"required"`
	Facilities   string `json:"facilities"`
	Status string `json:"status" binding:"required"`
}	

//UpdateFieldRequest untuk payload update lapangan(admin)
type UpdateFieldRequest struct {
	Name        *string `json:"name" binding:"omitempty"`
	Location    *string `json:"location" binding:"omitempty"`
	Description *string `json:"description" binding:"omitempty"`
	Price       *int    `json:"price" binding:"omitempty"`
	Facilities   *string `json:"facilities" binding:"omitempty"`
	Status string `json:"status" binding:"omitempty"`
}

//FieldFilter untuk payload pencarian lapangan(user/admin)
type FieldFilter struct{
	PaginationQuery
	Type string `form:"type" binding:"omitempty"`
	Status string `form:"status" binding:"omitempty"`
	Location string `form:"location" binding:"omitempty"`
	Search string `form:"search" binding:"omitempty"`
	PriceMax *int `form:"price_max" binding:"omitempty"`
	PriceMin *int `form:"price_min" binding:"omitempty"`	
}

//ScheduleRequest untuk payload pembuatan jadwal
type ScheduleRequest struct{
	Day string `json:"day" binding:"required, oneof=senin selasa rabu kamis jumat sabtu minggu"`
	OpenTime string `json:"open_time" binding:"required"`
	CloseTime string `json:"close_time" binding:"required"`
	IsClosed bool `json:"is_closed" binding:"required"`
}

//UpdateScheduleRequest untuk payload update jadwal
type UpdateScheduleRequest struct{
	ScheduleRequest []*ScheduleRequest `json:"schedule" binding:"omitempty"`
}

//ScheduleResponse untuk payload menampilkan jadwal
type ScheduleResponse struct {
	ID int `json:"id"`
	FieldID int `json:"field_id"`
	Day string `json:"day"`
	OpenTime string `json:"open_time"`
	CloseTime string `json:"close_time"`
	IsClosed bool `json:"is_closed"`
}	

//FieldResponse untuk payload menampilkan lapangan
type FieldResponse struct {
	ID int `json:"id"`
	Name string `json:"name"`
	Location string `json:"location"`
	Description string `json:"description"`
	Price int `json:"price"`
	Facilities []string `json:"facilities"`
	Status string `json:"status"`
	Schedule []ScheduleResponse `json:"schedule"`
}	

//SlotResponse untuk payload menampilkan slot satu jam 
type SlotResponse struct {
	StartTime string `json:"start_time"`
	EndTime string `json:"end_time"`
	IsAvailable bool `json:"is_available"`
	Reason string `json:"reason"`
}

//AvailableResponse untuk payload menampilkan lapangan yang tersedia pada satu tanggal
type AvailableResponse struct {
	FieldID int `json:"field_id"`
	FieldName string `json:"field_name"`
	Date string `json:"date"`
	Day string `json:"day"`
	Price int `json:"price"`
	OpenTime string `json:"open_time"`
	CloseTime string `json:"close_time"`	
	Slots []SlotResponse `json:"slots"`
}

//EncodeFasilities untuk menggabungkan fasilitas menjadi satu kolom teks
func EncodeFasilities(facilities string) []string {
	cleaned := strings.TrimSpace(facilities)
	for _, v := range facilities{
		if trimmed := strings.TrimSpace(v); trimeed != ""{
			cleaned = append(cleaned, trimmed)
		}
	}
	return strings.Join(cleaned, ",")
}

//DecodeFasilities untuk memecah fasilitas menjadi satu kolom teks
func DecodeFasilities(value string) []string {
	if strings.TrimSpace(value) == ""{
		return []string{}
	}
	values := strings.Split(value, ",")
	cleaned := make([]string, 0, len(values))
	for _, v := range values{
		if trimmed := strings.TrimSpace(v); trimmed != ""{
			cleaned = append(cleaned, trimmed)
		}
	}	
	return cleaned
}

//NewScheduleResponse untuk membuat satu schedule baru
func NewScheduleResponse(Day, OpenTime, CloseTime string, IsClosed bool)ScheduleResponse {
	return ScheduleResponse{
		Day: Day,
		OpenTime: OpenTime,
		CloseTime: CloseTime,
		IsClosed: IsClosed,
	}
}

// NewScheduleResponses memetakan daftar jadwal menjadi daftar response.
func NewScheduleResponses(Schedule []models.Schedule) []ScheduleResponse {
	result := make([]ScheduleResponse, 0, len(Schedule))
	for_, schedule := range Schedule{
		result.append(result, NewScheduleResponse(schedule))
	}
	return result
}

// NewFieldResponse memetakan model lapangan menjadi response.
func NewFieldResponse(field models.Field, schedule []ScheduleResponse) FieldResponse {
	return FieldResponse{
		ID: field.ID,
		Name: field.Name,
		Location: field.Location,
		Description: field.Description,
		Price: field.Price,
		Facilities: DecodeFasilities(field.Fasilities),
		Status: field.Status,
		Schedule: schedule,
	}
}

// NewFieldResponses memetakan daftar lapangan menjadi daftar response.
func NewFieldResponses(fields []models.Field, schedule []ScheduleResponse) []FieldResponse {
	result := make([]FieldResponse, 0, len(fields))
	for_, field := range fields{
		result.append(result, NewFieldResponse(field, schedule))
	}
	return result
}



	


	





