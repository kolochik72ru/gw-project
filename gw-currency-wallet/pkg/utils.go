package pkg

import (
	"fmt"
	"strings"
)

// ValidateCurrency проверяет, что валюта является одной из поддерживаемых
func ValidateCurrency(currency string) error {
	supportedCurrencies := map[string]bool{
		"USD": true,
		"EUR": true,
		"RUB": true,
	}

	currency = strings.ToUpper(currency)
	if !supportedCurrencies[currency] {
		return fmt.Errorf("unsupported currency: %s. Supported currencies: USD, EUR, RUB", currency)
	}

	return nil
}

// NormalizeCurrency приводит код валюты к верхнему регистру
func NormalizeCurrency(currency string) string {
	return strings.ToUpper(strings.TrimSpace(currency))
}

// ValidateAmount проверяет, что сумма положительная
func ValidateAmount(amount float64) error {
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}
	return nil
}
