-- ============================================================================
-- Migration 018: Fix Player Nationality Field Length
-- ============================================================================
-- Issue: The nationality field in the players table was defined as VARCHAR(10),
-- but SportRader API returns full country names (e.g., "United States", 
-- "South Africa") which can exceed 10 characters.
-- 
-- Solution: Extend the nationality field to VARCHAR(100) to accommodate
-- full country names from the SportRader API.
-- ============================================================================

-- Alter the players table to increase nationality field length
ALTER TABLE players 
ALTER COLUMN nationality TYPE VARCHAR(100);

-- Add a comment to document the change
COMMENT ON COLUMN players.nationality IS 'Country name from SportRader API (e.g., "United States", "South Africa")';
