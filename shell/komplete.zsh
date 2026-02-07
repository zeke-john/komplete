autoload -Uz add-zsh-hook 2>/dev/null
zmodload zsh/net/tcp 2>/dev/null
zmodload zsh/system 2>/dev/null

if [[ -z "$ZSH_VERSION" || "$TERM" == "dumb" ]]; then
    # not interactive zsh; skip
else

typeset -g _komplete_suggestion=""
typeset -g _komplete_bin="${KOMPLETE_BIN:-komplete}"
typeset -g _komplete_min_chars="${KOMPLETE_MIN_CHARS:-2}"
typeset -g _komplete_async_fd=""
typeset -g _komplete_child_pid=""
typeset -g _komplete_async_buffer=""
typeset -g _komplete_prev_buffer=""
typeset -gi _komplete_daemon_pid=0
typeset -g _komplete_port_file="/tmp/komplete-$(id -u).port"
typeset -gi _komplete_daemon_port=0
typeset -g _komplete_result_file="/tmp/komplete-result-$$.txt"

_komplete_remove_highlights() {
    region_highlight=("${(@)region_highlight:#*fg=8}")
}

_komplete_clear() {
    POSTDISPLAY=""
    _komplete_suggestion=""
    _komplete_remove_highlights
}

_komplete_display() {
    local suggestion="$1"

    [[ -z "$BUFFER" || -z "$suggestion" ]] && return 1
    [[ "$suggestion" != "${BUFFER}"* ]] && return 1
    [[ "$suggestion" == "$BUFFER" ]] && return 1

    local remainder="${suggestion#${BUFFER}}"
    POSTDISPLAY="$remainder"

    _komplete_remove_highlights
    local start=${#BUFFER}
    local end=$(( start + ${#remainder} ))
    region_highlight+=("${start} ${end} fg=8")
    return 0
}

_komplete_kill_async() {
    if [[ -n "$_komplete_async_fd" ]] && { true <&$_komplete_async_fd } 2>/dev/null; then
        local fd=$_komplete_async_fd
        builtin exec {_komplete_async_fd}<&-
        zle -F $fd 2>/dev/null
    fi

    if [[ -n "$_komplete_child_pid" ]]; then
        kill -TERM $_komplete_child_pid 2>/dev/null
    fi

    _komplete_async_fd=""
    _komplete_child_pid=""
    _komplete_async_buffer=""
    command rm -f "$_komplete_result_file" 2>/dev/null
}

_komplete_ensure_daemon() {
    if (( _komplete_daemon_port > 0 )); then
        if (( _komplete_daemon_pid > 0 )); then
            kill -0 "$_komplete_daemon_pid" 2>/dev/null && return 0
        elif [[ -f "$_komplete_port_file" ]]; then
            return 0
        fi
        _komplete_daemon_port=0
    fi

    if [[ -f "$_komplete_port_file" ]]; then
        local existing_port
        existing_port=$(<"$_komplete_port_file" 2>/dev/null)
        if [[ -n "$existing_port" ]] && ztcp 127.0.0.1 $existing_port 2>/dev/null; then
            ztcp -c $REPLY 2>/dev/null
            _komplete_daemon_port=$existing_port
            return 0
        fi
        command rm -f "$_komplete_port_file" 2>/dev/null
    fi

    "$_komplete_bin" daemon --port-file "$_komplete_port_file" &>/dev/null &!
    _komplete_daemon_pid=$!

    local i=0
    while (( i++ < 20 )) && [[ ! -f "$_komplete_port_file" ]]; do
        sleep 0.05
    done

    if [[ -f "$_komplete_port_file" ]]; then
        _komplete_daemon_port=$(<"$_komplete_port_file" 2>/dev/null)
        return 0
    fi

    return 1
}

_komplete_async_callback() {
    emulate -L zsh

    local fd=$1

    builtin exec {fd}<&-
    zle -F "$fd"
    _komplete_async_fd=""
    _komplete_async_buffer=""
}

_komplete_apply_result() {
    [[ ! -s "$_komplete_result_file" ]] && return 1

    local suggestion
    suggestion=$(<"$_komplete_result_file")
    command rm -f "$_komplete_result_file" 2>/dev/null
    _komplete_async_buffer=""

    [[ -z "$suggestion" || -z "$BUFFER" ]] && return 1

    if [[ "$suggestion" != "${BUFFER}"* || "$suggestion" == "$BUFFER" ]]; then
        _komplete_fetch
        return 1
    fi

    _komplete_suggestion="$suggestion"
    _komplete_display "$suggestion" && zle -R
    return 0
}

_komplete_query_daemon() {
    if [[ -n "$_komplete_async_fd" ]] && { true <&$_komplete_async_fd } 2>/dev/null; then
        if [[ -n "$_komplete_async_buffer" && "$BUFFER" == "$_komplete_async_buffer"* ]]; then
            return
        fi
    fi

    _komplete_kill_async
    _komplete_ensure_daemon || return

    _komplete_async_buffer="$BUFFER"
    local port=$_komplete_daemon_port
    local payload="{\"buffer\":\"$BUFFER\",\"cwd\":\"$PWD\",\"shell\":\"$SHELL\"}"
    local rfile="$_komplete_result_file"
    local ppid=$$

    builtin exec {_komplete_async_fd}< <(
        local result
        result=$(echo "$payload" | command nc -w 3 127.0.0.1 $port 2>/dev/null)
        if [[ -n "$result" ]]; then
            echo "$result" > "$rfile"
            kill -WINCH $ppid 2>/dev/null
        fi
        echo "$result"
    )

    command true

    zle -F "$_komplete_async_fd" _komplete_async_callback
}

_komplete_fetch() {
    if (( ${#BUFFER} < _komplete_min_chars )); then
        return
    fi

    [[ "$BUFFER" == cd\ * || "$BUFFER" == "cd" ]] && return

    _komplete_query_daemon
}

_komplete_line_pre_redraw() {
    _komplete_apply_result

    [[ "$BUFFER" == "$_komplete_prev_buffer" ]] && return
    _komplete_prev_buffer="$BUFFER"

    if [[ -n "$_komplete_suggestion" && -n "$BUFFER" && "$_komplete_suggestion" == "${BUFFER}"* && "$_komplete_suggestion" != "$BUFFER" ]]; then
        _komplete_display "$_komplete_suggestion"
        zle -R
        return
    fi

    _komplete_clear
    _komplete_fetch
}

_komplete_accept() {
    if [[ -n "$POSTDISPLAY" && -n "$_komplete_suggestion" ]]; then
        local accepted="$_komplete_suggestion"
        local orig_len=${#BUFFER}
        _komplete_kill_async
        _komplete_clear
        BUFFER="$accepted"
        CURSOR=${#BUFFER}
        _komplete_prev_buffer="$BUFFER"
        region_highlight+=("${orig_len} ${#BUFFER} fg=default")
    else
        _komplete_clear
        zle expand-or-complete
    fi
}

_komplete_accept_word() {
    if [[ -n "$POSTDISPLAY" && -n "$_komplete_suggestion" ]]; then
        local remaining="${_komplete_suggestion#${BUFFER}}"
        local orig_len=${#BUFFER}
        local next_word
        if [[ "$remaining" == *" "* ]]; then
            next_word="${remaining%% *} "
        else
            next_word="$remaining"
        fi

        _komplete_remove_highlights

        BUFFER="${BUFFER}${next_word}"
        CURSOR=${#BUFFER}
        _komplete_prev_buffer="$BUFFER"

        if _komplete_display "$_komplete_suggestion"; then
            region_highlight+=("${orig_len} ${#BUFFER} fg=default")
        else
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

_komplete_cleanup() {
    _komplete_kill_async
}
add-zsh-hook zshexit _komplete_cleanup 2>/dev/null

_komplete_ensure_daemon

alias k=komplete

fi
