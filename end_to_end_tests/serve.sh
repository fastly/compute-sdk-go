#!/bin/bash
#
# This script is designed to be used with `go run -exec` and `go test -exec`.
#
# It starts a `fastly compute serve` instance in the background, waits for it
# to be ready, and then executes the binary that was passed as an argument.
#
# The script handles cleanup of the `fastly` server and its child processes
# on exit.
#
# The --addr flag can be used to specify the address for the server, and
# this flag will be passed to the `fastly` server, but not to the executed
# binary.
#

set -e
set -u
set -m

# Globals that coordinate across functions.
addr="127.0.0.1:23456"
debug=false
fastly_pid=""
prefix="$(basename "$0")"

# Logs a message to stderr with a prefix.
# Globals:
#   prefix
# Arguments:
#   $1: The message to log.
log() {
  echo "${prefix}: $1" >&2
}

# Logs a message to stderr only if debug is enabled.
# Adds the [DEBUG] prefix to output.
# Globals:
#   prefix
# Arguments:
#   $1: The message to log.
log_debug() {
  if [[ "${debug}" == "true" ]]; then
    log "[DEBUG]: $1"
  fi
}

# Cleans up the server listening on addr and any child processes.
# Globals:
#   addr
cleanup() {
  set +e
  if [[ -n "${fastly_pid}" ]] && ps -p "${fastly_pid}" > /dev/null; then
    kill -HUP -"${fastly_pid}"
  fi

  wait_for_server "${addr}" stop
}
trap cleanup EXIT

# Waits for the server to start or stop listening on the specified address.
# Globals:
#   debug
# Arguments:
#   $1: The address to check (e.g., "127.0.0.1:23456")
#   $2: Optional. If set, waits for the server to stop.
wait_for_server() {
  local addr="$1"
  local negate="${2:-}"
  local host
  host=$(echo "${addr}" | cut -d: -f1)
  local port
  port=$(echo "${addr}" | cut -d: -f2)

  local verb="start"
  local past_verb="started"
  if [[ -n "${negate}" ]]; then
    verb="stop"
    past_verb="stopped"
  fi

  _check_server() {
    local host="$1"
    local port="$2"

    if [[ "${debug}" == "true" ]]; then
      log_debug "Checking port with /dev/tcp/${host}/${port}"
    fi

    # This uses a bash built-in command to check if a host & port is listening.
    (true &>/dev/null <>/dev/tcp/${host}/${port})
  }

  log_debug "Waiting for server to ${verb}..."
  for i in {1..30}; do
    sleep 1

    local condition_met=false
    if [[ -z "${negate}" ]]; then
      _check_server "${host}" "${port}" && condition_met=true
    else
      ! _check_server "${host}" "${port}" && condition_met=true
    fi

    if ${condition_met}; then
      log_debug "Server ${past_verb}."
      return 0
    fi

    if (( i == 30 )); then
      log "Server did not ${verb} after 30 seconds."
      return 1
    fi
  done
}

# Main entry point for the script.
# Parses arguments, starts the server, waits for it to be ready, and then
# executes the binary.
# Globals:
#   addr
#   fastly_pid
#   debug
# Arguments:
#   $@
main() {
  if [[ "$1" == "--debug" ]]; then
    debug=true
    shift
  fi

  if (( $# == 0 )); then
    log "Usage: $0 [--debug] <binary> [args...]"
    exit 1
  fi

  local binary="$1"
  shift

  local exec_args=()
  while (( $# > 0 )); do
    case "$1" in
      --addr)
        addr="$2"
        shift 2
        ;;
      *)
        exec_args+=("$1")
        shift
        ;;
    esac
  done

  local server_cmd=()
  local fastly_args=("--addr" "${addr}")
  if [[ "${debug}" == "true" ]]; then
    fastly_args+=("--verbose")
    fastly_args+=("--metadata-show")
  fi

  server_cmd+=("fastly" "compute" "serve")
  server_cmd+=("${fastly_args[@]}")
  log "server command >> ${server_cmd[*]} >&2 &"

  "${server_cmd[@]}" >&2 &
  fastly_pid=$!
  disown
  wait_for_server "${addr}"

  log "test command >> ${binary} ${exec_args[*]}"
  "${binary}" "${exec_args[@]}"
}

main "$@"
