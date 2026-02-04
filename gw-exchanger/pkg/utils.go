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

// FormatRate форматирует курс обмена для вывода
func FormatRate(rate float64) string {
	return fmt.Sprintf("%.8f", rate)
}
