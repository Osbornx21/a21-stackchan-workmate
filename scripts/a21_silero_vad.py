#!/usr/bin/env python3
"""A21 optional Silero VAD command runner.

Reads PCM16LE from stdin and writes a compact JSON VAD decision to stdout.
The runner intentionally does not download models or print local paths.
"""

from __future__ import annotations

import argparse
import json
import os
import sys
from typing import Sequence


UNAVAILABLE_MESSAGE = "a21 silero vad unavailable"
SUPPORTED_SAMPLE_RATES = {8000, 16000}


def main(argv: Sequence[str] | None = None) -> int:
    parser = build_parser()
    args = parser.parse_args(argv)

    if args.sample_rate not in SUPPORTED_SAMPLE_RATES:
        parser.error("--sample-rate must be 8000 or 16000")
    if args.channels <= 0:
        parser.error("--channels must be positive")
    if args.duration_ms <= 0:
        parser.error("--duration-ms must be positive")

    pcm = sys.stdin.buffer.read()
    expected_bytes = args.sample_rate * args.duration_ms * args.channels * 2 // 1000
    if len(pcm) != expected_bytes:
        parser.error("stdin PCM byte length does not match sample metadata")

    if not args.model or not os.path.isfile(args.model):
        return unavailable()

    try:
        import numpy as np
        import onnxruntime as ort
    except Exception:
        return unavailable()

    try:
        score = detect_silero_score(
            pcm=pcm,
            sample_rate=args.sample_rate,
            channels=args.channels,
            model=args.model,
            np=np,
            ort=ort,
        )
    except Exception:
        return unavailable()

    print(json.dumps({"speech_detected": score >= args.threshold, "score": score}, separators=(",", ":")))
    return 0


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description="Run optional A21 Silero VAD over one PCM16LE frame from stdin.",
    )
    parser.add_argument("--sample-rate", type=int, required=True, help="PCM sample rate: 8000 or 16000 Hz")
    parser.add_argument("--channels", type=int, required=True, help="PCM channel count")
    parser.add_argument("--duration-ms", type=int, required=True, help="PCM frame duration in milliseconds")
    parser.add_argument("--model", help="Optional local ONNX model file")
    parser.add_argument("--threshold", type=float, default=0.5, help="Speech score threshold")
    return parser


def unavailable() -> int:
    print(UNAVAILABLE_MESSAGE, file=sys.stderr)
    return 2


def detect_silero_score(*, pcm: bytes, sample_rate: int, channels: int, model: str, np, ort) -> float:
    samples = np.frombuffer(pcm, dtype=np.int16).astype(np.float32) / 32768.0
    if channels > 1:
        samples = samples.reshape((-1, channels)).mean(axis=1)
    samples = samples.reshape(1, -1)

    session = ort.InferenceSession(model, providers=["CPUExecutionProvider"])
    input_names = {item.name for item in session.get_inputs()}
    inputs = {}
    if "input" in input_names:
        inputs["input"] = samples
    elif input_names:
        first_audio_input = session.get_inputs()[0].name
        inputs[first_audio_input] = samples

    if "sr" in input_names:
        inputs["sr"] = np.array(sample_rate, dtype=np.int64)
    if "state" in input_names:
        inputs["state"] = np.zeros((2, 1, 128), dtype=np.float32)

    outputs = session.run(None, inputs)
    if not outputs:
        raise RuntimeError("empty silero output")
    return max(0.0, min(1.0, float(np.asarray(outputs[0]).reshape(-1)[0])))


if __name__ == "__main__":
    raise SystemExit(main())
