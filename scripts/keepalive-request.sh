#!/usr/bin/env bash
set -euo pipefail

###########################################################
# Free-tier keepalive                                     #
#                                                         #
# Sends periodic HTTP requests to a service to prevent it #
# from being considered idle.                             #
#                                                         #
# Examples:                                               #
#                                                         #
#   ./keepalive.sh \                                      #
#       --live-url https://example.onrender.com/health \  #
#       --interval 10m                                    #
#                                                         #
#   ./keepalive.sh \                                      #
#       --live-url https://example.onrender.com/health \  #
#       --interval 10m \                                  #
#       --quiet-hours 02:00-07:00                         #
#                                                         #
#   ./keepalive.sh \                                      #
#       --live-url https://example.onrender.com/health \  #
#       --interval 10m \                                  #
#       --quiet-hours 02:00-07:00,12:00-13:00             #
#                                                         #
#   ## Cron:                                              #
#   */10 * * * * /path/to/keepalive.sh \                  #
#       --live-url https://example.onrender.com/health \  #
#       --once                                            #
#                                                         #
# Environment variables:                                  #
#                                                         #
#   DASHBOARD_LIVE_URL                                    #
#   DASHBOARD_REQUEST_INTERVAL                            #
#   DASHBOARD_QUIET_HOURS                                 #
###########################################################

LIVE_URL="${DASHBOARD_LIVE_URL:-}"
INTERVAL="${DASHBOARD_REQUEST_INTERVAL:-10m}"
QUIET_HOURS="${DASHBOARD_QUIET_HOURS:-}"

HTTP_CLIENT=""
RUN_ONCE=false

# ---------------------------------------------------------
# Logging
# ---------------------------------------------------------

log() {
  printf '[%s] %s\n' "$(date '+%Y-%m-%d %H:%M:%S')" "$*"
}

warn() {
  printf '[%s] [WARN] %s\n' \
    "$(date '+%Y-%m-%d %H:%M:%S')" \
    "$*" >&2
}

errexit() {
  printf '[ERROR] %s\n' "$*" >&2
  exit 1
}

# ---------------------------------------------------------
# Usage
# ---------------------------------------------------------

usage() {
  cat <<EOF
Usage:
  ${0##*/} [OPTIONS]

Options:

  -u, --live-url     <url>       URL to send requests to.
  -i, --interval     <duration>  Time between requests.
      Supported units:
        s   seconds
        m   minutes
        h   hours
        d   days
      Examples:
        30s
        10m
        1h
        1d
  -q, --quiet-hours  <ranges>    One or more daily periods during which requests are suspended.
      Format:
        HH:MM-HH:MM

      Examples:
        02:00-07:00
        02:00-07:00,12:00-13:00
        23:00-06:00
  --once                         Send one request and exit. Useful for cron.
  -h, --help
      Print this help message.

Environment variables:

  DASHBOARD_LIVE_URL
  DASHBOARD_REQUEST_INTERVAL
  DASHBOARD_QUIET_HOURS

Examples:

  ${0##*/} \\
      --live-url https://example.onrender.com/health \\
      --interval 10m

  ${0##*/} \\
      --live-url https://example.onrender.com/health \\
      --interval 10m \\
      --quiet-hours 02:00-07:00

  ${0##*/} \\
      --live-url https://example.onrender.com/health \\
      --interval 10m \\
      --quiet-hours 02:00-07:00,12:00-13:00

EOF
}

## Parse a duration.
#
#  Examples:
#    30s -> 30
#    10m -> 600
#    1h  -> 3600
#    1d  -> 86400
parse_duration() {
  local value="$1"

  if [[ ! "$value" =~ ^([0-9]+)([smhd])$ ]]; then
    errexit \
      "Invalid interval '${value}'. Use values such as 30s, 10m, 1h, or 1d."
  fi

  local amount="${BASH_REMATCH[1]}"
  local unit="${BASH_REMATCH[2]}"

  case "$unit" in
    s)
      echo "$amount"
      ;;
    m)
      echo "$((amount * 60))"
      ;;
    h)
      echo "$((amount * 60 * 60))"
      ;;
    d)
      echo "$((amount * 60 * 60 * 24))"
      ;;
  esac
}

## Validate HH:MM.
validate_time() {
  local value="$1"

  [[ "$value" =~ ^([01][0-9]|2[0-3]):[0-5][0-9]$ ]]
}

## Convert HH:MM to minutes since midnight.
time_to_minutes() {
  local value="$1"

  local hours="${value%%:*}"
  local minutes="${value##*:}"

  echo "$((10#$hours * 60 + 10#$minutes))"
}

