#!/usr/bin/env python3
"""A21 isolated sherpa-onnx ASR smoke runner.

This runner decodes a local WAV fixture through sherpa-onnx and reports timing
and transcript length only. It intentionally never prints transcript text.
"""

from __future__ import annotations

import argparse
import json
import struct
import sys
import time
import wave
from pathlib import Path

from sherpa_onnx.offline_recognizer import OfflineRecognizer
from sherpa_onnx.online_recognizer import OnlineRecognizer


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Run A21 sherpa-onnx ASR smoke")
    parser.add_argument("--family", required=True, choices=("paraformer", "sense_voice", "streaming_zipformer"))
    parser.add_argument("--model-dir", required=True)
    parser.add_argument("--wav", required=True)
    parser.add_argument("--transcript-output")
    parser.add_argument("--num-threads", type=int, default=1)
    return parser.parse_args()


def read_wav(path: Path) -> tuple[int, list[float]]:
    with wave.open(str(path), "rb") as wav:
        sample_rate = wav.getframerate()
        channels = wav.getnchannels()
        sample_width = wav.getsampwidth()
        frames = wav.getnframes()
        data = wav.readframes(frames)
    if sample_width != 2:
        raise RuntimeError("only 16-bit PCM WAV is supported")
    values = struct.unpack("<%dh" % (len(data) // 2), data)
    if channels > 1:
        values = values[::channels]
    return sample_rate, [value / 32768.0 for value in values]


def decode_offline(args: argparse.Namespace, sample_rate: int, samples: list[float]) -> str:
    model_dir = Path(args.model_dir)
    if args.family == "paraformer":
        recognizer = OfflineRecognizer.from_paraformer(
            str(model_dir / "model.int8.onnx"),
            str(model_dir / "tokens.txt"),
            num_threads=args.num_threads,
        )
    elif args.family == "sense_voice":
        recognizer = OfflineRecognizer.from_sense_voice(
            str(model_dir / "model.int8.onnx"),
            str(model_dir / "tokens.txt"),
            num_threads=args.num_threads,
            language="zh",
            use_itn=True,
        )
    else:
        raise RuntimeError(f"unsupported offline family {args.family}")
    stream = recognizer.create_stream()
    stream.accept_waveform(sample_rate, samples)
    recognizer.decode_stream(stream)
    return stream.result.text.strip()


def decode_streaming_zipformer(args: argparse.Namespace, sample_rate: int, samples: list[float]) -> str:
    model_dir = Path(args.model_dir)
    recognizer = OnlineRecognizer.from_transducer(
        str(model_dir / "tokens.txt"),
        str(model_dir / "encoder.int8.onnx"),
        str(model_dir / "decoder.onnx"),
        str(model_dir / "joiner.int8.onnx"),
        num_threads=args.num_threads,
    )
    stream = recognizer.create_stream()
    chunk = max(1, int(sample_rate * 0.1))
    for start in range(0, len(samples), chunk):
        stream.accept_waveform(sample_rate, samples[start : start + chunk])
        while recognizer.is_ready(stream):
            recognizer.decode_stream(stream)
    stream.input_finished()
    while recognizer.is_ready(stream):
        recognizer.decode_stream(stream)
    return recognizer.get_result(stream).strip()


def main() -> int:
    args = parse_args()
    wav_path = Path(args.wav)
    started = time.perf_counter()
    sample_rate, samples = read_wav(wav_path)
    if not samples:
        raise RuntimeError("input WAV contains no samples")

    if args.family == "streaming_zipformer":
        text = decode_streaming_zipformer(args, sample_rate, samples)
    else:
        text = decode_offline(args, sample_rate, samples)
    if args.transcript_output:
        Path(args.transcript_output).write_text(text, encoding="utf-8")
    elapsed_ms = (time.perf_counter() - started) * 1000
    input_ms = len(samples) / sample_rate * 1000
    report = {
        "status": "passed",
        "input_duration_ms": round(input_ms, 3),
        "decode_duration_ms": round(elapsed_ms, 3),
        "real_time_factor": round(elapsed_ms / input_ms, 6),
        "text_chars": len(text),
    }
    json.dump(report, sys.stdout, ensure_ascii=True)
    sys.stdout.write("\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
