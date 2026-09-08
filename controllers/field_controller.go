package controllers

import (
	"Booking-Lapangan/dto"
	"Booking-Lapangan/models"
	"Booking-Lapangan/repository"
	"context"
	"errors"
	"strings"
)

// FieldControllerInterface mendefinisikan kontrak untuk operasi yang melibatkan field.
type FieldControllerInterface interface {
	List(ctx context.Context)
	Show(ctx context.Context, id uint) models.Field
	Create(ctx context.Context, field *models.Field) error
	Update(ctx context.Context, field *models.Field) error
	Delete(ctx context.Context, id uint) error
}

// FieldController mengimplementasikan FieldControllerInterface.
type FieldController struct {
	Repository repository.FieldRepository
}

// NewFieldController membuat instance baru dari FieldController.
func NewFieldController(repository repository.FieldRepository) FieldControllerInterface {
	return &FieldController{Repository: repository}
}

// List mengambil semua field.
func (c *FieldController) List(ctx context.Context, query dto.FieldFilter) ([]models.Field, error) {
	query.PaginationQuery = query.PaginationQuery.Normalize()

	if query.PriceMin != nil && *query.PriceMin > *query.PriceMax {
		return []models.Field{}, errors.New("price min must be less than price max")
	}

	filter := repository.FieldFilter{
		ListParams: query.ListParams,
		Type:       query.Type,
		Status:     query.Status,
		Location:   query.Location,
		Search:     query.Search,
		PriceMax:   query.PriceMax,
		PriceMin:   query.PriceMin,
	}

	fields, total, err := c.Repository.Paginate(filter)
	if err != nil {
		return nil, 0, err
	}
	return dto.NewFieldResponses(fields), total, nil
}

// show menampilkan detail lapangan
func (c *FieldController) detail(ctx context.Context, id uint) (dto.FieldResponse, error) {
	field, err := c.Repository.FindDetailByID(id)
	if err != nil {
		return nil, err
	}
	response := dto.NewFieldResponse(field)
	return response, nil
}

// Create membuat field baru beserta jadwalnya (admin)
func (c *FieldController) Create(ctx context.Context, req *dto.CreateFieldRequest) (*dto.FieldResponse, error) {
	status := models.FieldStatus(req.Status)
	if status == " " {
		status = models.FieldStatusActive
	}

	field := models.Field{
		Name:        strings.TrimSpace(req.Name),
		Location:    strings.TrimSpace(req.Location),
		Description: strings.TrimSpace(req.Description),
		Price:       req.Price,
		Facilities:  strings.TrimSpace(req.Facilities),
		Status:      status,
	}

	schedules, err := buildScheduleFromRequest(ctx, req.Schedule)
	if err != nil {
		return nil, err
	}

	err = c.createSchedule(ctx, func(r repository.Registry) error {
		if err := r.Schedule().Create(ctx, &field, req.Schedule); err != nil {
			return err
		}
		if len(schedules) == 0 {
			return nil
		}
		for i := 0; i < len(schedules); i++ {
			schedules[i].FieldID = field.ID
		}
		return r.Schedule().CreateMulti(ctx, schedules)
	})
	if err != nil {
		return nil, err
	}

	return dto.NewFieldResponse(field), nil
}