## Validate quiet-hour ranges.
validate_quiet_hours() {
  local ranges="$1"

  [[ -z "$ranges" ]] && return 0

  local range
  local start
  local end

  IFS=',' read -ra range_array <<< "$ranges"

  for range in "${range_array[@]}"; do

    if [[ ! "$range" =~ ^([0-9]{2}:[0-9]{2})-([0-9]{2}:[0-9]{2})$ ]]; then
      errexit \
        "Invalid quiet-hour range '${range}'. Expected HH:MM-HH:MM."
    fi

    start="${BASH_REMATCH[1]}"
    end="${BASH_REMATCH[2]}"

    if ! validate_time "$start"; then
      errexit "Invalid quiet-hour start time '${start}'."
    fi

    if ! validate_time "$end"; then
      errexit "Invalid quiet-hour end time '${end}'."
    fi

    if [[ "$start" == "$end" ]]; then
      errexit \
        "Quiet-hour range '${range}' has identical start and end times."
    fi
  done
}

## Convert a local date/time to epoch seconds.
date_to_epoch() {
  local value="$1"

  if date -d "$value" +%s 2>/dev/null; then
    return 0
  fi

  date -j -f '%Y-%m-%d %H:%M:%S' \
    "$value" \
    '+%s'
}

## Format epoch seconds as: YYYY-MM-DD HH:MM:SS
format_epoch() {
  local epoch="$1"

  if date -d "@${epoch}" '+%Y-%m-%d %H:%M:%S' 2>/dev/null; then
    return 0
  fi

  date -r "$epoch" '+%Y-%m-%d %H:%M:%S'
}

## Get the local date represented by an epoch timestamp.
#
#  Output format:
#    YYYY-MM-DD
epoch_date() {
  local epoch="$1"

  if date -d "@${epoch}" '+%Y-%m-%d' 2>/dev/null; then
    return 0
  fi

  date -r "$epoch" '+%Y-%m-%d'
}

## Shift a YYYY-MM-DD date by N calendar days.
shift_date() {
  local date_string="$1"
  local days="$2"

  if date -d "${date_string} ${days} day" '+%Y-%m-%d' 2>/dev/null; then
    return 0
  fi

  date -j \
    -v"${days}d" \
    -f '%Y-%m-%d' \
    "$date_string" \
    '+%Y-%m-%d'
}

## Determine whether an epoch timestamp falls inside any
#  configured quiet period.

is_epoch_quiet() {
  local epoch="$1"

  QUIET_END_EPOCH=""

  [[ -z "$QUIET_HOURS" ]] && return 1

  local current_date
  local previous_date
  local next_date

  current_date="$(epoch_date "$epoch")"
  previous_date="$(shift_date "$current_date" -1)"
  next_date="$(shift_date "$current_date" 1)"

  local range
  local start
  local end
  local start_minutes
  local end_minutes

  local candidate_start_epoch
  local candidate_end_epoch

  IFS=',' read -ra range_array <<< "$QUIET_HOURS"

  for range in "${range_array[@]}"; do

    start="${range%-*}"
    end="${range#*-}"

    start_minutes="$(time_to_minutes "$start")"
    end_minutes="$(time_to_minutes "$end")"

    ## Normal same-day range: i.e. 02:00-07:00
    if (( start_minutes < end_minutes )); then

      candidate_start_epoch="$(
        date_to_epoch "${current_date} ${start}:00"
      )"

      candidate_end_epoch="$(
        date_to_epoch "${current_date} ${end}:00"
      )"

      if (( epoch >= candidate_start_epoch &&
            epoch < candidate_end_epoch )); then

        QUIET_END_EPOCH="$candidate_end_epoch"
        return 0
      fi

    ## Cross-midnight range: 23:00-06:00
    #
    #  At 02:00 today, the active period started yesterday
    #  at 23:00.
    #
    #  At 23:30 today, the active period started today at
    #  23:00 and ends tomorrow at 06:00.
    else
      ## Yesterday -> today.
      candidate_start_epoch="$(
        date_to_epoch "${previous_date} ${start}:00"
      )"

      candidate_end_epoch="$(
        date_to_epoch "${current_date} ${end}:00"
      )"

      if (( epoch >= candidate_start_epoch &&
            epoch < candidate_end_epoch )); then

        QUIET_END_EPOCH="$candidate_end_epoch"
        return 0
      fi

      ## Today -> tomorrow.
      candidate_start_epoch="$(
        date_to_epoch "${current_date} ${start}:00"
      )"

      candidate_end_epoch="$(
        date_to_epoch "${next_date} ${end}:00"
      )"

      if (( epoch >= candidate_start_epoch &&
            epoch < candidate_end_epoch )); then

        QUIET_END_EPOCH="$candidate_end_epoch"
        return 0
      fi
    fi
  done

  return 1
}

