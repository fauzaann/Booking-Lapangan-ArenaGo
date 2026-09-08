package repository

import "Booking-Lapangan/models"

type FieldFilter struct {
	ListParams
	Type     string `json:"type"`
	Status   string `json:"status"`
	Location string `json:"location"`
	Search   string `json:"search"`
	PriceMax int    `json:"price_max"`
	PriceMin int    `json:"price_min"`
}

type FieldRepository interface {
	Paginate(filter FieldFilter) ([]models.Field, error)
	Count(filter FieldFilter) (int, error)
	FindByID(id uint) (models.Field, error)
	FindDetailByID(id uint) (models.Field, error)
	FindAll(filter FieldFilter) (*models.Field, error)
	Create(field *models.Field) error
	Update(field *models.Field) error
	Delete(id uint) error
	DeleteSoft(id uint) error
}
