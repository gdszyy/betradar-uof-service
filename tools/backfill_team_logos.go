package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	_ "github.com/lib/pq"
)

// TheSportsDBResponse TheSportsDB API 响应结构
type TheSportsDBResponse struct {
	Teams []struct {
		StrBadge string `json:"strBadge"`
		StrLogo  string `json:"strLogo"`
	} `json:"teams"`
}

func main() {
	// 从环境变量获取数据库连接字符串
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	// 连接数据库
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Println("✅ Connected to database")

	// 获取需要获取 Logo 的队伍列表（批量处理，每次 100 个）
	batchSize := 100
	maxRetries := 3
	processedCount := 0
	successCount := 0
	failureCount := 0

	for {
		teams, err := getTeamsNeedingLogoFetch(db, maxRetries, batchSize)
		if err != nil {
			log.Fatalf("Failed to get teams: %v", err)
		}

		if len(teams) == 0 {
			log.Println("✅ No more teams need logo fetch")
			break
		}

		log.Printf("📋 Processing batch of %d teams...", len(teams))

		for _, team := range teams {
			logoURL, err := fetchLogoFromTheSportsDB(team.TeamName)
			if err != nil {
				log.Printf("⚠️  Failed to fetch logo for team %s: %v", team.TeamName, err)
				updateTeamLogo(db, team.TeamID, "", false)
				failureCount++
			} else if logoURL == "" {
				log.Printf("⚠️  No logo found for team %s", team.TeamName)
				updateTeamLogo(db, team.TeamID, "", false)
				failureCount++
			} else {
				log.Printf("✅ Found logo for team %s: %s", team.TeamName, logoURL)
				updateTeamLogo(db, team.TeamID, logoURL, true)
				successCount++
			}

			processedCount++

			// 避免请求过快，添加延迟
			time.Sleep(500 * time.Millisecond)
		}

		log.Printf("📊 Progress: %d processed, %d success, %d failure", processedCount, successCount, failureCount)
	}

	log.Printf("🎉 Logo fetch completed! Total: %d, Success: %d, Failure: %d", processedCount, successCount, failureCount)
}

type Team struct {
	TeamID   string
	TeamName string
}

func getTeamsNeedingLogoFetch(db *sql.DB, maxRetries int, limit int) ([]Team, error) {
	query := `
		SELECT team_id, team_name
		FROM teams
		WHERE logo_fetched = false
		  AND (logo_fetch_attempted_at IS NULL OR logo_fetch_retry_count < $1)
		ORDER BY created_at ASC
		LIMIT $2
	`

	rows, err := db.Query(query, maxRetries, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var teams []Team
	for rows.Next() {
		var team Team
		if err := rows.Scan(&team.TeamID, &team.TeamName); err != nil {
			return nil, err
		}
		teams = append(teams, team)
	}

	return teams, nil
}

func fetchLogoFromTheSportsDB(teamName string) (string, error) {
	apiURL := fmt.Sprintf("https://www.thesportsdb.com/api/v1/json/123/searchteams.php?t=%s", url.QueryEscape(teamName))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status code %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var apiResponse TheSportsDBResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return "", err
	}

	if len(apiResponse.Teams) == 0 {
		return "", nil
	}

	logoURL := apiResponse.Teams[0].StrBadge
	if logoURL == "" {
		logoURL = apiResponse.Teams[0].StrLogo
	}

	return logoURL, nil
}

func updateTeamLogo(db *sql.DB, teamID string, logoURL string, success bool) error {
	query := `
		UPDATE teams
		SET logo_url = $1,
		    logo_fetched = $2,
		    logo_fetch_attempted_at = $3,
		    logo_fetch_retry_count = CASE WHEN $2 = true THEN 0 ELSE logo_fetch_retry_count + 1 END,
		    updated_at = $4
		WHERE team_id = $5
	`

	now := time.Now()
	_, err := db.Exec(query, logoURL, success, now, now, teamID)
	return err
}
