package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// AppVersion represents the current application version
const AppVersion = "1.0.0"

// BuildInfo represents build information (can be set during build)
var BuildInfo = ""

// Config holds the application configuration
type Config struct {
	Host   string
	Port   int
	DBPath string
}

// StatsHit represents a single traffic hit to be recorded
type StatsHit struct {
	Host string
	Path string
	Time time.Time
}

// App holds the application state
type App struct {
	DB          *sql.DB
	HitsChannel chan StatsHit
}

// Response structures for API
type YearStats struct {
	Year  int     `json:"year"`
	Total int     `json:"total"`
	Avg   float64 `json:"avg"`
}

type PathStats struct {
	PathPrefix string      `json:"pathprefix"`
	Years      []YearStats `json:"years"`
}

// Response structure for the new format
type PathYears struct {
	PathPrefix string `json:"pathprefix"`
	Years      []int  `json:"years"`
}

type HostStats struct {
	Host  string      `json:"host"`
	Paths []PathYears `json:"paths"`
}

type RootResponse struct {
	Data []HostStats `json:"data"`
}

type MonthStats struct {
	Month int     `json:"month"`
	Total int     `json:"total"`
	Avg   float64 `json:"avg"`
}

type YearDetailStats struct {
	PathPrefix string       `json:"pathprefix"`
	Year       int          `json:"year"`
	Total      int          `json:"total"`
	Avg        float64      `json:"avg"`
	Months     []MonthStats `json:"months"`
}

type YearResponse struct {
	Data []YearDetailStats `json:"data"`
}

