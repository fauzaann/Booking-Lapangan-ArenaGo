package controllers

import (
	"context"
	"strings"

	"Booking-Lapangan/dto"
	"Booking-Lapangan/pkg/openrouter"
)

type AssistantController interface {
	Chat(ctx context.Context, request dto.AssistantChatRequest) (*dto.AssistantChatResponse, error)
}

type assistantController struct {
	chat openrouter.ChatClient
}

const outOfScopeReply = "Maaf, saya tidak tahu. Saya hanya dapat membantu tentang booking lapangan, jadwal, availability, harga, lokasi, fasilitas, pembayaran, dan pembatalan."

var bookingTopicTerms = []string{
	"booking", "book", "pesan", "lapangan", "padel", "court", "jadwal", "jam", "slot",
	"harga", "tarif", "biaya", "tersedia", "availability", "lokasi", "alamat", "fasilitas",
	"sewa", "main", "bayar", "pembayaran", "invoice", "xendit", "batal", "cancel",
}

func NewAssistantController(chat openrouter.ChatClient) AssistantController {
	return &assistantController{chat: chat}
}

func (c *assistantController) Chat(ctx context.Context, request dto.AssistantChatRequest) (*dto.AssistantChatResponse, error) {
	if !isBookingTopic(request.Message) {
		return &dto.AssistantChatResponse{Reply: outOfScopeReply}, nil
	}

	messages := []openrouter.Message{{
		Role:    "system",
		Content: "Kamu adalah ArenaGO Assistant, concierge booking lapangan olahraga. Jawab hanya tentang booking lapangan, jadwal, availability, harga, lokasi, fasilitas, pembayaran, dan pembatalan. Untuk topik lain, jawab persis: 'Maaf, saya tidak tahu.' Jawab dalam Bahasa Indonesia secara ringkas. Jangan mengklaim booking atau pembayaran sudah dibuat; arahkan user ke halaman booking untuk konfirmasi. Jika informasi booking belum lengkap, tanyakan satu hal yang paling penting.",
	}}
	for _, item := range request.History {
		role := strings.ToLower(strings.TrimSpace(item.Role))
		if role != "user" && role != "assistant" {
			continue
		}
		content := strings.TrimSpace(item.Content)
		if content != "" {
			messages = append(messages, openrouter.Message{Role: role, Content: content})
		}
	}
	messages = append(messages, openrouter.Message{Role: "user", Content: strings.TrimSpace(request.Message)})

	reply, err := c.chat.Chat(ctx, messages)
	if err != nil {
		return nil, err
	}
	return &dto.AssistantChatResponse{Reply: reply}, nil
}

func isBookingTopic(message string) bool {
	message = strings.ToLower(strings.TrimSpace(message))
	for _, term := range bookingTopicTerms {
		if strings.Contains(message, term) {
			return true
		}
	}
	return false
}
