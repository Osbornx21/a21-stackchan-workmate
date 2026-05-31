#!/usr/bin/env python3
"""Host-only A21 virtual Xiaozhi WebSocket harness.

This script intentionally keeps the harness self-contained: it uses only the
Python standard library and never records raw audio, transcripts, provider
output, credentials, full URLs, proxy values, or local paths in reports.
"""

from __future__ import annotations

import argparse
import base64
import hashlib
import json
import math
import os
import socket
import struct
import sys
import time
import urllib.parse
from dataclasses import dataclass
from typing import Any


DEFAULT_TARGET_URL = "ws://127.0.0.1:21080/v1/xiaozhi"
DEFAULT_DEVICE_ID = "stackchan-virtual-a21-001"
DEFAULT_TRACE_ID = "a21-trace-virtual-xiaozhi"
DEFAULT_SESSION_ID = "a21-session-virtual-xiaozhi"
GUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
BUILTIN_SPEECH_OPUS_BASE64 = (
    "+wN9Fe8q/WweIrO8S6e3BS1VEZTVvOfWfIBmbF67EEX+tPRA8GZC+5Sr1DdpepXftLCgFnrYd7qX"
    "C3CgEU1P+mWzEPB6CuikBp4i80h/v5ZElM3JVM3JVn+LjVu/W8G/W8AxQpfypJthUGOaQtiEWod"
    "EhR/BQP4V181rZfIBwsl7tW4gDS91S9n3iAtH9R4JQaNYraLWjZgQY0xzadd4sn5p2JBpjk/gj1"
    "qW8BAzZp7OblAZp4KzmA0XC4mTIPVs70t+b3CwVujVZrf3QUvzXZLOuyxn6LPXK2VJPY7GfCxQg"
    "FPA/DDvwuMROoZSAE0WnRnwUY4cxwnCzY/40DIbrFI/oNIbEViu2GV+H0MMhxIRV9USaclvjzp2"
    "vVbi4Fr+bZHFd/yyhHjYLj8lMgIJXfVn323gAAPzHxXIU+1ZinogXdnvKTZez35gLCpTfzlrw22"
    "7xafE4ixdvVpmwlsXGPucZcz4dxRjaJFXScmUtaJD9EAt3cyvhVdXP754y4ja2ZOguiPrANR/jU"
    "RqlZgR2alef7EAY2Pgq2iiwhFPMvZxBiU742CMcsTryLt+S/vkN3GVSDAMb9M1J04zfHjKQ9M6T"
    "emRKqL2nWTOo8UqBy24aJcqmqgsAlzYXOA="
)


@dataclass
class HarnessOptions:
    target_url: str
    profile: str
    device_id: str
    trace_id: str
    session_id: str
    protocol_version: int
    expect_audio: bool
    require_metrics: bool
    abort_after_first_audio: bool
    timeout_ms: int
    json_report: bool
    opus_fixture: str | None
    repeat: int


class WebSocketError(RuntimeError):
    pass


