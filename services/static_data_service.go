package services

import (
	"database/sql"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"time"
	"uof-service/logger"
)

// StaticDataService 静态数据服务
type StaticDataService struct {
	db          *sql.DB
	apiBaseURL  string
	accessToken string
	client      *http.Client
}

// NewStaticDataService 创建静态数据服务
func NewStaticDataService(db *sql.DB, accessToken, apiBaseURL string) *StaticDataService {
	return &StaticDataService{
		db:          db,
		apiBaseURL:  apiBaseURL,
		accessToken: accessToken,
		client:      &http.Client{Timeout: 30 * time.Second},
	}
}

// Start 启动静态数据服务
func (s *StaticDataService) Start() error {
	logger.Println("[StaticData] Starting static data service...")

	// 启动时立即加载一次
	if err := s.LoadAllStaticData(); err != nil {
		logger.Errorf("[StaticData] ❌ Failed to load static data: %v", err)
		return err
	}

	// 每周刷新一次
	go func() {
		ticker := time.NewTicker(7 * 24 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			logger.Println("[StaticData] 🔄 Weekly refresh triggered")
			if err := s.LoadAllStaticData(); err != nil {
				logger.Errorf("[StaticData] ❌ Weekly refresh failed: %v", err)
			}
		}
	}()

	logger.Println("[StaticData] ✅ Static data service started (weekly refresh)")
	return nil
}

// LoadAllStaticData 加载所有静态数据
func (s *StaticDataService) LoadAllStaticData() error {
	logger.Println("[StaticData] 📥 Loading all static data...")

	// 加载 Sports
	if err := s.LoadSports(); err != nil {
		logger.Errorf("[StaticData] ⚠️  Failed to load sports: %v", err)
	}

	// 加载 Categories
	if err := s.LoadCategories(); err != nil {
		logger.Errorf("[StaticData] ⚠️  Failed to load categories: %v", err)
	}

	// 加载 Tournaments
	if err := s.LoadTournaments(); err != nil {
		logger.Errorf("[StaticData] ⚠️  Failed to load tournaments: %v", err)
	}

	// 加载 Void Reasons
	if err := s.LoadVoidReasons(); err != nil {
		logger.Errorf("[StaticData] ⚠️  Failed to load void reasons: %v", err)
	}

	// 加载 Betstop Reasons
	if err := s.LoadBetstopReasons(); err != nil {
		logger.Errorf("[StaticData] ⚠️  Failed to load betstop reasons: %v", err)
	}

	logger.Println("[StaticData] ✅ All static data loaded")
	return nil
}

