package main

import (
	"Booking-Lapangan/config"
	"Booking-Lapangan/controllers"
	"Booking-Lapangan/repository"
	"Booking-Lapangan/router"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	db := config.InitDB()
	defer func() {
		if err := config.CloseDatabase(db); err != nil {
			log.Printf("Gagal menutup database: %v", err)
		}
	}()

	repo := repository.NewRepository(db)
	controller := controllers.NewController(repo)

	routerEngine := gin.Default()
	router.RegisterRoutes(routerEngine, controller)

	port := "8080"
	fmt.Printf("Server berjalan di http://localhost:%s\n", port)
	if err := routerEngine.Run(":" + port); err != nil {
		log.Fatalf("Gagal menjalankan server: %v", err)
	}
}