class StdlibWebSocket:
    def __init__(self, raw_url: str, timeout_ms: int, headers: dict[str, str]):
        self.raw_url = raw_url
        self.timeout_s = max(timeout_ms, 1) / 1000.0
        self.headers = headers
        self.sock: socket.socket | None = None

    def __enter__(self) -> "StdlibWebSocket":
        parsed = urllib.parse.urlparse(self.raw_url)
        if parsed.scheme != "ws" or not parsed.hostname:
            raise WebSocketError("unsupported_target")
        port = parsed.port or 80
        path = parsed.path or "/"
        if parsed.query:
            path = f"{path}?{parsed.query}"
        sock = socket.create_connection((parsed.hostname, port), timeout=self.timeout_s)
        sock.settimeout(self.timeout_s)
        key = base64.b64encode(os.urandom(16)).decode("ascii")
        host = parsed.hostname if parsed.port is None else f"{parsed.hostname}:{port}"
        request_headers = {
            "Host": host,
            "Upgrade": "websocket",
            "Connection": "Upgrade",
            "Sec-WebSocket-Key": key,
            "Sec-WebSocket-Version": "13",
        }
        request_headers.update(self.headers)
        request = [f"GET {path} HTTP/1.1"]
        request.extend(f"{name}: {value}" for name, value in request_headers.items())
        request.extend(["", ""])
        sock.sendall("\r\n".join(request).encode("ascii"))
        response = self._read_http_response(sock)
        expected_accept = base64.b64encode(hashlib.sha1((key + GUID).encode("ascii")).digest()).decode("ascii")
        if " 101 " not in response.split("\r\n", 1)[0]:
            raise WebSocketError("handshake_rejected")
        if expected_accept.lower() not in response.lower():
            raise WebSocketError("handshake_accept_mismatch")
        self.sock = sock
        return self

    def __exit__(self, exc_type: Any, exc: Any, tb: Any) -> None:
        if self.sock is None:
            return
        try:
            self._send_frame(0x8, b"")
        except OSError:
            pass
        self.sock.close()
        self.sock = None

    @staticmethod
    def _read_http_response(sock: socket.socket) -> str:
        data = bytearray()
        while b"\r\n\r\n" not in data:
            chunk = sock.recv(4096)
            if not chunk:
                break
            data.extend(chunk)
            if len(data) > 65536:
                raise WebSocketError("handshake_too_large")
        return data.decode("iso-8859-1", errors="replace")

    def send_json(self, payload: dict[str, Any]) -> None:
        self._send_frame(0x1, json.dumps(payload, separators=(",", ":")).encode("utf-8"))

    def send_binary(self, payload: bytes) -> None:
        self._send_frame(0x2, payload)

    def recv(self) -> tuple[str, Any] | None:
        while True:
            frame = self._recv_frame()
            if frame is None:
                return None
            opcode, payload = frame
            if opcode == 0x1:
                try:
                    return "json", json.loads(payload.decode("utf-8"))
                except json.JSONDecodeError:
                    return "json", {"type": "invalid_json"}
            if opcode == 0x2:
                return "binary", len(payload)
            if opcode == 0x8:
                return "close", None
            if opcode == 0x9:
                self._send_frame(0xA, payload)

    def _send_frame(self, opcode: int, payload: bytes) -> None:
        if self.sock is None:
            raise WebSocketError("not_connected")
        header = bytearray([0x80 | opcode])
        length = len(payload)
        if length < 126:
            header.append(0x80 | length)
        elif length <= 0xFFFF:
            header.append(0x80 | 126)
            header.extend(struct.pack("!H", length))
        else:
            header.append(0x80 | 127)
            header.extend(struct.pack("!Q", length))
        mask = os.urandom(4)
        masked = bytes(payload[i] ^ mask[i % 4] for i in range(length))
        self.sock.sendall(bytes(header) + mask + masked)

    def _recv_frame(self) -> tuple[int, bytes] | None:
        if self.sock is None:
            raise WebSocketError("not_connected")
        header = self._recv_exact(2)
        if not header:
            return None
        opcode = header[0] & 0x0F
        masked = bool(header[1] & 0x80)
        length = header[1] & 0x7F
        if length == 126:
            length = struct.unpack("!H", self._recv_exact(2))[0]
        elif length == 127:
            length = struct.unpack("!Q", self._recv_exact(8))[0]
        mask = self._recv_exact(4) if masked else b""
        payload = self._recv_exact(length)
        if masked:
            payload = bytes(payload[i] ^ mask[i % 4] for i in range(length))
        return opcode, payload

    def _recv_exact(self, size: int) -> bytes:
        data = bytearray()
        while len(data) < size:
            chunk = self.sock.recv(size - len(data))  # type: ignore[union-attr]
            if not chunk:
                raise WebSocketError("connection_closed")
            data.extend(chunk)
        return bytes(data)


def reject_legacy_identity(value: str, field: str) -> None:
    lowered = value.lower()
    if "x21" in lowered or "v21" in lowered:
        raise ValueError(f"{field} contains legacy identity")


def build_hello_payload(
    *,
    profile: str,
    device_id: str,
    trace_id: str,
    session_id: str,
    protocol_version: int,
) -> dict[str, Any]:
    if profile not in {"xiaozhi", "a21-debug"}:
        raise ValueError("profile must be xiaozhi or a21-debug")
    if protocol_version not in {1, 2, 3}:
        raise ValueError("protocol_version must be 1, 2, or 3")
    for field, value in {
        "device_id": device_id,
        "trace_id": trace_id,
        "session_id": session_id,
    }.items():
        reject_legacy_identity(value, field)
    features: dict[str, bool] = {
        "mcp": True,
        "aec": True,
    }
    if profile == "a21-debug":
        features["device_events"] = True
        features["debug_metrics"] = True
    audio_params = {
        "format": "opus",
        "sample_rate": 16000,
        "channels": 1,
        "frame_duration": 60,
        "binary_protocol_version": protocol_version,
    }
    return {
        "type": "hello",
        "version": protocol_version,
        "transport": "websocket",
        "device_id": device_id,
        "trace_id": trace_id,
        "session_id": session_id,
        "features": features,
        "audio": dict(audio_params),
        "audio_params": audio_params,
    }


