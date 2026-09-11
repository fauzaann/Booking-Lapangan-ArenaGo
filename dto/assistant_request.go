package dto

type AssistantMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type AssistantChatRequest struct {
	Message string             `json:"message" validate:"required,max=2000"`
	History []AssistantMessage `json:"history" validate:"max=10,dive"`
}

type AssistantChatResponse struct {
	Reply string `json:"reply"`
}
