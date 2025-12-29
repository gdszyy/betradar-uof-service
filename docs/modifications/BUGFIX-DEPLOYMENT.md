# 🐛 Bug Fix Deployment Guide

## Issue Summary

**Critical Bug**: XML message parsing was failing, causing all messages to have empty `message_type` field.

### Root Cause
The XML parser was reading the first token (XML declaration `<?xml version="1.0"...?>`) instead of the root element. This caused:
- ❌ All messages stored with empty `message_type`
- ❌ Message type switch statement never matched any case
- ❌ Handler functions (`handleAlive`, `handleOddsChange`, etc.) never executed
- ❌ Specialized tables (`odds_changes`, `bet_stops`, `bet_settlements`, etc.) remained empty
- ❌ Event tracking never worked

### Impact
- **40,000+ messages** stored with empty `message_type`
- No data in specialized tables despite receiving messages
- WebSocket clients receiving messages but with incomplete metadata

### Fix Applied
Modified `services/amqp_consumer.go` line 249-261:

**Before (Broken)**:
```go
decoder := xml.NewDecoder(bytes.NewReader([]byte(xmlContent)))
token, _ := decoder.Token()
if startElement, ok := token.(xml.StartElement); ok {
    messageType = startElement.Name.Local
}
```

**After (Fixed)**:
```go
decoder := xml.NewDecoder(bytes.NewReader([]byte(xmlContent)))
// 循环读取token直到找到第一个StartElement(跳过XML声明等)
for {
    token, err := decoder.Token()
    if err != nil {
        break
    }
    if startElement, ok := token.(xml.StartElement); ok {
        messageType = startElement.Name.Local
        break
    }
}
```

### Test Results
All message types now parse correctly:
- ✅ `alive` messages
- ✅ `fixture_change` messages
- ✅ `odds_change` messages
- ✅ `bet_stop` messages
- ✅ `bet_settlement` messages
- ✅ All other message types

---

## Deployment Steps

### Option 1: Deploy via GitHub (Recommended)

If you've already published to GitHub:

1. **Commit and push the fix**:
   ```bash
   cd uof-go-service
   git add services/amqp_consumer.go
   git commit -m "Fix: XML parsing to correctly extract message type"
   git push origin main
   ```

2. **Railway will auto-deploy** (if connected to GitHub)
   - Go to Railway dashboard
   - Check deployment logs
   - Wait for build to complete

### Option 2: Deploy via Railway CLI

1. **Install Railway CLI** (if not already installed):
   ```bash
   npm i -g @railway/cli
   ```

2. **Login and link project**:
   ```bash
   railway login
   railway link
   ```

3. **Deploy**:
   ```bash
   cd /home/ubuntu/uof-go-service
   railway up
   ```

### Option 3: Manual Redeploy

1. **Create new deployment package**:
   ```bash
   cd /home/ubuntu
   tar -czf betradar-uof-go-service-fixed.tar.gz uof-go-service/
   ```

2. **Download and extract on your local machine**

3. **Push to your Git repository**

4. **Trigger Railway redeploy**

---

## Database Cleanup (Optional but Recommended)

The old 40,000+ messages have empty `message_type` and are not useful. You can clean them up:

### Method 1: Using the cleanup tool

```bash
cd /home/ubuntu/uof-go-service
export DATABASE_URL="your_database_url_here"
go run tools/cleanup_database.go
```

The tool will:
- Show statistics
- Ask for confirmation
- Delete messages with empty `message_type`
- Clean up related tables

### Method 2: Manual SQL cleanup

Connect to your Railway PostgreSQL database and run:

