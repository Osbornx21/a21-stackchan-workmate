#!/bin/zsh
set -u

GATEWAY_URL="${A21_GATEWAY_URL:-http://127.0.0.1:21081}"
CURL_MAX_TIME="${A21_CONTROL_CURL_MAX_TIME:-3}"
DEVICE_ID="${A21_DEVICE_ID:-}"

print_line() {
  print -r -- "$*"
}

usage() {
  cat <<'EOF'
A21 StackChan foreground control helper

Usage:
  a21-stackchan-control.command
  a21-stackchan-control.command status
  a21-stackchan-control.command volume
  a21-stackchan-control.command probe-action
  a21-stackchan-control.command face happy
  a21-stackchan-control.command motion nod
  a21-stackchan-control.command motion look_up 120
  a21-stackchan-control.command state idle
  a21-stackchan-control.command display status "A21 OK"
  a21-stackchan-control.command diagnostic-tone 255

Environment:
  A21_GATEWAY_URL=http://127.0.0.1:21081
  A21_DEVICE_ID=44:1b:f6:e2:6a:60

Important:
  - This is StackChan-focused. It does not change macOS system volume.
  - Current stock Xiaozhi Gateway path has no runtime StackChan speaker-volume
    setter. The volume command reports that boundary instead of pretending.
  - diagnostic-tone uses the legacy A21 /v1/devices/control audio WebSocket
    path if connected. It is a speaker/tone diagnostic, not product TTS volume.
  - This tool does not flash firmware, write NVS, call providers, or inject TTS.
EOF
}

curl_a21() {
  curl -sS --noproxy "*" --max-time "$CURL_MAX_TIME" "$@"
}

gateway_health() {
  curl_a21 "$GATEWAY_URL/healthz"
}

devices_json() {
  curl_a21 "$GATEWAY_URL/v1/devices"
}

show_status() {
  print_line "== A21 StackChan foreground status =="
  print_line "[config] gateway=$GATEWAY_URL device=${DEVICE_ID:-auto}"
  local health
  if health="$(gateway_health 2>&1)"; then
    print_line "[network] Gateway health ok: $health"
  else
    print_line "[network] Gateway health failed at $GATEWAY_URL/healthz"
    print_line "$health"
    return 1
  fi

  local json
  if ! json="$(devices_json 2>&1)"; then
    print_line "[device] failed to fetch devices from $GATEWAY_URL/v1/devices"
    print_line "$json"
    return 1
  fi
  print -r -- "$json" | /usr/bin/python3 -c '
import json, sys
data = json.load(sys.stdin)
devices = data.get("devices", [])
if not devices:
    print("[device] no devices registered")
    raise SystemExit(2)
for dev in devices:
    caps = dev.get("capabilities", {})
    runtime = dev.get("runtime_echo", {})
    print(
        "[device] id={id} status={status} age_ms={age} expression={expr} "
        "profile={profile} transport={transport} audio={audio} speaker={speaker} "
        "device_events={events} last_motion={motion}".format(
            id=dev.get("device_id", ""),
            status=dev.get("connection_status", ""),
            age=dev.get("device_age_ms", ""),
            expr=dev.get("current_expression", ""),
            profile=caps.get("xiaozhi_profile", ""),
            transport=caps.get("xiaozhi_transport", ""),
            audio=caps.get("xiaozhi_audio", ""),
            speaker=caps.get("speaker", ""),
            events=caps.get("xiaozhi_feature_device_events", ""),
            motion=runtime.get("last_motion_command", ""),
        )
    )
'
  show_volume_boundary
  show_action_boundary
}

