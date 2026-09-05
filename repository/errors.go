package repository

import (
	"errors"
	"fmt"

	"Booking-Lapangan/models"

	"gorm.io/gorm"
)

func translate(err error, defaultMessage string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &models.AppError{Code: 404, Message: defaultMessage}
	}
	return &models.AppError{Code: 500, Message: fmt.Sprintf("%s: %v", defaultMessage, err)}
}
