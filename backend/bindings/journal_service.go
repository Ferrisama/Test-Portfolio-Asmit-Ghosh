package bindings

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"changeme/backend/models"
	"changeme/backend/repo"
	"changeme/backend/services"
	"github.com/shopspring/decimal"
)

// JournalService exposes journal operations to the Wails frontend.
type JournalService struct {
	Repo       repo.TradeRepo
	AppVersion string
}

// Ping returns the app version to demonstrate an end-to-end call.
func (s *JournalService) Ping(ctx context.Context) (string, error) {
	if s.AppVersion == "" {
		return "dev", nil
	}
	return s.AppVersion, nil
}

// GetAnalytics calculates analytics for the given query with decimal precision.
func (s *JournalService) GetAnalytics(ctx context.Context, q models.Query) (models.AnalyticsSummary, error) {
	trades, err := s.Repo.List(ctx, q)
	if err != nil {
		return models.AnalyticsSummary{}, err
	}
	
	// Build returns slice using decimal precision
	returns := make([]float64, 0, len(trades))
	equityDecimal := decimal.Zero
	equity := make([]float64, 0, len(trades))
	
	for _, t := range trades {
		if t.ExitPrice != nil {
			// Calculate P&L using decimal precision
			entryPriceDecimal := decimal.NewFromFloat(t.EntryPrice)
			exitPriceDecimal := decimal.NewFromFloat(*t.ExitPrice)
			qtyDecimal := decimal.NewFromFloat(t.Quantity)
			feesDecimal := decimal.NewFromFloat(t.Fees)
			
			var pnlDecimal decimal.Decimal
			if strings.ToLower(t.Side) == "short" {
				pnlDecimal = entryPriceDecimal.Sub(exitPriceDecimal).Mul(qtyDecimal).Sub(feesDecimal)
			} else {
				pnlDecimal = exitPriceDecimal.Sub(entryPriceDecimal).Mul(qtyDecimal).Sub(feesDecimal)
			}
			
			pnl, _ := pnlDecimal.Float64()
			returns = append(returns, pnl)
			
			// Build equity curve
			equityDecimal = equityDecimal.Add(pnlDecimal)
			equityValue, _ := equityDecimal.Float64()
			equity = append(equity, equityValue)
		}
	}
	
	out := models.AnalyticsSummary{
		WinRate:      services.WinRate(returns),
		ProfitFactor: services.ProfitFactor(returns),
		MaxDD:        services.MaxDrawdown(equity),
		Sharpe:       services.Sharpe(returns),
		Sortino:      services.Sortino(returns),
		Expectancy:   services.Expectancy(returns),
	}
	return out, nil
}

