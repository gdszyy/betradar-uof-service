-- Add live_odds column to tracked_events table
ALTER TABLE tracked_events ADD COLUMN IF NOT EXISTS live_odds VARCHAR(50);

-- Add index for performance
CREATE INDEX IF NOT EXISTS idx_tracked_events_live_odds ON tracked_events(live_odds);
