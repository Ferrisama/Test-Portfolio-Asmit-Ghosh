package bindings

import (
	"context"
	"testing"
	"time"
	"changeme/backend/models"
)

// MockRepo implements TradeRepo for testing
type MockRepo struct {
	trades []models.Trade
}

func (m *MockRepo) Upsert(ctx context.Context, t models.Trade) error {
	// Simple append for testing - in real implementation would check for ID
	m.trades = append(m.trades, t)
	return nil
}

func (m *MockRepo) List(ctx context.Context, q models.Query) ([]models.Trade, error) {
	return m.trades, nil
}

func (m *MockRepo) Delete(ctx context.Context, id string) error {
	return nil
}

func TestCSVImport_UTCConversion(t *testing.T) {
	mockRepo := &MockRepo{}
	service := &JournalService{Repo: mockRepo}
	
	// CSV with different timezone inputs
	csvData := `symbol,side,entry_time,exit_time,entry_price,exit_price,qty,fees,notes
AAPL,long,2024-03-10T01:59:59-05:00,2024-03-10T03:00:01-04:00,100,110,1,0.5,EST/EDT boundary
MSFT,short,2024-01-01T10:30:00+05:30,2024-01-01T15:00:00+05:30,200,190,1,0.5,IST timezone
GOOGL,long,2024-01-01T12:00:00Z,,130,0,1,0.5,Already UTC`

	ctx := context.Background()
	report, err := service.ImportCSV(ctx, csvData)
	if err != nil {
		t.Fatalf("ImportCSV failed: %v", err)
	}

	if report.Imported != 3 {
		t.Fatalf("expected 3 imported trades, got %d", report.Imported)
	}

	if len(report.Errors) > 0 {
		t.Fatalf("unexpected errors: %v", report.Errors)
	}

	// Verify all timestamps are stored in UTC
	trades, err := mockRepo.List(ctx, models.Query{})
	if err != nil {
		t.Fatalf("Failed to list trades: %v", err)
	}

	testCases := []struct {
		symbol           string
		expectedEntryUTC string
		expectedExitUTC  string
	}{
		{"AAPL", "2024-03-10T06:59:59Z", "2024-03-10T07:00:01Z"}, // EST/EDT converted to UTC
		{"MSFT", "2024-01-01T05:00:00Z", "2024-01-01T09:30:00Z"}, // IST converted to UTC
		{"GOOGL", "2024-01-01T12:00:00Z", ""},                    // Already UTC, no exit time
	}

	for i, tc := range testCases {
		if i >= len(trades) {
			t.Fatalf("Expected trade %d not found", i)
		}

		trade := trades[i]
		
		// Check symbol
		if trade.Symbol != tc.symbol {
			t.Errorf("Expected symbol %s, got %s", tc.symbol, trade.Symbol)
		}

		// Check entry time is in UTC
		if trade.EntryTime.Location() != time.UTC {
			t.Errorf("Trade %s entry time not in UTC: %v", tc.symbol, trade.EntryTime.Location())
		}

		// Check entry time value
		if trade.EntryTime.Format(time.RFC3339) != tc.expectedEntryUTC {
			t.Errorf("Trade %s: expected entry time %s, got %s", 
				tc.symbol, tc.expectedEntryUTC, trade.EntryTime.Format(time.RFC3339))
		}

		// Check exit time if present
		if tc.expectedExitUTC != "" {
			if trade.ExitTime == nil {
				t.Errorf("Trade %s: expected exit time, got nil", tc.symbol)
				continue
			}
			
			if trade.ExitTime.Location() != time.UTC {
				t.Errorf("Trade %s exit time not in UTC: %v", tc.symbol, trade.ExitTime.Location())
			}
			
			if trade.ExitTime.Format(time.RFC3339) != tc.expectedExitUTC {
				t.Errorf("Trade %s: expected exit time %s, got %s", 
					tc.symbol, tc.expectedExitUTC, trade.ExitTime.Format(time.RFC3339))
			}
		} else if trade.ExitTime != nil {
			t.Errorf("Trade %s: expected no exit time, got %s", tc.symbol, trade.ExitTime)
		}
	}
}

