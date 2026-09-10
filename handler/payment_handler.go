package handler

import (
	"github.com/gin-gonic/gin"

	"Booking-Lapangan/controllers"
	"Booking-Lapangan/dto"
	"Booking-Lapangan/middleware"
	"Booking-Lapangan/utils"
)

// PaymentHandler menangani webhook Xendit dan pembacaan data pembayaran.
type PaymentHandler struct {
	Base
	payments controllers.PaymentController
}

// NewPaymentHandler membuat PaymentHandler.
func NewPaymentHandler(payments controllers.PaymentController, validate *utils.Validator) *PaymentHandler {
	return &PaymentHandler{Base: NewBase(validate), payments: payments}
}

// Webhook menangani POST /api/v1/payments/webhook.
// Endpoint ini dipanggil Xendit, bukan frontend, dan diautentikasi memakai
// header x-callback-token.
func (h *PaymentHandler) Webhook(c *gin.Context) {
	var payload utils.WebhookPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		utils.Error(c, utils.BadRequest("invalid webhook payload: "+err.Error()))
		return
	}

	token := c.GetHeader("x-callback-token")
	if err := h.payments.HandleWebhook(c.Request.Context(), token, payload); err != nil {
		utils.Error(c, err)
		return
	}
	utils.OK(c, "Webhook processed", nil)
}

// DetailByBooking menangani GET /api/v1/bookings/:id/payment.
func (h *PaymentHandler) DetailByBooking(c *gin.Context) {
	bookingID, ok := UintParam(c, "id")
	if !ok {
		return
	}

	actor := middleware.MustActor(c)
	payment, err := h.payments.DetailByBooking(c.Request.Context(), actor, bookingID)
	if err != nil {
		utils.Error(c, err)
		return
	}
	utils.OK(c, "Payment retrieved", payment)
}

// List menangani GET /api/v1/admin/payments.
func (h *PaymentHandler) List(c *gin.Context) {
	var query dto.PaymentFilterQuery
	if !h.BindQuery(c, &query) {
		return
	}
	query.PaginationQuery = query.PaginationQuery.Normalize()

	payments, total, err := h.payments.List(c.Request.Context(), query)
	if err != nil {
		utils.Error(c, err)
		return
	}
	utils.Paginated(c, "Payments retrieved", payments, utils.NewMeta(query.Page, query.Limit, total))
}
