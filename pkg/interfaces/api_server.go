package interfaces

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"uof-service/pkg/business"
	"uof-service/pkg/common"
)

// DefaultAPIServer 默认 API 服务器实现
type DefaultAPIServer struct {
	logger             common.Logger
	port               int
	oddsService        business.OddsService
	matchService       business.MatchService
	settlementService  business.SettlementService
	subscriptionService business.SubscriptionService
	bookingService     business.BookingService
	server             *http.Server
}

// NewAPIServer 创建 API 服务器
func NewAPIServer(
	logger common.Logger,
	port int,
	oddsService business.OddsService,
	matchService business.MatchService,
	settlementService business.SettlementService,
	subscriptionService business.SubscriptionService,
	bookingService business.BookingService,
) APIServer {
	return &DefaultAPIServer{
		logger:              logger,
		port:                port,
		oddsService:         oddsService,
		matchService:        matchService,
		settlementService:   settlementService,
		subscriptionService: subscriptionService,
		bookingService:      bookingService,
	}
}

// Start 启动 API 服务器
func (s *DefaultAPIServer) Start(ctx context.Context) error {
	s.logger.Info("Starting API server on port %d", s.port)

	mux := http.NewServeMux()

	// 注册路由
	s.registerRoutes(mux)

	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: s.corsMiddleware(s.loggingMiddleware(mux)),
	}

	// 启动服务器
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("API server error: %v", err)
		}
	}()

	s.logger.Info("API server started successfully on port %d", s.port)
	return nil
}

// Stop 停止 API 服务器
func (s *DefaultAPIServer) Stop(ctx context.Context) error {
	s.logger.Info("Stopping API server")

	if s.server != nil {
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		if err := s.server.Shutdown(ctx); err != nil {
			s.logger.Error("Failed to stop API server: %v", err)
			return err
		}
	}

	s.logger.Info("API server stopped successfully")
	return nil
}

// registerRoutes 注册路由
func (s *DefaultAPIServer) registerRoutes(mux *http.ServeMux) {
	// 健康检查
	mux.HandleFunc("/health", s.handleHealth)

	// 赔率相关
	mux.HandleFunc("/api/odds/", s.handleGetOdds)
	mux.HandleFunc("/api/odds/markets/", s.handleGetActiveMarkets)

	// 比赛相关
	mux.HandleFunc("/api/matches/", s.handleGetMatch)
	mux.HandleFunc("/api/matches/live", s.handleGetLiveMatches)
	mux.HandleFunc("/api/matches/upcoming", s.handleGetUpcomingMatches)
	mux.HandleFunc("/api/matches/search", s.handleSearchMatches)

	// 结算相关
	mux.HandleFunc("/api/settlements/", s.handleGetSettlements)
	mux.HandleFunc("/api/settlements/pending", s.handleGetPendingSettlements)

	// 订阅相关
	mux.HandleFunc("/api/subscriptions", s.handleSubscriptions)
	mux.HandleFunc("/api/subscriptions/stats", s.handleSubscriptionStats)

	// 预订相关
	mux.HandleFunc("/api/booking/matches", s.handleGetBookableMatches)
	mux.HandleFunc("/api/booking/book", s.handleBookMatch)
}

// 健康检查处理器
func (s *DefaultAPIServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"status": "ok",
		"time":   time.Now(),
	})
}

// 获取赔率处理器
func (s *DefaultAPIServer) handleGetOdds(w http.ResponseWriter, r *http.Request) {
	matchID := r.URL.Query().Get("match_id")
	if matchID == "" {
		s.writeError(w, http.StatusBadRequest, "match_id is required")
		return
	}

	odds, err := s.oddsService.GetOdds(r.Context(), matchID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, odds)
}

// 获取活跃市场处理器
func (s *DefaultAPIServer) handleGetActiveMarkets(w http.ResponseWriter, r *http.Request) {
	matchID := r.URL.Query().Get("match_id")
	if matchID == "" {
		s.writeError(w, http.StatusBadRequest, "match_id is required")
		return
	}

	markets, err := s.oddsService.GetActiveMarkets(r.Context(), matchID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, markets)
}

// 获取比赛处理器
func (s *DefaultAPIServer) handleGetMatch(w http.ResponseWriter, r *http.Request) {
	matchID := r.URL.Query().Get("match_id")
	if matchID == "" {
		s.writeError(w, http.StatusBadRequest, "match_id is required")
		return
	}

	match, err := s.matchService.GetMatch(r.Context(), matchID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, match)
}

