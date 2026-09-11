package database

import (
	"errors"
	"fmt"
	"log"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"Booking-Lapangan/models"
	"Booking-Lapangan/pkg/password"
)

// Seed mengisi data awal untuk development. Fungsi ini idempotent: menjalankan
// ulang tidak menggandakan data. Password selalu disimpan dalam bentuk bcrypt.
func Seed(db *gorm.DB) error {
	if err := seedUsers(db); err != nil {
		return err
	}
	if err := seedFields(db); err != nil {
		return err
	}
	log.Println("database: seeding completed")
	return nil
}

func seedUsers(db *gorm.DB) error {
	users := []struct {
		Name     string
		Email    string
		Password string
		Phone    string
		Role     models.Role
	}{
		{"Administrator", "admin@example.com", "admin123", "081200000001", models.RoleAdmin},
		{"John Doe", "user@example.com", "user123", "081200000002", models.RoleUser},
	}

	for _, item := range users {
		var count int64
		if err := db.Model(&models.User{}).Where("email = ?", item.Email).Count(&count).Error; err != nil {
			return fmt.Errorf("failed to check seeded user: %w", err)
		}
		if count > 0 {
			continue
		}

		hashed, err := password.Hash(item.Password)
		if err != nil {
			return fmt.Errorf("failed to hash seeded password: %w", err)
		}

		user := models.User{
			Name:     item.Name,
			Email:    item.Email,
			Password: hashed,
			Phone:    item.Phone,
			Role:     item.Role,
		}
		if err := db.Create(&user).Error; err != nil {
			return fmt.Errorf("failed to seed user %s: %w", item.Email, err)
		}
		log.Printf("database: seeded user %s (%s)", item.Email, item.Role)
	}
	return nil
}

func seedFields(db *gorm.DB) error {
	fields := []models.Field{
		{
			Name:         "Lapangan Padel A",
			Description:  "Lapangan padel indoor dengan ukuran standar internasional.",
			Type:         models.FieldTypePadel,
			Location:     "Jakarta Selatan",
			PricePerHour: 200000,
			Facilities:   "Ruang ganti,Parkir luas,Tribun,Wifi",
			Status:       models.FieldStatusActive,
		},
		{
			Name:         "Lapangan Padel B",
			Description:  "Lapangan padel indoor dengan lantai vinyl dan pencahayaan LED.",
			Type:         models.FieldTypePadel,
			Location:     "Bekasi Timur",
			PricePerHour: 60000,
			Facilities:   "AC,Ruang ganti,Kantin",
			Status:       models.FieldStatusActive,
		},
		{
			Name:         "Lapangan Padel C",
			Description:  "Lapangan padel outdoor ukuran penuh dengan ring standar FIBA.",
			Type:         models.FieldTypePadel,
			Location:     "Bekasi Utara",
			PricePerHour: 120000,
			Facilities:   "Parkir,Lampu malam,Toilet",
			Status:       models.FieldStatusActive,
		},
		{
			Name:         "Lapangan Padel D",
			Description:  "Lapangan padel outdoor dengan area pemanasan.",
			Type:         models.FieldTypePadel,
			Location:     "Bekasi Barat",
			PricePerHour: 100000,
			Facilities:   "Ruang ganti,Penyewaan raket,Toilet",
			Status:       models.FieldStatusActive,
		},
		{
			Name:         "Lapangan Padel E",
			Description:  "Lapangan padel indoor dengan jaring pengaman.",
			Type:         models.FieldTypePadel,
			Location:     "Tambun Selatan",
			PricePerHour: 250000,
			Facilities:   "Parkir luas,Tribun,Kantin,Lampu malam",
			Status:       models.FieldStatusActive,
		},
	}

	for i := range fields {
		var existing models.Field
		err := db.Where("name = ?", fields[i].Name).First(&existing).Error
		if err == nil {
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("failed to check seeded field: %w", err)
		}

		if err := db.Create(&fields[i]).Error; err != nil {
			return fmt.Errorf("failed to seed field %s: %w", fields[i].Name, err)
		}
		if err := seedSchedules(db, fields[i].ID); err != nil {
			return err
		}
		log.Printf("database: seeded field %s", fields[i].Name)
	}
	return nil
}

// seedSchedules membuat jam operasional Senin-Minggu untuk sebuah lapangan.
func seedSchedules(db *gorm.DB, fieldID uint) error {
	schedules := make([]models.Schedule, 0, 7)
	for _, day := range models.Days() {
		openTime, closeTime := "08:00", "23:00"
		if day == models.DaySaturday || day == models.DaySunday {
			openTime, closeTime = "07:00", "23:00"
		}
		schedules = append(schedules, models.Schedule{
			FieldID:   fieldID,
			Day:       day,
			OpenTime:  openTime,
			CloseTime: closeTime,
		})
	}

	err := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "field_id"}, {Name: "day"}},
		DoNothing: true,
	}).Create(&schedules).Error
	if err != nil {
		return fmt.Errorf("failed to seed schedules: %w", err)
	}
	return nil
}
