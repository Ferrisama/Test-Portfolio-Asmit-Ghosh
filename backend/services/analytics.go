package services

import (
	"math"

	"github.com/shopspring/decimal"
)

// MaxDrawdown calculates maximum percent-based drawdown from an equity curve using O(n) peak-tracking algorithm.
// This implementation properly handles the percent-based calculation: (peak - trough) / peak * 100
func MaxDrawdown(equity []float64) float64 {
	if len(equity) == 0 {
		return 0.0
	}

	maxDD := 0.0
	peak := equity[0]

	for _, value := range equity {
		// Update peak if current value is higher
		if value > peak {
			peak = value
		}
		
		// Calculate percent-based drawdown from peak
		if peak > 0 {
			percentDD := (peak - value) / peak * 100
			if percentDD > maxDD {
				maxDD = percentDD
			}
		}
	}

	return maxDD
}

// ProfitFactor calculates gross profits / gross losses (absolute) using decimal precision.
// Returns a safe finite value to prevent JSON serialization errors.
func ProfitFactor(returns []float64) float64 {
	if len(returns) == 0 {
		return 0
	}
	
	gpDecimal := decimal.Zero
	glDecimal := decimal.Zero
	
	for _, r := range returns {
		rDecimal := decimal.NewFromFloat(r)
		if r > 0 {
			gpDecimal = gpDecimal.Add(rDecimal)
		} else if r < 0 {
			glDecimal = glDecimal.Add(rDecimal.Neg())
		}
	}
	
	if glDecimal.IsZero() {
		if gpDecimal.IsZero() {
			return 0
		}
		// Return a large but finite number instead of +Inf to prevent JSON errors
		return 999.99
	}
	
	result, _ := gpDecimal.Div(glDecimal).Float64()
	
	// Ensure result is finite for JSON serialization
	if math.IsInf(result, 0) || math.IsNaN(result) {
		return 999.99
	}
	
	return result
}

// WinRate calculates wins / total trades.
func WinRate(returns []float64) float64 {
	if len(returns) == 0 {
		return 0
	}
	wins := 0.0
	for _, r := range returns {
		if r > 0 {
			wins += 1
		}
	}
	return wins / float64(len(returns))
}

// Expectancy calculates average return per trade using decimal precision.
func Expectancy(returns []float64) float64 {
	if len(returns) == 0 {
		return 0
	}
	
	sum := decimal.Zero
	for _, r := range returns {
		sum = sum.Add(decimal.NewFromFloat(r))
	}
	
	result, _ := sum.Div(decimal.NewFromInt(int64(len(returns)))).Float64()
	
	// Ensure result is finite for JSON serialization
	if math.IsInf(result, 0) || math.IsNaN(result) {
		return 0
	}
	
	return result
}

// Sharpe ratio using population stddev, risk-free = 0, with decimal precision for mean calculation.
// Returns a safe finite value to prevent JSON serialization errors.
func Sharpe(returns []float64) float64 {
	if len(returns) == 0 {
		return 0
	}
	
	mean := Expectancy(returns)
	meanDecimal := decimal.NewFromFloat(mean)
	
	var ss decimal.Decimal = decimal.Zero
	for _, r := range returns {
		rDecimal := decimal.NewFromFloat(r)
		diff := rDecimal.Sub(meanDecimal)
		ss = ss.Add(diff.Mul(diff))
	}
	
	variance, _ := ss.Div(decimal.NewFromInt(int64(len(returns)))).Float64()
	sd := math.Sqrt(variance)
	
	if sd == 0 {
		return 0
	}
	
	result := mean / sd
	
	// Ensure result is finite for JSON serialization
	if math.IsInf(result, 0) || math.IsNaN(result) {
		return 0
	}
	
	return result
}

// Sortino ratio using downside deviation only (negative returns vs mean).
// Returns a safe finite value to prevent JSON serialization errors.
func Sortino(returns []float64) float64 {
	if len(returns) == 0 {
		return 0
	}
	
	mean := Expectancy(returns)
	meanDecimal := decimal.NewFromFloat(mean)
	
	var downsideSS decimal.Decimal = decimal.Zero
	downsideCount := 0
	
	for _, r := range returns {
		if r < mean { // Below mean for downside deviation
			rDecimal := decimal.NewFromFloat(r)
			diff := rDecimal.Sub(meanDecimal)
			downsideSS = downsideSS.Add(diff.Mul(diff))
			downsideCount++
		}
	}
	
	if downsideCount == 0 {
		// Return a large but finite number instead of +Inf
		return 999.99
	}
	
	downsideVariance, _ := downsideSS.Div(decimal.NewFromInt(int64(downsideCount))).Float64()
	dd := math.Sqrt(downsideVariance)
	
	if dd == 0 {
		return 999.99
	}
	
	result := mean / dd
	
	// Ensure result is finite for JSON serialization
	if math.IsInf(result, 0) || math.IsNaN(result) {
		return 999.99
	}
	
	return result
}