// 获取直播比赛处理器
func (s *DefaultAPIServer) handleGetLiveMatches(w http.ResponseWriter, r *http.Request) {
	matches, err := s.matchService.GetLiveMatches(r.Context())
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, matches)
}

// 获取即将开始的比赛处理器
func (s *DefaultAPIServer) handleGetUpcomingMatches(w http.ResponseWriter, r *http.Request) {
	matches, err := s.matchService.GetUpcomingMatches(r.Context(), 24)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, matches)
}

// 搜索比赛处理器
func (s *DefaultAPIServer) handleSearchMatches(w http.ResponseWriter, r *http.Request) {
	query := business.MatchSearchQuery{
		SportID:  r.URL.Query().Get("sport_id"),
		Status:   r.URL.Query().Get("status"),
		TeamName: r.URL.Query().Get("team_name"),
		Limit:    50,
	}

	matches, err := s.matchService.SearchMatches(r.Context(), query)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, matches)
}

// 获取结算处理器
func (s *DefaultAPIServer) handleGetSettlements(w http.ResponseWriter, r *http.Request) {
	matchID := r.URL.Query().Get("match_id")
	if matchID == "" {
		s.writeError(w, http.StatusBadRequest, "match_id is required")
		return
	}

	settlements, err := s.settlementService.GetSettlements(r.Context(), matchID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, settlements)
}

// 获取待结算处理器
func (s *DefaultAPIServer) handleGetPendingSettlements(w http.ResponseWriter, r *http.Request) {
	settlements, err := s.settlementService.GetPendingSettlements(r.Context())
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, settlements)
}

// 订阅处理器
func (s *DefaultAPIServer) handleSubscriptions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		subscriptions, err := s.subscriptionService.GetSubscriptions(r.Context())
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		s.writeJSON(w, http.StatusOK, subscriptions)

	case http.MethodPost:
		matchID := r.URL.Query().Get("match_id")
		if matchID == "" {
			s.writeError(w, http.StatusBadRequest, "match_id is required")
			return
		}

		if err := s.subscriptionService.Subscribe(r.Context(), matchID); err != nil {
			s.writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		s.writeJSON(w, http.StatusOK, map[string]string{"status": "subscribed"})

	case http.MethodDelete:
		matchID := r.URL.Query().Get("match_id")
		if matchID == "" {
			s.writeError(w, http.StatusBadRequest, "match_id is required")
			return
		}

		if err := s.subscriptionService.Unsubscribe(r.Context(), matchID); err != nil {
			s.writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		s.writeJSON(w, http.StatusOK, map[string]string{"status": "unsubscribed"})

	default:
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// 订阅统计处理器
func (s *DefaultAPIServer) handleSubscriptionStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.subscriptionService.GetSubscriptionStats(r.Context())
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, stats)
}

// 获取可预订比赛处理器
func (s *DefaultAPIServer) handleGetBookableMatches(w http.ResponseWriter, r *http.Request) {
	date := time.Now()
	if dateStr := r.URL.Query().Get("date"); dateStr != "" {
		var err error
		date, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			s.writeError(w, http.StatusBadRequest, "invalid date format")
			return
		}
	}

	matches, err := s.bookingService.GetBookableMatches(r.Context(), date)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, matches)
}

// 预订比赛处理器
func (s *DefaultAPIServer) handleBookMatch(w http.ResponseWriter, r *http.Request) {
	matchID := r.URL.Query().Get("match_id")
	if matchID == "" {
		s.writeError(w, http.StatusBadRequest, "match_id is required")
		return
	}

	if err := s.bookingService.BookMatch(r.Context(), matchID); err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]string{"status": "booked"})
}

// 日志中间件
func (s *DefaultAPIServer) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		s.logger.Debug("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
		s.logger.Debug("%s %s completed in %v", r.Method, r.URL.Path, time.Since(start))
	})
}

// CORS 中间件
func (s *DefaultAPIServer) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// writeJSON 写入 JSON 响应
func (s *DefaultAPIServer) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError 写入错误响应
func (s *DefaultAPIServer) writeError(w http.ResponseWriter, status int, message string) {
	s.writeJSON(w, status, map[string]string{"error": message})
}

