# Betradar UOF Service WebSocket API Documentation

The WebSocket API provides real-time updates for odds changes, bet stops, settlements, and other UOF messages.

## Connection
- **URL**: `ws://{host}/ws`
- **Protocol**: Standard WebSocket

## Message Format
All messages exchanged are in JSON format.

### Server to Client (Broadcast)
The server pushes updates to connected clients.
```json
{
  "type": "uof_message",
  "message_type": "odds_change",
  "event_id": "sr:match:12345",
  "product_id": 1,
  "timestamp": 1672531200000,
  "data": { ... },
  "xml": "<xml_content_if_available>"
}
```

### Client to Server (Commands)

#### Subscribe
Clients can filter the messages they receive by subscribing to specific types, events, or markets.
```json
{
  "type": "subscribe",
  "message_types": ["odds_change", "bet_stop"],
  "event_ids": ["sr:match:12345", "sr:match:67890"],
  "market_ids": [1, 18, 52]
}
```
- `message_types`: Array of UOF message types (e.g., `odds_change`, `bet_stop`, `bet_settlement`, `fixture_change`).
- `event_ids`: Array of Betradar event IDs.
- `market_ids`: Array of market IDs (integers).

#### Unsubscribe
Clears all active filters for the connection.
```json
{
  "type": "unsubscribe"
}
```

## Heartbeat
The server sends a `ping` frame every 20 seconds. Clients should respond with a `pong` frame to keep the connection alive. The connection will be closed if no activity is detected for 120 seconds.

## Message Types
Common `message_type` values include:
- `odds_change`: Real-time odds updates.
- `bet_stop`: Notification to stop accepting bets for specific markets or the entire event.
- `bet_settlement`: Final results and settlement information.
- `bet_cancel`: Cancellation of previously sent bets.
- `fixture_change`: Updates to match schedules or metadata.
- `rollback_bet_settlement`: Reversal of a previous settlement.
- `rollback_bet_cancel`: Reversal of a previous cancellation.
