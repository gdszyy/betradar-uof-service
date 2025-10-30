package services

import (
"os"
"database/sql"
"encoding/xml"
"fmt"
"log"
)

// BetCancelProcessor Bet Cancel 消息处理器
type BetCancelProcessor struct {
db     *sql.DB
logger *log.Logger
}

// BetCancelMessage Bet Cancel 消息结构
type BetCancelMessage struct {
XMLName      xml.Name `xml:"bet_cancel"`
EventID      string   `xml:"event_id,attr"`
ProductID    int      `xml:"product,attr"`
Timestamp    int64    `xml:"timestamp,attr"`
StartTime    *int64   `xml:"start_time,attr"`
EndTime      *int64   `xml:"end_time,attr"`
SupercededBy *string  `xml:"superceded_by,attr"`
Market       []struct {
ID         string `xml:"id,attr"`
Specifiers string `xml:"specifiers,attr"`
VoidReason *int   `xml:"void_reason,attr"`
} `xml:"market"`
}

// NewBetCancelProcessor 创建 Bet Cancel 处理器
func NewBetCancelProcessor(db *sql.DB) *BetCancelProcessor {
return &BetCancelProcessor{
db:     db,
logger: log.New(os.Stdout, "", log.LstdFlags),
}
}

// ProcessBetCancel 处理 Bet Cancel 消息并更新 market status
func (p *BetCancelProcessor) ProcessBetCancel(xmlContent string) error {
var betCancel BetCancelMessage
if err := xml.Unmarshal([]byte(xmlContent), &betCancel); err != nil {
return fmt.Errorf("failed to parse bet_cancel message: %w", err)
}

// 开始事务
tx, err := p.db.Begin()
if err != nil {
return fmt.Errorf("failed to begin transaction: %w", err)
}
defer tx.Rollback()

// 遍历所有市场
for _, market := range betCancel.Market {
// 存储到 bet_cancels 表
query := `
INSERT INTO bet_cancels (
event_id, producer_id, timestamp,
market_id, specifiers, void_reason,
start_time, end_time, superceded_by,
created_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
ON CONFLICT (event_id, market_id, specifiers, producer_id) 
DO UPDATE SET
void_reason = EXCLUDED.void_reason,
start_time = EXCLUDED.start_time,
end_time = EXCLUDED.end_time,
superceded_by = EXCLUDED.superceded_by,
timestamp = EXCLUDED.timestamp,
created_at = NOW()
`

_, err := tx.Exec(
query,
betCancel.EventID,
betCancel.ProductID,
betCancel.Timestamp,
market.ID,
market.Specifiers,
market.VoidReason,
betCancel.StartTime,
betCancel.EndTime,
betCancel.SupercededBy,
)
if err != nil {
return fmt.Errorf("failed to insert bet_cancel: %w", err)
}

// 更新当前 market 的 status 为 -4 (Cancelled)
updateQuery := `
UPDATE markets 
SET status = -4, updated_at = NOW()
WHERE event_id = $1 AND market_id = $2 AND specifiers = $3
`
_, err = tx.Exec(updateQuery, betCancel.EventID, market.ID, market.Specifiers)
if err != nil {
p.logger.Printf("Warning: failed to update market status to cancelled: %v", err)
}
}

// 提交事务
if err := tx.Commit(); err != nil {
return fmt.Errorf("failed to commit transaction: %w", err)
}

// 输出自然语言日志
p.logger.Printf("[bet_cancel] 比赛 %s 的 %d个市场已取消",
betCancel.EventID, len(betCancel.Market))

return nil
}
