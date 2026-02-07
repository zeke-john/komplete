# Komplete

A lightweight AI shell assistant that converts natural language to shell commands and provides realtime autocomplete as you type.

So the two features are ->

- **Natural language to shell commands**: `k count all the .ts are in this folder` generates the command(s), asks for confirmation, and runs them.
- **Inline realtime autocomplete**: Non-intrusive and instant ghost-text autocomplete suggestions as you type, like copilot for your terminal.

A example below of **komplete** in action!

## Install

```bash
brew tap zeke-john/tap
brew install komplete
# macOS will block the binary the first time so run ->
xattr -dr com.apple.quarantine "$(which komplete)"
```

Add to your `.zshrc`:

```bash
eval "$(komplete init zsh)"
```

Then open a new terminal (or `source ~/.zshrc`). This gives you:

- The `k` shorthand (`k` = `komplete`)
- Inline autocomplete (ghost-text suggestions as you type)

## Setup

Set your API keys:

```bash
komplete config set openrouter_api_key sk-or-v1-your-key-here
komplete config set groq_api_key gsk_your-key-here
```

- **OpenRouter** key is for natural language commands - get one at [openrouter.ai](https://openrouter.ai)
- **Groq** key is for inline autocomplete - get one at [console.groq.com](https://console.groq.com)

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
Command ->
  1) docker volume prune -f

Run this command? [y/N/#]
```

Type `y` to run all, `n` to cancel, or a number to run a specific command.

### Flags

```bash
k --dry-run delete all node_modules       # show plan without running
k --verbose list files                    # show request/response metadata
k --model openai/gpt-oss-20b list files   # use a different model
```

## Inline Autocomplete

Ghost-text suggestions as you type, powered by Groq's fast inference with llama-3.1-8b-instant.

- **Tab** - accept the full suggestion
- **Shift+Tab** or **Option+F** - accept one word at a time

If you only want the `k` alias without autocomplete, use `eval "$(komplete init alias)"` instead.

## Config

All settings are stored in `~/.config/komplete/config.toml`.

```bash
komplete config list  # show all configured values
komplete config path  # print the config file path
```

### Available Commands

```bash
# API keys
komplete config set openrouter_api_key sk-or-v1-xxx   # for natural language commands
komplete config set groq_api_key gsk_xxx              # for inline autocomplete
```

```bash
# Model for natural language commands (default: openai/gpt-oss-safeguard-20b)
komplete config set model anthropic/claude-haiku-4.5
komplete config set model google/gemini-3-flash
```

```bash
# Autocomplete model on Groq (default: llama-3.1-8b-instant)
komplete config set groq_model llama-3.3-70b-versatile
```

```bash
# Shell and environment
komplete config set shell /bin/zsh    # override detected shell
komplete config set cwd /path/to/dir  # override working directory
```

## Other Commands

```bash
komplete version     # print version
komplete init zsh    # output the zsh autocomplete plugin
komplete init alias  # output alias k=komplete
```
