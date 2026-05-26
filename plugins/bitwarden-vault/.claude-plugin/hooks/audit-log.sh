#!/bin/zsh
# audit-log.sh — Log security-relevant bash commands for traceability

CMD="${CLD_BASH_COMMAND:-}"

# Only log potentially sensitive operations
case "$CMD" in
  *bw-plugin*auth*setup*|\
  *bw-plugin*auth*login*|\
  *bw-plugin*unlock*|\
  *bw-plugin*export*|\
  *bws*secret*|\
  *security*find-generic*|\
  *security*add-generic*)
    mkdir -p ~/.local/share/bw-plugin
    echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) $CMD" >> ~/.local/share/bw-plugin/audit.log
    ;;
esac

exit 0
