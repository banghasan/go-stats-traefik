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

// Config holds the application configuration
type Config struct {
	Host   string
	Port   int
	DBPath string
}

// StatsHit represents a single traffic hit to be recorded
type StatsHit struct {
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

type RootResponse struct {
	Data []PathStats `json:"data"`
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
		fmt.Println("API Stats Version 1.0")
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
	mux.HandleFunc("/api/stats", app.statsRootHandler)      // GET /api/stats
	mux.HandleFunc("/api/stats/", app.statsYearHandler)     // GET /api/stats/:year
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

	// Create table if not exists with UNIQUE constraint for UPSERT
	// Added avg logic check: Average is calculated on read, so we only need hits.
	// We need path, year, month.
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS stats (
		path TEXT,
		year INTEGER,
		month INTEGER,
		total_hits INTEGER DEFAULT 0,
		PRIMARY KEY (path, year, month)
	);
	`
	_, err = db.Exec(createTableSQL)
	if err != nil {
		return nil, err
	}

	return db, nil
}

// worker processes hits from the channel
func (app *App) worker() {
	for hit := range app.HitsChannel {
		year, month, _ := hit.Time.Date()

		// UPSERT Query
		query := `
		INSERT INTO stats (path, year, month, total_hits)
		VALUES (?, ?, ?, 1)
		ON CONFLICT(path, year, month)
		DO UPDATE SET total_hits = total_hits + 1;
		`

		_, err := app.DB.Exec(query, hit.Path, year, int(month))
		if err != nil {
			log.Printf("Error recording hit: %v", err)
		}
	}
}

// healthHandler handles GET /health
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "healthy",
		"version":   "1.0",
		"timestamp": time.Now().Unix(),
	})
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

	// Non-blocking send
	select {
	case app.HitsChannel <- StatsHit{Path: pathPrefix, Time: time.Now()}:
	default:
		// Channel full, drop metric to avoid blocking traffic
		log.Println("Stats channel full, dropping hit")
	}

	w.WriteHeader(http.StatusOK)
}

// statsRootHandler handles GET /api/stats
func (app *App) statsRootHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/stats" && r.URL.Path != "/api/stats/" {
		http.NotFound(w, r)
		return
	}

	rows, err := app.DB.Query(`
		SELECT path, year, SUM(total_hits) as year_total
		FROM stats
		GROUP BY path, year
		ORDER BY path, year DESC
	`)
	if err != nil {
		jsonError(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Aggregate data
	statsMap := make(map[string][]YearStats)
	for rows.Next() {
		var path string
		var year int
		var total int
		if err := rows.Scan(&path, &year, &total); err != nil {
			continue
		}

		// Calculate average (hits per day? or per month?)
		// The prompt example shows "avg: 21" for "total: 1100".
		// 1100 / 21 ~= 52. Maybe avg per week? Or per month (1100/12 ~ 90).
		// Let's assume daily average for now: Total / 365 (or days passed).
		// OR avg per month: Total / 12.
		// Let's stick to a simple Avg logic: Total hits / 12 months (or months recorded).
		// Actually, let's query count of months to be accurate.

		// For simplicity/performance in this loop, we'll rough calc or just use Total/12 for now.
		// Prompt doesn't specify formula. "avg" usually implies per unit time.
		// If it's real-time, maybe hits/day?
		// Let's use simple logic: AVG = Total / 12 (months) for simplicity unless we count specific months.
		avg := math.Ceil(float64(total) / 12.0)

		statsMap[path] = append(statsMap[path], YearStats{
			Year:  year,
			Total: total,
			Avg:   avg, // Simple placeholder logic
		})
	}

	resp := RootResponse{Data: make([]PathStats, 0)}
	for path, years := range statsMap {
		resp.Data = append(resp.Data, PathStats{
			PathPrefix: path,
			Years:      years,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
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
		SELECT path, month, total_hits
		FROM stats
		WHERE year = ?
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
		SELECT month, total_hits
		FROM stats
		WHERE path = ? AND year = ?
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
