-- =====================================================
-- Event (Match) Popularity Scoring Algorithm - SQL Implementation
-- Version: 1.1 (Adapted for betradar-uof-service)
-- Author: Manus AI
-- Date: 2025-12-01
-- =====================================================

-- This script calculates and updates the popularity_score field in tracked_events table.
-- Adapted from mts-service to work with betradar-uof-service database schema.

-- Note: Unlike mts-service which uses a separate event_popularity_scores table,
-- this version updates the popularity_score field directly in tracked_events table.

-- Step 1: Calculate and update event popularity scores
-- =====================================================

WITH 
-- 1.1 Calculate active market count for each event
-- Only consider events that are NOT ended and NOT cancelled
-- And only count markets that are NOT 'cancelled' or 'inactive'
event_active_market_counts AS (
    SELECT 
        te.event_id,
        COUNT(m.id) as active_market_count
    FROM tracked_events te
    LEFT JOIN markets m ON te.event_id = m.event_id AND m.status NOT IN ('cancelled', 'inactive', 'suspended')
    WHERE te.status NOT IN ('ended', 'closed', 'cancelled', 'abandoned')
    GROUP BY te.event_id
),

-- 1.2 Assign scores based on active market count
scored_events AS (
    SELECT 
        eamc.event_id,
        eamc.active_market_count,
        CASE 
            WHEN eamc.active_market_count > 400 THEN 10
            WHEN eamc.active_market_count > 300 THEN 9
            WHEN eamc.active_market_count > 200 THEN 8
            WHEN eamc.active_market_count > 150 THEN 7
            WHEN eamc.active_market_count > 100 THEN 6
            WHEN eamc.active_market_count > 50 THEN 5
            WHEN eamc.active_market_count > 30 THEN 4
            WHEN eamc.active_market_count > 10 THEN 3
            WHEN eamc.active_market_count > 5 THEN 2
            ELSE 1
        END as popularity_score
    FROM event_active_market_counts eamc
)

-- 1.3 Update the popularity_score in tracked_events table
-- Reset popularity_score to 0 for ended/cancelled matches
UPDATE tracked_events te
SET 
    popularity_score = CASE 
        WHEN te.status IN ('ended', 'closed', 'cancelled', 'abandoned') THEN 0
        ELSE COALESCE(se.popularity_score, 0)
    END,
    updated_at = CURRENT_TIMESTAMP
FROM (SELECT event_id FROM tracked_events) all_events
LEFT JOIN scored_events se ON all_events.event_id = se.event_id
WHERE te.event_id = all_events.event_id;

-- =====================================================
-- End of Script
-- =====================================================

-- Query to verify results:
-- SELECT event_id, home_team, away_team, popularity_score 
-- FROM tracked_events 
-- WHERE popularity_score > 0
-- ORDER BY popularity_score DESC 
-- LIMIT 10;

-- Query to check score distribution:
-- SELECT 
--     popularity_score,
--     COUNT(*) as count,
--     ROUND(COUNT(*) * 100.0 / SUM(COUNT(*)) OVER (), 2) as percentage
-- FROM tracked_events
-- WHERE popularity_score > 0
-- GROUP BY popularity_score
-- ORDER BY popularity_score DESC;
