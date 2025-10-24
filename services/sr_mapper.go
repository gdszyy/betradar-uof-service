package services

import (
	"fmt"
	"strings"
)

// SRMapper SportRadar 数据映射器
type SRMapper struct{}

// NewSRMapper 创建映射器实例
func NewSRMapper() *SRMapper {
	return &SRMapper{}
}

// MatchStatusMapping 比赛状态映射
var MatchStatusMapping = map[string]string{
	"0":   "not_started",      // 未开始
	"1":   "live",              // 进行中
	"2":   "suspended",         // 暂停
	"3":   "ended",             // 已结束
	"4":   "closed",            // 已关闭
	"5":   "cancelled",         // 已取消
	"6":   "delayed",           // 延迟
	"7":   "interrupted",       // 中断
	"8":   "postponed",         // 推迟
	"9":   "abandoned",         // 放弃
	"10":  "coverage_lost",     // 失去覆盖
	"11":  "about_to_start",    // 即将开始
	"20":  "first_half",        // 上半场
	"30":  "halftime",          // 中场休息
	"40":  "second_half",       // 下半场
	"50":  "awaiting_extra",    // 等待加时
	"60":  "extra_first_half",  // 加时上半场
	"70":  "extra_halftime",    // 加时中场
	"80":  "extra_second_half", // 加时下半场
	"90":  "awaiting_penalties", // 等待点球
	"100": "penalties",         // 点球大战
	"110": "after_penalties",   // 点球后
	"120": "after_extra",       // 加时后
	"140": "awaiting_golden_goal", // 等待金球
	"141": "golden_goal",       // 金球
}

// MatchStatusChineseName 比赛状态中文名称
var MatchStatusChineseName = map[string]string{
	"0":   "未开始",
	"1":   "进行中",
	"2":   "暂停",
	"3":   "已结束",
	"4":   "已关闭",
	"5":   "已取消",
	"6":   "延迟",
	"7":   "中断",
	"8":   "推迟",
	"9":   "放弃",
	"10":  "失去覆盖",
	"11":  "即将开始",
	"20":  "上半场",
	"30":  "中场休息",
	"40":  "下半场",
	"50":  "等待加时",
	"60":  "加时上半场",
	"70":  "加时中场",
	"80":  "加时下半场",
	"90":  "等待点球",
	"100": "点球大战",
	"110": "点球后",
	"120": "加时后",
	"140": "等待金球",
	"141": "金球",
}

// SportMapping 运动类型映射
var SportMapping = map[string]string{
	"sr:sport:1":  "football",      // 足球
	"sr:sport:2":  "basketball",    // 篮球
	"sr:sport:3":  "baseball",      // 棒球
	"sr:sport:4":  "ice_hockey",    // 冰球
	"sr:sport:5":  "tennis",        // 网球
	"sr:sport:6":  "handball",      // 手球
	"sr:sport:12": "rugby",         // 橄榄球
	"sr:sport:16": "aussie_rules",  // 澳式足球
	"sr:sport:21": "volleyball",    // 排球
	"sr:sport:22": "cricket",       // 板球
	"sr:sport:23": "darts",         // 飞镖
	"sr:sport:31": "badminton",     // 羽毛球
}

// SportChineseName 运动类型中文名称
var SportChineseName = map[string]string{
	"sr:sport:1":  "足球",
	"sr:sport:2":  "篮球",
	"sr:sport:3":  "棒球",
	"sr:sport:4":  "冰球",
	"sr:sport:5":  "网球",
	"sr:sport:6":  "手球",
	"sr:sport:12": "橄榄球",
	"sr:sport:16": "澳式足球",
	"sr:sport:21": "排球",
	"sr:sport:22": "板球",
	"sr:sport:23": "飞镖",
	"sr:sport:31": "羽毛球",
}

// EventStatusMapping 赛事状态映射
var EventStatusMapping = map[string]string{
	"not_started": "未开始",
	"live":        "进行中",
	"ended":       "已结束",
	"closed":      "已关闭",
}

