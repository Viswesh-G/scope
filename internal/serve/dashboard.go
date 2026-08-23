package serve

// DashboardHTML is the simple, modern, dark-themed HTML page served by our Live Dashboard.
// It uses Vanilla JS to fetch the history JSON and dynamically generate a table and stats.
const DashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Scope Live Dashboard</title>
    <style>
        :root {
            --bg-color: #0f111a;
            --surface-color: #1a1d2d;
            --text-main: #e2e8f0;
            --text-dim: #94a3b8;
            --accent: #38bdf8;
            --success: #10b981;
            --warning: #f59e0b;
            --border: #2e334d;
        }

        * { box-sizing: border-box; margin: 0; padding: 0; }

        body {
            font-family: 'Segoe UI', system-ui, -apple-system, sans-serif;
            background-color: var(--bg-color);
            color: var(--text-main);
            line-height: 1.6;
            padding: 2rem;
        }

        .container { max-width: 1200px; margin: 0 auto; }

        header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 2rem;
            border-bottom: 1px solid var(--border);
            padding-bottom: 1rem;
        }

        h1 { color: var(--accent); font-size: 2.5rem; }
        
        .live-indicator {
            display: flex;
            align-items: center;
            gap: 0.5rem;
            color: var(--success);
            font-weight: bold;
        }

        .pulse {
            width: 12px;
            height: 12px;
            background-color: var(--success);
            border-radius: 50%;
            animation: pulse-animation 2s infinite;
        }

        @keyframes pulse-animation {
            0% { box-shadow: 0 0 0 0 rgba(16, 185, 129, 0.7); }
            70% { box-shadow: 0 0 0 10px rgba(16, 185, 129, 0); }
            100% { box-shadow: 0 0 0 0 rgba(16, 185, 129, 0); }
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
            transition: transform 0.2s;
        }
        .stat-card:hover { transform: translateY(-3px); }

        .stat-value { font-size: 2rem; font-weight: bold; color: var(--accent); }
        .stat-label { color: var(--text-dim); font-size: 0.9rem; text-transform: uppercase; letter-spacing: 0.05em; }

        table {
            width: 100%;
            border-collapse: collapse;
            background-color: var(--surface-color);
            border-radius: 8px;
            overflow: hidden;
        }

        th, td { padding: 1rem; text-align: left; border-bottom: 1px solid var(--border); }
        th { background-color: rgba(255, 255, 255, 0.05); color: var(--text-dim); }
        tr:hover { background-color: rgba(255, 255, 255, 0.02); }

        .time-col { color: var(--text-dim); width: 220px; }
        .pattern-col { color: #f8fafc; font-family: monospace; font-size: 1.1rem; }
        .path-col { color: var(--text-dim); font-family: monospace; }
        .matches-col { color: var(--warning); font-weight: bold; }
        .duration-col { color: var(--success); }

        .search-form-card {
            background-color: var(--surface-color);
            padding: 1.5rem;
            border-radius: 8px;
            border: 1px solid var(--border);
            margin-bottom: 2rem;
        }

        .form-row {
            display: flex;
            gap: 1rem;
            align-items: center;
        }

        input[type="text"] {
            flex: 1;
            padding: 0.75rem;
            border-radius: 4px;
            border: 1px solid var(--border);
            background-color: rgba(255, 255, 255, 0.05);
            color: var(--text-main);
        }

        button {
            padding: 0.75rem 1.5rem;
            background-color: var(--accent);
            color: var(--bg-color);
            border: none;
            border-radius: 4px;
            font-weight: bold;
            cursor: pointer;
            transition: opacity 0.2s;
        }

        button:hover { opacity: 0.9; }
        button:disabled { opacity: 0.5; cursor: not-allowed; }

    </style>
</head>
<body>
    <div class="container">
        <header>
            <div>
                <h1>Scope Dashboard</h1>
                <p style="color: var(--text-dim)">Real-time search history & analytics</p>
            </div>
            <div class="live-indicator">
                <div class="pulse"></div>
                LIVE
            </div>
        </header>

        <div class="search-form-card">
            <h2 style="margin-top: 0; margin-bottom: 1rem; color: var(--accent); border: none; padding: 0;">New Search</h2>
            <form id="search-form">
                <div class="form-row">
                    <input type="text" id="search-pattern" placeholder="Regex pattern (e.g. func main)" required>
                    <input type="text" id="search-path" placeholder="Path (default: .)" value=".">
                    <label>
                        <input type="checkbox" id="search-ignore-case"> Ignore Case
                    </label>
                    <button type="submit" id="search-btn">Search</button>
                </div>
            </form>
        </div>

        <div class="stats-grid">
            <div class="stat-card">
                <div class="stat-value" id="stat-total-searches">-</div>
                <div class="stat-label">Total Searches</div>
            </div>
            <div class="stat-card">
                <div class="stat-value" id="stat-total-matches">-</div>
                <div class="stat-label">Total Matches Found</div>
            </div>
            <div class="stat-card">
                <div class="stat-value" id="stat-avg-duration">-</div>
                <div class="stat-label">Avg Duration</div>
            </div>
            <div class="stat-card">
                <div class="stat-value" id="stat-fastest">-</div>
                <div class="stat-label">Fastest Search</div>
            </div>
        </div>

        <h2 style="margin-bottom: 1rem; color: var(--accent);">Recent Searches</h2>
        <table>
            <thead>
                <tr>
                    <th>Timestamp</th>
                    <th>Pattern</th>
                    <th>Path</th>
                    <th>Matches</th>
                    <th>Workers</th>
                    <th>Time (ms)</th>
                </tr>
            </thead>
            <tbody id="history-body">
                <tr><td colspan="6" style="text-align: center; color: var(--text-dim);">Loading...</td></tr>
            </tbody>
        </table>
    </div>

    <script>
        // Formats the timestamp into a readable local string
        function formatTime(isoStr) {
            const d = new Date(isoStr);
            return d.toLocaleString();
        }

        // Fetches the latest history JSON and updates the UI
        async function fetchHistory() {
            try {
                const response = await fetch('/api/history');
                const data = await response.json();
                
                if (!data || data.length === 0) {
                    document.getElementById('history-body').innerHTML = 
                        '<tr><td colspan="6" style="text-align: center; color: var(--text-dim);">No history yet! Run a search in your terminal.</td></tr>';
                    return;
                }

                // 1. Update stats
                let totalMatches = 0;
                let totalDuration = 0;
                let fastest = data[0].duration_ms;

                data.forEach(record => {
                    totalMatches += record.matches;
                    totalDuration += record.duration_ms;
                    if (record.duration_ms < fastest) fastest = record.duration_ms;
                });

                document.getElementById('stat-total-searches').textContent = data.length;
                document.getElementById('stat-total-matches').textContent = totalMatches;
                document.getElementById('stat-avg-duration').textContent = (totalDuration / data.length).toFixed(1) + ' ms';
                document.getElementById('stat-fastest').textContent = fastest.toFixed(1) + ' ms';

                // 2. Update table (show newest first, max 50 for performance)
                const tbody = document.getElementById('history-body');
                tbody.innerHTML = ''; // clear old rows
                
                // Copy and reverse to show newest at top
                const recent = [...data].reverse().slice(0, 50);

                recent.forEach(r => {
                    const tr = document.createElement('tr');
                    
                    tr.innerHTML = ` + "`" + `
                        <td class="time-col">${formatTime(r.timestamp)}</td>
                        <td class="pattern-col">${r.pattern}</td>
                        <td class="path-col">${r.path}</td>
                        <td class="matches-col">${r.matches}</td>
                        <td>${r.workers}</td>
                        <td class="duration-col">${r.duration_ms.toFixed(2)}</td>
                    ` + "`" + `;
                    
                    tbody.appendChild(tr);
                });

            } catch (err) {
                console.error("Failed to fetch history:", err);
            }
        }

        // Handle the search form submission
        document.getElementById('search-form').addEventListener('submit', async (e) => {
            e.preventDefault();
            const btn = document.getElementById('search-btn');
            btn.disabled = true;
            btn.textContent = 'Searching...';

            const payload = {
                pattern: document.getElementById('search-pattern').value,
                path: document.getElementById('search-path').value,
                ignoreCase: document.getElementById('search-ignore-case').checked,
                workers: 0 // use default
            };

            try {
                await fetch('/api/search', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(payload)
                });
                
                // Fetch immediately to show the new result in the table
                setTimeout(fetchHistory, 500);
            } catch (err) {
                console.error(err);
            } finally {
                btn.disabled = false;
                btn.textContent = 'Search';
            }
        });

        // Fetch immediately, then poll every 2 seconds to keep it "Live"
        fetchHistory();
        setInterval(fetchHistory, 2000);
    </script>
</body>
</html>`
