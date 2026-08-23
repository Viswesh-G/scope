// Package output handles everything printed to the terminal, and now also HTML!
// This file generates a beautiful, standalone HTML report for a search run.
// We use Go's html/template package to inject our metrics into a premium HTML layout.
package output

import (
	"fmt"
	"html/template"
	"os"

	"github.com/Viswesh-G/scope/internal/metrics"
)

// HTMLMatch is a stripped-down version of search.Match to avoid cyclic dependencies.
type HTMLMatch struct {
	File    string
	LineNum int
	Line    string
}

// WriteHTMLReport takes the metrics report and the list of matches, compiles them
// into an HTML template, and writes it to the specified file.
func WriteHTMLReport(filename string, report metrics.Report, matches []HTMLMatch) error {
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("could not create HTML file: %w", err)
	}
	defer f.Close()

	// Parse our beautiful HTML template
	tmpl, err := template.New("report").Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("could not parse HTML template: %w", err)
	}

	// Prepare data for the template
	data := struct {
		Report  metrics.Report
		Matches []HTMLMatch
	}{
		Report:  report,
		Matches: matches,
	}

	// Execute the template and write to the file
	return tmpl.Execute(f, data)
}

// The HTML template itself. 
// It uses inline CSS (no external frameworks) to keep the file standalone and offline-friendly.
// It includes tabs for easy navigation and inline SVG bars for workload visualization.
const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Scope Search Report</title>
    <style>
        :root {
            --bg-color: #0f111a;
            --surface-color: #1a1d2d;
            --text-main: #e2e8f0;
            --text-dim: #94a3b8;
            --accent: #38bdf8;
            --success: #10b981;
            --warning: #f59e0b;
            --danger: #ef4444;
            --border: #2e334d;
        }

        * {
            box-sizing: border-box;
            margin: 0;
            padding: 0;
        }

        body {
            font-family: 'Segoe UI', system-ui, -apple-system, sans-serif;
            background-color: var(--bg-color);
            color: var(--text-main);
            line-height: 1.6;
            padding: 2rem;
        }

        .container {
            max-width: 1200px;
            margin: 0 auto;
        }

        header {
            margin-bottom: 2rem;
            border-bottom: 1px solid var(--border);
            padding-bottom: 1rem;
        }

        h1 {
            color: var(--accent);
            font-size: 2.5rem;
            margin-bottom: 0.5rem;
        }

        h2 {
            margin-top: 2rem;
            margin-bottom: 1rem;
            color: var(--text-main);
            border-bottom: 1px solid var(--border);
            padding-bottom: 0.5rem;
        }

        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 1rem;
            margin-bottom: 2rem;
        }

        .stat-card {
            background-color: var(--surface-color);
            padding: 1.5rem;
            border-radius: 8px;
            border: 1px solid var(--border);
            text-align: center;
        }

        .stat-value {
            font-size: 2rem;
            font-weight: bold;
            color: var(--accent);
        }

        .stat-label {
            color: var(--text-dim);
            font-size: 0.9rem;
            text-transform: uppercase;
            letter-spacing: 0.05em;
        }

        /* Tabs */
        .tabs {
            display: flex;
            gap: 1rem;
            margin-bottom: 1rem;
            border-bottom: 1px solid var(--border);
        }

        .tab-btn {
            background: none;
            border: none;
            color: var(--text-dim);
            padding: 0.5rem 1rem;
            cursor: pointer;
            font-size: 1rem;
            font-weight: bold;
            border-bottom: 2px solid transparent;
        }

        .tab-btn.active {
            color: var(--accent);
            border-bottom: 2px solid var(--accent);
        }

        .tab-content {
            display: none;
        }

        .tab-content.active {
            display: block;
        }

        /* Matches Table */
        table {
            width: 100%;
            border-collapse: collapse;
            background-color: var(--surface-color);
            border-radius: 8px;
            overflow: hidden;
        }

        th, td {
            padding: 1rem;
            text-align: left;
            border-bottom: 1px solid var(--border);
        }

        th {
            background-color: rgba(255, 255, 255, 0.05);
            color: var(--text-dim);
        }

        .file-col { color: #f8fafc; font-family: monospace; }
        .line-col { color: var(--success); text-align: center; width: 80px; }
        .match-col { font-family: monospace; }

        /* SVG Bar Chart */
        .worker-card {
            background-color: var(--surface-color);
            padding: 1.5rem;
            border-radius: 8px;
            border: 1px solid var(--border);
            margin-bottom: 1rem;
        }

        .worker-header {
            display: flex;
            justify-content: space-between;
            margin-bottom: 1rem;
        }
        
        .bar-container {
            width: 100%;
            height: 24px;
            background-color: var(--border);
            border-radius: 12px;
            overflow: hidden;
            margin-top: 8px;
        }

        .bar-fill {
            height: 100%;
            background-color: var(--accent);
            transition: width 0.5s ease;
        }

    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1>Scope Search Report</h1>
            <p style="color: var(--text-dim)">Self-profiling search engine results</p>
        </header>

        <div class="stats-grid">
            <div class="stat-card">
                <div class="stat-value">{{.Report.MatchesFound}}</div>
                <div class="stat-label">Matches</div>
            </div>
            <div class="stat-card">
                <div class="stat-value">{{.Report.FilesScanned}}</div>
                <div class="stat-label">Files Scanned</div>
            </div>
            <div class="stat-card">
                <div class="stat-value">{{.Report.TotalDuration}}</div>
                <div class="stat-label">Total Time</div>
            </div>
            <div class="stat-card">
                <div class="stat-value">{{printf "%.2f" .Report.Parallelism}}x</div>
                <div class="stat-label">Parallelism</div>
            </div>
        </div>

        <div class="tabs">
            <button class="tab-btn active" onclick="showTab('matches')">Matches</button>
            <button class="tab-btn" onclick="showTab('workers')">Worker Stats</button>
        </div>

        <div id="matches" class="tab-content active">
            {{if .Matches}}
            <table>
                <thead>
                    <tr>
                        <th>File</th>
                        <th>Line</th>
                        <th>Content</th>
                    </tr>
                </thead>
                <tbody>
                    {{range .Matches}}
                    <tr>
                        <td class="file-col">{{.File}}</td>
                        <td class="line-col">{{.LineNum}}</td>
                        <td class="match-col">{{.Line}}</td>
                    </tr>
                    {{end}}
                </tbody>
            </table>
            {{else}}
            <p style="color: var(--text-dim); text-align: center; padding: 2rem;">No matches collected.</p>
            {{end}}
        </div>

        <div id="workers" class="tab-content">
            {{range .Report.Balance.Ranked}}
            <div class="worker-card">
                <div class="worker-header">
                    <h3>Worker #{{.ID}}</h3>
                    <span style="color: var(--text-dim)">{{.WorkDuration}}</span>
                </div>
                <div>
                    <span style="color: var(--text-dim)">Files scanned:</span> {{.FilesScanned}} 
                    <span style="color: var(--text-dim); margin-left: 1rem">Matches:</span> <span style="color: var(--warning)">{{.MatchesFound}}</span>
                </div>
                
                <!-- Inline CSS bar to visualize workload -->
                {{if $.Report.Balance.MaxFiles}}
                <div class="bar-container" title="Relative workload (files)">
                    <!-- Calculate percentage relative to the max worker load -->
                    <div class="bar-fill" style="width: calc( ({{.FilesScanned}} / {{$.Report.Balance.MaxFiles}}) * 100% );"></div>
                </div>
                {{end}}
            </div>
            {{end}}
        </div>

    </div>

    <script>
        function showTab(tabId) {
            document.querySelectorAll('.tab-content').forEach(el => el.classList.remove('active'));
            document.querySelectorAll('.tab-btn').forEach(el => el.classList.remove('active'));
            
            document.getElementById(tabId).classList.add('active');
            event.target.classList.add('active');
        }
    </script>
</body>
</html>`
