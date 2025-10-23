package thesports

import "time"

// Category represents a sport category
type Category struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Country represents a country
type Country struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	ShortName string `json:"short_name"`
	Logo      string `json:"logo"`
}

// Competition represents a football competition/league
type Competition struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	ShortName   string `json:"short_name"`
	Logo        string `json:"logo"`
	CategoryID  int    `json:"category_id"`
	CountryID   int    `json:"country_id"`
	CurrentSeason *Season `json:"current_season,omitempty"`
}

// Team represents a football team
type Team struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	ShortName   string `json:"short_name"`
	Logo        string `json:"logo"`
	CountryID   int    `json:"country_id"`
	VenueID     int    `json:"venue_id"`
	Founded     int    `json:"founded"`
	NationalTeam bool  `json:"national_team"`
}

// Player represents a football player
type Player struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	ShortName   string    `json:"short_name"`
	Photo       string    `json:"photo"`
	CountryID   int       `json:"country_id"`
	DateOfBirth time.Time `json:"date_of_birth"`
	Height      int       `json:"height"` // in cm
	Weight      int       `json:"weight"` // in kg
	Position    string    `json:"position"`
	ShirtNumber int       `json:"shirt_number"`
}

// Coach represents a football coach
type Coach struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Photo       string    `json:"photo"`
	CountryID   int       `json:"country_id"`
	DateOfBirth time.Time `json:"date_of_birth"`
}

// Referee represents a referee
type Referee struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	CountryID int    `json:"country_id"`
}

// Venue represents a stadium/venue
type Venue struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	City     string `json:"city"`
	Capacity int    `json:"capacity"`
	Address  string `json:"address"`
}

// Season represents a competition season
type Season struct {
	ID            int       `json:"id"`
	Name          string    `json:"name"`
	Year          string    `json:"year"`
	StartDate     time.Time `json:"start_date"`
	EndDate       time.Time `json:"end_date"`
	CompetitionID int       `json:"competition_id"`
	IsCurrent     bool      `json:"is_current"`
}

// Stage represents a competition stage
type Stage struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"` // league, cup, playoff, etc.
	SeasonID int    `json:"season_id"`
}

// Match represents a football match
type Match struct {
	ID              int           `json:"id"`
	CompetitionID   int           `json:"competition_id"`
	SeasonID        int           `json:"season_id"`
	StageID         int           `json:"stage_id"`
	Round           int           `json:"round"`
	HomeTeamID      int           `json:"home_team_id"`
	AwayTeamID      int           `json:"away_team_id"`
	HomeTeam        *Team         `json:"home_team,omitempty"`
	AwayTeam        *Team         `json:"away_team,omitempty"`
	VenueID         int           `json:"venue_id"`
	RefereeID       int           `json:"referee_id"`
	StartTime       time.Time     `json:"start_time"`
	Status          string        `json:"status"` // not_started, live, finished, postponed, cancelled
	HomeScore       int           `json:"home_score"`
	AwayScore       int           `json:"away_score"`
	HalfTimeScore   string        `json:"half_time_score"`
	FullTimeScore   string        `json:"full_time_score"`
	ExtraTimeScore  string        `json:"extra_time_score"`
	PenaltyScore    string        `json:"penalty_score"`
	Minute          int           `json:"minute"`
	AddedTime       int           `json:"added_time"`
	Lineups         *MatchLineups `json:"lineups,omitempty"`
	Statistics      *MatchStats   `json:"statistics,omitempty"`
	Incidents       []Incident    `json:"incidents,omitempty"`
}

// MatchLineups represents match lineups
type MatchLineups struct {
	HomeTeam TeamLineup `json:"home_team"`
	AwayTeam TeamLineup `json:"away_team"`
}

// TeamLineup represents a team's lineup
type TeamLineup struct {
	Formation  string         `json:"formation"`
	StartingXI []LineupPlayer `json:"starting_xi"`
	Substitutes []LineupPlayer `json:"substitutes"`
	Coach      *Coach         `json:"coach,omitempty"`
}

// LineupPlayer represents a player in the lineup
type LineupPlayer struct {
	PlayerID    int    `json:"player_id"`
	Player      *Player `json:"player,omitempty"`
	ShirtNumber int    `json:"shirt_number"`
	Position    string `json:"position"`
	GridPosition string `json:"grid_position"` // e.g., "1:1" for goalkeeper
}

// MatchStats represents match statistics
type MatchStats struct {
	HomeTeam TeamStats `json:"home_team"`
	AwayTeam TeamStats `json:"away_team"`
}

// TeamStats represents team statistics
type TeamStats struct {
	Possession       int `json:"possession"`
	Shots            int `json:"shots"`
	ShotsOnTarget    int `json:"shots_on_target"`
	ShotsOffTarget   int `json:"shots_off_target"`
	Corners          int `json:"corners"`
	Offsides         int `json:"offsides"`
	Fouls            int `json:"fouls"`
	YellowCards      int `json:"yellow_cards"`
	RedCards         int `json:"red_cards"`
	Passes           int `json:"passes"`
	PassesAccurate   int `json:"passes_accurate"`
	Attacks          int `json:"attacks"`
	DangerousAttacks int `json:"dangerous_attacks"`
}

