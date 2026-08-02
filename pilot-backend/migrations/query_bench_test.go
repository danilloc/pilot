package migrations

import (
	"database/sql"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"
)

// This test seeds the trips table to >= 1,000,000 rows and times the three
// critical queries from design.md's "Queries Críticas" section. It is
// expensive (multi-minute seed) and mutates shared server state (slow query
// log settings), so it only runs when explicitly requested — never as part
// of a normal `go test ./...`.
const runPerfTestsEnv = "RUN_PERF_TESTS"

const (
	perfTargetTotalRows  = 1_000_000
	perfFillerDrivers    = 50
	perfFillerSeedPerDrv = 40
	perfHeroDays         = 30
	perfHeroTripsPerDay  = 60
	perfThreshold        = 100 * time.Millisecond
)

func TestQueryPerformance_Under1MRows(t *testing.T) {
	if os.Getenv(runPerfTestsEnv) != "1" {
		t.Skipf("skipping expensive 1M-row performance test; set %s=1 to run it", runPerfTestsEnv)
	}

	db := testDB(t)
	heroDriverID := seedPerformanceData(t, db)

	restoreSlowLog := enableSlowQueryLog(t, db, perfThreshold)
	defer restoreSlowLog()

	t.Run("daily_earnings", func(t *testing.T) {
		query := `SELECT
			SUM(fare_value) as total_earned,
			COUNT(*) as total_trips,
			AVG(fare_value) as avg_fare,
			SUM(distance_km) as total_distance,
			SUM(fare_value) / (SUM(duration_minutes) / 60) as earnings_per_hour
		FROM trips
		WHERE driver_id = ?
		  AND ended_at >= CURDATE()
		  AND ended_at < CURDATE() + INTERVAL 1 DAY
		  AND status = 'COMPLETED'`

		var totalEarned, avgFare, totalDistance, earningsPerHour sql.NullFloat64
		var totalTrips int
		elapsed := timedScan(t, db, query, []interface{}{heroDriverID}, &totalEarned, &totalTrips, &avgFare, &totalDistance, &earningsPerHour)

		if totalTrips == 0 {
			t.Fatal("daily earnings query matched 0 trips, want > 0 (seed didn't include today's trips)")
		}
		if elapsed > perfThreshold {
			t.Errorf("query took %v, want < %v", elapsed, perfThreshold)
		}

		// The half-open range is sargable, but for a window this narrow
		// (~1 day) the cost-based optimizer picks idx_ended_at over the
		// composite idx_driver_ended: across the whole table, "today"
		// matches so few rows database-wide that scanning by ended_at
		// alone (then filtering driver_id) is cheaper than seeking into
		// idx_driver_ended for one driver's slice. Confirmed via EXPLAIN;
		// this is data-distribution-dependent optimizer behavior, not a
		// missing index — weekly_stats below, with a wider 8-day window,
		// does use idx_driver_ended.
		assertExplainUsesIndex(t, db,
			"SELECT * FROM trips WHERE driver_id = ? AND ended_at >= CURDATE() AND ended_at < CURDATE() + INTERVAL 1 DAY AND status = 'COMPLETED'",
			[]interface{}{heroDriverID}, "trips", "idx_ended_at")
	})

	t.Run("weekly_stats", func(t *testing.T) {
		query := `SELECT
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
		ORDER BY day DESC`

		start := time.Now()
		rows, err := db.Query(query, heroDriverID)
		if err != nil {
			t.Fatalf("query error = %v", err)
		}
		dayCount := 0
		for rows.Next() {
			dayCount++
		}
		elapsed := time.Since(start)
		if err := rows.Err(); err != nil {
			t.Fatalf("rows.Err() = %v", err)
		}
		rows.Close()

		if dayCount == 0 {
			t.Fatal("weekly stats query matched 0 days, want > 0")
		}
		if elapsed > perfThreshold {
			t.Errorf("query took %v, want < %v", elapsed, perfThreshold)
		}

		// Same sargable date-range form as daily_earnings above.
		assertExplainUsesIndex(t, db,
			"SELECT * FROM trips WHERE driver_id = ? AND ended_at >= DATE_SUB(CURDATE(), INTERVAL 7 DAY) AND ended_at < CURDATE() + INTERVAL 1 DAY AND status = 'COMPLETED'",
			[]interface{}{heroDriverID}, "trips", "idx_driver_ended")
	})

	t.Run("goal_progress", func(t *testing.T) {
		query := `SELECT
			dg.goal_amount,
			COALESCE(SUM(t.fare_value), 0) as actual_amount,
			(COALESCE(SUM(t.fare_value), 0) / dg.goal_amount * 100) as percent
		FROM daily_goals dg
		LEFT JOIN trips t ON t.driver_id = dg.driver_id
		    AND t.ended_at >= dg.goal_date
		    AND t.ended_at < dg.goal_date + INTERVAL 1 DAY
		    AND t.status = 'COMPLETED'
		WHERE dg.driver_id = ? AND dg.goal_date = CURDATE()
		GROUP BY dg.id`

		var goalAmount, actualAmount, percent sql.NullFloat64
		elapsed := timedScan(t, db, query, []interface{}{heroDriverID}, &goalAmount, &actualAmount, &percent)

		if !goalAmount.Valid || goalAmount.Float64 <= 0 {
			t.Fatalf("goal_amount = %v, want a positive seeded value", goalAmount)
		}
		if elapsed > perfThreshold {
			t.Errorf("query took %v, want < %v", elapsed, perfThreshold)
		}

		// EXPLAIN reports table_name as the query's alias, not the base
		// table name, when one is used (dg = daily_goals, t = trips).
		assertExplainUsesIndex(t, db,
			"SELECT dg.goal_amount, COALESCE(SUM(t.fare_value), 0) as actual_amount FROM daily_goals dg LEFT JOIN trips t ON t.driver_id = dg.driver_id AND DATE(t.ended_at) = dg.goal_date AND t.status = 'COMPLETED' WHERE dg.driver_id = ? AND dg.goal_date = CURDATE() GROUP BY dg.id",
			[]interface{}{heroDriverID}, "dg", "unique_driver_date")
	})

	assertSlowLogEmpty(t, db)
}

