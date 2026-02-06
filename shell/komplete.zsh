autoload -Uz add-zsh-hook 2>/dev/null

if [[ -z "$ZSH_VERSION" || "$TERM" == "dumb" ]]; then
    # not interactive zsh; skip
else

typeset -g _komplete_suggestion=""
typeset -g _komplete_bin="${KOMPLETE_BIN:-komplete}"
typeset -g _komplete_min_chars="${KOMPLETE_MIN_CHARS:-2}"
typeset -gF _komplete_last_query=0
typeset -gF _komplete_throttle="${KOMPLETE_THROTTLE:-0.3}"

_komplete_clear() {
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
    return 0
}

_komplete_request() {
    if (( ${#BUFFER} < _komplete_min_chars )); then
        _komplete_clear
        return
    fi

    if [[ -n "$_komplete_suggestion" ]] && _komplete_display "$_komplete_suggestion"; then
        return
    fi

    _komplete_clear

    local now=${EPOCHREALTIME:-0}
    if (( now - _komplete_last_query < _komplete_throttle )); then
        return
    fi
    _komplete_last_query=$now

    local suggestion
    suggestion=$("$_komplete_bin" suggest --cwd "$PWD" -- "$BUFFER" 2>/dev/null)

    if [[ -n "$suggestion" ]]; then
        _komplete_suggestion="$suggestion"
        _komplete_display "$suggestion"
    else
        _komplete_suggestion=""
        POSTDISPLAY=""
    fi
}

_komplete_accept() {
    if [[ -n "$POSTDISPLAY" && -n "$_komplete_suggestion" ]]; then
        local accepted="$_komplete_suggestion"
        POSTDISPLAY=""
        _komplete_suggestion=""
        region_highlight=()
        BUFFER="$accepted"
        CURSOR=${#BUFFER}
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

        if ! _komplete_display "$_komplete_suggestion"; then
            _komplete_clear
        fi
        zle -R
    else
        zle forward-word
    fi
}

_komplete_self_insert() {
    zle .self-insert
    _komplete_request
}

_komplete_backward_delete_char() {
    zle .backward-delete-char
    _komplete_request
}

_komplete_kill_whole_line() {
    zle .kill-whole-line
    _komplete_clear
}

if (( ${+widgets[accept-line]} )); then
    zle -A accept-line _komplete_orig_accept_line
fi

_komplete_accept_line() {
    _komplete_clear
    if (( ${+widgets[_komplete_orig_accept_line]} )); then
        zle _komplete_orig_accept_line
    else
        zle .accept-line
    fi
}

zle -N self-insert _komplete_self_insert
zle -N backward-delete-char _komplete_backward_delete_char
zle -N kill-whole-line _komplete_kill_whole_line
zle -N accept-line _komplete_accept_line
zle -N _komplete_accept
zle -N _komplete_accept_word

bindkey '^I' _komplete_accept
bindkey '\e[Z' _komplete_accept_word
bindkey '^[f' _komplete_accept_word

_komplete_precmd() {
    _komplete_suggestion=""
}
add-zsh-hook precmd _komplete_precmd 2>/dev/null

fi