is_quiet_time() {
  local now_epoch

  now_epoch="$(date +%s)"

  is_epoch_quiet "$now_epoch"
}

echo_sleep_until() {
  local now_epoch
  local next_epoch
  local target_epoch
  local sleep_seconds
  local target_timestamp

  now_epoch="$(date +%s)"

  ## Normal next-request time.
  next_epoch=$((now_epoch + INTERVAL_SECONDS))

  ## Default to the normal interval.
  target_epoch="$next_epoch"

  ## If the normal next request lands in quiet hours, move
  #  the request to the end of that quiet period.
  if [[ -n "$QUIET_HOURS" ]]; then
    if is_epoch_quiet "$next_epoch"; then
      target_epoch="$QUIET_END_EPOCH"
      target_timestamp="$(format_epoch "$target_epoch")"

      log "Next request falls during quiet hours; sleeping until ${target_timestamp}."

    fi
  fi

  sleep_seconds=$((target_epoch - now_epoch))

  ## Protect against clock adjustments or an extremely short
  #  interval resulting in a zero/negative sleep.
  if (( sleep_seconds < 1 )); then
    sleep_seconds=1
  fi

  ## If not redirected to the end of quiet hours,
  #  report the ordinary next-request timestamp.
  if (( target_epoch == next_epoch )); then
    target_timestamp="$(format_epoch "$target_epoch")"

    log "Sleeping until ${target_timestamp}."
  fi

  sleep "$sleep_seconds"
}

check_dependencies() {
  if command -v curl >/dev/null 2>&1; then
    HTTP_CLIENT="curl"
    return
  fi

  if command -v wget >/dev/null 2>&1; then
    HTTP_CLIENT="wget"
    return
  fi

  errexit "Neither curl nor wget is installed."
}

request() {
  log "Requesting ${LIVE_URL}"

  case "$HTTP_CLIENT" in
    curl)
      curl \
        --fail \
        --silent \
        --show-error \
        --location \
        --connect-timeout 10 \
        --max-time 60 \
        --output /dev/null \
        "$LIVE_URL"
      ;;
    wget)
      wget \
        --quiet \
        --timeout=60 \
        --output-document=/dev/null \
        "$LIVE_URL"
      ;;
    *)
      errexit "Unknown HTTP client '${HTTP_CLIENT}'."
      ;;
  esac
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    -u|--live-url)
      [[ $# -ge 2 ]] || errexit "$1 requires an argument"

      LIVE_URL="$2"

      shift 2
      ;;
    -i|--interval)
      [[ $# -ge 2 ]] || errexit "$1 requires an argument"
      INTERVAL="$2"

      shift 2
      ;;
    -q|--quiet-hours)
      [[ $# -ge 2 ]] || errexit "$1 requires an argument"
      QUIET_HOURS="$2"

      shift 2
      ;;
    --once)
      RUN_ONCE=true

      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "[ERROR] Invalid option: $1" >&2
      usage

      exit 1
      ;;

  esac

done

[[ -n "$LIVE_URL" ]] || \
  errexit "--live-url cannot be empty."

if [[ ! "$LIVE_URL" =~ ^https?:// ]]; then
  errexit "--live-url must begin with http:// or https://."
fi

INTERVAL_SECONDS="$(parse_duration "$INTERVAL")"

if (( INTERVAL_SECONDS <= 0 )); then
  errexit "Interval must be greater than zero."
fi

validate_quiet_hours "$QUIET_HOURS"

check_dependencies

echo
echo "HTTP keepalive request script"
echo "-----------------------------"
echo "URL:           ${LIVE_URL}"
echo "HTTP client:   ${HTTP_CLIENT}"
echo "Interval:      ${INTERVAL} (${INTERVAL_SECONDS}s)"

if [[ -n "$QUIET_HOURS" ]]; then
  echo "Quiet hours:   ${QUIET_HOURS}"
else
  echo "Quiet hours:   none"
fi

echo

if [[ "$RUN_ONCE" == true ]]; then

  if request; then
    log "Request succeeded."
    exit 0
  else
    warn "Request failed."
    exit 1
  fi

fi

shutdown() {
  echo
  log "Stopping keepalive."
  exit 0
}

trap shutdown SIGINT SIGTERM

while true; do
  if is_quiet_time; then
    target_timestamp="$(format_epoch "$QUIET_END_EPOCH")"

    log "Quiet hours active; sleeping until ${target_timestamp}."

    now_epoch="$(date +%s)"
    sleep_seconds=$((QUIET_END_EPOCH - now_epoch))

    if (( sleep_seconds > 0 )); then
      sleep "$sleep_seconds"
    fi

    continue
  fi

  if request; then
    log "Request succeeded."
  else
    warn "Request failed."
  fi

  echo_sleep_until

done