def build_listen_payload(state: str, options: HarnessOptions) -> dict[str, str]:
    return {
        "type": "listen",
        "state": state,
        "device_id": options.device_id,
        "trace_id": options.trace_id,
        "session_id": options.session_id,
    }


def build_abort_payload(options: HarnessOptions) -> dict[str, str]:
    return {
        "type": "abort",
        "reason": "barge_in",
        "device_id": options.device_id,
        "trace_id": options.trace_id,
        "session_id": options.session_id,
    }


def wrap_opus_payload(payload: bytes, protocol_version: int) -> bytes:
    if protocol_version == 1:
        return payload
    if protocol_version == 2:
        return struct.pack("!HHIII", 2, 0, 0, int(time.time() * 1000) & 0xFFFFFFFF, len(payload)) + payload
    if protocol_version == 3:
        return bytes([0, 0]) + struct.pack("!H", len(payload)) + payload
    raise ValueError("protocol_version must be 1, 2, or 3")


def redact_target(raw_url: str) -> str:
    parsed = urllib.parse.urlparse(raw_url)
    if parsed.scheme != "ws" or not parsed.hostname:
        return "redacted"
    port = "" if parsed.port is None else f":{parsed.port}"
    path = parsed.path or "/"
    return f"{parsed.hostname}{port}{path}"


def build_report(
    *,
    target_url: str,
    profile: str,
    device_id: str,
    trace_id: str,
    session_id: str,
    protocol_version: int,
) -> dict[str, Any]:
    return {
        "schema": "a21.virtual_xiaozhi_harness.v1",
        "target": redact_target(target_url),
        "profile": profile,
        "device_id": device_id,
        "trace_id": trace_id,
        "session_id": session_id,
        "protocol_version": protocol_version,
        "hello_accepted": False,
        "listen_ack": False,
        "binary_downlink_frames": 0,
        "first_audio_ms": None,
        "tts_stop_received": False,
        "abort_sent": False,
        "abort_stop_ms": None,
        "metrics_observed": False,
        "opus_fixture": "none",
        "prd_accepted": False,
    }


def percentile95(samples: list[int]) -> int | None:
    if not samples:
        return None
    ordered = sorted(samples)
    index = max(0, math.ceil(len(ordered) * 0.95) - 1)
    return ordered[index]


def build_aggregate_report(
    *,
    base_report: dict[str, Any],
    run_reports: list[dict[str, Any]],
    exit_codes: list[int],
    abort_after_first_audio: bool,
) -> dict[str, Any]:
    aggregate = dict(base_report)
    if run_reports:
        aggregate.update(
            {
                key: value
                for key, value in run_reports[-1].items()
                if key
                not in {
                    "run_index",
                    "target",
                    "profile",
                    "device_id",
                    "trace_id",
                    "session_id",
                    "protocol_version",
                    "schema",
                }
            }
        )
    first_audio_samples = [
        report["first_audio_ms"]
        for report in run_reports
        if isinstance(report.get("first_audio_ms"), int)
    ]
    abort_stop_samples = [
        report["abort_stop_ms"]
        for report in run_reports
        if isinstance(report.get("abort_stop_ms"), int)
    ]
    aggregate.update(
        {
            "runs": len(run_reports),
            "successful_runs": sum(1 for code in exit_codes if code == 0),
            "exit_codes": list(exit_codes),
            "run_reports": run_reports,
            "first_audio_samples_ms": first_audio_samples,
            "first_audio_p95_ms": percentile95(first_audio_samples),
            "abort_stop_samples_ms": abort_stop_samples,
            "abort_stop_p95_ms": percentile95(abort_stop_samples),
            "host_candidate": False,
            "prd_accepted": False,
        }
    )
    all_runs_green = bool(run_reports) and all(code == 0 for code in exit_codes)
    all_runs_have_audio = len(first_audio_samples) == len(run_reports)
    first_audio_ok = aggregate["first_audio_p95_ms"] is not None and aggregate["first_audio_p95_ms"] < 1500
    abort_ok = True
    if abort_after_first_audio:
        abort_ok = (
            len(abort_stop_samples) == len(run_reports)
            and aggregate["abort_stop_p95_ms"] is not None
            and aggregate["abort_stop_p95_ms"] < 300
        )
    repeat_window_ok = len(run_reports) >= 3
    aggregate["host_candidate"] = (
        repeat_window_ok and all_runs_green and all_runs_have_audio and first_audio_ok and abort_ok
    )
    return aggregate


