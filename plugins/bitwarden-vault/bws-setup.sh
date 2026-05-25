#!/usr/bin/env bash
# bws-setup.sh — Interactive Bitwarden Secrets Manager CLI account setup
# Adds a BWS profile + access token to the machine (keychain or shell profile)

set -euo pipefail

# ── Colors ─────────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
RESET='\033[0m'

info()  { printf "${BLUE}ℹ${RESET}  %s\n" "$1"; }
ok()    { printf "${GREEN}✓${RESET}  %s\n" "$1"; }
warn()  { printf "${YELLOW}⚠${RESET}  %s\n" "$1"; }
err()   { printf "${RED}✗${RESET}  %s\n" "$1" >&2; }
ask()   { printf "${CYAN}?${RESET}  %s" "$1"; }

# ── Detect shell profile ───────────────────────────────────────────
detect_shell_profile() {
    if [[ -n "${ZSH_VERSION:-}" ]] || [[ "$SHELL" == */zsh ]]; then
        echo "$HOME/.zshrc"
    elif [[ -n "${BASH_VERSION:-}" ]] || [[ "$SHELL" == */bash ]]; then
        if [[ -f "$HOME/.bash_profile" ]]; then
            echo "$HOME/.bash_profile"
        else
            echo "$HOME/.bashrc"
        fi
    else
        echo "$HOME/.profile"
    fi
}

# ── Keychain helpers ───────────────────────────────────────────────
has_security() {
    command -v security &>/dev/null && [[ "$OSTYPE" == darwin* ]]
}

kc_store() {
    local svc="$1" val="$2"
    security add-generic-password -a "$USER" -s "$svc" -w "$val" -U 2>/dev/null
}

kc_get() {
    local svc="$1"
    security find-generic-password -a "$USER" -s "$svc" -w 2>/dev/null | tr -d '\n'
}

kc_delete() {
    local svc="$1"
    security delete-generic-password -a "$USER" -s "$svc" &>/dev/null || true
}

# ── bws helpers ────────────────────────────────────────────────────
BWS_BIN="${BWS_BIN:-$HOME/bin/bws}"
BWS_CONFIG_DIR="$HOME/.config/bws"
BWS_CONFIG="$BWS_CONFIG_DIR/config"

ensure_bws() {
    if [[ ! -x "$BWS_BIN" ]]; then
        err "bws binary not found at $BWS_BIN"
        ask "Path to bws binary [$HOME/bin/bws]: "
        read -r bws_path
        bws_path="${bws_path:-$HOME/bin/bws}"
        if [[ ! -x "$bws_path" ]]; then
            err "bws not found at $bws_path"
            echo "    Install: https://bitwarden.com/help/secrets-manager-cli/"
            exit 1
        fi
        BWS_BIN="$bws_path"
    fi
    ok "bws found: $BWS_BIN ($("$BWS_BIN" --version 2>/dev/null | head -1))"
}

show_current_config() {
    echo
    info "Current bws config ($BWS_CONFIG):"
    if [[ -f "$BWS_CONFIG" ]]; then
        cat "$BWS_CONFIG" | sed 's/^/    /'
    else
        echo "    (none)"
    fi
    echo
    info "Existing keychain entries:"
    if has_security; then
        security dump-keychain 2>/dev/null | grep -o 'bws\.[a-zA-Z0-9_-]*\.token' | sort -u | sed 's/^/    /' || echo "    (none)"
    else
        echo "    (keychain not available)"
    fi
    echo
}

