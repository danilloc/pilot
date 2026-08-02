package service

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"gorm.io/gorm"

	"pilot-backend/internal/models"
	"pilot-backend/pkg/cache"
	"pilot-backend/pkg/logger"
)

const statsCacheTTL = 5 * time.Minute

// DailyStats is the response shape for GET /api/stats/today.
type DailyStats struct {
	Date            string  `json:"date"`
	TotalEarned     float64 `json:"total_earned"`
	TotalTrips      int     `json:"total_trips"`
	AvgFare         float64 `json:"avg_fare"`
	TotalDistance   float64 `json:"total_distance"`
	EarningsPerHour float64 `json:"earnings_per_hour"`
	ActiveHours     float64 `json:"active_hours"`
}

// DayStat is one day's entry within GET /api/stats/week.
type DayStat struct {
	Date     string  `json:"date"`
	Earnings float64 `json:"earnings"`
	Trips    int     `json:"trips"`
}

// StatsSummary aggregates a period (week or month) of DayStat/WeekStat data.
type StatsSummary struct {
	TotalEarned float64 `json:"total_earned"`
	TotalTrips  int     `json:"total_trips"`
	AvgFare     float64 `json:"avg_fare"`
}

// WeeklyStats is the response shape for GET /api/stats/week.
type WeeklyStats struct {
	Period  string       `json:"period"`
	Days    []DayStat    `json:"days"`
	Summary StatsSummary `json:"summary"`
}

// WeekStat is one calendar week's entry within GET /api/stats/month.
type WeekStat struct {
	WeekStart string  `json:"week_start"`
	Earnings  float64 `json:"earnings"`
	Trips     int     `json:"trips"`
}

// MonthlyStats is the response shape for GET /api/stats/month.
type MonthlyStats struct {
	Month   string       `json:"month"`
	Weeks   []WeekStat   `json:"weeks"`
	Summary StatsSummary `json:"summary"`
}

// StatsService computes earnings statistics from trip history, caching
// each response for statsCacheTTL to keep repeated dashboard loads cheap.
type StatsService struct {
	db    *gorm.DB
	cache *cache.Redis
	log   *logger.Logger
}

// NewStatsService builds a StatsService from its collaborators.
func NewStatsService(db *gorm.DB, redisCache *cache.Redis, log *logger.Logger) *StatsService {
	return &StatsService{db: db, cache: redisCache, log: log}
}

type dailyStatsRow struct {
	TotalEarned     *float64
	TotalTrips      int
	AvgFare         *float64
	TotalDistance   *float64
	EarningsPerHour *float64
	ActiveHours     *float64
}

// Today computes today's earnings stats for driverID, using the same
// sargable date-range predicate as design.md's confirmed daily_earnings
// query (see database feature) so idx_driver_ended is used in full.
func (s *StatsService) Today(ctx context.Context, driverID int64) (*DailyStats, error) {
	today := time.Now().Format("2006-01-02")
	cacheKey := statsCacheKey("today", driverID, today)

	var cached DailyStats
	if s.readCache(ctx, cacheKey, &cached) {
		return &cached, nil
	}

	var row dailyStatsRow
	err := s.db.WithContext(ctx).Raw(`
		SELECT
			SUM(fare_value) as total_earned,
			COUNT(*) as total_trips,
			AVG(fare_value) as avg_fare,
			SUM(distance_km) as total_distance,
			SUM(fare_value) / NULLIF(SUM(duration_minutes) / 60, 0) as earnings_per_hour,
			SUM(duration_minutes) / 60 as active_hours
		FROM trips
		WHERE driver_id = ?
		  AND ended_at >= CURDATE()
		  AND ended_at < CURDATE() + INTERVAL 1 DAY
		  AND status = 'COMPLETED'`, driverID).Scan(&row).Error
	if err != nil {
		s.log.Errorw("stats.today_query_failed", "driver_id", driverID, "error", err.Error())
		return nil, models.NewAPIError(models.ErrCodeDatabaseError, "database error")
	}

	stats := &DailyStats{
		Date:            today,
		TotalEarned:     derefOr(row.TotalEarned, 0),
		TotalTrips:      row.TotalTrips,
		AvgFare:         derefOr(row.AvgFare, 0),
		TotalDistance:   derefOr(row.TotalDistance, 0),
		EarningsPerHour: derefOr(row.EarningsPerHour, 0),
		ActiveHours:     derefOr(row.ActiveHours, 0),
	}

	s.writeCache(ctx, cacheKey, stats)
	return stats, nil
}

type dayStatRow struct {
	Day      time.Time
	Earnings float64
	Trips    int
	AvgFare  float64
}