def mark_json_event(report: dict[str, Any], message: dict[str, Any]) -> None:
    msg_type = message.get("type")
    state = message.get("state")
    if msg_type == "hello" and message.get("transport") == "websocket":
        report["hello_accepted"] = True
    if msg_type == "listen" and state == "start" and message.get("status") == "accepted":
        report["listen_ack"] = True
    if msg_type == "tts" and isinstance(message.get("audio_ingress"), dict):
        report["metrics_observed"] = True
    if msg_type == "tts" and isinstance(message.get("voice_pipeline"), dict):
        report["metrics_observed"] = True
    if msg_type == "tts" and state == "stop":
        report["tts_stop_received"] = True


def read_fixture(path: str | None) -> bytes | None:
    if path is None:
        return None
    with open(path, "rb") as handle:
        data = handle.read()
    if not data:
        raise ValueError("opus fixture is empty")
    return data


def builtin_speech_opus_fixture() -> bytes:
    return base64.b64decode(BUILTIN_SPEECH_OPUS_BASE64)


def resolve_opus_fixture(path: str | None, *, expect_audio: bool) -> tuple[bytes | None, str]:
    if path in ("builtin_speech", "builtin:speech"):
        return builtin_speech_opus_fixture(), "builtin_speech"
    if path is not None:
        return read_fixture(path), "provided"
    if expect_audio:
        return builtin_speech_opus_fixture(), "builtin_speech"
    return None, "none"


def run_self_test() -> int:
    stock = build_hello_payload(
        profile="xiaozhi",
        device_id=DEFAULT_DEVICE_ID,
        trace_id=DEFAULT_TRACE_ID,
        session_id=DEFAULT_SESSION_ID,
        protocol_version=3,
    )
    debug = build_hello_payload(
        profile="a21-debug",
        device_id="stackchan-virtual-a21-debug-001",
        trace_id="a21-trace-virtual-xiaozhi-debug",
        session_id="a21-session-virtual-xiaozhi-debug",
        protocol_version=2,
    )
    report = build_report(
        target_url="ws://127.0.0.1:21080/v1/xiaozhi?token=secret",
        profile="xiaozhi",
        device_id=DEFAULT_DEVICE_ID,
        trace_id=DEFAULT_TRACE_ID,
        session_id=DEFAULT_SESSION_ID,
        protocol_version=1,
    )
    rendered_stock = json.dumps(stock, sort_keys=True)
    rendered_report = json.dumps(report, sort_keys=True)
    assert "device_events" not in rendered_stock
    assert "debug_metrics" not in rendered_stock
    assert debug["features"]["device_events"] is True
    assert debug["features"]["debug_metrics"] is True
    assert report["target"] == "127.0.0.1:21080/v1/xiaozhi"
    assert report["prd_accepted"] is False
    assert "secret" not in rendered_report

    assert percentile95([]) is None
    assert percentile95([120, 80, 200, 160, 40]) == 200

    run_a = dict(report)
    run_a.update({"first_audio_ms": 80, "abort_stop_ms": 42, "binary_downlink_frames": 1})
    run_b = dict(report)
    run_b.update({"first_audio_ms": 120, "abort_stop_ms": 58, "binary_downlink_frames": 2})
    run_c = dict(report)
    run_c.update({"first_audio_ms": None, "abort_stop_ms": None, "binary_downlink_frames": 0})
    aggregate = build_aggregate_report(
        base_report=report,
        run_reports=[run_a, run_b, run_c],
        exit_codes=[0, 0, 2],
        abort_after_first_audio=True,
    )
    rendered_aggregate = json.dumps(aggregate, sort_keys=True)
    assert aggregate["runs"] == 3
    assert aggregate["successful_runs"] == 2
    assert aggregate["first_audio_samples_ms"] == [80, 120]
    assert aggregate["first_audio_p95_ms"] == 120
    assert aggregate["abort_stop_samples_ms"] == [42, 58]
    assert aggregate["abort_stop_p95_ms"] == 58
    assert aggregate["host_candidate"] is False
    assert aggregate["prd_accepted"] is False
    assert "secret" not in rendered_aggregate
    assert "data_base64" not in rendered_aggregate

    host_run_a = dict(report)
    host_run_a.update({"first_audio_ms": 820, "abort_stop_ms": 180, "binary_downlink_frames": 1})
    host_run_b = dict(report)
    host_run_b.update({"first_audio_ms": 930, "abort_stop_ms": 220, "binary_downlink_frames": 1})
    host_run_c = dict(report)
    host_run_c.update({"first_audio_ms": 1200, "abort_stop_ms": 260, "binary_downlink_frames": 1})
    host_candidate = build_aggregate_report(
        base_report=report,
        run_reports=[host_run_a, host_run_b, host_run_c],
        exit_codes=[0, 0, 0],
        abort_after_first_audio=True,
    )
    assert host_candidate["host_candidate"] is True
    assert host_candidate["prd_accepted"] is False
    print("a21 virtual xiaozhi self-test passed")
    return 0