// LoadSports 加载体育类型
func (s *StaticDataService) LoadSports() error {
	url := fmt.Sprintf("%s/sports/en/sports.xml", s.apiBaseURL)
	logger.Printf("[StaticData] 📥 Loading sports from: %s", url)

	body, err := s.fetchAPI(url)
	if err != nil {
		return fmt.Errorf("failed to fetch sports: %w", err)
	}

	var sportsData struct {
		Sports []struct {
			ID   string `xml:"id,attr"`
			Name string `xml:"name,attr"`
		} `xml:"sport"`
	}

	if err := xml.Unmarshal(body, &sportsData); err != nil {
		return fmt.Errorf("failed to parse sports XML: %w", err)
	}

	// 存储到数据库
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	count := 0
	for _, sport := range sportsData.Sports {
		_, err := tx.Exec(`
			INSERT INTO sports (id, name, updated_at)
			VALUES ($1, $2, NOW())
			ON CONFLICT (id) DO UPDATE SET
				name = EXCLUDED.name,
				updated_at = NOW()
		`, sport.ID, sport.Name)

		if err != nil {
			logger.Errorf("[StaticData] ⚠️  Failed to insert sport %s: %v", sport.ID, err)
			continue
		}
		count++
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	logger.Printf("[StaticData] ✅ Loaded %d sports", count)
	return nil
}

// LoadCategories 加载分类
func (s *StaticDataService) LoadCategories() error {
	// 先从数据库获取所有 sports
	rows, err := s.db.Query("SELECT id FROM sports")
	if err != nil {
		return fmt.Errorf("failed to query sports: %w", err)
	}
	defer rows.Close()

	var sportIDs []string
	for rows.Next() {
		var sportID string
		if err := rows.Scan(&sportID); err != nil {
			continue
		}
		sportIDs = append(sportIDs, sportID)
	}

	if len(sportIDs) == 0 {
		return fmt.Errorf("no sports found in database, please load sports first")
	}

	logger.Printf("[StaticData] 📥 Loading categories for %d sports...", len(sportIDs))

	totalCount := 0
	for _, sportID := range sportIDs {
// 按 sport 查询 categories
			// 官方文档规范: /sports/{language}/sports/{sport_id}/categories.xml
			url := fmt.Sprintf("%s/sports/en/sports/%s/categories.xml", s.apiBaseURL, sportID)
		
		body, err := s.fetchAPI(url)
		if err != nil {
			logger.Errorf("[StaticData] ⚠️  Failed to fetch categories for %s: %v", sportID, err)
			continue
		}

			var categoriesData struct {
				XMLName xml.Name `xml:"sport_categories"`
				Categories []struct {
					ID          string `xml:"id,attr"`
					Name        string `xml:"name,attr"`
					CountryCode string `xml:"country_code,attr"`
				} `xml:"categories>category"`
			}

		if err := xml.Unmarshal(body, &categoriesData); err != nil {
			logger.Errorf("[StaticData] ⚠️  Failed to parse categories XML for %s: %v", sportID, err)
			continue
		}

		// 存储到数据库
		tx, err := s.db.Begin()
		if err != nil {
			logger.Errorf("[StaticData] ⚠️  Failed to begin transaction for %s: %v", sportID, err)
			continue
		}

		count := 0
		for _, category := range categoriesData.Categories {
			_, err := tx.Exec(`
				INSERT INTO categories (id, sport_id, name, country_code, updated_at)
				VALUES ($1, $2, $3, $4, NOW())
				ON CONFLICT (id) DO UPDATE SET
					sport_id = EXCLUDED.sport_id,
					name = EXCLUDED.name,
					country_code = EXCLUDED.country_code,
					updated_at = NOW()
			`, category.ID, sportID, category.Name, category.CountryCode)

			if err != nil {
				logger.Errorf("[StaticData] ⚠️  Failed to insert category %s: %v", category.ID, err)
				continue
			}
			count++
		}

		if err := tx.Commit(); err != nil {
			logger.Errorf("[StaticData] ⚠️  Failed to commit transaction for %s: %v", sportID, err)
			tx.Rollback()
			continue
		}

		logger.Printf("[StaticData] ✅ Loaded %d categories for %s", count, sportID)
		totalCount += count
	}

	logger.Printf("[StaticData] ✅ Total loaded %d categories", totalCount)
	return nil
}

// LoadTournaments 加载锦标赛
func (s *StaticDataService) LoadTournaments() error {
	url := fmt.Sprintf("%s/descriptions/en/tournaments.xml", s.apiBaseURL)
	logger.Printf("[StaticData] 📥 Loading tournaments from: %s", url)

	body, err := s.fetchAPI(url)
	if err != nil {
		return fmt.Errorf("failed to fetch tournaments: %w", err)
	}

	var tournamentsData struct {
		Tournaments []struct {
			ID       string `xml:"id,attr"`
			Name     string `xml:"name,attr"`
			Sport    struct {
				ID string `xml:"id,attr"`
			} `xml:"sport"`
			Category struct {
				ID string `xml:"id,attr"`
			} `xml:"category"`
		} `xml:"tournament"`
	}

	if err := xml.Unmarshal(body, &tournamentsData); err != nil {
		return fmt.Errorf("failed to parse tournaments XML: %w", err)
	}

	// 存储到数据库
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	count := 0
	for _, tournament := range tournamentsData.Tournaments {
		_, err := tx.Exec(`
			INSERT INTO tournaments (id, sport_id, category_id, name, updated_at)
			VALUES ($1, $2, $3, $4, NOW())
			ON CONFLICT (id) DO UPDATE SET
				sport_id = EXCLUDED.sport_id,
				category_id = EXCLUDED.category_id,
				name = EXCLUDED.name,
				updated_at = NOW()
		`, tournament.ID, tournament.Sport.ID, tournament.Category.ID, tournament.Name)

		if err != nil {
			logger.Errorf("[StaticData] ⚠️  Failed to insert tournament %s: %v", tournament.ID, err)
			continue
		}
		count++
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	logger.Printf("[StaticData] ✅ Loaded %d tournaments", count)
	return nil
}

// LoadVoidReasons 加载作废原因
func (s *StaticDataService) LoadVoidReasons() error {
	url := fmt.Sprintf("%s/descriptions/void_reasons.xml", s.apiBaseURL)
	logger.Printf("[StaticData] 📥 Loading void reasons from: %s", url)

	body, err := s.fetchAPI(url)
	if err != nil {
		return fmt.Errorf("failed to fetch void reasons: %w", err)
	}

	var voidReasonsData struct {
		VoidReasons []struct {
			ID          int    `xml:"id,attr"`
			Description string `xml:"description,attr"`
		} `xml:"void_reason"`
	}

	if err := xml.Unmarshal(body, &voidReasonsData); err != nil {
		return fmt.Errorf("failed to parse void reasons XML: %w", err)
	}

	// 存储到数据库
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	count := 0
	for _, reason := range voidReasonsData.VoidReasons {
		_, err := tx.Exec(`
			INSERT INTO void_reasons (id, description, updated_at)
			VALUES ($1, $2, NOW())
			ON CONFLICT (id) DO UPDATE SET
				description = EXCLUDED.description,
				updated_at = NOW()
		`, reason.ID, reason.Description)

		if err != nil {
			logger.Errorf("[StaticData] ⚠️  Failed to insert void reason %d: %v", reason.ID, err)
			continue
		}
		count++
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	logger.Printf("[StaticData] ✅ Loaded %d void reasons", count)
	return nil
}

// LoadBetstopReasons 加载停止投注原因
func (s *StaticDataService) LoadBetstopReasons() error {
	url := fmt.Sprintf("%s/descriptions/betstop_reasons.xml", s.apiBaseURL)
	logger.Printf("[StaticData] 📥 Loading betstop reasons from: %s", url)

	body, err := s.fetchAPI(url)
	if err != nil {
		return fmt.Errorf("failed to fetch betstop reasons: %w", err)
	}

	var betstopReasonsData struct {
		BetstopReasons []struct {
			ID          int    `xml:"id,attr"`
			Description string `xml:"description,attr"`
		} `xml:"betstop_reason"`
	}

	if err := xml.Unmarshal(body, &betstopReasonsData); err != nil {
		return fmt.Errorf("failed to parse betstop reasons XML: %w", err)
	}

	// 存储到数据库
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	count := 0
	for _, reason := range betstopReasonsData.BetstopReasons {
		_, err := tx.Exec(`
			INSERT INTO betstop_reasons (id, description, updated_at)
			VALUES ($1, $2, NOW())
			ON CONFLICT (id) DO UPDATE SET
				description = EXCLUDED.description,
				updated_at = NOW()
		`, reason.ID, reason.Description)

		if err != nil {
			logger.Errorf("[StaticData] ⚠️  Failed to insert betstop reason %d: %v", reason.ID, err)
			continue
		}
		count++
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	logger.Printf("[StaticData] ✅ Loaded %d betstop reasons", count)
	return nil
}

// fetchAPI 调用 API 并返回响应体
func (s *StaticDataService) fetchAPI(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("x-access-token", s.accessToken)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return body, nil
}

