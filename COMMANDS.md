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
