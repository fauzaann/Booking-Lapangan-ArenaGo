package main

import (
	"Booking-Lapangan/config"
	"Booking-Lapangan/controllers"
	"Booking-Lapangan/repository"
	"Booking-Lapangan/router"
	"context"
	"fmt"
	"log"
	"os"

	openrouter "github.com/OpenRouterTeam/go-sdk"
	"github.com/OpenRouterTeam/go-sdk/models/components"

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

	ctx := context.Background()

	s := openrouter.New(
		openrouter.WithSecurity(os.Getenv("OPENROUTER_API_KEY")),
	)

	res, err := s.Chat.Send(ctx, components.ChatRequest{
		Messages: []components.ChatMessages{
			components.CreateChatMessagesUser(
				components.ChatUserMessage{
					Content: components.CreateChatUserMessageContentStr(
						"What is the meaning of life?",
					),
					Role: components.ChatUserMessageRoleUser,
				},
			),
		},
		Model: openrouter.Pointer("openai/gpt-4o-mini"),
	}, nil)
	if err != nil {
		log.Fatal(err)
	}
	if res.ChatResult != nil {
		fmt.Println(res.ChatResult.Choices[0].Message.Content)
	}
}