// timedScan executes query (measuring wall-clock time from dispatch through
// the row becoming available) and scans the single resulting row into dest.
func timedScan(t *testing.T, db *sql.DB, query string, args []interface{}, dest ...interface{}) time.Duration {
	t.Helper()
	start := time.Now()
	row := db.QueryRow(query, args...)
	err := row.Scan(dest...)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("scan error = %v", err)
	}
	return elapsed
}

// seedPerformanceData seeds a hero driver with a realistic, bounded trip
// history (so query result sets stay realistic) and pads the trips table
// to perfTargetTotalRows total via server-side self-duplication of a small
// filler seed — much faster than sending 1M rows from Go. Idempotent: reuses
// existing data if a prior run already seeded it.
func seedPerformanceData(t *testing.T, db *sql.DB) int64 {
	t.Helper()

	const heroEmail = "perf-hero@example.com"
	var heroDriverID int64
	err := db.QueryRow("SELECT id FROM drivers WHERE email = ?", heroEmail).Scan(&heroDriverID)
	if err == sql.ErrNoRows {
		res, err := db.Exec("INSERT INTO drivers (uuid, uber_id, name, email, rating) VALUES (UUID(), 'perf-hero', 'Perf Hero', ?, 4.9)", heroEmail)
		if err != nil {
			t.Fatalf("insert hero driver error = %v", err)
		}
		heroDriverID, err = res.LastInsertId()
		if err != nil {
			t.Fatalf("LastInsertId() error = %v", err)
		}
		seedHeroTrips(t, db, heroDriverID)
		seedHeroGoal(t, db, heroDriverID)
	} else if err != nil {
		t.Fatalf("query hero driver error = %v", err)
	}

	var total int
	if err := db.QueryRow("SELECT COUNT(*) FROM trips WHERE driver_id != ?", heroDriverID).Scan(&total); err != nil {
		t.Fatalf("count filler trips error = %v", err)
	}

	if total == 0 {
		seedFillerDrivers(t, db)
		if err := db.QueryRow("SELECT COUNT(*) FROM trips WHERE driver_id != ?", heroDriverID).Scan(&total); err != nil {
			t.Fatalf("count filler trips error = %v", err)
		}
	}

	for total < perfTargetTotalRows {
		res, err := db.Exec(`
			INSERT INTO trips (uuid, driver_id, uber_trip_id, started_at, ended_at, distance_km, fare_value, currency, city, status)
			SELECT UUID(), driver_id,
			       CONCAT('filler-', driver_id, '-', SUBSTRING(MD5(RAND()), 1, 24), '-', SUBSTRING(MD5(RAND()), 1, 8)),
			       started_at, ended_at, distance_km, fare_value, currency, city, status
			FROM trips WHERE driver_id != ?`, heroDriverID)
		if err != nil {
			t.Fatalf("doubling insert error = %v", err)
		}
		added, _ := res.RowsAffected()
		total += int(added)
		t.Logf("seed: filler trips now ~%d", total)
	}

	return heroDriverID
}

func seedHeroTrips(t *testing.T, db *sql.DB, heroDriverID int64) {
	t.Helper()
	now := time.Now()

	const batchSize = 200
	values := make([]string, 0, batchSize)
	args := make([]interface{}, 0, batchSize*6)
	flush := func() {
		if len(values) == 0 {
			return
		}
		query := fmt.Sprintf(
			"INSERT INTO trips (uuid, driver_id, uber_trip_id, started_at, ended_at, distance_km, fare_value, status) VALUES %s",
			strings.Join(values, ","),
		)
		if _, err := db.Exec(query, args...); err != nil {
			t.Fatalf("insert hero trips batch error = %v", err)
		}
		values = values[:0]
		args = args[:0]
	}

	tripN := 0
	for day := 0; day < perfHeroDays; day++ {
		dayStart := now.AddDate(0, 0, -day)
		for i := 0; i < perfHeroTripsPerDay; i++ {
			started := dayStart.Add(-time.Duration(i) * time.Hour / 4)
			ended := started.Add(20 * time.Minute)
			tripN++
			values = append(values, "(UUID(), ?, ?, ?, ?, ?, ?, 'COMPLETED')")
			args = append(args,
				heroDriverID,
				fmt.Sprintf("hero-trip-%d", tripN),
				started.Format("2006-01-02 15:04:05"),
				ended.Format("2006-01-02 15:04:05"),
				5+rand.Float64()*20,
				15+rand.Float64()*60,
			)
			if len(values) >= batchSize {
				flush()
			}
		}
	}
	flush()
}

