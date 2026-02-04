package cache

import (
	"sync"
	"time"
)

// RatesCache кеш для курсов валют
type RatesCache struct {
	rates  map[string]float32
	mu     sync.RWMutex
	ttl    time.Duration
	lastUp time.Time
}

// NewRatesCache создает новый кеш
func NewRatesCache(ttl time.Duration) *RatesCache {
	return &RatesCache{
		rates: make(map[string]float32),
		ttl:   ttl,
	}
}

// Set сохраняет курсы в кеш
func (c *RatesCache) Set(rates map[string]float32) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.rates = rates
	c.lastUp = time.Now()
}

// Get возвращает курсы из кеша, если они актуальны
func (c *RatesCache) Get() (map[string]float32, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Проверяем, не истек ли TTL
	if time.Since(c.lastUp) > c.ttl {
		return nil, false
	}

	// Возвращаем копию, чтобы избежать race condition
	ratesCopy := make(map[string]float32, len(c.rates))
	for k, v := range c.rates {
		ratesCopy[k] = v
	}

	return ratesCopy, true
}

// GetRate возвращает конкретный курс из кеша
func (c *RatesCache) GetRate(fromCurrency, toCurrency string) (float32, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Проверяем, не истек ли TTL
	if time.Since(c.lastUp) > c.ttl {
		return 0, false
	}

	key := fromCurrency + "_" + toCurrency
	rate, exists := c.rates[key]
	return rate, exists
}

// Clear очищает кеш
func (c *RatesCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.rates = make(map[string]float32)
	c.lastUp = time.Time{}
}

// IsValid проверяет, актуален ли кеш
func (c *RatesCache) IsValid() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return time.Since(c.lastUp) <= c.ttl && len(c.rates) > 0
}