def run_harness(options: HarnessOptions) -> tuple[int, dict[str, Any]]:
    report = build_report(
        target_url=options.target_url,
        profile=options.profile,
        device_id=options.device_id,
        trace_id=options.trace_id,
        session_id=options.session_id,
        protocol_version=options.protocol_version,
    )
    start_time = time.monotonic()
    abort_at: float | None = None
    try:
        fixture, fixture_label = resolve_opus_fixture(options.opus_fixture, expect_audio=options.expect_audio)
        report["opus_fixture"] = fixture_label
        headers = {
            "Device-Id": options.device_id,
            "Protocol-Version": str(options.protocol_version),
        }
        with StdlibWebSocket(options.target_url, options.timeout_ms, headers) as conn:
            conn.send_json(
                build_hello_payload(
                    profile=options.profile,
                    device_id=options.device_id,
                    trace_id=options.trace_id,
                    session_id=options.session_id,
                    protocol_version=options.protocol_version,
                )
            )
            hello = conn.recv()
            if hello and hello[0] == "json":
                mark_json_event(report, hello[1])
            conn.send_json(build_listen_payload("start", options))
            start_ack = conn.recv()
            if start_ack and start_ack[0] == "json":
                mark_json_event(report, start_ack[1])
            if fixture is not None:
                conn.send_binary(wrap_opus_payload(fixture, options.protocol_version))
            conn.send_json(build_listen_payload("stop", options))
            deadline = time.monotonic() + max(options.timeout_ms, 1) / 1000.0
            while time.monotonic() < deadline:
                remaining = deadline - time.monotonic()
                if conn.sock is not None:
                    conn.sock.settimeout(max(remaining, 0.001))
                try:
                    event = conn.recv()
                except socket.timeout:
                    break
                if event is None:
                    break
                kind, payload = event
                if kind == "json":
                    mark_json_event(report, payload)
                    if report["abort_sent"] and report["tts_stop_received"] and abort_at is not None:
                        report["abort_stop_ms"] = round((time.monotonic() - abort_at) * 1000)
                        break
                    if report["tts_stop_received"] and not options.abort_after_first_audio:
                        break
                elif kind == "binary":
                    report["binary_downlink_frames"] += 1
                    if report["first_audio_ms"] is None:
                        report["first_audio_ms"] = round((time.monotonic() - start_time) * 1000)
                    if options.abort_after_first_audio and not report["abort_sent"]:
                        conn.send_json(build_abort_payload(options))
                        report["abort_sent"] = True
                        abort_at = time.monotonic()
                elif kind == "close":
                    break
    except Exception as exc:  # Report only error class, never full local/URL context.
        report["error"] = type(exc).__name__
        return 1, report

    exit_code = 0
    if options.expect_audio and report["binary_downlink_frames"] == 0:
        report["failure"] = "expected_audio_not_observed"
        exit_code = 2
    if options.require_metrics and not report["metrics_observed"]:
        report["failure"] = "required_metrics_not_observed"
        exit_code = 2
    return exit_code, report


