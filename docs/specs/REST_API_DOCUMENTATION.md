# Betradar UOF Service REST API Documentation

This document provides a comprehensive guide to the business-related REST API endpoints available in the Betradar UOF Service.

## Table of Contents
1. [Event & Match APIs](#event--match-apis)
2. [Odds & Market APIs](#odds--market-apis)
3. [Leagues & Categories APIs](#leagues--categories-apis)
4. [Booking & Subscription APIs](#booking--subscription-apis)
5. [Recovery & Replay APIs](#recovery--replay-apis)
6. [System & Monitoring APIs](#system--monitoring-apis)

---

## Event & Match APIs

### Get Events (Unified Filter)
Returns a list of events based on various filters.
- **URL**: `/api/events`
- **Method**: `GET`
- **Query Parameters**:
  - `is_live` (boolean): Filter by live status.
  - `status` (string): Filter by event status (e.g., `active`).
  - `sport_id` (string): Comma-separated sport IDs.
  - `start_time_from` (ISO8601): Filter by start time.
  - `start_time_to` (ISO8601): Filter by end time.
  - `team_name` (string): Search by team name.
  - `tournament_id` (string): Comma-separated tournament IDs.
  - `search` (string): Search by team name or event ID.
  - `popular` (boolean): Filter by popularity.

### Get Live Matches
Returns currently live matches.
- **URL**: `/api/matches/live`
- **Method**: `GET`

### Get Upcoming Matches
Returns upcoming matches within a specified timeframe.
- **URL**: `/api/matches/upcoming`
- **Method**: `GET`
- **Query Parameters**:
  - `hours` (int): Timeframe in hours (default: 24).

### Get Match Detail
Returns detailed information for a specific match.
- **URL**: `/api/matches/{event_id}`
- **Method**: `GET`

### Get Event Detail (Comprehensive)
Returns full event details including all markets, specifiers, and outcomes.
- **URL**: `/api/events/{event_id}`
- **Method**: `GET`

---

## Odds & Market APIs

### Get All Booked Markets Odds
Returns current odds for all markets of all booked matches.
- **URL**: `/api/odds/all`
- **Method**: `GET`

### Get Event Markets
Returns all available markets for a specific event.
- **URL**: `/api/odds/{event_id}/markets`
- **Method**: `GET`

### Get Market Odds
Returns current odds for a specific market in an event.
- **URL**: `/api/odds/{event_id}/{market_id}`
- **Method**: `GET`

### Get Odds History
Returns historical odds changes for a specific outcome.
- **URL**: `/api/odds/{event_id}/{market_id}/{outcome_id}/history`
- **Method**: `GET`
- **Query Parameters**:
  - `limit` (int): Number of records to return (default: 50, max: 200).

### Get Market Tabs Configuration
Returns the configuration for market grouping (tabs and chips).
- **URL**: `/api/config/market-tabs`
- **Method**: `GET`

---

## Leagues & Categories APIs

### Get Categories
Returns a list of sport categories (countries/regions).
- **URL**: `/api/categories`
- **Method**: `GET`
- **Query Parameters**:
  - `page`, `page_size` (int): Pagination parameters.

### Get Tournaments
Returns a list of tournaments (leagues) for a specific category.
- **URL**: `/api/tournaments`
- **Method**: `GET`
- **Query Parameters**:
  - `category_id` (string): Required. The category ID.
  - `sort` (string): Sorting criteria (`name`, `match_count_asc`, `match_count_desc`, `popularity_desc`).

---

## Booking & Subscription APIs

### Get Booked Matches
Returns a list of matches currently booked in the Betradar system.
- **URL**: `/api/booking/booked`
- **Method**: `GET`

### Get Bookable Matches
Returns a list of matches available for booking.
- **URL**: `/api/booking/bookable`
- **Method**: `GET`

### Trigger Auto Booking
Manually triggers the auto-booking process for live matches.
- **URL**: `/api/booking/trigger`
- **Method**: `POST`

### Book Specific Match
Books a specific match by ID.
- **URL**: `/api/booking/match/{match_id}`
- **Method**: `POST`

### Sync Subscriptions
Synchronizes the local subscription state with Betradar.
- **URL**: `/api/booking/sync`
- **Method**: `POST`

---

## Recovery & Replay APIs

### Trigger Full Recovery
Triggers a full recovery of all messages from Betradar.
- **URL**: `/api/recovery/trigger`
- **Method**: `POST`

### Trigger Event Recovery
Triggers recovery for a specific event.
- **URL**: `/api/recovery/event/{event_id}`
- **Method**: `POST`
- **Query Parameters**:
  - `product` (string): The product to recover (default: `liveodds`).

### Get Recovery Status
Returns the status of recent recovery requests.
- **URL**: `/api/recovery/status`
- **Method**: `GET`

### Start Replay
Starts a replay test for a specific event.
- **URL**: `/api/replay/start`
- **Method**: `POST`
- **Body**:
  ```json
  {
    "event_id": "sr:match:12345",
    "speed": 20,
    "duration": 300
  }
  ```

### Stop Replay
Stops the current replay test.
- **URL**: `/api/replay/stop`
- **Method**: `POST`

---

## System & Monitoring APIs

### Get Producer Status
Returns the health status of all Betradar producers.
- **URL**: `/api/producer/status`
- **Method**: `GET`

### Get Bet Acceptance Status
Checks if the system is currently in a state to accept bets.
- **URL**: `/api/producer/bet-acceptance`
- **Method**: `GET`

### Get Message History
Returns recent UOF messages.
- **URL**: `/api/messages/recent`
- **Method**: `GET`
- **Query Parameters**:
  - `limit` (int): Number of messages.
  - `type` (string): Filter by message type (e.g., `odds_change`).