# ── Main setup ─────────────────────────────────────────────────────
main() {
    echo
    printf "  ${BOLD}┌─ Bitwarden Secrets Manager CLI Setup ─────────────┐${RESET}\n"
    echo

    ensure_bws
    show_current_config

    # --- Profile name ---
    ask "Profile name [default]: "
    read -r profile_name
    profile_name="${profile_name:-default}"

    # Validate profile name (TOML-safe)
    if [[ "$profile_name" =~ [^a-zA-Z0-9_-] ]]; then
        err "Profile name must be alphanumeric, hyphen, or underscore"
        exit 1
    fi

    # --- Server URL ---
    ask "Server base URL [https://api.bitwarden.com]: "
    read -r server_url
    server_url="${server_url:-https://api.bitwarden.com}"

    # --- Access token ---
    ask "Access token (from Bitwarden Secrets Manager web app): "
    read -rs access_token
    echo
    if [[ -z "$access_token" ]]; then
        err "Access token is required"
        exit 1
    fi

    # Mask for display
    token_mask="${access_token:0:8}****${access_token: -4}"

    # --- Storage method ---
    echo
    info "Where should the access token be stored?"
    echo "    1) macOS Keychain (secure, recommended)"
    echo "    2) Shell profile as env var (~/.zshrc or ~/.bash_profile)"
    echo "    3) Both"
    ask "Choice [1]: "
    read -r storage_choice
    storage_choice="${storage_choice:-1}"

    use_keychain=false
    use_profile=false

    case "$storage_choice" in
        1) use_keychain=true ;;
        2) use_profile=true ;;
        3) use_keychain=true; use_profile=true ;;
        *) warn "Invalid choice, defaulting to keychain"; use_keychain=true ;;
    esac

    # --- Apply ---
    echo
    info "Applying configuration..."

    # Ensure config dir exists
    mkdir -p "$BWS_CONFIG_DIR"

    # Update TOML config
    local tmp_config
    tmp_config=$(mktemp)
    if [[ -f "$BWS_CONFIG" ]]; then
        # Remove existing profile section if present (portable awk)
        awk -v prof="$profile_name" '
            /^\[profiles\./ {
                in_profile = 0
                name = $0
                gsub(/^\[profiles\.|\]$/, "", name)
                if (name == prof) in_profile = 1
                else print
                next
            }
            in_profile { next }
            { print }
        ' "$BWS_CONFIG" > "$tmp_config" || cp "$BWS_CONFIG" "$tmp_config"
    fi

    # Append new profile
    {
        echo ""
        echo "[profiles.$profile_name]"
        echo "server_base = \"$server_url\""
    } >> "$tmp_config"

    mv "$tmp_config" "$BWS_CONFIG"
    ok "Updated $BWS_CONFIG with profile '$profile_name'"

    # Store access token
    kc_service="bws.$profile_name.token"

    if $use_keychain; then
        if has_security; then
            kc_delete "$kc_service"
            kc_store "$kc_service" "$access_token"
            ok "Stored access token in macOS Keychain ($kc_service)"
        else
            warn "macOS Keychain not available, skipping keychain storage"
            use_keychain=false
        fi
    fi

    if $use_profile; then
        profile_file=$(detect_shell_profile)
        # Remove old entry for this profile if exists
        if [[ -f "$profile_file" ]]; then
            grep -v "BWS_ACCESS_TOKEN_$profile_name=" "$profile_file" > "${profile_file}.tmp" 2>/dev/null || true
            mv "${profile_file}.tmp" "$profile_file"
        fi
        {
            echo ""
            echo "# BWS profile: $profile_name"
            echo "export BWS_ACCESS_TOKEN_$profile_name='$access_token'"
        } >> "$profile_file"
        ok "Stored access token in $profile_file"
    fi

    # --- Test ---
    echo
    ask "Test the connection now? [Y/n]: "
    read -r test_it
    test_it="${test_it:-Y}"

    if [[ "$test_it" =~ ^[Yy] ]]; then
        echo
        export BWS_ACCESS_TOKEN="$access_token"
        if "$BWS_BIN" secret list -t "$access_token" &>/dev/null; then
            ok "Connection successful! Secrets are accessible."
        else
            # Try with profile
            export BWS_PROFILE="$profile_name"
            if "$BWS_BIN" secret list &>/dev/null; then
                ok "Connection successful via profile '$profile_name'!"
            else
                err "Connection failed. Check your access token and server URL."
                echo "    Common issues:"
                echo "      • Access token may be expired or revoked"
                echo "      • Wrong server URL (EU users need https://api.bitwarden.eu)"
                echo "      • Service account lacks permissions on any project"
            fi
        fi
    fi

    # --- Summary ---
    echo
    printf "  ${BOLD}┌─ Setup Complete ──────────────────────────────────┐${RESET}\n"
    echo
    printf "  Profile:     ${BOLD}%s${RESET}\n" "$profile_name"
    printf "  Server:      %s\n" "$server_url"
    printf "  Token:       %s\n" "$token_mask"
    printf "  Keychain:    %s\n" "$($use_keychain && echo "✓ stored" || echo "— not used")"
    printf "  Shell env:   %s\n" "$($use_profile && echo "✓ stored" || echo "— not used")"
    echo
    echo "  Usage examples:"
    echo "    bws -p $profile_name secret list"
    echo "    bws -p $profile_name secret get <secret-id>"
    echo "    bws -p $profile_name project list"
    echo "    BWS_PROFILE=$profile_name bws secret list"
    echo

    if $use_keychain; then
        echo "  To load token from keychain in scripts:"
        echo "    export BWS_ACCESS_TOKEN=\$(security find-generic-password -a \"\$USER\" -s \"$kc_service\" -w)"
        echo
    fi

    if $use_profile; then
        echo "  ${YELLOW}Reload your shell to use the new env var:${RESET}"
        echo "    source $profile_file"
        echo
    fi

    ok "Done!"
}

main "$@"
