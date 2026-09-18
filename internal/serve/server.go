// Package serve sets up the HTTP server for the scp live dashboard.
//
// The dashboard is a single HTML page served at "/" that polls for history
// and receives live updates via Server-Sent Events (SSE).
//
// Endpoints:
//
//	GET  /           → the dashboard HTML page
//	GET  /api/history → search history as JSON
//	POST /api/search  → trigger a search (async, result visible in history)
//	GET  /api/stream  → SSE stream of new search records
package serve

import (
	// embed makes the Go compiler bake the web/ directory into the binary itself.
	// This means the dashboard HTML is always available, even when deployed somewhere
	// without the source files next to the binary.
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/Viswesh-G/scope/internal/analysis"
	"github.com/Viswesh-G/scope/internal/output"
	"github.com/fsnotify/fsnotify"
)

//go:embed web/index.html
var dashboardHTML []byte

// Broker is a simple publish-subscribe hub. Every SSE client gets its own channel,
// and when a new search record arrives, we send it to all connected channels.
type Broker struct {
	mu      sync.Mutex
	clients map[chan analysis.SearchRecord]bool
}

func newBroker() *Broker {
	return &Broker{clients: make(map[chan analysis.SearchRecord]bool)}
}

// addClient registers a new SSE client and returns a channel it reads from.
func (b *Broker) addClient() chan analysis.SearchRecord {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan analysis.SearchRecord, 5)
	b.clients[ch] = true
	return ch
}

// removeClient deregisters a client and closes its channel.
func (b *Broker) removeClient(ch chan analysis.SearchRecord) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.clients, ch)
	close(ch)
}

// Broadcast sends a new record to every connected SSE client.
// If a client is too slow to consume, we drop the message to avoid blocking.
func (b *Broker) Broadcast(record analysis.SearchRecord) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for ch := range b.clients {
		select {
		case ch <- record:
		default:
			// the client's buffer is full — skip this message for them
		}
	}
}

var broker = newBroker()

// StartServer starts the HTTP server on the given port and never returns unless it errors.
func StartServer(port int) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/", handleDashboard)
	mux.HandleFunc("/api/history", handleHistoryAPI)
	mux.HandleFunc("/api/search", handleSearchAPI)
	mux.HandleFunc("/api/stream", handleStreamAPI)

	addr := fmt.Sprintf(":%d", port)

	fmt.Println()
	output.TitleColor.Println("SCP Live Dashboard [SYS_ONLINE]")
	output.DimColor.Println("───────────────────────────────")
	output.SuccessColor.Printf("🚀  http://localhost%s\n", addr)
	output.DimColor.Println("Press Ctrl+C to stop")
	fmt.Println()

	// Start the file watcher in the background.
	// It will broadcast new search records to all SSE clients via the broker.
	go watchHistoryFile()

	srv := &http.Server{
		Addr:        addr,
		Handler:     addCORSHeaders(mux),
		ReadTimeout: 10 * time.Second,
		// WriteTimeout stays at 0 because SSE connections must stay open indefinitely.
		// For other routes, ReadTimeout already limits how long we wait for request headers.
		WriteTimeout: 0,
	}

	return srv.ListenAndServe()
}

// addCORSHeaders wraps a handler and adds CORS headers to every response.
// This is needed if anyone wants to query the API from a browser extension or external page.
func addCORSHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func handleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(dashboardHTML)
}

func handleHistoryAPI(w http.ResponseWriter, r *http.Request) {
	records, err := analysis.Load()
	if err != nil {
		http.Error(w, `{"error": "could not load history"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(records)
}

// SearchRequest is the JSON payload for POST /api/search.
type SearchRequest struct {
	Pattern    string `json:"pattern"`
	Path       string `json:"path"`
	IgnoreCase bool   `json:"ignoreCase"`
	Workers    int    `json:"workers"`
}

func handleSearchAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Search requests come from the local dashboard, so keep the payload small.
	// This prevents a malformed request from being held in memory indefinitely.
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}
	if req.Pattern == "" {
		http.Error(w, "Pattern is required", http.StatusBadRequest)
		return
	}

	// Find the scp binary we're currently running as, so we can run it again
	exe, err := os.Executable()
	if err != nil {
		http.Error(w, "Could not find scp executable", http.StatusInternalServerError)
		return
	}

	path := req.Path
	if path == "" {
		path = "."
	}

	args := []string{"search", "-p", req.Pattern, "--path", path, "-q"}
	if req.IgnoreCase {
		args = append(args, "-i")
	}
	if req.Workers > 0 {
		args = append(args, "-w", fmt.Sprintf("%d", req.Workers))
	}

	// Run the search in the background. The SSE stream will pick up the
	// new history record once it's written.
	go func() {
		cmd := exec.Command(exe, args...)
		_ = cmd.Run()
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_, _ = w.Write([]byte(`{"status": "searching"}`))
}

func handleStreamAPI(w http.ResponseWriter, r *http.Request) {
	// SSE requires the response to stay open and flush individual events.
	// http.Flusher is the interface that enables per-write flushing.
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported by this server", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := broker.addClient()
	defer broker.removeClient(ch)

	for {
		select {
		case record := <-ch:
			data, _ := json.Marshal(record)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		case <-r.Context().Done():
			// Client disconnected
			return
		}
	}
}

// watchHistoryFile uses fsnotify to watch the history JSON file for changes.
// When a new record is written by a search run, we broadcast it to all SSE clients.
// This is more efficient than polling with time.Sleep: instead of waking up
// every 500ms regardless, we only do work when the file actually changes.
func watchHistoryFile() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		// Fall back to polling if fsnotify setup fails (e.g. too many open files)
		pollHistoryFileFallback()
		return
	}
	defer watcher.Close()

	histDir := filepath.Join(analysis.ScopeDir)
	// Make sure the .scope directory exists before watching it
	os.MkdirAll(histDir, 0755)

	if err := watcher.Add(histDir); err != nil {
		pollHistoryFileFallback()
		return
	}

	var lastCount int
	if records, err := analysis.Load(); err == nil {
		lastCount = len(records)
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			// Only react to write events on the history file
			if !event.Has(fsnotify.Write) && !event.Has(fsnotify.Create) {
				continue
			}
			if filepath.Base(event.Name) != analysis.HistoryFile {
				continue
			}

			// Small sleep to let the writer finish flushing
			time.Sleep(50 * time.Millisecond)

			records, err := analysis.Load()
			if err != nil || len(records) <= lastCount {
				continue
			}

			// Broadcast all the new records
			for i := lastCount; i < len(records); i++ {
				broker.Broadcast(records[i])
			}
			lastCount = len(records)

		case _, ok := <-watcher.Errors:
			if !ok {
				return
			}
		}
	}
}

// pollHistoryFileFallback is the backup watcher for systems where fsnotify doesn't work.
// It polls every 500ms, which is less efficient but always works.
func pollHistoryFileFallback() {
	var lastCount int
	if records, err := analysis.Load(); err == nil {
		lastCount = len(records)
	}

	for {
		time.Sleep(500 * time.Millisecond)
		records, err := analysis.Load()
		if err != nil {
			continue
		}
		if len(records) > lastCount {
			for i := lastCount; i < len(records); i++ {
				broker.Broadcast(records[i])
			}
			lastCount = len(records)
		} else if len(records) < lastCount {
			lastCount = len(records)
		}
	}
}
