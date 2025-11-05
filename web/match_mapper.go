package web

import (
	"uof-service/services"
)

// EnhancedMatchDetail 增强的比赛详情结构(包含映射后的字段)
type EnhancedMatchDetail struct {
	// 原始字段
	EventID        string  `json:"event_id"`
	SRNID          *string `json:"srn_id"`
	SportID        *string `json:"sport_id"`
	Status         string  `json:"status"`
	ScheduleTime   *string `json:"schedule_time"`
	HomeTeamID     *string `json:"home_team_id"`
	HomeTeamName   *string `json:"home_team_name"`
	AwayTeamID     *string `json:"away_team_id"`
	AwayTeamName   *string `json:"away_team_name"`
	HomeScore      *int    `json:"home_score"`
	AwayScore      *int    `json:"away_score"`
	MatchStatus    *string `json:"match_status"`
	MatchTime      *string `json:"match_time"`
	MessageCount   int     `json:"message_count"`
	LastMessageAt  *string `json:"last_message_at"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`

	// 映射后的字段
		Sport              string `json:"sport"`               // "football"
		SportName          string `json:"sport_name"`          // "Football" (改为英文)
	MatchStatusMapped  string `json:"match_status_mapped"` // "first_half"
	MatchStatusName    string `json:"match_status_name"`   // "上半场"
	MatchTimeMapped    string `json:"match_time_mapped"`   // "23分15秒"
	HomeTeamIDMapped   string `json:"home_team_id_mapped"` // "1001"
	AwayTeamIDMapped   string `json:"away_team_id_mapped"` // "1002"
	IsLive             bool   `json:"is_live"`             // true/false
	IsEnded            bool   `json:"is_ended"`            // true/false
}

// MapMatchDetail 将原始比赛数据映射为增强的比赛详情
func MapMatchDetail(match MatchDetail, mapper *services.SRMapper) EnhancedMatchDetail {
	enhanced := EnhancedMatchDetail{
		EventID:       match.EventID,
		SRNID:         match.SRNID,
		SportID:       match.SportID,
		Status:        match.Status,
		ScheduleTime:  match.ScheduleTime,
		HomeTeamID:    match.HomeTeamID,
		HomeTeamName:  match.HomeTeamName,
		AwayTeamID:    match.AwayTeamID,
		AwayTeamName:  match.AwayTeamName,
		HomeScore:     match.HomeScore,
		AwayScore:     match.AwayScore,
		MatchStatus:   match.MatchStatus,
		MatchTime:     match.MatchTime,
		MessageCount:  match.MessageCount,
		LastMessageAt: match.LastMessageAt,
		CreatedAt:     match.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:     match.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}

	// 映射运动类型
	if match.SportID != nil && *match.SportID != "" {
		enhanced.Sport = mapper.MapSport(*match.SportID)
			// 修复中文 sport_name 问题，改为使用英文名称
			enhanced.SportName = mapper.MapSport(*match.SportID)
	}

	// 映射比赛状态
	if match.MatchStatus != nil && *match.MatchStatus != "" {
			enhanced.MatchStatusMapped = mapper.MapMatchStatus(*match.MatchStatus)
			enhanced.MatchStatusName = mapper.MapMatchStatusChinese(*match.MatchStatus)
			
			// 修复 is_live 逻辑：当 Status 为 "live" 或 MatchStatus 为 live 状态时，IsLive 为 true
			isLiveFromStatus := match.Status == "live"
			isLiveFromMatchStatus := mapper.IsMatchLive(*match.MatchStatus)
			enhanced.IsLive = isLiveFromStatus || isLiveFromMatchStatus
			
			enhanced.IsEnded = mapper.IsMatchEnded(*match.MatchStatus)
	}

	// 映射比赛时间
	if match.MatchTime != nil && *match.MatchTime != "" {
		enhanced.MatchTimeMapped = mapper.FormatMatchTime(*match.MatchTime)
	}

	// 映射球队 ID
	if match.HomeTeamID != nil && *match.HomeTeamID != "" {
		enhanced.HomeTeamIDMapped = mapper.ExtractCompetitorIDFromURN(*match.HomeTeamID)
	}
	if match.AwayTeamID != nil && *match.AwayTeamID != "" {
		enhanced.AwayTeamIDMapped = mapper.ExtractCompetitorIDFromURN(*match.AwayTeamID)
	}

	return enhanced
}

// MapMatchList 批量映射比赛列表
func MapMatchList(matches []MatchDetail, mapper *services.SRMapper) []EnhancedMatchDetail {
	enhanced := make([]EnhancedMatchDetail, len(matches))
	for i, match := range matches {
		enhanced[i] = MapMatchDetail(match, mapper)
	}
	return enhanced
}