// MapMatchStatus 映射比赛状态
func (m *SRMapper) MapMatchStatus(srStatus string) string {
	if mapped, ok := MatchStatusMapping[srStatus]; ok {
		return mapped
	}
	return "unknown"
}

// MapMatchStatusChinese 映射比赛状态(中文)
func (m *SRMapper) MapMatchStatusChinese(srStatus string) string {
	if mapped, ok := MatchStatusChineseName[srStatus]; ok {
		return mapped
	}
	return "未知"
}

// MapSport 映射运动类型
func (m *SRMapper) MapSport(sportID string) string {
	if mapped, ok := SportMapping[sportID]; ok {
		return mapped
	}
	return "unknown"
}

// MapSportChinese 映射运动类型(中文)
func (m *SRMapper) MapSportChinese(sportID string) string {
	if mapped, ok := SportChineseName[sportID]; ok {
		return mapped
	}
	return "未知"
}

// MapEventStatus 映射赛事状态(中文)
func (m *SRMapper) MapEventStatus(status string) string {
	if mapped, ok := EventStatusMapping[status]; ok {
		return mapped
	}
	return status
}

// IsMatchEnded 判断比赛是否已结束
func (m *SRMapper) IsMatchEnded(matchStatus string) bool {
	endedStatuses := []string{"3", "4", "5", "9", "110", "120"}
	for _, status := range endedStatuses {
		if matchStatus == status {
			return true
		}
	}
	return false
}

// IsMatchLive 判断比赛是否进行中
func (m *SRMapper) IsMatchLive(matchStatus string) bool {
	liveStatuses := []string{"1", "20", "30", "40", "60", "70", "80", "100", "141"}
	for _, status := range liveStatuses {
		if matchStatus == status {
			return true
		}
	}
	return false
}

// ExtractMatchIDFromURN 从 URN 提取比赛 ID
// 例如: sr:match:12345 -> 12345
func (m *SRMapper) ExtractMatchIDFromURN(urn string) string {
	parts := strings.Split(urn, ":")
	if len(parts) >= 3 {
		return parts[2]
	}
	return urn
}

// ExtractCompetitorIDFromURN 从 URN 提取球队 ID
// 例如: sr:competitor:1001 -> 1001
func (m *SRMapper) ExtractCompetitorIDFromURN(urn string) string {
	parts := strings.Split(urn, ":")
	if len(parts) >= 3 {
		return parts[2]
	}
	return urn
}

// FormatMatchTime 格式化比赛时间
// 例如: "23:15" -> "23分15秒"
func (m *SRMapper) FormatMatchTime(matchTime string) string {
	if matchTime == "" {
		return ""
	}
	
	parts := strings.Split(matchTime, ":")
	if len(parts) == 2 {
		return fmt.Sprintf("%s分%s秒", parts[0], parts[1])
	}
	
	return matchTime
}

// GetMatchStatusCategory 获取比赛状态分类
func (m *SRMapper) GetMatchStatusCategory(matchStatus string) string {
	if m.IsMatchEnded(matchStatus) {
		return "ended"
	}
	if m.IsMatchLive(matchStatus) {
		return "live"
	}
	return "scheduled"
}

