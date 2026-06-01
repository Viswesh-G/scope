# Scope CLI Commands Guide

This document provides a comprehensive list of all commands available in the `scope` CLI, along with their usage and examples.

## Basic Search

### `scope search`
Searches files for a regex pattern within a specified directory.

**Flags:**
* `-p, --pattern`: The regular expression pattern to search for (Required)
* `--path`: The path to search in (Default: `.`)
* `-r, --recursive`: Search directories recursively (Default: `true`)
* `-i, --ignore-case`: Perform a case-insensitive search (Default: `false`)
* `-w, --workers`: Number of concurrent workers to use (Default: Number of CPU cores)

**Examples:**
```bash
# Basic search in the current directory
scope search --pattern "TODO"

# Case-insensitive search recursively in a specific path
scope search -p "func main" --path ./cmd -r -i

# Run a search with a specific number of workers
scope search -p "fmt.Println" -w 4

# Search only in filenames instead of contents
scope search -p "engine" --fname

# Find files with the most matches for a pattern
scope search -p "github" --hotspots
```

---

## Benchmarking

### `scope compare`
Benchmarks `scope` against `ripgrep` for a specific search pattern and path. It runs both tools multiple times to compute an average duration and determine a winner.

**Flags:**
* `-p, --pattern`: The regular expression pattern to search for (Required)
* `--path`: The path to search in (Default: `.`)
* `--runs`: Number of benchmark iterations to run for an accurate average (Default: `20`)

**Examples:**
```bash
# Compare scope and ripgrep using 20 runs
scope compare -p "github" --runs 20
```

---

## Configuration & Personalization

### `scope config set-color`
Personalizes the CLI aesthetic by allowing you to change the color of different output elements.

**Elements:** `title`, `section`, `success`, `warning`, `error`, `dim`, `file`, `match`, `time`
**Colors:** `black`, `red`, `green`, `yellow`, `blue`, `magenta`, `cyan`, `white`, `faint`, `bold`, and high-intensity variants like `hired`, `higreen`, `hicyan`, etc.

**Examples:**
```bash
# Change the title color to high-intensity cyan
scope config set-color title hicyan

# Change the match highlight color to red
scope config set-color match red
```

### `scope config list`
Displays a list of all output elements that can be styled, their current configured color, and a list of all valid color names you can choose from. The output text is rendered in the actual colors to serve as a preview!

**Examples:**
```bash
scope config list
```

### `scope config reset`
Resets all personalized aesthetics and colors back to their original defaults.

**Examples:**
```bash
scope config reset
```

---

## Ignore File Management

### `scope ignore init`
Initializes a new `.scope-ignore` file in the current directory with standard default ignore patterns (like `.git`, `node_modules`, `vendor`).

**Examples:**
```bash
scope ignore init
```

### `scope ignore add`
Adds a new pattern or path to the current directory's `.scope-ignore` file.

**Examples:**
```bash
# Ignore all .log files
scope ignore add "*.log"

# Ignore a specific directory
scope ignore add "build/"
```

### `scope ignore remove`
Removes a specific pattern from the current directory's `.scope-ignore` file.

**Examples:**
```bash
scope ignore remove "*.log"
```

### `scope ignore list`
Lists all patterns currently defined in the local `.scope-ignore` file.

**Examples:**
```bash
scope ignore list
```

---

## Codebase Analysis

### `scope dupes`
Find duplicate files by hashing their contents (SHA256).

**Examples:**
```bash
# Find duplicates in the current directory
scope dupes
```

### `scope deps`
Analyze and count Go dependencies across the repository based on import statements.

**Examples:**
```bash
scope deps
```

### `scope graph`
Build and display a visual directory tree graph of the repository, respecting ignore rules.

**Flags:**
* `--dot`: Export to Graphviz DOT format

**Examples:**
```bash
# Display directory tree
scope graph

# Export to DOT format
scope graph --dot > graph.dot
```

### `scope stats`
Show statistics for file extensions across the repository.

**Examples:**
```bash
scope stats
```

---

## Search History & Observability

### `scope history`
Displays a chronological list of recent searches executed, including the pattern, matches found, worker count, and total execution time.

**Examples:**
```bash
scope history
```

### `scope history stats`
Displays aggregated metrics over your search history, such as total searches, unique patterns, and average/fastest/slowest search times.

**Examples:**
```bash
scope history stats
```

### `scope history top`
Lists the most frequently searched patterns ranked by occurrence count.

**Examples:**
```bash
scope history top
```

### `scope history slowest`
Displays the top 10 slowest search queries you have run.

**Examples:**
```bash
scope history slowest
```

### `scope history fastest`
Displays the top 10 fastest search queries you have run.

**Examples:**
```bash
scope history fastest
```

### `scope history recent`
Lists a specific number of your most recent searches.

**Flags:**
* `--limit`: Number of recent searches to display (Default: `10`)

**Examples:**
```bash
scope history recent --limit 5
```

### `scope history pattern`
Filters and lists historical searches by a specific pattern string.

**Examples:**
```bash
scope history pattern "TODO"
```

### `scope history path`
Filters and lists historical searches by a specific target path.

**Examples:**
```bash
scope history path "./cmd"
```

### `scope history export`
Exports your entire search history to a JSON file.

**Flags:**
* `-o, --output`: Destination file path for the export (Required)

**Examples:**
```bash
scope history export -o /tmp/hist_test1.json
```

### `scope history prune`
Prunes the search history to keep only the most recent N records.

**Examples:**
```bash
scope history prune 50
```

### `scope history clear`
Deletes all recorded search history.

**Examples:**
```bash
scope history clear
```