detect_device() {
  if [[ -n "$DEVICE_ID" ]]; then
    print -r -- "$DEVICE_ID"
    return 0
  fi
  local json
  if ! json="$(devices_json 2>/dev/null)"; then
    return 1
  fi
  print -r -- "$json" | /usr/bin/python3 -c '
import json, sys
data = json.load(sys.stdin)
devices = data.get("devices", [])
online = [
    d for d in devices
    if d.get("connection_status") == "online"
    and d.get("capabilities", {}).get("xiaozhi_transport") == "websocket"
]
physical = [
    d for d in online
    if "virtual" not in d.get("device_id", "").lower()
    and "bench" not in d.get("device_id", "").lower()
]
chosen = (physical or online or devices or [{}])[0]
device_id = chosen.get("device_id", "")
if not device_id:
    raise SystemExit(2)
print(device_id)
'
}

show_volume_boundary() {
  print_line "[volume] StackChan runtime volume setter: not exposed on current stock /v1/xiaozhi Gateway path."
  print_line "[volume] Current path can report speaker=available_xiaozhi_opus_downlink, but not set speaker gain."
  print_line "[volume] Known firmware knobs are code-level: official codec SetOutputVolume(...) or old M5.Speaker.setVolume(...)."
  print_line "[volume] To actually raise StackChan TTS loudness, use a planned firmware/protocol transition, then build/flash/record evidence."
}

show_action_boundary() {
  print_line "[action] /v1/xiaozhi/control requires debug profile negotiation on the live socket."
  print_line "[action] A stock device can be online for audio while still rejecting host-pushed face/motion events with HTTP 409."
}

volume_command() {
  show_status || true
  print_line "[volume] No StackChan volume change was sent."
  return 2
}

validate_control() {
  local kind="$1"
  local value="$2"
  case "$kind:$value" in
    state:idle|state:listening|state:thinking|state:speaking|state:error) return 0 ;;
    face:idle|face:attentive|face:thinking|face:speaking|face:happy|face:error) return 0 ;;
    motion:look_up|motion:nod|motion:shake|motion:stop|motion:dance) return 0 ;;
    display:status|display:asr|display:tts) return 0 ;;
  esac
  print_line "[control] unsupported $kind value '$value'"
  return 2
}

control_payload() {
  local device="$1"
  local kind="$2"
  local value="$3"
  local extra="${4:-}"
  local stamp
  stamp="$(date +%s)"
  /usr/bin/python3 - "$device" "$kind" "$value" "$extra" "$stamp" <<'PY'
import json, sys
device, kind, value, extra, stamp = sys.argv[1:6]
payload = {
    "device_id": device,
    "event": kind,
    "trace_id": f"a21-trace-stackchan-desktop-control-{stamp}",
    "session_id": f"a21-session-stackchan-desktop-control-{stamp}",
    "reason": "stackchan_desktop_control",
}
if kind == "face":
    payload["emotion"] = value
elif kind == "motion":
    payload["name"] = value
    if extra:
        try:
            payload["y_angle"] = int(extra)
        except ValueError:
            payload["text"] = extra
elif kind == "state":
    payload["state"] = value
elif kind == "display":
    payload["slot"] = value
    if extra:
        payload["text"] = extra
print(json.dumps(payload, ensure_ascii=True, separators=(",", ":")))
PY
}

post_json() {
  local url="$1"
  local payload="$2"
  curl_a21 -X POST -H 'Content-Type: application/json' --data "$payload" -w $'\nHTTP_STATUS:%{http_code}' "$url" 2>&1
}

send_control() {
  local kind="$1"
  local value="$2"
  local extra="${3:-}"
  if [[ -z "$value" ]]; then
    print_line "[control] missing value for $kind"
    return 2
  fi
  validate_control "$kind" "$value" || return $?
  local device
  if ! device="$(detect_device)"; then
    print_line "[control] could not detect an online xiaozhi device. Set A21_DEVICE_ID explicitly."
    return 1
  fi
  local payload response code body
  payload="$(control_payload "$device" "$kind" "$value" "$extra")"
  response="$(post_json "$GATEWAY_URL/v1/xiaozhi/control" "$payload")"
  code="${response##*HTTP_STATUS:}"
  body="${response%$'\n'HTTP_STATUS:*}"
  if [[ "$code" == 2* ]]; then
    print_line "[control] delivered $kind=$value to device=$device"
    print_line "$body"
    return 0
  fi
  print_line "[control] failed HTTP $code for $kind=$value device=$device"
  print_line "$body"
  return 1
}

