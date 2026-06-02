#!/usr/bin/env python3
"""A21 sherpa-onnx streaming ASR JSONL helper.

The helper speaks newline-delimited JSON on stdin/stdout. It intentionally
keeps stderr quiet and emits only stable error codes; transcripts are returned
to the parent process as ASR events and must be redacted by reports upstream.
"""

from __future__ import annotations

import base64
import json
import os
import struct
import sys
from pathlib import Path
from typing import Any


def emit(event: dict[str, Any]) -> None:
    sys.stdout.write(json.dumps(event, ensure_ascii=True, separators=(",", ":")) + "\n")
    sys.stdout.flush()


def error(code: str) -> int:
    emit({"type": "error", "code": code})
    return 1


def pcm16le_to_float_samples(payload: bytes) -> list[float]:
    if not payload or len(payload) % 2:
        raise ValueError("invalid_pcm")
    values = struct.unpack("<%dh" % (len(payload) // 2), payload)
    return [value / 32768.0 for value in values]


class FakeRecognizer:
    def __init__(self) -> None:
        self.append_count = 0

    def start(self, command: dict[str, Any]) -> bool:
        emit({"type": "ready"})
        return True

    def append(self, command: dict[str, Any]) -> None:
        self.append_count += 1
        emit({"type": "partial", "text": "partial"})

    def commit(self) -> None:
        emit({"type": "final", "text": "final"})


class StreamingZipformerRecognizer:
    def __init__(self, model_dir: str) -> None:
        try:
            from sherpa_onnx.online_recognizer import OnlineRecognizer
        except Exception as exc:  # pragma: no cover - depends on local model env
            raise RuntimeError("sherpa_onnx_unavailable") from exc

        root = Path(model_dir)
        required = [
            root / "tokens.txt",
            root / "encoder.int8.onnx",
            root / "decoder.onnx",
            root / "joiner.int8.onnx",
        ]
        if not all(path.exists() for path in required):
            raise RuntimeError("model_files_missing")
        self.recognizer = OnlineRecognizer.from_transducer(
            str(root / "tokens.txt"),
            str(root / "encoder.int8.onnx"),
            str(root / "decoder.onnx"),
            str(root / "joiner.int8.onnx"),
            num_threads=1,
        )
        self.stream = self.recognizer.create_stream()
        self.last_partial = ""

    def start(self, command: dict[str, Any]) -> bool:
        emit({"type": "ready"})
        return True

    def append(self, command: dict[str, Any]) -> None:
        sample_rate = int(command.get("sample_rate_hz") or 16000)
        payload = base64.b64decode(str(command.get("pcm16le_b64") or ""), validate=True)
        samples = pcm16le_to_float_samples(payload)
        self.stream.accept_waveform(sample_rate, samples)
        while self.recognizer.is_ready(self.stream):
            self.recognizer.decode_stream(self.stream)
        text = self.recognizer.get_result(self.stream).strip()
        if text and text != self.last_partial:
            self.last_partial = text
            emit({"type": "partial", "text": text})

    def commit(self) -> None:
        self.stream.input_finished()
        while self.recognizer.is_ready(self.stream):
            self.recognizer.decode_stream(self.stream)
        text = self.recognizer.get_result(self.stream).strip()
        emit({"type": "final", "text": text})


def recognizer_for_start(command: dict[str, Any]):
    if os.environ.get("A21_SHERPA_STREAMING_ASR_FAKE") == "1":
        return FakeRecognizer()
    family = str(command.get("family") or "streaming_zipformer").strip()
    if family != "streaming_zipformer":
        raise RuntimeError("unsupported_streaming_family")
    model_dir = str(command.get("model_dir") or "").strip()
    if not model_dir:
        raise RuntimeError("model_dir_missing")
    return StreamingZipformerRecognizer(model_dir)


def main() -> int:
    recognizer = None
    for raw in sys.stdin:
        try:
            command = json.loads(raw)
            command_type = str(command.get("type") or "").strip().lower()
            if command_type == "start":
                recognizer = recognizer_for_start(command)
                recognizer.start(command)
            elif command_type == "append":
                if recognizer is None:
                    return error("session_not_started")
                recognizer.append(command)
            elif command_type == "commit":
                if recognizer is None:
                    return error("session_not_started")
                recognizer.commit()
                return 0
            elif command_type == "cancel":
                return 0
            else:
                return error("unknown_command")
        except Exception as exc:
            code = str(exc) if str(exc) else "sherpa_streaming_helper_failed"
            stable = {
                "invalid_pcm",
                "model_dir_missing",
                "model_files_missing",
                "session_not_started",
                "sherpa_onnx_unavailable",
                "unsupported_streaming_family",
                "unknown_command",
            }
            if code not in stable:
                code = "sherpa_streaming_helper_failed"
            return error(code)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
