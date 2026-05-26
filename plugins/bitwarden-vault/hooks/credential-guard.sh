#!/bin/zsh
# credential-guard.sh — Block accidental writes to credential/session files

TARGET="${CLD_EDIT_TARGET:-${CLD_WRITE_FILE:-}}"

case "$TARGET" in
  *bw-plugin/state.json*|
  *bw-plugin/config.json*|
  *.Bitwarden*data*|
  *keychain.plist*|
  *security find-generic-password*)
    echo "SECURITY: Direct edit/write to credential/session file blocked: $TARGET"
    echo "Use bw-plugin auth setup/login/test commands instead."
    exit 1
    ;;
esac

exit 0
