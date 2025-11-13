# dtop - Docker Container Monitor

A terminal UI tool for monitoring Docker containers, inspired by htop. Displays containers grouped by project with an interactive tree interface.

![dtop screenshot](.github/assets/screenshot.png)

## Features

- **Project Grouping**: Automatically groups containers by their name prefix (Docker Compose convention)
- **Tree Navigation**: Expandable/collapsible project view
- **Real-time Monitoring**: Auto-refreshes container status every 2 seconds
- **Interactive Actions**: Context-aware menu system for managing containers
- **Keyboard-driven**: Full keyboard navigation with vim-style keybindings
- **Viewport Scrolling**: Efficient handling of hundreds of containers with automatic scrolling
- **Sticky Footer**: Help text always visible at bottom of screen
- **Scroll Indicator**: Shows current position when content exceeds screen height
- **List Mode**: Non-interactive output for scripts and CI/CD pipelines (`--list` / `-l`)

## Installation

### Via go install (recommended)

```bash
go install github.com/ekinertac/dtop@latest
```

This installs `dtop` to your `$GOPATH/bin` directory (usually `~/go/bin`). 

**If the command is not found after installation**, add Go's bin directory to your PATH:

```bash
# For bash/zsh, add to ~/.bashrc or ~/.zshrc
export PATH="$PATH:$(go env GOPATH)/bin"

# Or if go command is not available, typically it's:
export PATH="$PATH:$HOME/go/bin"
```

After adding the export line, restart your terminal or run `source ~/.bashrc` (or `~/.zshrc`).

### From source

```bash
git clone https://github.com/ekinertac/dtop.git
cd dtop
go build -o dtop
```

## Usage

### Interactive mode (default)

```bash
dtop
```

Launches the full interactive TUI with real-time monitoring and keyboard navigation.

### List mode (non-interactive)

```bash
dtop --list
# or short form
dtop -l
```

Lists current containers and exits immediately. Useful for:
- Shell scripts and automation
- CI/CD pipelines
- Quick status checks in non-interactive environments
- Logging/monitoring systems

Example output:
```
dtop - Docker Container Monitor

NAME                                               STATUS                         CPU %    MEM %    UPTIME
------------------------------------------------------------------------------------------------------------------------
▼ myproject (3)
    myproject-web-1                                Up 2 hours                     0.0%     0.0%     02h 15m
    myproject-db-1                                 Up 2 hours (healthy)           0.0%     0.0%     02h 15m
    myproject-worker-1                             Up 2 hours                     0.0%     0.0%     02h 15m
```

## Keyboard Shortcuts

### Navigation
- `↑` / `k` - Move up
- `↓` / `j` - Move down
- `PgUp` - Page up
- `PgDn` - Page down
- `Home` - Jump to top
- `End` - Jump to bottom
- `←` / `h` - Collapse project
- `→` / `l` - Expand project
- `Enter` - Open action menu
- `q` / `Ctrl+C` - Quit

### Menu Navigation
- `↑` / `↓` - Select menu item
- `Enter` - Execute action
- `Esc` - Close menu

## Actions

### Project-level Actions
- Restart All - Restart all containers (`docker compose restart`)
- Stop All - Stop all running containers (`docker compose stop`)
- Down - Stop and remove all containers (`docker compose down`, **keeps volumes**)
- Start All - Start all stopped containers (`docker compose start`)

### Container-level Actions
- Restart - Restart the container (`docker restart`)
- Stop - Stop the container (`docker stop`)
- Remove - Remove the container (`docker rm`, **keeps volumes**)
- Logs - View container logs (last 1000 lines, scrollable)

**Note:** All operations preserve volumes by default. To remove volumes, use `docker volume rm` or `docker compose down --volumes` from the terminal.

## How It Works

**Project Grouping**: dtop automatically groups containers based on their naming convention:
- `myproject_web_1` → project: `myproject`
- `myproject-db-1` → project: `myproject`
- Standalone containers are shown under their own project name

**Docker Integration**: Uses Docker API directly for all operations, no docker-compose dependency required.

## Requirements

- Go 1.21+
- Docker running on local machine
- Docker socket accessible (typically `/var/run/docker.sock`)

## Development

```bash
# Install dependencies
go mod download

# Build
go build

# Run
./dtop
```

## Publishing to GitHub

To make your tool available via `go install`, push to GitHub:

```bash
# Initialize git if not already done
git init
git add .
git commit -m "Initial commit"

# Create repo on GitHub, then:
git remote add origin https://github.com/ekinertac/dtop.git
git branch -M main
git push -u origin main

# Create a release tag (optional, for versioning)
git tag v0.1.0
git push origin v0.1.0
```

After pushing, others can install with:
```bash
go install github.com/ekinertac/dtop@latest
```

## Use Cases

**Interactive monitoring:**
```bash
dtop
```
Monitor containers in real-time with keyboard control.

**Scripts and automation:**
```bash
# Check container status
dtop --list
# or
dtop -l

# Save to file
dtop -l > containers.txt

# Pipe to other tools
dtop -l | grep "myproject"

# CI/CD health check
if dtop -l | grep -q "Exit"; then
    echo "Some containers are down!"
    exit 1
fi
```

## Roadmap

- [x] List mode for non-interactive use (`--list` / `-l`)
- [x] Real-time CPU/Memory statistics
- [x] Log viewer with scrolling
- [ ] Container inspect view
- [ ] Exec into container
- [ ] Filter/search functionality
- [ ] Color themes
- [ ] Configuration file support

