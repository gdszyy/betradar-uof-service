package models

import (
	"time"
)

// LDEvent Live Data 事件记录
type LDEvent struct {
	ID          uint      `gorm:"primaryKey"`
	CreatedAt   time.Time `gorm:"index"`
	
	// 事件标识
	UUID        string    `gorm:"uniqueIndex;size:36"`
	EventID     string    `gorm:"index;size:50"`  // 已弃用的 ID
	
	// 比赛信息
	MatchID     string    `gorm:"index;size:50"`
	SportID     int       `gorm:"index"`
	
	// 事件信息
	Type        int       `gorm:"index"`
	TypeName    string    `gorm:"size:100"`
	Info        string    `gorm:"type:text"`
	Side        string    `gorm:"size:10"`  // home/away/none
	
	// 时间信息
	MTime       string    `gorm:"size:10"`  // 比赛时间 MM:SS
	STime       int64     `gorm:"index"`    // Scout 输入时间戳
	
	// 比赛状态
	MatchStatus string    `gorm:"size:20"`
	T1Score     int
	T2Score     int
	
	// 球员信息
	Player1     string    `gorm:"size:100"`
	Player2     string    `gorm:"size:100"`
	ExtraInfo   string    `gorm:"type:text"`
	
	// 重要性标记
	IsImportant bool      `gorm:"index"`
}

// TableName 指定表名
func (LDEvent) TableName() string {
	return "ld_events"
}

// LDMatch Live Data 比赛记录
type LDMatch struct {
	ID          uint      `gorm:"primaryKey"`
	CreatedAt   time.Time `gorm:"index"`
	UpdatedAt   time.Time
	
	// 比赛标识
	MatchID     string    `gorm:"uniqueIndex;size:50"`
	SportID     int       `gorm:"index"`
	
	// 球队信息
	T1ID        string    `gorm:"size:50"`
	T2ID        string    `gorm:"size:50"`
	T1Name      string    `gorm:"size:200"`
	T2Name      string    `gorm:"size:200"`
	
	// 状态信息
	MatchStatus string    `gorm:"index;size:20"`
	MatchTime   string    `gorm:"size:10"`
	
	// 比分
	T1Score     int
	T2Score     int
	
	// 时间信息
	MatchDate   string    `gorm:"size:20"`
	StartTime   string    `gorm:"size:20"`
	
	// 其他信息
	CoverageType string   `gorm:"size:50"`
	DeviceID     string   `gorm:"size:50"`
	
	// 订阅状态
	Subscribed  bool      `gorm:"index"`
	LastEventAt *time.Time
}

// TableName 指定表名
func (LDMatch) TableName() string {
	return "ld_matches"
}

// LDLineup Live Data 阵容记录
type LDLineup struct {
	ID          uint      `gorm:"primaryKey"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	
	// 比赛标识
	MatchID     string    `gorm:"index;size:50"`
	
	// 球员信息 (JSON 存储)
	Team1Players string   `gorm:"type:text"`  // JSON array
	Team2Players string   `gorm:"type:text"`  // JSON array
}

// TableName 指定表名
func (LDLineup) TableName() string {
	return "ld_lineups"
}

