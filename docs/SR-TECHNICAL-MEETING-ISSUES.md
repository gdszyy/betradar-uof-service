# Sportradar UOF Technical Issues

**Date**: 2025-10-21  
**Account**: afunmts982  
**Environment**: Production (stgmq.betradar.com)

---

## Issue #1: Replay API - Playlist Empty on Play

### What We Did

1. **API Calls** (按文档顺序):
   ```
   POST /replay/reset                              → 200 OK
   PUT  /replay/events/test:match:21797788         → 200 OK
   GET  /replay/                                   → 200 OK (event in list ✓)
   POST /replay/play?speed=30&node_id=1            → 400 "Playlist is empty"
   ```

2. **Timing**:
   - Reset → Add: immediate
   - Add → Verify: 3 seconds wait
   - Verify → Play: <1 second
   - Retried with 5-10 seconds wait: same result

3. **Key Details from Docs**:
   - ✓ Used `x-access-token` header (not Basic Auth)
   - ✓ Added `node_id` parameter
   - ✓ Verified event in playlist before Play
   - ✓ Aware of 48-hour rule (only events >48h old are replayable)

### Observed Behavior

```
08:19:30  PUT /replay/events/test:match:21797788  → 200 OK
08:19:33  GET /replay/                            → Contains test:match:21797788 ✓
08:19:33  POST /replay/play                       → 400 "Playlist is empty" ✗
```

**Gap**: 3 seconds between verify and play, event confirmed but play fails.

### Questions for SR

1. Is `test:match:21797788` valid for Replay API? (may be AMQP-only)
2. Is there a minimum wait time between AddEvent and Play?
3. Can you provide a list of currently available test events?
4. Is there server-side logging we can check?

---

## Issue #2: Recovery API - Rate Limiting

### What We Did

**Auto-recovery on startup**:
```
POST /liveodds/recovery/initiate_request?request_id=X&node_id=1
POST /pre/recovery/initiate_request?request_id=Y&node_id=1
```

**Result**: 403 Forbidden after multiple restarts
```xml
<response response_code="FORBIDDEN">
  <action>Too many requests. Limits are:
    - 4 requests per 120 minutes [Recovery length: 1440 minutes]
    - 2 requests per 30 minutes [Recovery length: 1440 minutes]
    - 4 requests per 10 minutes [Recovery length: 30 minutes]
    ...
  </action>
</response>
```

### Our Solution

- Detect 403 rate limit errors
- Schedule async retry (15min, then 30min)
- Don't block service startup

### Questions for SR

1. Are the 6 limits evaluated independently or in combination?
2. What's the recommended recovery frequency for production?
3. Should we trigger recovery on every service restart?
4. Is there an API to query remaining quota?

---

## Issue #3: Replay Messages Not Received

### Current Setup

```
Our Service → stgmq.betradar.com:5671 (Production AMQP)
Replay API  → global.replaymq.betradar.com:5671 (Replay AMQP)
```

**Result**: Replay API calls succeed, but we don't receive replay messages.

### Questions for SR

1. Confirm: Replay messages go to `global.replaymq.betradar.com` only?
2. Can we receive replay messages on production AMQP connection?
3. Does `node_id` affect message routing between connections?

---

## Current Implementation Status

### Working ✓
- AMQP message receiving (45,000+ messages processed)
- XML parsing (alive, fixture_change, odds_change, etc.)
- Recovery API with `request_id` and `node_id` tracking
- Replay API client (all endpoints implemented)
- Rate limit handling with smart retry

### Not Working ✗
- Replay playlist persistence (event disappears between verify and play)
- Receiving replay messages (architecture issue)

---

## Request

1. **Replay**: Provide working test event IDs or explain correct API flow
2. **Recovery**: Clarify rate limit rules and best practices
3. **AMQP**: Confirm replay message routing architecture

**Priority**: Replay API issue > AMQP architecture > Recovery rate limits

