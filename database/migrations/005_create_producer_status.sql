-- Migration 005: Create producer_status table
-- Purpose: Track producer health status for monitoring and bet acceptance checks

CREATE TABLE IF NOT EXISTS producer_status (
    product_id INTEGER PRIMARY KEY,
    status VARCHAR(20) NOT NULL DEFAULT 'online',
    last_alive BIGINT NOT NULL,
    last_alive_at TIMESTAMP,
    subscribed INTEGER NOT NULL DEFAULT 1,
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create index for faster queries
CREATE INDEX IF NOT EXISTS idx_producer_status_last_alive 
ON producer_status(last_alive);

CREATE INDEX IF NOT EXISTS idx_producer_status_last_alive_at 
ON producer_status(last_alive_at);

-- Add comments
COMMENT ON TABLE producer_status IS 'Tracks producer health status from alive messages';
COMMENT ON COLUMN producer_status.product_id IS 'Producer/Product ID from UOF';
COMMENT ON COLUMN producer_status.status IS 'Producer status (online/offline)';
COMMENT ON COLUMN producer_status.last_alive IS 'Last alive timestamp (Unix epoch in milliseconds)';
COMMENT ON COLUMN producer_status.last_alive_at IS 'Last alive timestamp (PostgreSQL timestamp)';
COMMENT ON COLUMN producer_status.subscribed IS 'Subscription status (1=subscribed, 0=cancelled)';