// Incident represents a match incident/event
type Incident struct {
	ID          int       `json:"id"`
	MatchID     int       `json:"match_id"`
	Type        string    `json:"type"` // goal, yellow_card, red_card, substitution, etc.
	Time        int       `json:"time"` // minute
	AddedTime   int       `json:"added_time"`
	TeamID      int       `json:"team_id"`
	PlayerID    int       `json:"player_id"`
	Player      *Player   `json:"player,omitempty"`
	RelatedPlayerID int   `json:"related_player_id"` // for assists, substitutions
	RelatedPlayer *Player `json:"related_player,omitempty"`
	Description string    `json:"description"`
	HomeScore   int       `json:"home_score"`
	AwayScore   int       `json:"away_score"`
	GifURL      string    `json:"gif_url,omitempty"`
}

// MatchLive represents live match data
type MatchLive struct {
	Match      *Match      `json:"match"`
	Events     []Incident  `json:"events"`
	Statistics *MatchStats `json:"statistics"`
}

// Commentary represents match commentary
type Commentary struct {
	ID          int       `json:"id"`
	MatchID     int       `json:"match_id"`
	Time        int       `json:"time"`
	AddedTime   int       `json:"added_time"`
	Text        string    `json:"text"`
	Important   bool      `json:"important"`
	CreatedAt   time.Time `json:"created_at"`
}

// H2H represents head-to-head data
type H2H struct {
	HomeTeamID   int      `json:"home_team_id"`
	AwayTeamID   int      `json:"away_team_id"`
	TotalMatches int      `json:"total_matches"`
	HomeWins     int      `json:"home_wins"`
	AwayWins     int      `json:"away_wins"`
	Draws        int      `json:"draws"`
	LastMatches  []Match  `json:"last_matches"`
}

// Odds represents match odds
type Odds struct {
	MatchID      int           `json:"match_id"`
	BookmakerID  int           `json:"bookmaker_id"`
	Bookmaker    *Bookmaker    `json:"bookmaker,omitempty"`
	Markets      []OddsMarket  `json:"markets"`
	UpdatedAt    time.Time     `json:"updated_at"`
}

// Bookmaker represents a bookmaker
type Bookmaker struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Logo string `json:"logo"`
}

// OddsMarket represents an odds market
type OddsMarket struct {
	ID        int         `json:"id"`
	Name      string      `json:"name"` // 1x2, over_under, asian_handicap, etc.
	Outcomes  []OddsOutcome `json:"outcomes"`
}

// OddsOutcome represents an odds outcome
type OddsOutcome struct {
	Name  string  `json:"name"`
	Odds  float64 `json:"odds"`
	Line  string  `json:"line,omitempty"` // for handicap, over/under
}

// Prediction represents match prediction
type Prediction struct {
	MatchID          int     `json:"match_id"`
	HomeWinProb      float64 `json:"home_win_probability"`
	DrawProb         float64 `json:"draw_probability"`
	AwayWinProb      float64 `json:"away_win_probability"`
	PredictedScore   string  `json:"predicted_score"`
	Confidence       float64 `json:"confidence"`
}

// PlayerStatistics represents player statistics
type PlayerStatistics struct {
	PlayerID        int     `json:"player_id"`
	SeasonID        int     `json:"season_id"`
	Appearances     int     `json:"appearances"`
	Goals           int     `json:"goals"`
	Assists         int     `json:"assists"`
	YellowCards     int     `json:"yellow_cards"`
	RedCards        int     `json:"red_cards"`
	MinutesPlayed   int     `json:"minutes_played"`
	Rating          float64 `json:"rating"`
}

// TeamStatistics represents team statistics
type TeamStatistics struct {
	TeamID          int     `json:"team_id"`
	SeasonID        int     `json:"season_id"`
	MatchesPlayed   int     `json:"matches_played"`
	Wins            int     `json:"wins"`
	Draws           int     `json:"draws"`
	Losses          int     `json:"losses"`
	GoalsFor        int     `json:"goals_for"`
	GoalsAgainst    int     `json:"goals_against"`
	CleanSheets     int     `json:"clean_sheets"`
	Points          int     `json:"points"`
	Position        int     `json:"position"`
	Form            string  `json:"form"` // e.g., "WWDLW"
}

// MatchVideo represents match video
type MatchVideo struct {
	ID          int       `json:"id"`
	MatchID     int       `json:"match_id"`
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	Thumbnail   string    `json:"thumbnail"`
	Duration    int       `json:"duration"` // in seconds
	Type        string    `json:"type"` // highlights, full_match, etc.
	CreatedAt   time.Time `json:"created_at"`
}

