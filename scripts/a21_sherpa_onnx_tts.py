#!/usr/bin/env python3
"""A21 isolated sherpa-onnx VITS TTS runner.

The Go wrapper passes user text through a private temp file so the text does
not appear in the process command line or A21 evidence reports.
"""

from __future__ import annotations

import argparse
import json
import sys
import time
from pathlib import Path

import sherpa_onnx


REQUIRED_MODEL_FILES = (
    "model.onnx",
    "lexicon.txt",
    "tokens.txt",
    "phone.fst",
    "date.fst",
    "number.fst",
)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Run A21 sherpa-onnx local TTS")
    parser.add_argument("--model-dir", required=True)
    parser.add_argument("--text-file", required=True)
    parser.add_argument("--output", required=True)
    parser.add_argument("--speaker-id", type=int, default=21)
    parser.add_argument("--speed", type=float, default=1.0)
    parser.add_argument("--num-threads", type=int, default=1)
    return parser.parse_args()


def validate_model_dir(model_dir: Path) -> None:
    if not model_dir.is_dir():
        raise RuntimeError("model dir is unavailable")
    missing = [name for name in REQUIRED_MODEL_FILES if not (model_dir / name).is_file()]
    if missing:
        raise RuntimeError("model files are missing")


def main() -> int:
    args = parse_args()
    model_dir = Path(args.model_dir)
    text_file = Path(args.text_file)
    output = Path(args.output)
    validate_model_dir(model_dir)
    text = text_file.read_text(encoding="utf-8").strip()
    if not text:
        raise RuntimeError("text is required")

    started = time.perf_counter()
    config = sherpa_onnx.OfflineTtsConfig(
        model=sherpa_onnx.OfflineTtsModelConfig(
            vits=sherpa_onnx.OfflineTtsVitsModelConfig(
                model=str(model_dir / "model.onnx"),
                lexicon=str(model_dir / "lexicon.txt"),
                tokens=str(model_dir / "tokens.txt"),
            ),
            provider="cpu",
            debug=False,
            num_threads=args.num_threads,
        ),
        rule_fsts=",".join(
            str(model_dir / name) for name in ("phone.fst", "date.fst", "number.fst")
        ),
        max_num_sentences=1,
    )
    if not config.validate():
        raise RuntimeError("invalid sherpa-onnx TTS config")

    tts = sherpa_onnx.OfflineTts(config)
    generation = sherpa_onnx.GenerationConfig()
    generation.sid = args.speaker_id
    generation.speed = args.speed
    generation.silence_scale = 0.2

    audio = tts.generate(text, generation)
    if getattr(audio, "samples", None) is None or len(audio.samples) == 0:
        raise RuntimeError("sherpa-onnx produced empty audio")
    output.parent.mkdir(parents=True, exist_ok=True)
    sherpa_onnx.write_wave(str(output), audio.samples, audio.sample_rate)

    elapsed_ms = (time.perf_counter() - started) * 1000
    json.dump(
        {
            "status": "ok",
            "engine": "vits_icefall_zh_aishell3",
            "sample_rate": audio.sample_rate,
            "duration_ms": round(elapsed_ms, 3),
        },
        sys.stdout,
        ensure_ascii=True,
    )
    sys.stdout.write("\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
