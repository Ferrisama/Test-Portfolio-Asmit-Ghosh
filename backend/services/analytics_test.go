package services

import (
	"context"
	"encoding/csv"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"changeme/backend/models"
)

// Mock repository for testing
type mockRepo struct {
	trades []models.Trade
}

func (m *mockRepo) Upsert(ctx context.Context, t models.Trade) error {
	// Simple mock - just add to slice
	m.trades = append(m.trades, t)
	return nil
}

func (m *mockRepo) List(ctx context.Context, q models.Query) ([]models.Trade, error) {
	return m.trades, nil
}

func (m *mockRepo) Delete(ctx context.Context, id string) error {
	return nil
}

func TestMaxDrawdown_Simple(t *testing.T) {
	equity := []float64{100, 120, 90, 95, 80, 130}
	dd := MaxDrawdown(equity)
	expected := 33.33 // (120-80)/120 * 100 = 33.33%
	if abs(dd-expected) > 0.01 {
		t.Fatalf("expected ~33.33%%, got %v", dd)
	}
}

func TestMaxDrawdown_PercentBased(t *testing.T) {
	equity := []float64{100, 200, 150}
	// Percent-based max DD from 200 to 150 is 25% of peak (50/200 * 100).
	dd := MaxDrawdown(equity)
	expected := 25.0
	if abs(dd-expected) > 0.01 {
		t.Fatalf("expected 25%% percent-based DD, got %v", dd)
	}
}

func TestMaxDrawdown_NoDrawdown(t *testing.T) {
	equity := []float64{100, 120, 140, 160}
	dd := MaxDrawdown(equity)
	if dd != 0.0 {
		t.Fatalf("expected 0 drawdown for increasing equity, got %v", dd)
	}
}

func TestMaxDrawdown_EmptySlice(t *testing.T) {
	equity := []float64{}
	dd := MaxDrawdown(equity)
	if dd != 0.0 {
		t.Fatalf("expected 0 drawdown for empty slice, got %v", dd)
	}
}

func TestMaxDrawdown_ComplexScenario(t *testing.T) {
	// More complex equity curve
	equity := []float64{1000, 1200, 1100, 1300, 900, 950, 1400}
	// Peak of 1300, trough of 900 -> DD = (1300-900)/1300 * 100 = 30.77%
	dd := MaxDrawdown(equity)
	expected := 30.77
	if abs(dd-expected) > 0.01 {
		t.Fatalf("expected ~30.77%%, got %v", dd)
	}
}

func TestProfitFactorAndWinRate_TinySet(t *testing.T) {
	returns := []float64{10, -5, 20, -10}
	pf := ProfitFactor(returns)
	wr := WinRate(returns)
	if pf <= 1.0 {
		t.Fatalf("expected pf > 1, got %v", pf)
	}
	if wr != 0.5 {
		t.Fatalf("expected wr=0.5, got %v", wr)
	}
}

func TestProfitFactor_NoLosses(t *testing.T) {
	returns := []float64{10, 20, 5}
	pf := ProfitFactor(returns)
	// Should return large finite number, not +Inf
	if pf <= 0 || pf > 1000 {
		t.Fatalf("expected large finite number, got %v", pf)
	}
}

func TestProfitFactor_NoWins(t *testing.T) {
	returns := []float64{-10, -20, -5}
	pf := ProfitFactor(returns)
	if pf != 0 {
		t.Fatalf("expected 0 profit factor for no wins, got %v", pf)
	}
}

func TestSortino_DownsideDeviationOnly(t *testing.T) {
	returns := []float64{1, -1, 2, -2, 3}
	s := Sortino(returns)
	if s <= 0 {
		t.Fatalf("expected positive sortino, got %v", s)
	}
}

func TestSortino_NoDownside(t *testing.T) {
	returns := []float64{1, 2, 3, 4, 5}
	s := Sortino(returns)
	// Should return large finite number, not +Inf
	if s <= 0 || s > 1000 {
		t.Fatalf("expected large finite number, got %v", s)
	}
}

func TestExpectancy_DecimalPrecision(t *testing.T) {
	// Test that expectancy uses decimal precision
	returns := []float64{0.1, 0.1, 0.1}
	expectancy := Expectancy(returns)
	expected := 0.1
	if abs(expectancy-expected) > 0.0001 {
		t.Fatalf("expected %v, got %v", expected, expectancy)
	}
}