diagnostic_tone_payload() {
  local device="$1"
  local volume="$2"
  local stamp
  stamp="$(date +%s)"
  /usr/bin/python3 - "$device" "$volume" "$stamp" <<'PY'
import json, sys
device, volume, stamp = sys.argv[1:4]
payload = {
    "device_id": device,
    "state": "speaking",
    "mode": "workmate",
    "text": "DIAGNOSTIC TONE",
    "trace_id": f"a21-trace-stackchan-tone-{stamp}",
    "session_id": f"a21-session-stackchan-tone-{stamp}",
    "stream_id": f"a21-stackchan-tone-{stamp}",
    "diagnostic_tone_hz": 1000,
    "diagnostic_tone_duration_ms": 1200,
    "diagnostic_tone_volume": int(volume),
}
print(json.dumps(payload, ensure_ascii=True, separators=(",", ":")))
PY
}

diagnostic_tone() {
  local volume="${1:-255}"
  if ! [[ "$volume" == <1-255> ]]; then
    print_line "[tone] volume must be 1-255"
    return 2
  fi
  local device
  if ! device="$(detect_device)"; then
    print_line "[tone] could not detect an online device. Set A21_DEVICE_ID explicitly."
    return 1
  fi
  print_line "[tone] trying legacy /v1/devices/control diagnostic tone at volume=$volume"
  print_line "[tone] This is not product TTS volume. Current stock xiaozhi devices are expected to reject this if audio_ws is not connected."
  local payload response code body
  payload="$(diagnostic_tone_payload "$device" "$volume")"
  response="$(post_json "$GATEWAY_URL/v1/devices/control" "$payload")"
  code="${response##*HTTP_STATUS:}"
  body="${response%$'\n'HTTP_STATUS:*}"
  if [[ "$code" == 2* ]]; then
    print_line "[tone] diagnostic tone delivered through legacy audio_ws"
    print_line "$body"
    return 0
  fi
  print_line "[tone] diagnostic tone failed HTTP $code"
  print_line "$body"
  return 1
}

probe_action() {
  show_status || true
  send_control face happy || true
  send_control motion nod || true
}

menu() {
  while true; do
    print_line ""
    print_line "A21 StackChan foreground control helper"
    print_line "Gateway: $GATEWAY_URL"
    print_line "Device: ${DEVICE_ID:-auto-detect}"
    print_line "1) Status and StackChan volume boundary"
    print_line "2) Explain StackChan volume control"
    print_line "3) Probe face happy"
    print_line "4) Probe motion nod"
    print_line "5) Motion look_up 120"
    print_line "6) Try legacy diagnostic tone volume 255"
    print_line "7) Help"
    print_line "0) Quit"
    local choice
    read "choice?Choose: "
    case "$choice" in
      1) show_status ;;
      2) volume_command ;;
      3) send_control face happy ;;
      4) send_control motion nod ;;
      5) send_control motion look_up 120 ;;
      6) diagnostic_tone 255 ;;
      7) usage ;;
      0|q|quit|exit) return 0 ;;
      *) print_line "Unknown choice: $choice" ;;
    esac
  done
}

cmd="${1:-menu}"
case "$cmd" in
  -h|--help|help) usage ;;
  menu) menu ;;
  status) show_status ;;
  volume) volume_command ;;
  probe-action) probe_action ;;
  diagnostic-tone) diagnostic_tone "${2:-255}" ;;
  face|motion|state|display)
    send_control "$cmd" "${2:-}" "${3:-}"
    ;;
  *)
    print_line "Unknown command: $cmd"
    usage
    exit 2
    ;;
esac
