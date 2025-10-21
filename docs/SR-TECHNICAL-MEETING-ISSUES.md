# Sportradar UOF Technical Issue

**Date**: 2025-10-21  
**Account**: afunmts982  
**Environment**: Production (stgmq.betradar.com)

---

## Issue: Replay API - Playlist Empty on Play

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

3. **Key Details from Documentation**:
   - ✓ Used `x-access-token` header (not Basic Auth)
   - ✓ Added `node_id` parameter for routing
   - ✓ Verified event in playlist before Play
   - ✓ Aware of 48-hour rule (only events >48h old are replayable)

### Observed Behavior

```
08:19:30  PUT /replay/events/test:match:21797788  → 200 OK
08:19:33  GET /replay/                            → Contains test:match:21797788 ✓
08:19:33  POST /replay/play                       → 400 "Playlist is empty" ✗
```

**Timeline**: 3 seconds between verify and play. Event confirmed in playlist, but play fails immediately after.

### Actual Logs

```
2025/10/21 08:19:29 🔄 Resetting replay...
2025/10/21 08:19:29 [ReplayClient] Making POST request to /replay/reset
2025/10/21 08:19:29 ✅ Replay reset successfully

2025/10/21 08:19:29 ➕ Adding event test:match:21797788...
2025/10/21 08:19:29 [ReplayClient] Making PUT request to /replay/events/test:match:21797788
2025/10/21 08:19:30 ✅ Event test:match:21797788 added successfully

2025/10/21 08:19:30 ⏳ Waiting for event to be added to playlist...

2025/10/21 08:19:33 📋 Listing replay events...
2025/10/21 08:19:33 [ReplayClient] Making GET request to /replay/
2025/10/21 08:19:33 ✅ Event test:match:21797788 confirmed in playlist

2025/10/21 08:19:33 ▶️  Starting replay...
2025/10/21 08:19:33 [ReplayClient] Making POST request to /replay/play?speed=30&node_id=1
2025/10/21 08:19:33 ❌ Failed: API error (status 400): 
<response>
  <action>One or more parameter values are invalid</action>
  <message>Playlist is empty or not found.</message>
</response>
```

### Questions for Sportradar

1. **Event Validity**: Is `test:match:21797788` valid for Replay API?
   - Does it meet the 48-hour age requirement?
   - Is it AMQP-only or Replay-compatible?

2. **API Timing**: Is there a minimum wait time between AddEvent and Play?
   - Our 3-second wait seems insufficient
   - Should we poll GET /replay/ until certain condition?

3. **Test Events**: Can you provide a list of currently available test events?
   - Which `test:match:*` IDs are guaranteed to work?
   - Any recommended events for testing?

4. **Debugging**: Is there server-side logging we can check?
   - Why does the event disappear between verify and play?
   - Any internal state we should be aware of?

---

## Implementation Details

### HTTP Client
```go
func (r *ReplayClient) doRequest(method, path string, body io.Reader) ([]byte, error) {
    url := r.baseURL + path  // https://api.betradar.com/v1
    req, _ := http.NewRequest(method, url, body)
    req.Header.Set("x-access-token", r.accessToken)
    
    resp, err := r.client.Do(req)
    // ... error handling
}
```

### Replay Flow
```go
func QuickReplay(eventID string, speed int, nodeID int) error {
    // 1. Reset
    POST /replay/reset
    
    // 2. Add event
    PUT /replay/events/{eventID}
    
    // 3. Wait and verify (retry up to 5 times)
    for i := 0; i < 5; i++ {
        time.Sleep(2 * time.Second)
        eventsXML := GET /replay/
        if contains(eventsXML, eventID) {
            break
        }
    }
    
    // 4. Play
    POST /replay/play?speed={speed}&node_id={nodeID}
    
    // 5. Wait for ready
    for {
        status := GET /replay/status
        if status != "SETTING_UP" {
            break
        }
    }
}
```

---

## Request

Please help us understand:
1. Why the event disappears from playlist between verify and play
2. The correct API flow and timing requirements
3. Which test events are currently available and working

**Priority**: High - Blocking our Replay testing capability