func TestCSVImport_Deduplication(t *testing.T) {
	mockRepo := &MockRepo{}
	service := &JournalService{Repo: mockRepo}

	// CSV with duplicate trades (same symbol, times, entry_price, qty)
	csvData := `symbol,side,entry_time,exit_time,entry_price,exit_price,qty,fees,notes
AAPL,long,2024-01-01T10:00:00Z,2024-01-01T11:00:00Z,100,110,1,0.5,Original trade
AAPL,long,2024-01-01T10:00:00Z,2024-01-01T11:00:00Z,100,110,1,1.0,Duplicate with different fees and notes
MSFT,short,2024-01-01T10:00:00Z,2024-01-01T11:00:00Z,200,190,2,0.5,Different trade
AAPL,long,2024-01-01T10:00:00Z,2024-01-01T11:00:00Z,100,110,2,0.5,Different qty - should not be duplicate`

	ctx := context.Background()
	report, err := service.ImportCSV(ctx, csvData)
	if err != nil {
		t.Fatalf("ImportCSV failed: %v", err)
	}

	// Should import 3 trades (1 duplicate skipped)
	if report.Imported != 3 {
		t.Errorf("expected 3 imported trades, got %d", report.Imported)
	}

	// Should skip 1 duplicate
	if report.Skipped != 1 {
		t.Errorf("expected 1 skipped trade, got %d", report.Skipped)
	}

	if len(report.Errors) > 0 {
		t.Errorf("unexpected errors: %v", report.Errors)
	}

	// Verify the correct trades were imported
	trades, err := mockRepo.List(ctx, models.Query{})
	if err != nil {
		t.Fatalf("Failed to list trades: %v", err)
	}

	if len(trades) != 3 {
		t.Fatalf("expected 3 trades in repo, got %d", len(trades))
	}

	// Check that we have the expected unique trades
	symbolCounts := make(map[string]int)
	for _, trade := range trades {
		symbolCounts[trade.Symbol]++
	}

	if symbolCounts["AAPL"] != 2 { // Original + different qty
		t.Errorf("expected 2 AAPL trades, got %d", symbolCounts["AAPL"])
	}

	if symbolCounts["MSFT"] != 1 {
		t.Errorf("expected 1 MSFT trade, got %d", symbolCounts["MSFT"])
	}
}

func TestCSVImport_ValidationErrors(t *testing.T) {
	mockRepo := &MockRepo{}
	service := &JournalService{Repo: mockRepo}

	// CSV with various validation errors
	csvData := `symbol,side,entry_time,exit_time,entry_price,exit_price,qty,fees,notes
,long,2024-01-01T10:00:00Z,2024-01-01T11:00:00Z,100,110,1,0.5,Empty symbol
AAPL,invalid,2024-01-01T10:00:00Z,2024-01-01T11:00:00Z,100,110,1,0.5,Invalid side
MSFT,long,invalid-date,2024-01-01T11:00:00Z,100,110,1,0.5,Invalid entry time
GOOGL,short,2024-01-01T10:00:00Z,invalid-date,100,110,1,0.5,Invalid exit time
TSLA,long,2024-01-01T10:00:00Z,2024-01-01T11:00:00Z,-100,110,1,0.5,Negative entry price
NVDA,long,2024-01-01T10:00:00Z,2024-01-01T11:00:00Z,100,-110,1,0.5,Negative exit price
META,short,2024-01-01T10:00:00Z,2024-01-01T11:00:00Z,100,110,-1,0.5,Negative quantity
NFLX,long,2024-01-01T10:00:00Z,2024-01-01T11:00:00Z,100,110,1,-0.5,Negative fees
AMZN,long,2024-01-01T10:00:00Z,2024-01-01T11:00:00Z,100,110,1,0.5,Valid trade`

	ctx := context.Background()
	report, err := service.ImportCSV(ctx, csvData)
	if err != nil {
		t.Fatalf("ImportCSV failed: %v", err)
	}

	// Should import only 1 valid trade
	if report.Imported != 1 {
		t.Errorf("expected 1 imported trade, got %d", report.Imported)
	}

	// Should have 8 errors
	if len(report.Errors) != 8 {
		t.Errorf("expected 8 errors, got %d: %v", len(report.Errors), report.Errors)
	}

	// Verify the valid trade was imported
	trades, err := mockRepo.List(ctx, models.Query{})
	if err != nil {
		t.Fatalf("Failed to list trades: %v", err)
	}

	if len(trades) != 1 {
		t.Fatalf("expected 1 trade in repo, got %d", len(trades))
	}

	if trades[0].Symbol != "AMZN" {
		t.Errorf("expected AMZN trade, got %s", trades[0].Symbol)
	}
}

func TestCSVImport_EmptyAndComments(t *testing.T) {
	mockRepo := &MockRepo{}
	service := &JournalService{Repo: mockRepo}

	// CSV with comments and empty lines
	csvData := `symbol,side,entry_time,exit_time,entry_price,exit_price,qty,fees,notes
# This is a comment line
AAPL,long,2024-01-01T10:00:00Z,2024-01-01T11:00:00Z,100,110,1,0.5,Valid trade

# Another comment
MSFT,short,2024-01-01T12:00:00Z,2024-01-01T13:00:00Z,200,190,1,0.5,Another valid trade`

	ctx := context.Background()
	report, err := service.ImportCSV(ctx, csvData)
	if err != nil {
		t.Fatalf("ImportCSV failed: %v", err)
	}

	// Should import 2 valid trades, ignoring comments and empty lines
	if report.Imported != 2 {
		t.Errorf("expected 2 imported trades, got %d", report.Imported)
	}

	if len(report.Errors) > 0 {
		t.Errorf("unexpected errors: %v", report.Errors)
	}

	// Verify trades were imported
	trades, err := mockRepo.List(ctx, models.Query{})
	if err != nil {
		t.Fatalf("Failed to list trades: %v", err)
	}

	if len(trades) != 2 {
		t.Fatalf("expected 2 trades in repo, got %d", len(trades))
	}
}