package services

import (
	"encoding/xml"
	"fmt"
)

// LDEvent Live Data 事件
type LDEvent struct {
	XMLName xml.Name `xml:"event"`
	
	// 通用属性
	ID          string `xml:"id,attr"`           // 已弃用
	UUID        string `xml:"uuid,attr"`         // 唯一标识符
	MatchID     string `xml:"matchid,attr"`      // 比赛 ID
	SportID     int    `xml:"sportid,attr"`      // 体育项目 ID
	Info        string `xml:"info,attr"`         // 事件描述
	MTime       string `xml:"mtime,attr"`        // 比赛时间 (MM:SS)
	Side        string `xml:"side,attr"`         // home/away/none
	STime       int64  `xml:"stime,attr"`        // Scout 输入时间 (timestamp)
	Type        int    `xml:"type,attr"`         // 事件类型 ID
	MatchStatus string `xml:"matchstatus,attr"`  // 比赛状态
	
	// 足球特定属性
	Player1     string `xml:"player1,attr"`      // 球员1
	Player2     string `xml:"player2,attr"`      // 球员2
	ExtraInfo   string `xml:"extrainfo,attr"`    // 额外信息
	
	// 比分相关
	T1Score     int    `xml:"t1s,attr"`          // 主队得分
	T2Score     int    `xml:"t2s,attr"`          // 客队得分
}

// LDMatchInfo Live Data 比赛信息
type LDMatchInfo struct {
	XMLName xml.Name `xml:"match"`
	
	// 基本信息
	MatchID     string `xml:"matchid,attr"`
	T1ID        string `xml:"t1id,attr"`
	T2ID        string `xml:"t2id,attr"`
	T1Name      string `xml:"t1name,attr"`
	T2Name      string `xml:"t2name,attr"`
	SportID     int    `xml:"sportid,attr"`
	
	// 状态信息
	MatchStatus string `xml:"matchstatus,attr"`  // not_started/live/ended/cancelled等
	MatchTime   string `xml:"matchtime,attr"`    // 比赛时间 (MM:SS)
	
	// 比分
	T1Score     int    `xml:"t1s,attr"`
	T2Score     int    `xml:"t2s,attr"`
	
	// 时间信息
	MatchDate   string `xml:"matchdate,attr"`    // 比赛日期
	StartTime   string `xml:"starttime,attr"`    // 开始时间
	
	// 其他信息
	CoverageType string `xml:"coveragetype,attr"`
	DeviceID     string `xml:"deviceid,attr"`
}

// LDLineup Live Data 阵容
type LDLineup struct {
	XMLName xml.Name `xml:"lineup"`
	
	MatchID string     `xml:"matchid,attr"`
	Team1   *LDTeam    `xml:"team1"`
	Team2   *LDTeam    `xml:"team2"`
}

// LDTeam 球队阵容
type LDTeam struct {
	XMLName xml.Name    `xml:"team1,omitempty"`
	
	Players []LDPlayer  `xml:"player"`
}

// LDPlayer 球员信息
type LDPlayer struct {
	XMLName xml.Name `xml:"player"`
	
	ID       string `xml:"id,attr"`
	Name     string `xml:"name,attr"`
	Number   int    `xml:"number,attr"`
	Position string `xml:"position,attr"`
	Status   string `xml:"status,attr"`  // starting/substitute/bench
}

// LDMatchList 比赛列表
type LDMatchList struct {
	XMLName xml.Name      `xml:"matchlist"`
	
	Matches []LDMatchInfo `xml:"match"`
}

// LDAlive 心跳消息
type LDAlive struct {
	XMLName xml.Name `xml:"alive"`
}

// LDLogin 登录消息
type LDLogin struct {
	XMLName  xml.Name `xml:"login"`
	Username string   `xml:"username,attr"`
	Password string   `xml:"password,attr"`
}

// LDBookMatch 订阅比赛
type LDBookMatch struct {
	XMLName xml.Name `xml:"match"`
	MatchID string   `xml:"matchid,attr"`
}

// LDUnsubscribe 取消订阅
type LDUnsubscribe struct {
	XMLName xml.Name `xml:"unsubscribe"`
	MatchID string   `xml:"matchid,attr"`
}

// 事件类型常量 (足球)
const (
	// 进球相关
	EventTypeGoal              = 1
	EventTypeOwnGoal           = 2
	EventTypePenaltyGoal       = 3
	EventTypePenaltyMissed     = 666
	
	// 卡牌
	EventTypeYellowCard        = 17
	EventTypeRedCard           = 18
	EventTypeYellowRedCard     = 19
	
	// 角球
	EventTypeCorner            = 11
	
	// 换人
	EventTypeSubstitution      = 20
	
	// 比赛状态
	EventTypeMatchStart        = 32
	EventTypePeriodStart       = 33
	EventTypePeriodEnd         = 34
	EventTypeMatchEnd          = 35
	
	// VAR
	EventTypeVARStart          = 1001
	EventTypeVAREnd            = 1002
)

// 比赛状态常量
const (
	MatchStatusNotStarted = "not_started"
	MatchStatusLive       = "live"
	MatchStatusEnded      = "ended"
	MatchStatusCancelled  = "cancelled"
	MatchStatusAbandoned  = "abandoned"
	MatchStatusPostponed  = "postponed"
)

// GetEventTypeName 获取事件类型名称
func GetEventTypeName(eventType int) string {
	switch eventType {
	case EventTypeGoal:
		return "Goal"
	case EventTypeOwnGoal:
		return "Own Goal"
	case EventTypePenaltyGoal:
		return "Penalty Goal"
	case EventTypePenaltyMissed:
		return "Penalty Missed"
	case EventTypeYellowCard:
		return "Yellow Card"
	case EventTypeRedCard:
		return "Red Card"
	case EventTypeYellowRedCard:
		return "Yellow-Red Card"
	case EventTypeCorner:
		return "Corner"
	case EventTypeSubstitution:
		return "Substitution"
	case EventTypeMatchStart:
		return "Match Start"
	case EventTypePeriodStart:
		return "Period Start"
	case EventTypePeriodEnd:
		return "Period End"
	case EventTypeMatchEnd:
		return "Match End"
	case EventTypeVARStart:
		return "VAR Check Start"
	case EventTypeVAREnd:
		return "VAR Check End"
	default:
		return fmt.Sprintf("Unknown (%d)", eventType)
	}
}

// IsImportantEvent 判断是否为重要事件
func IsImportantEvent(eventType int) bool {
	switch eventType {
	case EventTypeGoal, EventTypeOwnGoal, EventTypePenaltyGoal,
		EventTypeRedCard, EventTypeYellowRedCard,
		EventTypeMatchStart, EventTypeMatchEnd:
		return true
	default:
		return false
	}
}