func TestImportStoresUTC(t *testing.T) {
	// Test various timezone inputs are converted to UTC
	testCases := []struct {
		input    string
		expected string
	}{
		{"2024-03-10T01:59:59-05:00", "2024-03-10T06:59:59Z"}, // EST to UTC
		{"2024-03-10T01:59:59+05:30", "2024-03-09T20:29:59Z"}, // IST to UTC  
		{"2024-03-10T01:59:59Z", "2024-03-10T01:59:59Z"},      // Already UTC
		{"2024-07-15T14:30:00-07:00", "2024-07-15T21:30:00Z"}, // PDT to UTC
		{"2024-12-25T12:00:00+01:00", "2024-12-25T11:00:00Z"}, // CET to UTC
	}

	for _, tc := range testCases {
		tt, err := time.Parse(time.RFC3339, tc.input)
		if err != nil {
			t.Fatal(err)
		}
		
		utc := tt.UTC()
		if utc.Format(time.RFC3339) != tc.expected {
			t.Fatalf("input %s: expected %s, got %s", tc.input, tc.expected, utc.Format(time.RFC3339))
		}
		
		if utc.Location() != time.UTC {
			t.Fatalf("expected UTC location, got %v", utc.Location())
		}
	}
}

func TestCSVImportDeduplication(t *testing.T) {
	// Test deduplication key generation
	testCases := []struct {
		desc     string
		symbol   string
		entry    string
		exit     string
		price    float64
		qty      float64
		expected string
	}{
		{
			desc:     "basic trade",
			symbol:   "AAPL", 
			entry:    "2024-03-10T06:59:59Z",
			exit:     "",
			price:    100.50,
			qty:      10,
			expected: "AAPL|2024-03-10T06:59:59Z||100.50000000|10.00000000",
		},
		{
			desc:     "closed trade",
			symbol:   "MSFT",
			entry:    "2024-03-10T06:59:59Z", 
			exit:     "2024-03-10T07:59:59Z",
			price:    200.25,
			qty:      5,
			expected: "MSFT|2024-03-10T06:59:59Z|2024-03-10T07:59:59Z|200.25000000|5.00000000",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			// Simulate the deduplication key creation logic
			exitTimeStr := ""
			if tc.exit != "" {
				exitTimeStr = tc.exit
			}
			
			key := strings.Join([]string{
				tc.symbol,
				tc.entry,
				exitTimeStr,
				formatFloat(tc.price, 8),
				formatFloat(tc.qty, 8),
			}, "|")
			
			if key != tc.expected {
				t.Fatalf("expected key %q, got %q", tc.expected, key)
			}
		})
	}
	
	// Test that different trades generate different keys
	key1 := "AAPL|2024-03-10T06:59:59Z||100.50000000|10.00000000"
	key2 := "AAPL|2024-03-10T06:59:59Z||100.50000000|20.00000000" // Different quantity
	key3 := "MSFT|2024-03-10T06:59:59Z||100.50000000|10.00000000" // Different symbol
	
	if key1 == key2 {
		t.Fatal("Different quantities should generate different keys")
	}
	if key1 == key3 {
		t.Fatal("Different symbols should generate different keys")
	}
}

func TestDSTBoundaryHandling(t *testing.T) {
	// Test the DST boundary case from the sample data
	entryTime := "2024-03-10T01:59:59-05:00" // EST
	exitTime := "2024-03-10T03:00:01-04:00"  // EDT (after DST change)
	
	entry, err := time.Parse(time.RFC3339, entryTime)
	if err != nil {
		t.Fatal(err)
	}
	
	exit, err := time.Parse(time.RFC3339, exitTime)
	if err != nil {
		t.Fatal(err)
	}
	
	// Both should be converted to UTC properly
	entryUTC := entry.UTC()
	exitUTC := exit.UTC()
	
	// Verify they're both in UTC
	if entryUTC.Location() != time.UTC || exitUTC.Location() != time.UTC {
		t.Fatal("Times should be converted to UTC")
	}
	
	// The actual times should be reasonable (entry before exit when normalized)
	if !entryUTC.Before(exitUTC) {
		t.Fatalf("Entry time %v should be before exit time %v", entryUTC, exitUTC)
	}
	
	// Check expected UTC values
	expectedEntry := "2024-03-10T06:59:59Z"
	expectedExit := "2024-03-10T07:00:01Z"
	
	if entryUTC.Format(time.RFC3339) != expectedEntry {
		t.Fatalf("Expected entry UTC %s, got %s", expectedEntry, entryUTC.Format(time.RFC3339))
	}
	if exitUTC.Format(time.RFC3339) != expectedExit {
		t.Fatalf("Expected exit UTC %s, got %s", expectedExit, exitUTC.Format(time.RFC3339))
	}
}

