# Komplete

Komplete is a CLI assistant that converts natural-language requests into shell commands. You type what you want to do, Komplete calls an LLM to generate a command plan, shows you the plan, asks for confirmation, then executes the commands.

## Quick Start

### 1. Set up your API key

Create a `.env` file in the project directory:

```bash
OPENROUTER_API_KEY=your-key-here
```

Or export it in your shell:

```bash
export OPENROUTER_API_KEY=your-key-here
```

### 2. Run with Go

```bash
go run . "your request here"
```

### 3. Build the binary

```bash
go build -o komplete .
./komplete "your request here"
```

### 4. Set up the `/k` alias (optional)

Add to your `~/.zshrc` or `~/.bashrc`:

```bash
alias /k="/path/to/komplete"
```

Then use:

```bash
/k "delete node_modules and reinstall"
```

## Examples

```text
/k clean up all my docker containers

Plan:
1) docker system prune -af --volumes

Run these commands? [y/N]
```

```text
/k delete node_modules then do a fresh yarn install

Plan:
1) rm -rf node_modules
2) yarn install --no-cache

Run these commands? [y/N]
```

```text
/k stop whatever is running on port 3000

Plan:
1) lsof -ti tcp:3000 | xargs kill -9

Run these commands? [y/N]
```

## Flags

| Flag                   | Description                                          |
| ---------------------- | ---------------------------------------------------- |
| `--dry-run`            | Show the plan without executing                      |
| `--model <name>`       | Use a different BAML client (e.g., `CustomGPT5Mini`) |
| `--shell <shell>`      | Override detected shell                              |
| `--cwd <path>`         | Override working directory                           |
| `--timeout <duration>` | Model request timeout (default: `10s`)               |
| `--verbose`            | Show request/response metadata                       |

Examples:

```bash
./komplete --dry-run "delete all log files"
./komplete --model CustomGPT5Mini "show disk usage"
./komplete --timeout 30s "complex request"
```

## Subcommands

```bash
komplete version              # Print version
komplete config get <key>     # Get a config value
komplete config set <key> <value>  # Set a config value
komplete config list          # List all config values
komplete config path          # Print config file path
```

Config keys: `model`, `shell`, `timeout`, `cwd`

## Configuration

Config file location: `~/.config/komplete/config.toml`

Example:

```toml
model = "CustomGPT5Mini"
shell = "zsh"
timeout = "15s"
```

## Exit Codes

| Code | Meaning              |
| ---- | -------------------- |
| 0    | Success              |
| 1    | Generic error        |
| 2    | User aborted         |
| 3    | Model request failed |

## Tech Stack

- Go + Cobra (CLI)
- BAML (typed LLM output)
- OpenRouter (default LLM provider)

## Notes

- Binary name is `komplete`. `/k` is an alias.
- Targets macOS and Linux.

---

baml-cli generate
go build -o k .

---

list files in this folder
show hidden files
show disk usage for this folder
what is my current directory
show git status
show last 5 git commits
list running processes
show listening ports
show top 10 largest files in this folder
find all .go files in this repo
show environment variables
show my shell version
what apps do ihave on my mac and count them
