-- Create trades table with proper SQLite syntax
CREATE TABLE IF NOT EXISTS trades (
    id TEXT PRIMARY KEY,
    symbol TEXT NOT NULL,
    side TEXT NOT NULL,
    entry_time DATETIME NOT NULL,
    exit_time DATETIME,
    entry_price REAL NOT NULL,
    exit_price REAL,
    qty REAL NOT NULL,
    fees REAL DEFAULT 0.0,
    notes TEXT DEFAULT '',
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);

-- Create indexes for query performance
CREATE INDEX IF NOT EXISTS idx_trades_symbol ON trades(symbol);
CREATE INDEX IF NOT EXISTS idx_trades_side ON trades(side);
CREATE INDEX IF NOT EXISTS idx_trades_entry_time ON trades(entry_time);
CREATE INDEX IF NOT EXISTS idx_trades_exit_time ON trades(exit_time);
CREATE INDEX IF NOT EXISTS idx_trades_symbol_entry_time ON trades(symbol, entry_time);
CREATE INDEX IF NOT EXISTS idx_trades_closed ON trades(exit_time) WHERE exit_time IS NOT NULL;

-- Create unique index for deduplication
CREATE UNIQUE INDEX IF NOT EXISTS idx_trades_dedupe ON trades(
    symbol, 
    entry_time, 
    COALESCE(exit_time, ''), 
    entry_price, 
    qty
);