// Week computes the last 7 days of earnings stats for driverID, using the
// same sargable date-range predicate as design.md's confirmed weekly_stats
// query.
func (s *StatsService) Week(ctx context.Context, driverID int64) (*WeeklyStats, error) {
	now := time.Now()
	periodStart := now.AddDate(0, 0, -7)
	cacheKey := statsCacheKey("week", driverID, now.Format("2006-01-02"))

	var cached WeeklyStats
	if s.readCache(ctx, cacheKey, &cached) {
		return &cached, nil
	}

	var rows []dayStatRow
	err := s.db.WithContext(ctx).Raw(`
		SELECT
			DATE(ended_at) as day,
			SUM(fare_value) as earnings,
			COUNT(*) as trips,
			AVG(fare_value) as avg_fare
		FROM trips
		WHERE driver_id = ?
		  AND ended_at >= DATE_SUB(CURDATE(), INTERVAL 7 DAY)
		  AND ended_at < CURDATE() + INTERVAL 1 DAY
		  AND status = 'COMPLETED'
		GROUP BY DATE(ended_at)
		ORDER BY day DESC`, driverID).Scan(&rows).Error
	if err != nil {
		s.log.Errorw("stats.week_query_failed", "driver_id", driverID, "error", err.Error())
		return nil, models.NewAPIError(models.ErrCodeDatabaseError, "database error")
	}

	days := make([]DayStat, 0, len(rows))
	summary := StatsSummary{}
	for _, r := range rows {
		days = append(days, DayStat{
			Date:     r.Day.Format("2006-01-02"),
			Earnings: r.Earnings,
			Trips:    r.Trips,
		})
		summary.TotalEarned += r.Earnings
		summary.TotalTrips += r.Trips
	}
	if summary.TotalTrips > 0 {
		summary.AvgFare = summary.TotalEarned / float64(summary.TotalTrips)
	}

	stats := &WeeklyStats{
		Period:  periodStart.Format("2006-01-02") + " to " + now.Format("2006-01-02"),
		Days:    days,
		Summary: summary,
	}

	s.writeCache(ctx, cacheKey, stats)
	return stats, nil
}

type weekStatRow struct {
	WeekStart time.Time
	Earnings  float64
	Trips     int
}

// Month computes calendar-week-grouped earnings stats for the given month
// (format "2006-01"; defaults to the current month when empty). Weeks are
// grouped by their Monday (WEEKDAY() offset), matching common calendar-week
// semantics distinct from the rolling 7-day window Week() uses.
func (s *StatsService) Month(ctx context.Context, driverID int64, month string) (*MonthlyStats, error) {
	if month == "" {
		month = time.Now().Format("2006-01")
	}
	monthStart, err := time.Parse("2006-01", month)
	if err != nil {
		return nil, models.NewAPIError(models.ErrCodeInvalidQueryParam, "invalid month (expected YYYY-MM)")
	}
	monthEnd := monthStart.AddDate(0, 1, 0)

	cacheKey := statsCacheKey("month", driverID, month)
	var cached MonthlyStats
	if s.readCache(ctx, cacheKey, &cached) {
		return &cached, nil
	}

	var rows []weekStatRow
	dbErr := s.db.WithContext(ctx).Raw(`
		SELECT
			DATE(DATE_SUB(ended_at, INTERVAL WEEKDAY(ended_at) DAY)) as week_start,
			SUM(fare_value) as earnings,
			COUNT(*) as trips
		FROM trips
		WHERE driver_id = ?
		  AND ended_at >= ?
		  AND ended_at < ?
		  AND status = 'COMPLETED'
		GROUP BY week_start
		ORDER BY week_start ASC`, driverID, monthStart, monthEnd).Scan(&rows).Error
	if dbErr != nil {
		s.log.Errorw("stats.month_query_failed", "driver_id", driverID, "error", dbErr.Error())
		return nil, models.NewAPIError(models.ErrCodeDatabaseError, "database error")
	}

	weeks := make([]WeekStat, 0, len(rows))
	summary := StatsSummary{}
	for _, r := range rows {
		weeks = append(weeks, WeekStat{
			WeekStart: r.WeekStart.Format("2006-01-02"),
			Earnings:  r.Earnings,
			Trips:     r.Trips,
		})
		summary.TotalEarned += r.Earnings
		summary.TotalTrips += r.Trips
	}
	if summary.TotalTrips > 0 {
		summary.AvgFare = summary.TotalEarned / float64(summary.TotalTrips)
	}

	stats := &MonthlyStats{
		Month:   month,
		Weeks:   weeks,
		Summary: summary,
	}

	s.writeCache(ctx, cacheKey, stats)
	return stats, nil
}

func (s *StatsService) readCache(ctx context.Context, key string, dest interface{}) bool {
	raw, err := s.cache.Get(ctx, key)
	if err != nil {
		if !errors.Is(err, cache.ErrCacheMiss) {
			s.log.Warnw("stats.cache_read_failed", "key", key, "error", err.Error())
		}
		return false
	}
	if err := json.Unmarshal([]byte(raw), dest); err != nil {
		s.log.Warnw("stats.cache_decode_failed", "key", key, "error", err.Error())
		return false
	}
	return true
}

func (s *StatsService) writeCache(ctx context.Context, key string, value interface{}) {
	raw, err := json.Marshal(value)
	if err != nil {
		s.log.Warnw("stats.cache_encode_failed", "key", key, "error", err.Error())
		return
	}
	if err := s.cache.Set(ctx, key, string(raw), statsCacheTTL); err != nil {
		s.log.Warnw("stats.cache_write_failed", "key", key, "error", err.Error())
	}
}

func statsCacheKey(kind string, driverID int64, period string) string {
	return "stats:" + kind + ":" + period + ":" + strconv.FormatInt(driverID, 10)
}

func derefOr(v *float64, fallback float64) float64 {
	if v == nil {
		return fallback
	}
	return *v
}
