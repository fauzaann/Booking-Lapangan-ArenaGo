package config

import (
	"gorm.io/gorm"


)
func newDatabaseConfig(cfg *gorm.Config)(*gorm.Config, error) {
	gormConfig := &gorm.Config{
		// Konfigurasi GORM lainnya dapat ditambahkan di sini
	}

	return gormConfig, nil
}	

func closeDatabase(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}