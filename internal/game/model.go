package game

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"
)

const (
	MicrosPerStonky = int64(1_000_000)

	StarterBalanceMicros = int64(25_000) * MicrosPerStonky
	BusinessUnlockMicros = int64(250_000) * MicrosPerStonky

	MinDebtLimitMicros = int64(5_000) * MicrosPerStonky
	MaxDebtLimitMicros = int64(100_000) * MicrosPerStonky

	ShareScale = int64(10_000) // 1 share = 10_000 units.
)

var (
	ErrInvalidSymbol        = errors.New("symbol must be exactly 6 uppercase letters")
	ErrStockNotFound        = errors.New("stock not found")
	ErrDuplicateIdempotency = errors.New("duplicate idempotency key")
	ErrInsufficientFunds    = errors.New("insufficient funds")
	ErrInsufficientShares   = errors.New("insufficient shares")
	ErrBusinessLocked       = errors.New("business feature locked: net worth below requirement")
	ErrUnauthorized         = errors.New("unauthorized")
)

var symbolRE = regexp.MustCompile(`^[A-Z]{6}$`)

func ValidateSymbol(symbol string) error {
	if !symbolRE.MatchString(strings.TrimSpace(symbol)) {
		return ErrInvalidSymbol
	}
	return nil
}

func StonkyToMicros(v float64) int64 {
	return int64(math.Round(v * float64(MicrosPerStonky)))
}

func MicrosToStonky(v int64) float64 {
	return float64(v) / float64(MicrosPerStonky)
}

func SharesToUnits(v float64) (int64, error) {
	if v <= 0 {
		return 0, fmt.Errorf("shares must be > 0")
	}
	return int64(math.Round(v * float64(ShareScale))), nil
}

func UnitsToShares(v int64) float64 {
	return float64(v) / float64(ShareScale)
}

func DebtLimitFromPeak(peakNetWorthMicros int64) int64 {
	limit := int64(math.Round(float64(peakNetWorthMicros) * 0.35))
	if limit < MinDebtLimitMicros {
		return MinDebtLimitMicros
	}
	if limit > MaxDebtLimitMicros {
		return MaxDebtLimitMicros
	}
	return limit
}