def run_repeated_harness(options: HarnessOptions) -> tuple[int, dict[str, Any]]:
    base_report = build_report(
        target_url=options.target_url,
        profile=options.profile,
        device_id=options.device_id,
        trace_id=options.trace_id,
        session_id=options.session_id,
        protocol_version=options.protocol_version,
    )
    run_reports: list[dict[str, Any]] = []
    exit_codes: list[int] = []
    for index in range(options.repeat):
        exit_code, report = run_harness(options)
        report["run_index"] = index + 1
        run_reports.append(report)
        exit_codes.append(exit_code)
    aggregate = build_aggregate_report(
        base_report=base_report,
        run_reports=run_reports,
        exit_codes=exit_codes,
        abort_after_first_audio=options.abort_after_first_audio,
    )
    overall_exit_code = 0
    for code in exit_codes:
        if code != 0:
            overall_exit_code = code
            break
    return overall_exit_code, aggregate


def parse_args(argv: list[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="A21 virtual Xiaozhi WebSocket harness")
    parser.add_argument("--target-url", default=DEFAULT_TARGET_URL)
    parser.add_argument("--profile", choices=["xiaozhi", "a21-debug"], default="xiaozhi")
    parser.add_argument("--device-id", default=DEFAULT_DEVICE_ID)
    parser.add_argument("--trace-id", default=DEFAULT_TRACE_ID)
    parser.add_argument("--session-id", default=DEFAULT_SESSION_ID)
    parser.add_argument("--protocol-version", type=int, choices=[1, 2, 3], default=1)
    parser.add_argument("--expect-audio", action="store_true")
    parser.add_argument("--require-metrics", action="store_true")
    parser.add_argument("--abort-after-first-audio", action="store_true")
    parser.add_argument("--timeout-ms", type=int, default=2000)
    parser.add_argument("--json-report", action="store_true")
    parser.add_argument("--opus-fixture", help="Path to a raw Opus packet, or builtin_speech")
    parser.add_argument("--repeat", type=int, default=1, help="Run N sequential virtual turns")
    parser.add_argument("--self-test", action="store_true")
    return parser.parse_args(argv)


def main(argv: list[str]) -> int:
    args = parse_args(argv)
    if args.self_test:
        return run_self_test()
    if args.repeat < 1:
        print("repeat must be >= 1", file=sys.stderr)
        return 2
    options = HarnessOptions(
        target_url=args.target_url,
        profile=args.profile,
        device_id=args.device_id,
        trace_id=args.trace_id,
        session_id=args.session_id,
        protocol_version=args.protocol_version,
        expect_audio=args.expect_audio,
        require_metrics=args.require_metrics,
        abort_after_first_audio=args.abort_after_first_audio,
        timeout_ms=args.timeout_ms,
        json_report=args.json_report,
        opus_fixture=args.opus_fixture,
        repeat=args.repeat,
    )
    try:
        for field in ("device_id", "trace_id", "session_id"):
            reject_legacy_identity(getattr(options, field), field)
    except ValueError as exc:
        print(str(exc), file=sys.stderr)
        return 2
    exit_code, report = run_repeated_harness(options)
    if args.json_report:
        print(json.dumps(report, sort_keys=True, separators=(",", ":")))
    else:
        print(f"target={report['target']}")
        print(f"profile={report['profile']} protocol_version={report['protocol_version']}")
        print(f"runs={report['runs']} successful_runs={report['successful_runs']}")
        print(
            "hello_accepted={hello} listen_ack={listen} binary_downlink_frames={frames} "
            "first_audio_ms={first} first_audio_p95_ms={first_p95} "
            "tts_stop_received={stop} abort_sent={abort} abort_stop_ms={abort_stop} "
            "abort_stop_p95_ms={abort_p95} metrics_observed={metrics} "
            "host_candidate={host_candidate} prd_accepted={prd}".format(
                hello=report["hello_accepted"],
                listen=report["listen_ack"],
                frames=report["binary_downlink_frames"],
                first=report["first_audio_ms"],
                first_p95=report["first_audio_p95_ms"],
                stop=report["tts_stop_received"],
                abort=report["abort_sent"],
                abort_stop=report["abort_stop_ms"],
                abort_p95=report["abort_stop_p95_ms"],
                metrics=report["metrics_observed"],
                host_candidate=report["host_candidate"],
                prd=report["prd_accepted"],
            )
        )
        if "failure" in report:
            print(f"failure={report['failure']}", file=sys.stderr)
        if "error" in report:
            print(f"error={report['error']}", file=sys.stderr)
    return exit_code


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