func seedHeroGoal(t *testing.T, db *sql.DB, heroDriverID int64) {
	t.Helper()
	if _, err := db.Exec(
		"INSERT INTO daily_goals (uuid, driver_id, goal_date, goal_amount) VALUES (UUID(), ?, CURDATE(), 350.00)",
		heroDriverID,
	); err != nil {
		t.Fatalf("insert hero goal error = %v", err)
	}
}

func seedFillerDrivers(t *testing.T, db *sql.DB) {
	t.Helper()
	now := time.Now()

	for d := 0; d < perfFillerDrivers; d++ {
		email := fmt.Sprintf("perf-filler-%d@example.com", d)
		res, err := db.Exec(
			"INSERT INTO drivers (uuid, uber_id, name, email, rating) VALUES (UUID(), ?, ?, ?, 4.5)",
			fmt.Sprintf("perf-filler-%d", d), fmt.Sprintf("Filler Driver %d", d), email,
		)
		if err != nil {
			t.Fatalf("insert filler driver error = %v", err)
		}
		driverID, err := res.LastInsertId()
		if err != nil {
			t.Fatalf("LastInsertId() error = %v", err)
		}

		values := make([]string, 0, perfFillerSeedPerDrv)
		args := make([]interface{}, 0, perfFillerSeedPerDrv*6)
		for i := 0; i < perfFillerSeedPerDrv; i++ {
			started := now.AddDate(0, 0, -rand.Intn(730)).Add(-time.Duration(rand.Intn(24)) * time.Hour)
			ended := started.Add(20 * time.Minute)
			values = append(values, "(UUID(), ?, ?, ?, ?, ?, ?, 'COMPLETED')")
			args = append(args,
				driverID,
				fmt.Sprintf("filler-seed-%d-%d", d, i),
				started.Format("2006-01-02 15:04:05"),
				ended.Format("2006-01-02 15:04:05"),
				5+rand.Float64()*20,
				15+rand.Float64()*60,
			)
		}
		query := fmt.Sprintf(
			"INSERT INTO trips (uuid, driver_id, uber_trip_id, started_at, ended_at, distance_km, fare_value, status) VALUES %s",
			strings.Join(values, ","),
		)
		if _, err := db.Exec(query, args...); err != nil {
			t.Fatalf("insert filler trips batch error = %v", err)
		}
	}
}

// enableSlowQueryLog configures MySQL to log queries slower than threshold
// into mysql.slow_log (log_output=TABLE), returning a func that restores
// the server's original settings. Local dev instance only.
func enableSlowQueryLog(t *testing.T, db *sql.DB, threshold time.Duration) func() {
	t.Helper()

	var origEnabled, origOutput string
	var origTime float64
	if err := db.QueryRow("SELECT @@global.slow_query_log, @@global.log_output, @@global.long_query_time").
		Scan(&origEnabled, &origOutput, &origTime); err != nil {
		t.Fatalf("reading original slow log settings error = %v", err)
	}

	exec := func(stmt string) {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("exec %q error = %v", stmt, err)
		}
	}

	exec("SET GLOBAL log_output = 'TABLE'")
	exec(fmt.Sprintf("SET GLOBAL long_query_time = %f", threshold.Seconds()))
	exec("SET GLOBAL slow_query_log = 'ON'")
	exec("TRUNCATE TABLE mysql.slow_log")

	// slow_query_log rejects the quoted string form of its boolean value
	// (e.g. SET GLOBAL slow_query_log = '0' errors with 42000); '0'/'1' as
	// read back from @@global.slow_query_log must be normalized to the
	// ON/OFF keyword form before being used in a SET statement.
	origEnabledKeyword := "OFF"
	if origEnabled == "1" {
		origEnabledKeyword = "ON"
	}

	return func() {
		exec("SET GLOBAL slow_query_log = " + origEnabledKeyword)
		exec("SET GLOBAL log_output = '" + origOutput + "'")
		exec(fmt.Sprintf("SET GLOBAL long_query_time = %f", origTime))
	}
}

func assertSlowLogEmpty(t *testing.T, db *sql.DB) {
	t.Helper()
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM mysql.slow_log WHERE sql_text LIKE '%%trips%%'").Scan(&count); err != nil {
		t.Fatalf("querying mysql.slow_log error = %v", err)
	}
	if count != 0 {
		t.Errorf("mysql.slow_log has %d entries above the %v threshold, want 0", count, perfThreshold)
	}
}
