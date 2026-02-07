# Komplete

AI shell assistant. Type what you want in plain english, get real shell commands.

Two features:

- **Natural language commands** - `k clean up docker storage` generates commands, asks for confirmation, runs them
- **Inline autocomplete** - instant ghost-text suggestions as you type, like copilot for your terminal (optional)

## Install

```bash
brew tap zeke-john/tap
brew install komplete
xattr -dr com.apple.quarantine "$(which komplete)"
```

## Setup

Set your OpenRouter API key (used for natural language commands):

```bash
komplete config set openrouter_api_key sk-or-v1-your-key-here
```

Get a key at [openrouter.ai](https://openrouter.ai) :)

## Usage

You can use either `komplete` or `k` ->

```bash
k clean up docker storage
k show disk usage for this folder
k find all .go files in this repo
k list running processes on port 3000
k show last 5 git commits
```

Komplete generates a command plan, shows it to you, and asks for confirmation before running anything.

```
Commands -->
  1) docker volume prune -f

Run these commands? [y/N/#]
```

Type `y` to run all, `n` to cancel, or a number to run a specific command.

### Flags

```bash
k --dry-run delete all node_modules              # show plan without running
k --verbose list files                           # show request/response metadata
k --model openai/gpt-oss-20b list files          # use a different model
```

## Short Alias

Add to your `.zshrc` for a shorter command:

```bash
eval "$(komplete init alias)"
```

Then use `k` instead of `komplete`:

```bash
k clean up docker storage
k show git log
```

## Inline Autocomplete (optional)

Ghost-text suggestions as you type, powered by Groq.

Set your Groq API key (get one at [console.groq.com](https://console.groq.com)):

```bash
komplete config set groq_api_key gsk_your-key-here
```

Add to your `.zshrc`:

```bash
eval "$(komplete init zsh)"
```

Open a new terminal. Start typing and suggestions appear as grey text.

- **Tab** - accept the full suggestion
- **Shift+Tab** or **Option+F** - accept one word at a time

The autocomplete runs a lightweight background daemon that caches suggestions. It uses Groq's fast inference w/ openai/gpt-oss-20b.

## Config

All settings are stored in `~/.config/komplete/config.toml`.

```bash
komplete config list                 # show all configured values
komplete config path                 # print the config file path
```

### Available keys

```bash
# API keys
komplete config set openrouter_api_key sk-or-v1-xxx   # for natural language commands
komplete config set groq_api_key gsk_xxx               # for inline autocomplete
```

```bash
# Model for natural language commands (default: openai/gpt-oss-safeguard-20b)

# Set any OpenRouter model ID:
komplete config set model anthropic/claude-haiku-4.5
komplete config set model google/gemini-3-flash
```

```bash
# Autocomplete model on Groq (default: openai/gpt-oss-20b)
komplete config set groq_model llama-3.3-70b-versatile
```

```bash
# Shell and environment
komplete config set shell /bin/zsh       # override detected shell
komplete config set cwd /path/to/dir     # override working directory
komplete config set timeout 15s          # model request timeout
```

API keys set via config are picked up automatically. Environment variables (`OPENROUTER_API_KEY`, `GROQ_API_KEY`) take priority if set.

## Other Commands

```bash
komplete version          # print version
komplete init zsh         # output the zsh autocomplete plugin
komplete init alias       # output alias k=komplete
```