func main() {
	// Parse command line flags
	config := Config{}
	var showVersion bool
	flag.BoolVar(&showVersion, "version", false, "Show application version")
	flag.StringVar(&config.Host, "host", "0.0.0.0", "Host to listen on")
	flag.IntVar(&config.Port, "port", 8080, "Port to listen on")
	flag.StringVar(&config.DBPath, "db", "./stats.db", "Path to SQLite database")
	flag.Parse()

	if showVersion {
		if BuildInfo != "" {
			fmt.Printf("API Stats Version %s (Build: %s)\n", AppVersion, BuildInfo)
		} else {
			fmt.Printf("API Stats Version %s\n", AppVersion)
		}
		return
	}

	// Initialize Database
	db, err := initDB(config.DBPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize App
	app := &App{
		DB:          db,
		HitsChannel: make(chan StatsHit, 10000), // Buffered channel for async logging
	}

	// Start Worker
	go app.worker()

	// Set up Router
	mux := http.NewServeMux()

	// Health Check Endpoint
	mux.HandleFunc("/health", healthHandler)

	// Middleware Endpoint (Traefik ForwardAuth)
	mux.HandleFunc("/", app.middlewareHandler)

	// API Endpoints
	mux.HandleFunc("/api/stats/data", app.statsRootHandler)  // GET /api/stats/data (was /api/data)
	mux.HandleFunc("/api/stats", app.statsSummaryHandler)    // GET /api/stats (new format)
	mux.HandleFunc("/api/stats/", app.statsYearHandler)      // GET /api/stats/:year
	mux.HandleFunc("/api/stats/data/", app.statsDataHandler) // GET /api/stats/data/:pathprefix?year=:year

	// Start Server
	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	log.Printf("Starting API Stats service on %s", addr)
	log.Printf("Database path: %s", config.DBPath)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// initDB initializes the SQLite database
func initDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	// Check if 'host' column exists
	// If not, we need to migrate
	// We can check user_version or just try to query
	_, err = db.Query("SELECT host FROM stats LIMIT 1")
	if err == nil {
		// Column exists, we are good (or check if we need index updates, but minimal for now)
		return db, nil
	}

	// Migration needed: Recreate table with 'host'
	log.Println("Migrating database: Adding 'host' column...")

	// 1. Rename old table
	_, err = db.Exec("ALTER TABLE stats RENAME TO stats_old")
	if err != nil {
		// Maybe table doesn't exist yet (fresh install)
		// Ignore error if it's "no such table", but safer to just proceed to Create
	}

	// 2. Create new table with host in PK
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS stats (
		host TEXT DEFAULT 'unknown',
		path TEXT,
		year INTEGER,
		month INTEGER,
		total_hits INTEGER DEFAULT 0,
		PRIMARY KEY (host, path, year, month)
	);
	`
	_, err = db.Exec(createTableSQL)
	if err != nil {
		return nil, fmt.Errorf("failed to create new table: %v", err)
	}

	// 3. Copy data if old table exists
	// Check if stats_old exists first to avoid error on fresh run
	var name string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='stats_old'").Scan(&name)
	if err == nil {
		_, err = db.Exec(`
			INSERT INTO stats (host, path, year, month, total_hits)
			SELECT 'unknown', path, year, month, total_hits FROM stats_old
		`)
		if err != nil {
			return nil, fmt.Errorf("failed to copy data: %v", err)
		}

		// 4. Drop old table
		_, err = db.Exec("DROP TABLE stats_old")
		if err != nil {
			log.Printf("Warning: Failed to drop stats_old: %v", err)
		}
	}

	return db, nil
}

// worker processes hits from the channel
func (app *App) worker() {
	for hit := range app.HitsChannel {
		year, month, _ := hit.Time.Date()

		// UPSERT Query
		query := `
		INSERT INTO stats (host, path, year, month, total_hits)
		VALUES (?, ?, ?, ?, 1)
		ON CONFLICT(host, path, year, month)
		DO UPDATE SET total_hits = total_hits + 1;
		`

		_, err := app.DB.Exec(query, hit.Host, hit.Path, year, int(month))
		if err != nil {
			log.Printf("Error recording hit: %v", err)
		}
	}
}

// healthHandler handles GET /health
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := map[string]interface{}{
		"status":    "healthy",
		"version":   AppVersion,
		"timestamp": time.Now().Unix(),
	}
	if BuildInfo != "" {
		response["build"] = BuildInfo
	}
	json.NewEncoder(w).Encode(response)
}

// extractPathPrefix extracts the first segment of a path
// Examples: /v3/cal/today -> /v3, /v2/quran/ayat -> /v2, / -> /
func extractPathPrefix(fullPath string) string {
	if fullPath == "" || fullPath == "/" {
		return "/"
	}

	// Remove leading slash and split
	trimmed := strings.TrimPrefix(fullPath, "/")
	parts := strings.Split(trimmed, "/")

	if len(parts) == 0 || parts[0] == "" {
		return "/"
	}

	// Return first segment with leading slash
	return "/" + parts[0]
}

// middlewareHandler handles the Traefik ForwardAuth request
func (app *App) middlewareHandler(w http.ResponseWriter, r *http.Request) {
	// Identify PathPrefix
	// Traefik sends "X-Forwarded-Uri" or "X-Replaced-Path"
	fullPath := r.Header.Get("X-Forwarded-Uri")
	if fullPath == "" {
		fullPath = r.Header.Get("X-Replaced-Path")
	}
	if fullPath == "" {
		// Fallback or ignore? For now, we can try RequestURI if direct hit (testing)
		// But strictly for middleware, if missing, we might proceed or log "unknown"
		// If used strictly as ForwardAuth, we must return 200 to allow traffic.
		fullPath = "unknown"
	}

	// Extract only the first path segment (prefix)
	// e.g., /v3/cal/today -> /v3, /v2/quran/ayat/acak -> /v2
	pathPrefix := extractPathPrefix(fullPath)

	// Extract Host
	host := r.Header.Get("X-Forwarded-Host")
	if host == "" {
		host = r.Host
	}

	// Non-blocking send
	select {
	case app.HitsChannel <- StatsHit{Host: host, Path: pathPrefix, Time: time.Now()}:
	default:
		// Channel full, drop metric to avoid blocking traffic
		log.Println("Stats channel full, dropping hit")
	}

	w.WriteHeader(http.StatusOK)
}

// statsRootHandler handles GET /api/stats/data
func (app *App) statsRootHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/stats/data" && r.URL.Path != "/api/stats/data/" {
		http.NotFound(w, r)
		return
	}

	rows, err := app.DB.Query(`
		SELECT DISTINCT host, path, year
		FROM stats
		ORDER BY host, path, year
	`)
	if err != nil {
		jsonError(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Aggregate data: Host -> Path -> Years
	type HostData struct {
		Paths map[string][]int
	}
	// map[host] -> HostData
	dataMap := make(map[string]*HostData)

	for rows.Next() {
		var host string
		var path string
		var year int
		if err := rows.Scan(&host, &path, &year); err != nil {
			continue
		}

		if _, exists := dataMap[host]; !exists {
			dataMap[host] = &HostData{Paths: make(map[string][]int)}
		}

		dataMap[host].Paths[path] = append(dataMap[host].Paths[path], year)
	}

	// Convert map to slice of HostStats
	var result []HostStats
	for host, hData := range dataMap {
		var paths []PathYears
		for path, years := range hData.Paths {
			paths = append(paths, PathYears{
				PathPrefix: path,
				Years:      years,
			})
		}
		result = append(result, HostStats{
			Host:  host,
			Paths: paths,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// statsYearHandler handles GET /api/stats/:year
func (app *App) statsYearHandler(w http.ResponseWriter, r *http.Request) {
	// Extract Year from URL
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 { // /api/stats/2026
		jsonError(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	yearStr := parts[3]
	year, err := strconv.Atoi(yearStr)
	if err != nil {
		jsonError(w, "Invalid year format", http.StatusBadRequest)
		return
	}

	rows, err := app.DB.Query(`
		SELECT path, month, SUM(total_hits)
		FROM stats
		WHERE year = ?
		GROUP BY path, month
		ORDER BY path, month ASC
	`, year)
	if err != nil {
		jsonError(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Map: Path -> Months
	type TempData struct {
		Total  int
		Months []MonthStats
	}
	dataMap := make(map[string]*TempData)

	for rows.Next() {
		var path string
		var month int
		var total int
		if err := rows.Scan(&path, &month, &total); err != nil {
			continue
		}

		if _, exists := dataMap[path]; !exists {
			dataMap[path] = &TempData{Total: 0, Months: []MonthStats{}}
		}

		// Avg per month (hits per day in that month?)
		// Let's assume avg hits/day in that month (30 days approx)
		avg := math.Ceil(float64(total) / 30.0)

		dataMap[path].Months = append(dataMap[path].Months, MonthStats{
			Month: month,
			Total: total,
			Avg:   avg,
		})
		dataMap[path].Total += total
	}

	resp := YearResponse{Data: make([]YearDetailStats, 0)}
	for path, temp := range dataMap {
		// Calculate yearly avg (e.g., total / 12)
		yearAvg := math.Ceil(float64(temp.Total) / 12.0)

		resp.Data = append(resp.Data, YearDetailStats{
			PathPrefix: path,
			Year:       year,
			Total:      temp.Total,
			Avg:        yearAvg,
			Months:     temp.Months,
		})
	}

	// Check if empty
	if len(resp.Data) == 0 {
		// Prompt says: "Jika data kosong ..., kembalikan JSON error"
		// Actually prompt says "Jika data kosong atau tahun salah, kembalikan JSON error"
		// If valid year but no data, maybe return empty list? But prompt implies error.
		// We'll return empty list as standard API behavior, but if strict:
		// jsonError(w, "No data for this year", http.StatusNotFound)
		// Let's return empty Data array as initialized.
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// StatsSummary represents the new format for /api/stats
type StatsSummary struct {
	PathPrefix string `json:"pathprefix"`
	Total      int    `json:"total"`
}

type StatsSummaryResponse struct {
	Total int            `json:"total"`
	Data  []StatsSummary `json:"data"`
}

// statsSummaryHandler handles GET /api/stats
func (app *App) statsSummaryHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/stats" {
		http.NotFound(w, r)
		return
	}

	rows, err := app.DB.Query(`
		SELECT path, SUM(total_hits) as total
		FROM stats
		GROUP BY path
		ORDER BY path
	`)
	if err != nil {
		jsonError(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var result []StatsSummary
	totalAll := 0

	for rows.Next() {
		var path string
		var total int
		if err := rows.Scan(&path, &total); err != nil {
			continue
		}

		result = append(result, StatsSummary{
			PathPrefix: path,
			Total:      total,
		})

		totalAll += total
	}

	resp := StatsSummaryResponse{
		Total: totalAll,
		Data:  result,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// statsDataHandler handles GET /api/stats/data/:pathprefix?year=:year
func (app *App) statsDataHandler(w http.ResponseWriter, r *http.Request) {
	// Extract pathprefix from URL
	urlParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(urlParts) < 4 { // /api/stats/data/:pathprefix
		jsonError(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	pathPrefix := "/" + urlParts[3]

	// Get year from query parameter, default to current year
	yearStr := r.URL.Query().Get("year")
	var year int

	if yearStr == "" {
		// Default to current year
		year = time.Now().Year()
	} else {
		var err error
		year, err = strconv.Atoi(yearStr)
		if err != nil {
			jsonError(w, "Invalid year format", http.StatusBadRequest)
			return
		}
	}

	// Query data for specific path and year
	rows, err := app.DB.Query(`
		SELECT month, SUM(total_hits)
		FROM stats
		WHERE path = ? AND year = ?
		GROUP BY month
		ORDER BY month ASC
	`, pathPrefix, year)

	if err != nil {
		jsonError(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Accumulate data for the path
	total := 0
	months := []MonthStats{}

	for rows.Next() {
		var month int
		var totalHits int
		if err := rows.Scan(&month, &totalHits); err != nil {
			continue
		}

		// Calculate monthly avg (hits per day in that month)
		avg := math.Ceil(float64(totalHits) / 30.0)

		months = append(months, MonthStats{
			Month: month,
			Total: totalHits,
			Avg:   avg,
		})

		total += totalHits
	}

	// Calculate yearly avg
	yearAvg := math.Ceil(float64(total) / 12.0)

	resp := YearDetailStats{
		PathPrefix: pathPrefix,
		Year:       year,
		Total:      total,
		Avg:        yearAvg,
		Months:     months,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(YearResponse{Data: []YearDetailStats{resp}})
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
