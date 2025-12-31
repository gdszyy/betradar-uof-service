-- Migration: Add live_odds field to tracked_events table
-- Date: 2025-12-31
-- Purpose: Store liveodds status from Sportradar API (booked/bookable/not_available)

-- Add live_odds column
ALTER TABLE tracked_events ADD COLUMN IF NOT EXISTS live_odds VARCHAR(50);

-- Create index for better query performance
CREATE INDEX IF NOT EXISTS idx_tracked_events_live_odds ON tracked_events(live_odds);

-- Update existing records based on subscribed status
-- Note: This is a best-effort migration. Actual values should be synced from Sportradar API.
UPDATE tracked_events 
SET live_odds = CASE 
    WHEN subscribed = true THEN 'booked'
    ELSE NULL
END
WHERE live_odds IS NULL;

-- Verify migration
SELECT 
    live_odds, 
    COUNT(*) as count,
    COUNT(*) * 100.0 / (SELECT COUNT(*) FROM tracked_events) as percentage
FROM tracked_events
GROUP BY live_odds
ORDER BY count DESC;