```sql
-- Check current state
SELECT 
    CASE WHEN message_type = '' OR message_type IS NULL THEN 'empty' ELSE 'valid' END as type_status,
    COUNT(*) as count
FROM uof_messages
GROUP BY type_status;

-- Delete messages with empty type
DELETE FROM uof_messages WHERE message_type = '' OR message_type IS NULL;

-- Clean up other tables (should be empty anyway)
DELETE FROM odds_changes;
DELETE FROM bet_stops;
DELETE FROM bet_settlements;
DELETE FROM tracked_events;
DELETE FROM producer_status;

-- Verify cleanup
SELECT COUNT(*) FROM uof_messages;
```

---

## Verification After Deployment

### 1. Check Service Logs

In Railway dashboard, check the logs for:
```
Started consuming messages
Auto recovery is enabled, triggering full recovery...
Odds change for event sr:match:xxxxx: X markets, status=...
Bet stop for event sr:match:xxxxx: market_status=...
```

### 2. Monitor Database

Run the data check tool:
```bash
cd /home/ubuntu/uof-go-service
export DATABASE_URL="your_database_url_here"
go run tools/check_data.go
```

You should see:
- ✅ `uof_messages` with valid `message_type` values
- ✅ `odds_changes` table filling up
- ✅ `bet_stops` table filling up
- ✅ `bet_settlements` table filling up
- ✅ `tracked_events` table filling up
- ✅ `producer_status` table updating

### 3. Test API Endpoints

```bash
# Get recent messages (should show message_type)
curl https://your-service.railway.app/api/messages

# Get tracked events (should show events)
curl https://your-service.railway.app/api/events

# Get specific event messages
curl https://your-service.railway.app/api/events/sr:match:12345/messages
```

### 4. Test WebSocket

Open the web interface:
```
https://your-service.railway.app/
```

You should see:
- ✅ Messages with correct `message_type` displayed
- ✅ Event tracking working
- ✅ Real-time updates showing parsed data

---

## Expected Behavior After Fix

### Message Flow
1. **AMQP receives message** → XML with declaration
2. **Parser extracts type** → `alive`, `odds_change`, `bet_stop`, etc.
3. **Base message stored** → `uof_messages` table with correct type
4. **Type-specific handler called** → Based on switch statement
5. **Specialized data stored** → `odds_changes`, `bet_stops`, etc.
6. **Event tracking updated** → `tracked_events` table
7. **WebSocket broadcast** → Clients receive full data

### Database Tables

**uof_messages**:
- All messages with correct `message_type`
- `event_id`, `product_id`, `sport_id` properly extracted
- Full XML content preserved

**odds_changes**:
- One row per odds_change message
- `event_id`, `product_id`, `timestamp` populated
- `markets_count` showing number of markets
- Full XML content for detailed parsing

**bet_stops**:
- One row per bet_stop message
- Event and timing information

**bet_settlements**:
- One row per bet_settlement message
- Settlement results preserved

**tracked_events**:
- One row per unique event
- `message_count` incrementing
- `last_message_at` updating

**producer_status**:
- One row per product (1=liveodds, 3=pre)
- Status 'online'
- `last_alive` timestamp updating every 10 seconds

---

## Rollback Plan

If issues occur after deployment:

1. **Check Railway logs** for errors
2. **Revert to previous deployment** in Railway dashboard
3. **Report issues** with log excerpts

The fix is minimal and well-tested, so rollback should not be necessary.

---

## Support

If you encounter any issues:

1. **Check logs first**: Railway dashboard → Deployments → Logs
2. **Run diagnostic tools**: `db_diagnostic.go`, `check_data.go`, `examine_messages.go`
3. **Verify environment variables**: All required vars set in Railway

---

## Summary

✅ **Bug identified**: XML parsing skipped root element  
✅ **Fix applied**: Loop through tokens to find StartElement  
✅ **Testing complete**: All message types parse correctly  
✅ **Deployment ready**: Code ready for Railway deployment  
✅ **Cleanup available**: Tool to remove old invalid data  
✅ **Verification tools**: Scripts to confirm fix is working  

**Next Step**: Deploy to Railway and monitor for 5-10 minutes to confirm data is flowing correctly.

