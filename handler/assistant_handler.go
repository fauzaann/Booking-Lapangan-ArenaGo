package handler

import (
	"github.com/gin-gonic/gin"

	"Booking-Lapangan/controllers"
	"Booking-Lapangan/dto"
	"Booking-Lapangan/pkg/response"
	"Booking-Lapangan/pkg/validator"
)

type AssistantHandler struct {
	Base
	assistant controllers.AssistantController
}

func NewAssistantHandler(assistant controllers.AssistantController, validate *validator.Validator) *AssistantHandler {
	return &AssistantHandler{Base: NewBase(validate), assistant: assistant}
}

func (h *AssistantHandler) Chat(c *gin.Context) {
	var request dto.AssistantChatRequest
	if !h.bindJSON(c, &request) {
		return
	}

	result, err := h.assistant.Chat(c.Request.Context(), request)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, "Assistant response generated", result)
}