// ImportCSV imports trades from a CSV payload string with deduplication and UTC conversion.
func (s *JournalService) ImportCSV(ctx context.Context, csvPayload string) (models.ImportReport, error) {
	r := csv.NewReader(strings.NewReader(csvPayload))
	r.FieldsPerRecord = -1
	r.Comment = '#' // Allow comments in CSV
	// Expect header: symbol,side,entry_time,exit_time,entry_price,exit_price,qty,fees,notes
	
	var report models.ImportReport
	duplicateSet := make(map[string]bool) // Track duplicates
	
	// Read and validate header
	header, err := r.Read()
	if err != nil {
		return report, fmt.Errorf("read header: %w", err)
	}
	
	// Validate expected header format
	expectedHeaders := []string{"symbol", "side", "entry_time", "exit_time", "entry_price", "exit_price", "qty", "fees", "notes"}
	if len(header) < len(expectedHeaders) {
		return report, fmt.Errorf("invalid header: expected at least %d columns, got %d", len(expectedHeaders), len(header))
	}
	
	lineNum := 1 // Start counting from data lines (header is line 0)
	
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		lineNum++
		
		if err != nil {
			report.Errors = append(report.Errors, fmt.Sprintf("line %d: %v", lineNum, err))
			continue
		}
		
		// Skip empty lines or comments
		if len(rec) == 0 || (len(rec) > 0 && strings.HasPrefix(strings.TrimSpace(rec[0]), "#")) {
			continue
		}
		
		if len(rec) < 9 {
			report.Errors = append(report.Errors, fmt.Sprintf("line %d: invalid record length, expected 9 fields, got %d", lineNum, len(rec)))
			continue
		}
		
		// Parse and validate symbol
		symbol := strings.TrimSpace(rec[0])
		if symbol == "" {
			report.Errors = append(report.Errors, fmt.Sprintf("line %d: empty symbol", lineNum))
			continue
		}
		
		// Parse and validate side
		side := strings.ToLower(strings.TrimSpace(rec[1]))
		if side != "long" && side != "short" {
			report.Errors = append(report.Errors, fmt.Sprintf("line %d: invalid side '%s', must be 'long' or 'short'", lineNum, rec[1]))
			continue
		}
		
		// Parse entry time and convert to UTC
		entryTime, err := time.Parse(time.RFC3339, strings.TrimSpace(rec[2]))
		if err != nil {
			report.Errors = append(report.Errors, fmt.Sprintf("line %d: bad entry_time '%s': %v", lineNum, rec[2], err))
			continue
		}
		entryTimeUTC := entryTime.UTC()
		
		// Parse exit time and convert to UTC
		var exitTimeUTC *time.Time
		exitTimeStr := strings.TrimSpace(rec[3])
		if exitTimeStr != "" {
			et, err := time.Parse(time.RFC3339, exitTimeStr)
			if err != nil {
				report.Errors = append(report.Errors, fmt.Sprintf("line %d: bad exit_time '%s': %v", lineNum, rec[3], err))
				continue
			}
			etUTC := et.UTC()
			exitTimeUTC = &etUTC
		}
		
		// Parse entry price
		entryPrice, err := strconv.ParseFloat(strings.TrimSpace(rec[4]), 64)
		if err != nil || entryPrice <= 0 {
			report.Errors = append(report.Errors, fmt.Sprintf("line %d: bad entry_price '%s'", lineNum, rec[4]))
			continue
		}
		
		// Parse exit price
		var exitPrice *float64
		exitPriceStr := strings.TrimSpace(rec[5])
		if exitPriceStr != "" && exitPriceStr != "0" {
			ep, err := strconv.ParseFloat(exitPriceStr, 64)
			if err != nil || ep <= 0 {
				report.Errors = append(report.Errors, fmt.Sprintf("line %d: bad exit_price '%s'", lineNum, rec[5]))
				continue
			}
			exitPrice = &ep
		}
		
		// Parse quantity
		qty, err := strconv.ParseFloat(strings.TrimSpace(rec[6]), 64)
		if err != nil || qty <= 0 {
			report.Errors = append(report.Errors, fmt.Sprintf("line %d: bad quantity '%s'", lineNum, rec[6]))
			continue
		}
		
		// Parse fees
		fees, err := strconv.ParseFloat(strings.TrimSpace(rec[7]), 64)
		if err != nil || fees < 0 {
			report.Errors = append(report.Errors, fmt.Sprintf("line %d: bad fees '%s'", lineNum, rec[7]))
			continue
		}
		
		// Create deduplication key: (symbol, entry_time, exit_time, entry_price, qty)
		exitTimeStr = ""
		if exitTimeUTC != nil {
			exitTimeStr = exitTimeUTC.Format(time.RFC3339)
		}
		
		dedupKey := fmt.Sprintf("%s|%s|%s|%s|%s", 
			symbol, 
			entryTimeUTC.Format(time.RFC3339), 
			exitTimeStr,
			fmt.Sprintf("%.8f", entryPrice), // Use fixed precision to avoid floating point issues
			fmt.Sprintf("%.8f", qty),
		)
		
		if duplicateSet[dedupKey] {
			report.Skipped++
			continue
		}
		duplicateSet[dedupKey] = true
		
		// Create trade object
		t := models.Trade{
			Symbol:     symbol,
			Side:       side,
			EntryTime:  entryTimeUTC,
			ExitTime:   exitTimeUTC,
			EntryPrice: entryPrice,
			ExitPrice:  exitPrice,
			Quantity:   qty,
			Fees:       fees,
			Notes:      strings.TrimSpace(rec[8]),
		}
		
		// Attempt to save trade
		if err := s.Repo.Upsert(ctx, t); err != nil {
			report.Errors = append(report.Errors, fmt.Sprintf("line %d: failed to save trade: %v", lineNum, err))
		} else {
			report.Imported++
		}
	}
	
	return report, nil
}

// ListTrades returns trades matching the query. Useful for populating the table in UI.
func (s *JournalService) ListTrades(ctx context.Context, q models.Query) ([]models.Trade, error) {
	if s.Repo == nil {
		return nil, fmt.Errorf("repo not configured")
	}
	return s.Repo.List(ctx, q)
}

// GetEquityPoints returns equity curve points for charting.
// This creates a proper cumulative P&L curve over time using only closed trades.
func (s *JournalService) GetEquityPoints(ctx context.Context, q models.Query) ([]models.EquityPoint, error) {
	trades, err := s.Repo.List(ctx, q)
	if err != nil {
		return nil, err
	}
	
	// Filter and sort closed trades by exit time
	type closedTrade struct {
		ExitTime time.Time
		PnL      decimal.Decimal
	}
	
	var closedTrades []closedTrade
	
	for _, t := range trades {
		// Only include closed trades (have exit time and exit price)
		if t.ExitTime != nil && t.ExitPrice != nil {
			// Calculate P&L using decimal precision
			entryPriceDecimal := decimal.NewFromFloat(t.EntryPrice)
			exitPriceDecimal := decimal.NewFromFloat(*t.ExitPrice)
			qtyDecimal := decimal.NewFromFloat(t.Quantity)
			feesDecimal := decimal.NewFromFloat(t.Fees)
			
			var pnlDecimal decimal.Decimal
			if strings.ToLower(t.Side) == "short" {
				pnlDecimal = entryPriceDecimal.Sub(exitPriceDecimal).Mul(qtyDecimal).Sub(feesDecimal)
			} else {
				pnlDecimal = exitPriceDecimal.Sub(entryPriceDecimal).Mul(qtyDecimal).Sub(feesDecimal)
			}
			
			closedTrades = append(closedTrades, closedTrade{
				ExitTime: *t.ExitTime,
				PnL:      pnlDecimal,
			})
		}
	}
	
	// Sort by exit time to create chronological equity curve
	sort.Slice(closedTrades, func(i, j int) bool {
		return closedTrades[i].ExitTime.Before(closedTrades[j].ExitTime)
	})
	
	// Build cumulative equity points
	points := make([]models.EquityPoint, 0, len(closedTrades))
	cumulativeEquity := decimal.Zero
	
	for _, trade := range closedTrades {
		cumulativeEquity = cumulativeEquity.Add(trade.PnL)
		equityValue, _ := cumulativeEquity.Float64()
		
		points = append(points, models.EquityPoint{
			Time:  trade.ExitTime.Format(time.RFC3339),
			Value: equityValue,
		})
	}
	
	return points, nil
}