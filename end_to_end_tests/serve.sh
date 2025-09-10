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

fastly_pid=""
addr="127.0.0.1:23456"
debug=false

# Cleans up the server listening on addr and any child processes.
# Globals:
#   addr
cleanup() {
  set +e
  if [[ -n "${fastly_pid}" ]] && ps -p "${fastly_pid}" > /dev/null; then
    if command -v pkill >/dev/null; then
      pkill -INT -g "$(ps -o pgid= -p "${fastly_pid}")"
    else
      kill -INT -"${fastly_pid}"
    fi
  fi

  local port
  port=$(echo "${addr}" | cut -d: -f2)
  local pids
  pids=$(lsof -i :${port} -t)
  if [[ -n "${pids}" ]]; then
    kill -TERM ${pids}
  fi
}
trap cleanup EXIT

# Waits for the server to start listening on the specified address.
# Globals:
#   debug
# Arguments:
#   $1: The address to check (e.g., "127.0.0.1:23456")
wait_for_server() {
  local addr="$1"
  local host
  host=$(echo "${addr}" | cut -d: -f1)
  local port
  port=$(echo "${addr}" | cut -d: -f2)

  [[ "${debug}" == "true" ]] && echo "Waiting for server to start..." >&2
  for i in {1..5}; do
    sleep 2
    if nc -z "${host}" "${port}"; then
      [[ "${debug}" == "true" ]] && echo "Server started." >&2
      return 0
    fi
    if (( i == 10 )); then
      echo "Server did not start after 10 seconds." >&2
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
    echo "Usage: $0 [--debug] <binary> [args...]" >&2
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
        ;;      *)
        exec_args+=("$1")
        shift
        ;;    esac
  done

  local fastly_args="--verbose"
  # local fastly_args="--metadata-show --verbose"

  # TODO: --addr (find self in fastly.toml)
  fastly compute serve --addr "${addr}" ${fastly_args} >&2 &
  fastly_pid=$!

  if [[ "${debug}" == "true" ]]; then
    echo "Fastly PID: ${fastly_pid}" >&2
    echo "Fastly command: fastly compute serve --addr \"${addr}\" ${fastly_args} &" >&2
  fi

  wait_for_server "${addr}"

  if [[ "${debug}" == "true" ]]; then
    echo "-- Executing --" >&2
    echo "Cmd: ${binary}" >&2
    echo "Args: ${exec_args[@]}" >&2
  fi

  "${binary}" "${exec_args[@]}"
}

main "$@"
