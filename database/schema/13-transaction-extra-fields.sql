-- Migration: Add extra fields to transaction table for Mintscan-like explorer support
-- These fields are available from the Cosmos SDK TxResponse but were not previously saved.

-- Numeric error code (0 = success). More granular than the boolean 'success' column.
ALTER TABLE transaction ADD COLUMN IF NOT EXISTS code INTEGER DEFAULT 0;

-- The namespace for the error code (which module returned the error)
ALTER TABLE transaction ADD COLUMN IF NOT EXISTS codespace TEXT DEFAULT '';

-- Hex-encoded result data returned by the transaction
ALTER TABLE transaction ADD COLUMN IF NOT EXISTS data TEXT;

-- Additional information about the transaction
ALTER TABLE transaction ADD COLUMN IF NOT EXISTS info TEXT DEFAULT '';

-- Transaction timestamp from the TxResponse
ALTER TABLE transaction ADD COLUMN IF NOT EXISTS timestamp TIMESTAMP WITHOUT TIME ZONE;

-- Raw events emitted by the transaction (different from logs)
ALTER TABLE transaction ADD COLUMN IF NOT EXISTS events JSONB;

-- Timeout height from the transaction body
ALTER TABLE transaction ADD COLUMN IF NOT EXISTS timeout_height BIGINT DEFAULT 0;

-- Extension options from the transaction body (JSON array)
ALTER TABLE transaction ADD COLUMN IF NOT EXISTS extension_options JSONB DEFAULT '[]'::JSONB;

-- Non-critical extension options from the transaction body (JSON array)  
ALTER TABLE transaction ADD COLUMN IF NOT EXISTS non_critical_extension_options JSONB DEFAULT '[]'::JSONB;

-- Tip information from AuthInfo (if applicable)
ALTER TABLE transaction ADD COLUMN IF NOT EXISTS tip JSONB;

-- Full raw JSON of the transaction (for explorer raw view like Mintscan)
ALTER TABLE transaction ADD COLUMN IF NOT EXISTS raw_json JSONB;

-- Create index on timestamp for time-based queries
CREATE INDEX IF NOT EXISTS transaction_timestamp_index ON transaction (timestamp);

-- Create index on code for filtering by error status
CREATE INDEX IF NOT EXISTS transaction_code_index ON transaction (code);
