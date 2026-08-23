// Package serve sets up a simple local web server that provides a live dashboard.
// It serves an HTML page at the root (/) and exposes the search history as JSON (/api/history).
package serve

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/Viswesh-G/scope/internal/analysis"
	"github.com/Viswesh-G/scope/internal/output"
)

// StartServer begins listening on the requested port.
func StartServer(port int) error {
	mux := http.NewServeMux()

	// 1. The dashboard UI route
	mux.HandleFunc("/", handleDashboard)

	// 2. The JSON API route that the dashboard polls
	mux.HandleFunc("/api/history", handleHistoryAPI)

	// 3. The search API route for the Web UI
	mux.HandleFunc("/api/search", handleSearchAPI)

	addr := fmt.Sprintf(":%d", port)
	
	fmt.Println()
	output.TitleColor.Println("Scope Live Dashboard")
	output.DimColor.Println("────────────────────")
	output.SuccessColor.Printf("🚀 Server running on http://localhost%s\n", addr)
	output.DimColor.Println("Press Ctrl+C to stop")
	fmt.Println()

	// Create a custom server with timeouts for good practice
	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	return srv.ListenAndServe()
}

// handleDashboard serves the HTML string that makes up our beautiful frontend.
func handleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// DashboardHTML comes from dashboard.go
	w.Write([]byte(DashboardHTML))
}

// handleHistoryAPI reads the latest history from .scope/history.json and returns it as JSON.
func handleHistoryAPI(w http.ResponseWriter, r *http.Request) {
	records, err := analysis.Load()
	if err != nil {
		http.Error(w, `{"error": "could not load history"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	
	// We use an encoder to efficiently write the JSON directly to the response
	json.NewEncoder(w).Encode(records)
}

// SearchRequest represents the incoming JSON payload from the web dashboard
type SearchRequest struct {
	Pattern    string `json:"pattern"`
	Path       string `json:"path"`
	IgnoreCase bool   `json:"ignoreCase"`
	Workers    int    `json:"workers"`
}

// handleSearchAPI runs the scope search CLI in the background
func handleSearchAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	if req.Pattern == "" {
		http.Error(w, "Pattern is required", http.StatusBadRequest)
		return
	}

	// Find the current executable path so we can invoke ourselves
	exe, err := os.Executable()
	if err != nil {
		http.Error(w, "Could not find scope executable", http.StatusInternalServerError)
		return
	}

	// Prepare the arguments
	args := []string{"search", "-p", req.Pattern}
	
	if req.Path != "" {
		args = append(args, "--path", req.Path)
	} else {
		args = append(args, "--path", ".")
	}
	
	if req.IgnoreCase {
		args = append(args, "-i")
	}
	
	if req.Workers > 0 {
		args = append(args, "-w", fmt.Sprintf("%d", req.Workers))
	}

	// We run it quietly (-q) since nobody is watching the terminal output
	args = append(args, "-q")

	// Execute it in the background! The user's dashboard will pick it up 
	// from history.json automatically when it finishes.
	go func() {
		cmd := exec.Command(exe, args...)
		// We don't care about the output for now, just run it
		_ = cmd.Run()
	}()

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(`{"status": "searching"}`))
}