// MappedMatchData 映射后的比赛数据
type MappedMatchData struct {
	EventID            string  `json:"event_id"`
	MatchID            string  `json:"match_id"`             // 纯数字 ID
	SRNID              *string `json:"srn_id"`
	
	// 运动类型
	SportID            *string `json:"sport_id"`             // 原始 SR ID
	Sport              string  `json:"sport"`                // 映射后的英文名
	SportName          string  `json:"sport_name"`           // 中文名称
	
	// 比赛状态
	Status             string  `json:"status"`               // active/ended/scheduled
	StatusName         string  `json:"status_name"`          // 中文状态名
	MatchStatus        *string `json:"match_status"`         // 原始 SR 状态码
	MatchStatusMapped  string  `json:"match_status_mapped"`  // 映射后的英文状态
	MatchStatusName    string  `json:"match_status_name"`    // 中文状态名
	MatchStatusCategory string `json:"match_status_category"` // live/ended/scheduled
	
	// 比赛时间
	ScheduleTime       *string `json:"schedule_time"`
	MatchTime          *string `json:"match_time"`           // 原始时间
	MatchTimeFormatted string  `json:"match_time_formatted"` // 格式化后的时间
	
	// 球队信息
	HomeTeamID         *string `json:"home_team_id"`
	HomeTeamIDShort    string  `json:"home_team_id_short"`   // 纯数字 ID
	HomeTeamName       *string `json:"home_team_name"`
	AwayTeamID         *string `json:"away_team_id"`
	AwayTeamIDShort    string  `json:"away_team_id_short"`   // 纯数字 ID
	AwayTeamName       *string `json:"away_team_name"`
	
	// 比分
	HomeScore          *int    `json:"home_score"`
	AwayScore          *int    `json:"away_score"`
	
	// 元数据
	MessageCount       int     `json:"message_count"`
	LastMessageAt      *string `json:"last_message_at"`
	CreatedAt          string  `json:"created_at"`
	UpdatedAt          string  `json:"updated_at"`
}

// MapMatchData 映射比赛数据
func (m *SRMapper) MapMatchData(
	eventID string,
	srnID *string,
	sportID *string,
	status string,
	scheduleTime *string,
	homeTeamID *string,
	homeTeamName *string,
	awayTeamID *string,
	awayTeamName *string,
	homeScore *int,
	awayScore *int,
	matchStatus *string,
	matchTime *string,
	messageCount int,
	lastMessageAt *string,
	createdAt string,
	updatedAt string,
) *MappedMatchData {
	mapped := &MappedMatchData{
		EventID:       eventID,
		MatchID:       m.ExtractMatchIDFromURN(eventID),
		SRNID:         srnID,
		SportID:       sportID,
		Status:        status,
		StatusName:    m.MapEventStatus(status),
		ScheduleTime:  scheduleTime,
		HomeTeamID:    homeTeamID,
		HomeTeamName:  homeTeamName,
		AwayTeamID:    awayTeamID,
		AwayTeamName:  awayTeamName,
		HomeScore:     homeScore,
		AwayScore:     awayScore,
		MatchStatus:   matchStatus,
		MatchTime:     matchTime,
		MessageCount:  messageCount,
		LastMessageAt: lastMessageAt,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
	}
	
	// 映射运动类型
	if sportID != nil {
		mapped.Sport = m.MapSport(*sportID)
		mapped.SportName = m.MapSportChinese(*sportID)
	} else {
		mapped.Sport = "unknown"
		mapped.SportName = "未知"
	}
	
	// 映射比赛状态
	if matchStatus != nil {
		mapped.MatchStatusMapped = m.MapMatchStatus(*matchStatus)
		mapped.MatchStatusName = m.MapMatchStatusChinese(*matchStatus)
		mapped.MatchStatusCategory = m.GetMatchStatusCategory(*matchStatus)
	} else {
		mapped.MatchStatusMapped = "unknown"
		mapped.MatchStatusName = "未知"
		mapped.MatchStatusCategory = "scheduled"
	}
	
	// 格式化比赛时间
	if matchTime != nil {
		mapped.MatchTimeFormatted = m.FormatMatchTime(*matchTime)
	}
	
	// 提取球队 ID
	if homeTeamID != nil {
		mapped.HomeTeamIDShort = m.ExtractCompetitorIDFromURN(*homeTeamID)
	}
	if awayTeamID != nil {
		mapped.AwayTeamIDShort = m.ExtractCompetitorIDFromURN(*awayTeamID)
	}
	
	return mapped
}

