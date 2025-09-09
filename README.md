## Trading Journal – Take‑Home Assignment Scaffold

This repo is a starter for a Wails v3 app (Go backend + Vite/JS frontend) used as a take‑home assignment. It compiles and runs. A few tests intentionally fail and a handful of TODOs are left for you to complete.

### Tech

- Go 1.22+
- Wails v3 alpha 12
- SQLite (pure Go driver `modernc.org/sqlite`)
- Frontend: vanilla JS + Vite (minimal shell)

### Project layout

- `backend/` bindings, repo (SQLite), services (analytics), models
- `migrations/` SQLite schema and indexes
- `frontend/` minimal UI and generated Wails bindings
- `testdata/` CSV fixtures (tiny + large)
- `scripts/` verify, data generator
- `.github/workflows/ci.yml` CI pipeline

### Quickstart

- Build once: `wails3 build`
- Dev mode: `wails3 dev`
- Verify locally: `scripts/verify.sh`

### What you need to implement

1. Analytics (backend/services/analytics.go)

- Implement O(n) percent‑based MaxDrawdown (peak‑tracking). Current naive absolute DD is intentional; tests expect percent‑based.
- Validate or adjust ProfitFactor, Expectancy, Sharpe, Sortino (Sortino uses downside deviation only).

2. Precision

- Switch P&L aggregation from float64 to a decimal library (e.g., shopspring/decimal). Keep JSON shapes numeric; convert at the edge.

3. CSV import (backend/bindings/journal_service.go)

- Deduplicate on key: (symbol, entry_time, exit_time, entry_price, qty).
- Ensure all timestamps are stored in UTC.
- Return correct ImportReport counts.

4. Trades listing + filters

- Add a UI table backed by `ListTrades(ctx, q)` (already exposed) and wire the existing filters to refresh analytics + table.

5. Equity chart

- Render a minimalist equity line chart. Provide `equityPoints: Array<{t:string; v:number}>`.

6. Timezone test

- Add/adjust a unit test that ensures storage is UTC (import converts local offsets to UTC).

7. Make CI green

- The repo includes a failing test by design (percent‑based MaxDD). Implement it and ensure CI passes: vet, lint, tests, frontend build.

See `REVIEW_RUBRIC.md` for scoring criteria.

### Commands

- Run app: `wails3 dev`
- Build: `wails3 build`
- Tests: `go test ./...`
- Generate large CSV: `python3 scripts/gen_large_csv.py --rows 5000`
- Verify: `bash scripts/verify.sh`

### Notes

- Fixtures: `testdata/sample_trades.csv` (tiny) and `testdata/large_trades.csv` (generated).
- Intentional gotchas: naive O(n²) MaxDD, float precision, UTC enforcement, dedupe rules.
