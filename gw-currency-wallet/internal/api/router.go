package api

import (
	"github.com/gin-gonic/gin"
	"gw-currency-wallet/internal/api/handlers"
	"gw-currency-wallet/internal/api/middleware"
	"gw-currency-wallet/internal/service"
	"github.com/sirupsen/logrus"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// SetupRouter настраивает и возвращает роутер с всеми эндпоинтами
func SetupRouter(
	walletService *service.WalletService,
	jwtMiddleware *middleware.JWTMiddleware,
	logger *logrus.Logger,
	ginMode string,
) *gin.Engine {
	// Установка режима Gin
	gin.SetMode(ginMode)

	router := gin.New()

	// Middleware
	router.Use(gin.Recovery())
	router.Use(middleware.Logger(logger))

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Swagger documentation
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Инициализация handlers
	authHandler := handlers.NewAuthHandler(walletService, jwtMiddleware, logger)
	walletHandler := handlers.NewWalletHandler(walletService, logger)
	exchangeHandler := handlers.NewExchangeHandler(walletService, logger)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Public routes (без авторизации)
		v1.POST("/register", authHandler.Register)
		v1.POST("/login", authHandler.Login)

		// Protected routes (требуют авторизации)
		authorized := v1.Group("")
		authorized.Use(jwtMiddleware.Auth())
		{
			// Wallet operations
			authorized.GET("/balance", walletHandler.GetBalance)
			authorized.POST("/wallet/deposit", walletHandler.Deposit)
			authorized.POST("/wallet/withdraw", walletHandler.Withdraw)

			// Exchange operations
			authorized.GET("/exchange/rates", exchangeHandler.GetRates)
			authorized.POST("/exchange", exchangeHandler.Exchange)
		}
	}

	return router
}
