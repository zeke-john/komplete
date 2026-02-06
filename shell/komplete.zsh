autoload -Uz add-zsh-hook 2>/dev/null

if [[ -z "$ZSH_VERSION" || "$TERM" == "dumb" ]]; then
    # not interactive zsh; skip
else

typeset -g _komplete_suggestion=""
typeset -g _komplete_bin="${KOMPLETE_BIN:-komplete}"
typeset -g _komplete_min_chars="${KOMPLETE_MIN_CHARS:-2}"
typeset -gF _komplete_last_query=0
typeset -gF _komplete_throttle="${KOMPLETE_THROTTLE:-0.3}"
typeset -gi _komplete_async_fd=0
typeset -g _komplete_async_buffer=""
typeset -g _komplete_prev_buffer=""
typeset -g _komplete_debug_file="/tmp/komplete_debug.log"

_komplete_dbg() {
    print -r -- "$(date +%H:%M:%S.%N) $1" >> "$_komplete_debug_file"
}

_komplete_clear() {
    _komplete_dbg "clear: POSTDISPLAY was '${POSTDISPLAY}', suggestion was '${_komplete_suggestion}'"
    POSTDISPLAY=""
    _komplete_suggestion=""
    region_highlight=("${region_highlight[@]:#*komplete*}")
}

_komplete_display() {
    local suggestion="$1"

    [[ -z "$BUFFER" || -z "$suggestion" ]] && return 1
    [[ "$suggestion" != "${BUFFER}"* ]] && return 1
    [[ "$suggestion" == "$BUFFER" ]] && return 1

    local remainder="${suggestion#${BUFFER}}"
    POSTDISPLAY="$remainder"

    local start=${#BUFFER}
    local end=$(( start + ${#remainder} ))
    region_highlight=("${region_highlight[@]:#*komplete*}")
    region_highlight+=("${start} ${end} fg=8 # komplete")
    _komplete_dbg "display: set POSTDISPLAY='${remainder}' for suggestion='${suggestion}' buffer='${BUFFER}'"
    return 0
}

_komplete_kill_async() {
    if (( _komplete_async_fd > 0 )); then
        zle -F $_komplete_async_fd 2>/dev/null
        exec {_komplete_async_fd}<&- 2>/dev/null
        _komplete_async_fd=0
    fi
}

_komplete_async_callback() {
    local fd=$1
    local suggestion=""
    read -r suggestion <&$fd 2>/dev/null

    zle -F $fd 2>/dev/null
    exec {fd}<&- 2>/dev/null
    _komplete_async_fd=0

    _komplete_dbg "async_callback: suggestion='${suggestion}' buffer='${BUFFER}' async_buffer='${_komplete_async_buffer}'"

    if [[ -n "$suggestion" && "$BUFFER" == "$_komplete_async_buffer" ]]; then
        _komplete_suggestion="$suggestion"
        if _komplete_display "$suggestion"; then
            zle -R
        fi
    fi
}

_komplete_fetch() {
    _komplete_kill_async

    if (( ${#BUFFER} < _komplete_min_chars )); then
        return
    fi

    local now=${EPOCHREALTIME:-0}
    if (( now - _komplete_last_query < _komplete_throttle )); then
        _komplete_dbg "fetch: THROTTLED buffer='${BUFFER}'"
        return
    fi
    _komplete_last_query=$now

    _komplete_async_buffer="$BUFFER"
    _komplete_dbg "fetch: launching async for buffer='${BUFFER}'"

    exec {_komplete_async_fd} < <("$_komplete_bin" suggest --cwd "$PWD" -- "$BUFFER" 2>/dev/null)
    zle -F $_komplete_async_fd _komplete_async_callback
}

_komplete_line_pre_redraw() {
    [[ "$BUFFER" == "$_komplete_prev_buffer" ]] && return
    _komplete_dbg "pre_redraw: buffer changed '${_komplete_prev_buffer}' -> '${BUFFER}' POSTDISPLAY='${POSTDISPLAY}'"
    _komplete_prev_buffer="$BUFFER"
    _komplete_clear
    _komplete_fetch
}

_komplete_accept() {
    if [[ -n "$POSTDISPLAY" && -n "$_komplete_suggestion" ]]; then
        local accepted="$_komplete_suggestion"
        _komplete_clear
        BUFFER="$accepted"
        CURSOR=${#BUFFER}
        _komplete_prev_buffer="$BUFFER"
        zle redisplay
    else
        _komplete_clear
        zle expand-or-complete
    fi
}

_komplete_accept_word() {
    if [[ -n "$POSTDISPLAY" && -n "$_komplete_suggestion" ]]; then
        local remaining="${_komplete_suggestion#${BUFFER}}"
        local next_word
        if [[ "$remaining" == *" "* ]]; then
            next_word="${remaining%% *} "
        else
            next_word="$remaining"
        fi
        BUFFER="${BUFFER}${next_word}"
        CURSOR=${#BUFFER}
        _komplete_prev_buffer="$BUFFER"

        if ! _komplete_display "$_komplete_suggestion"; then
            _komplete_clear
        fi
        zle -R
    else
        zle forward-word
    fi
}

_komplete_kill_whole_line() {
    zle .kill-whole-line
    _komplete_clear
    _komplete_kill_async
    _komplete_prev_buffer="$BUFFER"
}

if (( ${+widgets[accept-line]} )); then
    zle -A accept-line _komplete_orig_accept_line
fi

_komplete_accept_line() {
    _komplete_clear
    _komplete_kill_async
    _komplete_prev_buffer=""
    if (( ${+widgets[_komplete_orig_accept_line]} )); then
        zle _komplete_orig_accept_line
    else
        zle .accept-line
    fi
}

if (( ${+widgets[zle-line-pre-redraw]} )); then
    _komplete_dbg "init: existing zle-line-pre-redraw found, chaining"
    zle -A zle-line-pre-redraw _komplete_orig_line_pre_redraw
fi
zle -N zle-line-pre-redraw _komplete_line_pre_redraw

zle -N kill-whole-line _komplete_kill_whole_line
zle -N accept-line _komplete_accept_line
zle -N _komplete_accept
zle -N _komplete_accept_word

bindkey '^I' _komplete_accept
bindkey '\e[Z' _komplete_accept_word
bindkey '^[f' _komplete_accept_word

_komplete_precmd() {
    _komplete_suggestion=""
    _komplete_prev_buffer=""
    _komplete_kill_async
}
add-zsh-hook precmd _komplete_precmd 2>/dev/null

_komplete_dbg "init: komplete.zsh loaded successfully"

fi
