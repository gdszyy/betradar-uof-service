-- =====================================================
-- Tournament (League) Popularity Scoring Algorithm - SQL Implementation
-- Version: 1.2 (Adapted for betradar-uof-service)
-- Author: Manus AI
-- Date: 2025-12-01
-- =====================================================

-- This script calculates and stores the popularity score for tournaments.
-- Adapted from mts-service to work with betradar-uof-service database schema.

-- Note: The tournament_popularity_scores table already exists in the database
-- (created by migration 014_create_tournament_popularity_scores.sql)

-- Step 1: Calculate and upsert tournament popularity scores
-- =====================================================

WITH 
-- 1.1 Calculate market count for each event
event_market_counts AS (
    SELECT 
        event_id,
        COUNT(id) as market_count
    FROM markets
    GROUP BY event_id
),

-- 1.2 Calculate statistics for each tournament
tournament_stats AS (
    SELECT 
        te.tournament_id,
        MAX(te.tournament_name) as tournament_name,
        MAX(te.category_id) as category_id,
        MAX(te.sport_id) as sport_id,
        COUNT(DISTINCT te.event_id) as match_count,
        COALESCE(SUM(emc.market_count), 0) as total_markets,
        COALESCE(AVG(emc.market_count), 0) as avg_markets,
        COUNT(DISTINCT CASE WHEN te.feature_match = TRUE THEN te.event_id END) as feature_match_count,
        COUNT(DISTINCT CASE WHEN te.sellout = TRUE THEN te.event_id END) as sellout_count,
        COALESCE(SUM(te.broadcasts_count), 0) as total_broadcasts,
        COALESCE(AVG(te.attendance), 0) as avg_attendance
    FROM tracked_events te
    LEFT JOIN event_market_counts emc ON te.event_id = emc.event_id
    WHERE te.tournament_id IS NOT NULL 
        AND te.tournament_id != ''
    GROUP BY te.tournament_id
    HAVING COUNT(DISTINCT te.event_id) > 0
),

-- 1.3 Calculate tier scores based on tournament characteristics
scored_tournaments AS (
    SELECT 
        ts.*,
        -- Calculate Tournament Tier Score (1-10) based on name patterns
        CASE 
            WHEN ts.tournament_name ILIKE '%World Cup%' OR ts.tournament_name ILIKE '%Olympics%' THEN 10
            WHEN ts.tournament_name ILIKE '%Champions League%' THEN 9
            WHEN ts.tournament_name ILIKE '%Premier League%' OR ts.tournament_name ILIKE '%NBA%' OR ts.tournament_name ILIKE '%UEFA%' THEN 8
            WHEN ts.tournament_name ILIKE '%International%' OR ts.tournament_name ILIKE '%Europa League%' THEN 7
            WHEN ts.tournament_name ILIKE '%Serie A%' OR ts.tournament_name ILIKE '%La Liga%' OR ts.tournament_name ILIKE '%Bundesliga%' THEN 7
            WHEN ts.avg_markets > 100 THEN 6
            WHEN ts.avg_markets > 50 THEN 5
            ELSE 4
        END as tier_score,
        
        -- Calculate Market Depth Score (1-10) based on average markets per match
        CASE 
            WHEN ts.avg_markets > 300 THEN 10
            WHEN ts.avg_markets > 200 THEN 9
            WHEN ts.avg_markets > 150 THEN 8
            WHEN ts.avg_markets > 100 THEN 7
            WHEN ts.avg_markets > 50 THEN 6
            WHEN ts.avg_markets > 30 THEN 5
            WHEN ts.avg_markets > 20 THEN 4
            WHEN ts.avg_markets > 10 THEN 3
            WHEN ts.avg_markets > 5 THEN 2
            ELSE 1
        END as market_score
    FROM tournament_stats ts
)

-- 1.4 Insert or update the final scores
INSERT INTO tournament_popularity_scores (
    tournament_id,
    tournament_name,
    category_id,
    sport_id,
    match_count,
    total_broadcasts,
    avg_attendance,
    feature_match_count,
    sellout_count,
    final_popularity_score,
    updated_at
)
SELECT 
    tournament_id,
    tournament_name,
    category_id,
    sport_id,
    match_count,
    total_broadcasts,
    ROUND(avg_attendance, 2),
    feature_match_count,
    sellout_count,
    -- Calculate the final weighted score (tier 50% + market depth 50%)
    ROUND((tier_score * 0.5 + market_score * 0.5), 2) as final_popularity_score,
    CURRENT_TIMESTAMP
FROM scored_tournaments
-- Use ON CONFLICT to perform an upsert operation
ON CONFLICT (tournament_id) 
DO UPDATE SET
    tournament_name = EXCLUDED.tournament_name,
    category_id = EXCLUDED.category_id,
    sport_id = EXCLUDED.sport_id,
    match_count = EXCLUDED.match_count,
    total_broadcasts = EXCLUDED.total_broadcasts,
    avg_attendance = EXCLUDED.avg_attendance,
    feature_match_count = EXCLUDED.feature_match_count,
    sellout_count = EXCLUDED.sellout_count,
    final_popularity_score = EXCLUDED.final_popularity_score,
    updated_at = CURRENT_TIMESTAMP;

-- =====================================================
-- End of Script
-- =====================================================

-- Query to verify results:
-- SELECT tournament_name, match_count, final_popularity_score 
-- FROM tournament_popularity_scores 
-- ORDER BY final_popularity_score DESC 
-- LIMIT 10;
