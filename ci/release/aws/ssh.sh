#!/bin/sh
set -eu
. "$(dirname "$0")/../../../ci/sub/lib.sh"
cd -- "$(dirname "$0")/../../.."

help() {
  cat <<EOF
usage: $0 [--dry-run] [--run=regex] ...

Run a command on every builder instance.
EOF
}

main() {
  while flag_parse "$@"; do
    case "$FLAG" in
      h|help)
        help
        return 0
        ;;
      dry-run)
        flag_noarg && shift "$FLAGSHIFT"
        DRY_RUN=1
        ;;
      run)
        flag_reqarg && shift "$FLAGSHIFT"
        JOBFILTER="$FLAGARG"
        ;;
      *)
        flag_errusage "unrecognized flag $FLAGRAW"
        ;;
    esac
  done
  shift "$FLAGSHIFT"

  REMOTE_HOST=$CI_HOST_D2_LINUX_AMD64 && runjob linux-amd64 ssh "$REMOTE_HOST" "$@"
  REMOTE_HOST=$CI_HOST_D2_LINUX_ARM64 && runjob linux-arm64 ssh "$REMOTE_HOST" "$@"
  REMOTE_HOST=$CI_HOST_D2_MACOS_AMD64 && runjob macos-amd64 ssh "$REMOTE_HOST" "$@"
  REMOTE_HOST=$CI_HOST_D2_MACOS_ARM64 && runjob macos-arm64 ssh "$REMOTE_HOST" "$@"
  REMOTE_HOST=$CI_HOST_D2_WINDOWS_AMD64 && runjob macos-arm64 ssh "$REMOTE_HOST" "$@"
}

main "$@"
