package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gw-currency-wallet/internal/api/middleware"
	"gw-currency-wallet/internal/service"
	"github.com/sirupsen/logrus"
)

// ExchangeHandler обработчик для обмена валют
type ExchangeHandler struct {
	service *service.WalletService
	logger  *logrus.Logger
}

// NewExchangeHandler создает новый обработчик обмена
func NewExchangeHandler(service *service.WalletService, logger *logrus.Logger) *ExchangeHandler {
	return &ExchangeHandler{
		service: service,
		logger:  logger,
	}
}

// ExchangeRequest запрос на обмен валюты
type ExchangeRequest struct {
	FromCurrency string  `json:"from_currency" binding:"required,oneof=USD EUR RUB"`
	ToCurrency   string  `json:"to_currency" binding:"required,oneof=USD EUR RUB"`
	Amount       float64 `json:"amount" binding:"required,gt=0"`
}

// GetRates возвращает курсы валют
// @Summary Get exchange rates
// @Description Get current exchange rates for all currency pairs
// @Tags exchange
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/exchange/rates [get]
func (h *ExchangeHandler) GetRates(c *gin.Context) {
	_, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	rates, err := h.service.GetExchangeRates(c.Request.Context())
	if err != nil {
		h.logger.Errorf("Failed to get exchange rates: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve exchange rates"})
		return
	}

	// Преобразуем карту в более удобный формат
	formattedRates := make(map[string]float32)
	for key, value := range rates {
		formattedRates[key] = value
	}

	c.JSON(http.StatusOK, gin.H{"rates": formattedRates})
}

// Exchange обменивает валюту
// @Summary Exchange currency
// @Description Exchange one currency for another
// @Tags exchange
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body ExchangeRequest true "Exchange data"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /api/v1/exchange [post]
func (h *ExchangeHandler) Exchange(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req ExchangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Проверка, что валюты разные
	if req.FromCurrency == req.ToCurrency {
		c.JSON(http.StatusBadRequest, gin.H{"error": "from_currency and to_currency must be different"})
		return
	}

	exchangedAmount, newBalances, err := h.service.ExchangeCurrency(
		c.Request.Context(),
		userID,
		req.FromCurrency,
		req.ToCurrency,
		req.Amount,
	)

	if err != nil {
		h.logger.Errorf("Failed to exchange currency: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":          "Exchange successful",
		"exchanged_amount": exchangedAmount,
		"new_balance":      newBalances,
	})
}
