package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
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
	Host     string
	Port     int
	DBPath   string
	Timezone string
}

// StatsHit represents a single traffic hit to be recorded
type StatsHit struct {
	Host string
	Path string
	Time time.Time
}

// App holds the application state
type App struct {
	DB           *sql.DB
	HitsChannel  chan StatsHit
	TimeLocation *time.Location
}

// Response structure for the new format
type PrefixData struct {
	Prefix string `json:"prefix"`
	Total  int    `json:"total"`
	Tahun  []int  `json:"tahun"`
}

type HostData struct {
	Host  string       `json:"host"`
	Total int          `json:"total"`
	Data  []PrefixData `json:"data"`
}

func main() {
	// Parse command line flags
	config := Config{}
	var showVersion bool
	flag.BoolVar(&showVersion, "version", false, "Show application version")
	flag.StringVar(&config.Host, "host", "0.0.0.0", "Host to listen on")
	flag.IntVar(&config.Port, "port", 8080, "Port to listen on")
	flag.StringVar(&config.DBPath, "db", "./stats.db", "Path to SQLite database")
	flag.StringVar(&config.Timezone, "tz", "UTC", "Timezone (e.g., Asia/Jakarta)")
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

	// Load Timezone
	loc, err := time.LoadLocation(config.Timezone)
	if err != nil {
		log.Fatalf("Invalid timezone %s: %v", config.Timezone, err)
	}

	// Initialize App
	app := &App{
		DB:           db,
		HitsChannel:  make(chan StatsHit, 10000), // Buffered channel for async logging
		TimeLocation: loc,
	}

	// Start Worker
	go app.worker()

	// Set up Router
	mux := http.NewServeMux()

	// Health Check Endpoint
	mux.HandleFunc("/health", healthHandler)

	// Middleware Endpoint (Traefik ForwardAuth) - Main Path
	mux.HandleFunc("/", app.middlewareHandler)

	// API Endpoints
	mux.HandleFunc("/api", app.apiInfoHandler) // GET /api (App Info, JSON)
	mux.HandleFunc("/api/", app.statsHandler)  // GET /api/ (catch-all for stats)

	// Start Server
	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	log.Printf("Starting API Stats service on %s", addr)
	log.Printf("Database path: %s", config.DBPath)
	log.Printf("Timezone: %s", config.Timezone)

	// Wrap mux with logging middleware
	loggedMux := loggingMiddleware(mux, loc)

	if err := http.ListenAndServe(addr, loggedMux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// statusWriter wraps http.ResponseWriter to capture the status code
type statusWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (w *statusWriter) WriteHeader(code int) {
	if !w.written {
		w.statusCode = code
		w.written = true
		w.ResponseWriter.WriteHeader(code)
	}
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if !w.written {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}

// loggingMiddleware logs requests in the specified format
func loggingMiddleware(next http.Handler, loc *time.Location) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		sw := &statusWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(sw, r)

		// Get Client IP
		// Priority: X-Forwarded-For -> RemoteAddr
		clientIP := r.Header.Get("X-Forwarded-For")
		if clientIP == "" {
			clientIP, _, _ = strings.Cut(r.RemoteAddr, ":")
			// Strip IPv6 brackets if present
			clientIP = strings.Trim(clientIP, "[]")
		} else {
			// If multiple IPs, take the first one
			if i := strings.Index(clientIP, ","); i != -1 {
				clientIP = strings.TrimSpace(clientIP[:i])
			}
		}

		// Extract Host from header (as used by middlewareHandler)
		host := r.Header.Get("X-Forwarded-Host")
		if host == "" {
			host = r.Host
		}

		// Extract Original Path from Traefik (as used by middlewareHandler)
		originalPath := r.Header.Get("X-Forwarded-Uri")
		if originalPath == "" {
			originalPath = r.Header.Get("X-Replaced-Path")
		}
		if originalPath == "" {
			originalPath = r.URL.Path
		}

		// Skip logging for /health endpoint from localhost
		if r.URL.Path == "/health" && (clientIP == "127.0.0.1" || clientIP == "::1" || strings.HasPrefix(clientIP, "127.")) {
			return
		}

		// Format Time: (11 Jan 2026, 11.50.53)
		timestamp := start.In(loc).Format("02 Jan 2006, 15.04.05")

		// Colorize Status Code
		// Green: 200-299, Yellow: 300-399, Red: >= 400
		var colorReset = "\033[0m"
		var colorStatus string

		switch {
		case sw.statusCode >= 200 && sw.statusCode < 300:
			colorStatus = "\033[32m" // Green
		case sw.statusCode >= 300 && sw.statusCode < 400:
			colorStatus = "\033[33m" // Yellow
		default:
			colorStatus = "\033[31m" // Red
		}

		// Output Format: (Time) [Status] IP Method InternalPath Host OriginalPath
		fmt.Printf("(%s) [%s%d%s] %s %s %s %s %s\n",
			timestamp,
			colorStatus, sw.statusCode, colorReset,
			clientIP,
			r.Method,
			r.URL.Path,
			host,
			originalPath,
		)
	})
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

// statsHandler handles GET /api/ and GET /api/:host
func (app *App) statsHandler(w http.ResponseWriter, r *http.Request) {
	// 1. Identify Host from URL Path
	// URL: /api/ -> host="" (all hosts)
	// URL: /api/api.test.com -> host="api.test.com"

	// Strip /api/ prefix
	pathParam := strings.TrimPrefix(r.URL.Path, "/api/")
	pathParam = strings.Trim(pathParam, "/") // Remove trailing slash if any

	targetHost := ""
	if pathParam != "" {
		targetHost = pathParam
	}

	// 2. Parse Query Params
	query := r.URL.Query()
	yearParam := query.Get("year") // Empty means all years
	prefixParam := query.Get("prefix")
	allParam := query.Get("all")
	isAll := allParam == "1" || allParam == "true"

	// 3a. Build Host Totals Query (ignores year/prefix filters)
	var totalQuery strings.Builder
	totalQuery.WriteString(`
		SELECT host, SUM(total_hits)
		FROM stats
		WHERE 1=1
	`)
	var totalArgs []interface{}
	if targetHost != "" {
		totalQuery.WriteString(" AND host = ?")
		totalArgs = append(totalArgs, targetHost)
	}
	totalQuery.WriteString(" GROUP BY host")

	// Execute Totals Query
	hostTotals := make(map[string]int)
	rowsTotal, err := app.DB.Query(totalQuery.String(), totalArgs...)
	if err != nil {
		jsonError(w, "Database error", http.StatusInternalServerError)
		log.Printf("DB Error (Totals): %v", err)
		return
	}
	defer rowsTotal.Close()

	for rowsTotal.Next() {
		var h string
		var t int
		if err := rowsTotal.Scan(&h, &t); err == nil {
			hostTotals[h] = t
		}
	}

	// 3b. Build Detailed Query (respects all filters)
	var sqlQuery strings.Builder
	sqlQuery.WriteString(`
		SELECT host, path, year, SUM(total_hits) as total
		FROM stats
		WHERE 1=1
	`)

	var args []interface{}

	if targetHost != "" {
		sqlQuery.WriteString(" AND host = ?")
		args = append(args, targetHost)
	}

	if yearParam != "" {
		year, err := strconv.Atoi(yearParam)
		if err == nil {
			sqlQuery.WriteString(" AND year = ?")
			args = append(args, year)
		}
	}

	if prefixParam != "" {
		sqlQuery.WriteString(" AND path = ?")
		args = append(args, prefixParam)
	}

	sqlQuery.WriteString(" GROUP BY host, path, year")

	// 4. Execute Query
	rows, err := app.DB.Query(sqlQuery.String(), args...)
	if err != nil {
		jsonError(w, "Database error", http.StatusInternalServerError)
		log.Printf("DB Error: %v", err)
		return
	}
	defer rows.Close()

	// 5. Aggregate Data In-Memory
	// Structure: map[host] -> map[path] -> {Total, Years[]}
	type TempPathData struct {
		Total int
		Years []int
	}
	// map[host]map[path]*TempPathData
	agg := make(map[string]map[string]*TempPathData)

	for rows.Next() {
		var h, p string
		var y, t int
		if err := rows.Scan(&h, &p, &y, &t); err != nil {
			continue
		}

		if agg[h] == nil {
			agg[h] = make(map[string]*TempPathData)
		}
		if agg[h][p] == nil {
			agg[h][p] = &TempPathData{Years: []int{}}
		}

		agg[h][p].Total += t
		agg[h][p].Years = append(agg[h][p].Years, y)
	}

	// 6. Format Response
	// 6. Format Response
	response := make([]HostData, 0)

	for h, paths := range agg {
		hostTotal := hostTotals[h]

		// Fallback if needed
		if hostTotal == 0 && len(paths) > 0 {
			for _, d := range paths {
				hostTotal += d.Total
			}
		}

		var prefixList []PrefixData

		for p, data := range paths {
			// hostTotal += data.Total (removed)

			// Deduplicate years (just in case, though Group By host,path,year implies unique rows)
			// But Go appends.

			prefixList = append(prefixList, PrefixData{
				Prefix: p,
				Total:  data.Total,
				Tahun:  data.Years,
			})
		}

		// Sort PrefixList by Total DESC
		// Simple bubble sort or Slice sort
		// We'll use a simple selection sort logic or implement sort.Interface if huge.
		// For brevity, let's use a quick inline sort wrapper?
		// Actually best to import "sort".
		// Since I cannot add import easily without messing up lines, I will assume "sort" is needed.
		// Wait, "sort" is NOT in imports. "strings" is.
		// I'll stick to simple logic or just output unsorted?
		// User asked: "order by total request, top 20 saja (default)"
		// I MUST sort. I will add "sort" to imports in a separate step or try to bubble sort here if list small.
		// Bubble sort is fine for API response sizes usually.
		for i := 0; i < len(prefixList)-1; i++ {
			for j := 0; j < len(prefixList)-i-1; j++ {
				if prefixList[j].Total < prefixList[j+1].Total {
					prefixList[j], prefixList[j+1] = prefixList[j+1], prefixList[j]
				}
			}
		}

		// Limit to top 20
		if !isAll && len(prefixList) > 20 {
			prefixList = prefixList[:20]
		}

		response = append(response, HostData{
			Host:  h,
			Total: hostTotal,
			Data:  prefixList,
		})
	}

	// Sort Response by Host (optional, but good for consistency)
	// (Bubble sort hosts)
	for i := 0; i < len(response)-1; i++ {
		for j := 0; j < len(response)-i-1; j++ {
			if response[j].Host < response[j+1].Host {
				response[j], response[j+1] = response[j+1], response[j]
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// apiInfoHandler handles GET /api
func (app *App) apiInfoHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api" {
		// Just in case, though ServeMux handles precise match if registered
		http.NotFound(w, r)
		return
	}

	response := map[string]interface{}{
		"app":     "Go Stats Traefik",
		"version": AppVersion,
		"date":    time.Now().In(app.TimeLocation).Format("02 Jan 2006, 15.04.05"),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
