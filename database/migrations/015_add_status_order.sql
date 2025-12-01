-- ============================================================================
-- Migration 015: Add status_order field to tracked_events
-- ============================================================================
-- Date: 2025-12-01
-- Author: System
-- Issue: Fix odds_change message processing failure
-- 
-- Problem:
--   odds_change messages were not updating tracked_events table because
--   the SQL statements referenced a non-existent status_order field.
--   This caused all live match status updates to fail silently.
--
-- Solution:
--   Add status_order field to enable proper status progression tracking
--   and fix odds_change message processing.
--
-- Status Order Values:
--   0  = unknown
--   10 = not_started, postponed, cancelled, abandoned
--   20 = suspended, interrupted, delayed
--   30 = live
--   40 = ended
--   50 = closed
-- ============================================================================

BEGIN;

-- Add status_order column
ALTER TABLE tracked_events 
ADD COLUMN IF NOT EXISTS status_order INTEGER DEFAULT 0;

-- Create index for better query performance
CREATE INDEX IF NOT EXISTS idx_tracked_events_status_order 
ON tracked_events(status_order);

-- Add column comment
COMMENT ON COLUMN tracked_events.status_order IS 
'Numeric order for status progression: 0=unknown, 10=not_started/postponed/cancelled/abandoned, 20=suspended/interrupted/delayed, 30=live, 40=ended, 50=closed. Used to ensure status only moves forward.';

-- Update existing records based on current status
UPDATE tracked_events 
SET status_order = CASE 
    WHEN status = 'closed' THEN 50
    WHEN status = 'ended' THEN 40
    WHEN status = 'live' THEN 30
    WHEN status IN ('suspended', 'interrupted', 'delayed') THEN 20
    WHEN status IN ('not_started', 'postponed', 'cancelled', 'abandoned', 'scheduled') THEN 10
    ELSE 0
END
WHERE status_order = 0;

COMMIT;

-- Verification
SELECT 
    'Migration 015 completed successfully' as message,
    COUNT(*) as total_records,
    COUNT(CASE WHEN status_order > 0 THEN 1 END) as records_with_order
FROM tracked_events;

-- Show status distribution
SELECT status, status_order, COUNT(*) as count
FROM tracked_events
GROUP BY status, status_order
ORDER BY status_order DESC, count DESC;
