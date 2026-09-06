package controllers

import (
	"context"
	"time"

	"Booking-Lapangan/repository"

	"github.com/gin-gonic/gin"
)

var timeNow = time.Now

// Actor adalah identitas yang sudah terautentikasi
type Actor struct {
	ID       uint
	Username string
	Role     string
}

// IsAdmin memeriksa apakah aktor memiliki peran admin
func (a *Actor) IsAdmin() bool {
	return a.Role == "ADMIN"
}

// Controller is the main application controller used by router.
type Controller struct {
	auth *AuthController
}

func NewController(repo repository.UserRepository) *Controller {
	return &Controller{auth: NewAuthController(repo)}
}

func (c *Controller) Register(ctx *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"success": false, "message": err.Error()})
		return
	}

	user, err := c.auth.Register(context.Background(), req.Username, req.Email, req.Password)
	if err != nil {
		ctx.JSON(400, gin.H{"success": false, "message": err.Error()})
		return
	}

	ctx.JSON(201, gin.H{"success": true, "data": user})
}

func (c *Controller) Login(ctx *gin.Context) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"success": false, "message": err.Error()})
		return
	}

	user, err := c.auth.Login(context.Background(), req.Email, req.Password)
	if err != nil {
		ctx.JSON(401, gin.H{"success": false, "message": err.Error()})
		return
	}

	ctx.JSON(200, gin.H{"success": true, "data": user})
}

func (c *Controller) GetProfile(ctx *gin.Context) {
	actor, ok := ctx.Get("actor")
	if !ok {
		ctx.JSON(401, gin.H{"success": false, "message": "Unauthorized"})
		return
	}
	userID := actor.(Actor).ID
	user, err := c.auth.Profile(context.Background(), userID)
	if err != nil {
		ctx.JSON(404, gin.H{"success": false, "message": err.Error()})
		return
	}
	ctx.JSON(200, gin.H{"success": true, "data": user})
}

func (c *Controller) UpdateProfile(ctx *gin.Context) {
	ctx.JSON(200, gin.H{"success": true, "message": "update profile belum diimplementasi"})
}

func (c *Controller) Logout(ctx *gin.Context) {
	ctx.JSON(200, gin.H{"success": true, "message": "logout success"})
}
