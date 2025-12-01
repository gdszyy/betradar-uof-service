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
-- 1.1 Calculate market count for each event
event_market_counts AS (
    SELECT 
        te.event_id,
        COUNT(m.id) as market_count
    FROM tracked_events te
    LEFT JOIN markets m ON te.event_id = m.event_id
    GROUP BY te.event_id
),

-- 1.2 Assign scores based on market count
scored_events AS (
    SELECT 
        emc.event_id,
        emc.market_count,
        CASE 
            WHEN emc.market_count > 400 THEN 10
            WHEN emc.market_count > 300 THEN 9
            WHEN emc.market_count > 200 THEN 8
            WHEN emc.market_count > 150 THEN 7
            WHEN emc.market_count > 100 THEN 6
            WHEN emc.market_count > 50 THEN 5
            WHEN emc.market_count > 30 THEN 4
            WHEN emc.market_count > 10 THEN 3
            WHEN emc.market_count > 5 THEN 2
            ELSE 1
        END as popularity_score
    FROM event_market_counts emc
)

-- 1.3 Update the popularity_score in tracked_events table
UPDATE tracked_events te
SET 
    popularity_score = se.popularity_score,
    updated_at = CURRENT_TIMESTAMP
FROM scored_events se
WHERE te.event_id = se.event_id;

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