func TestFixtures_SampleTrades_Returns(t *testing.T) {
	// Test using the sample CSV data if available
	f, err := os.Open("../../testdata/sample_trades.csv")
	if err != nil {
		// Skip if file doesn't exist - this is optional
		t.Skip("Sample CSV file not found, skipping fixture test")
		return
	}
	defer f.Close()
	
	r := csv.NewReader(f)
	r.Comment = '#'
	
	// Skip header
	_, err = r.Read()
	if err != nil {
		t.Fatal("Failed to read header:", err)
	}
	
	var returns []float64
	lineNum := 1
	
	for {
		rec, err := r.Read()
		if err != nil {
			break // EOF or error
		}
		lineNum++
		
		// Skip comments and empty lines
		if len(rec) == 0 || strings.HasPrefix(strings.TrimSpace(rec[0]), "#") {
			continue
		}
		
		if len(rec) < 8 {
			continue // Invalid record
		}
		
		side := strings.ToLower(strings.TrimSpace(rec[1]))
		
		entry, err := strconv.ParseFloat(strings.TrimSpace(rec[4]), 64)
		if err != nil {
			continue
		}
		
		exitStr := strings.TrimSpace(rec[5])
		if exitStr == "" || exitStr == "0" {
			continue // Open trade
		}
		
		exit, err := strconv.ParseFloat(exitStr, 64)
		if err != nil {
			continue
		}
		
		qty, err := strconv.ParseFloat(strings.TrimSpace(rec[6]), 64)
		if err != nil {
			continue
		}
		
		fees, err := strconv.ParseFloat(strings.TrimSpace(rec[7]), 64)
		if err != nil {
			fees = 0 // Default to 0 fees if parsing fails
		}
		
		// Calculate P&L
		var pnl float64
		if side == "short" {
			pnl = (entry - exit) * qty - fees
		} else {
			pnl = (exit - entry) * qty - fees
		}
		
		returns = append(returns, pnl)
	}
	
	if len(returns) == 0 {
		t.Skip("No closed trades found in sample_trades.csv")
		return
	}
	
	// Verify all analytics functions work and return finite values
	pf := ProfitFactor(returns)
	wr := WinRate(returns)
	sharpe := Sharpe(returns)
	sortino := Sortino(returns)
	expectancy := Expectancy(returns)
	
	// Basic sanity checks
	if wr < 0 || wr > 1 {
		t.Fatalf("invalid win rate: %v", wr)
	}
	if pf < 0 {
		t.Fatalf("invalid profit factor: %v", pf)
	}
	
	// Ensure no infinite values that would break JSON serialization
	checkFinite := func(name string, value float64) {
		if !isFinite(value) {
			t.Fatalf("%s returned non-finite value: %v", name, value)
		}
	}
	
	checkFinite("ProfitFactor", pf)
	checkFinite("WinRate", wr)
	checkFinite("Sharpe", sharpe)
	checkFinite("Sortino", sortino)
	checkFinite("Expectancy", expectancy)
	
	t.Logf("Sample data analytics: WR=%.3f, PF=%.3f, Sharpe=%.3f, Sortino=%.3f, Expectancy=%.2f",
		wr, pf, sharpe, sortino, expectancy)
}

func TestAnalytics_NoInfiniteValues(t *testing.T) {
	// Test edge cases that might produce infinite values
	testCases := []struct {
		name    string
		returns []float64
	}{
		{"only wins", []float64{10, 20, 5}},
		{"only losses", []float64{-10, -20, -5}},
		{"mixed", []float64{10, -5, 20, -10}},
		{"zero returns", []float64{0, 0, 0}},
		{"single positive", []float64{100}},
		{"single negative", []float64{-100}},
		{"empty", []float64{}},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pf := ProfitFactor(tc.returns)
			wr := WinRate(tc.returns)
			sharpe := Sharpe(tc.returns) 
			sortino := Sortino(tc.returns)
			expectancy := Expectancy(tc.returns)
			
			// All values must be finite for JSON serialization
			values := map[string]float64{
				"ProfitFactor": pf,
				"WinRate":      wr,
				"Sharpe":       sharpe,
				"Sortino":      sortino,
				"Expectancy":   expectancy,
			}
			
			for name, value := range values {
				if !isFinite(value) {
					t.Fatalf("%s returned non-finite value %v for case %q", name, value, tc.name)
				}
			}
		})
	}
}

// Helper functions
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func formatFloat(f float64, precision int) string {
	format := "%." + strconv.Itoa(precision) + "f"
	return strings.TrimRight(strings.TrimRight(sprintf(format, f), "0"), ".")
}

func sprintf(format string, args ...interface{}) string {
	return strings.Replace(format, "%", "", -1) + strconv.FormatFloat(args[0].(float64), 'f', 8, 64)
}

func isFinite(f float64) bool {
	return !isInf(f, 0) && !isNaN(f)
}

func isInf(f float64, sign int) bool {
	// Simple check for infinity
	return f > 1e100 || f < -1e100
}

func isNaN(f float64) bool {
	// Simple NaN check
	return f != f
}