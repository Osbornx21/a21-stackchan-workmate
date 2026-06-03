#!/usr/bin/env python3
"""A21 DashScope CosyVoice TTS wrapper for the voice_clone_cli contract.

The wrapper receives A21's normalized voice_clone_cli arguments, calls
DashScope's CosyVoice WebSocket TTS API, and writes PCM16 mono WAV to --output.
It does not print prompt text, provider output, API keys, or URLs.
"""

from __future__ import annotations

import argparse
import json
import os
import sys
import uuid
import wave

try:
    import websocket
except Exception:  # pragma: no cover - runtime dependency guard
    websocket = None


DEFAULT_WS_URL = "wss://dashscope.aliyuncs.com/api-ws/v1/inference/"
DEFAULT_MODEL = "cosyvoice-v3-flash"
DEFAULT_VOICE = "longanyang"


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="A21 DashScope CosyVoice TTS wrapper")
    parser.add_argument("--text-file", required=True)
    parser.add_argument("--output", required=True)
    parser.add_argument("--sample-rate", type=int, default=24000)
    parser.add_argument("--ref-audio", default="")
    parser.add_argument("--ref-text-file", default="")
    parser.add_argument("--model", default="")
    parser.add_argument("--persona", default="")
    parser.add_argument("--style", default="")
    return parser.parse_args()


def env_first(*names: str) -> str:
    for name in names:
        value = os.environ.get(name, "").strip()
        if value:
            return value
    return ""


def normalize_model(value: str) -> str:
    value = (value or "").strip()
    if not value:
        return DEFAULT_MODEL
    if value.startswith("cosyvoice_"):
        value = value.replace("_", "-")
    return value


def send_json(ws, payload: dict) -> None:
    ws.send(json.dumps(payload, ensure_ascii=False))


def run_task(ws, task_id: str, model: str, voice: str, sample_rate: int) -> None:
    send_json(
        ws,
        {
            "header": {
                "action": "run-task",
                "task_id": task_id,
                "streaming": "duplex",
            },
            "payload": {
                "task_group": "audio",
                "task": "tts",
                "function": "SpeechSynthesizer",
                "model": model,
                "parameters": {
                    "text_type": "PlainText",
                    "voice": voice,
                    "format": "pcm",
                    "sample_rate": sample_rate,
                    "volume": 50,
                    "rate": 1,
                    "pitch": 1,
                    "enable_ssml": False,
                },
                "input": {},
            },
        },
    )


def continue_task(ws, task_id: str, text: str) -> None:
    send_json(
        ws,
        {
            "header": {
                "action": "continue-task",
                "task_id": task_id,
                "streaming": "duplex",
            },
            "payload": {"input": {"text": text}},
        },
    )


def finish_task(ws, task_id: str) -> None:
    send_json(
        ws,
        {
            "header": {
                "action": "finish-task",
                "task_id": task_id,
                "streaming": "duplex",
            },
            "payload": {"input": {}},
        },
    )


def write_wav(path: str, pcm: bytes, sample_rate: int) -> None:
    with wave.open(path, "wb") as wav:
        wav.setnchannels(1)
        wav.setsampwidth(2)
        wav.setframerate(sample_rate)
        wav.writeframes(pcm)


def main() -> int:
    args = parse_args()
    if websocket is None:
        print("dashscope_tts_dependency_missing", file=sys.stderr)
        return 3
    api_key = env_first("A21_DASHSCOPE_API_KEY", "A21_LAB_DASHSCOPE_API_KEY", "DASHSCOPE_API_KEY")
    if not api_key:
        print("dashscope_tts_api_key_missing", file=sys.stderr)
        return 3
    with open(args.text_file, "r", encoding="utf-8") as handle:
        text = handle.read().strip()
    if not text:
        print("dashscope_tts_text_missing", file=sys.stderr)
        return 2

    ws_url = env_first("A21_DASHSCOPE_TTS_URL", "TTS_BASE_URL") or DEFAULT_WS_URL
    model = env_first("A21_DASHSCOPE_TTS_MODEL", "TTS_MODEL") or normalize_model(args.model)
    voice = env_first("A21_DASHSCOPE_TTS_VOICE", "TTS_VOICE") or DEFAULT_VOICE
    sample_rate = args.sample_rate if args.sample_rate > 0 else 24000
    task_id = str(uuid.uuid4())
    pcm = bytearray()

    headers = [
        "Authorization: bearer " + api_key,
        "X-DashScope-DataInspection: enable",
        "User-Agent: a21-dashscope-cosyvoice-wrapper/0.1",
    ]
    try:
        ws = websocket.create_connection(ws_url, header=headers, timeout=20)
        try:
            run_task(ws, task_id, model, voice, sample_rate)
            started = False
            finish_sent = False
            while True:
                message = ws.recv()
                if isinstance(message, bytes):
                    pcm.extend(message)
                    continue
                try:
                    event = json.loads(message)
                except json.JSONDecodeError:
                    continue
                header = event.get("header", {})
                event_name = header.get("event", "")
                if event_name == "task-started" and not started:
                    started = True
                    continue_task(ws, task_id, text)
                    finish_task(ws, task_id)
                    finish_sent = True
                elif event_name == "task-failed":
                    print("dashscope_tts_task_failed", file=sys.stderr)
                    return 4
                elif event_name == "task-finished":
                    break
                if started and finish_sent and len(pcm) > 0 and event_name == "":
                    continue
        finally:
            ws.close()
    except Exception:
        print("dashscope_tts_websocket_failed", file=sys.stderr)
        return 4

    if not pcm:
        print("dashscope_tts_audio_empty", file=sys.stderr)
        return 4
    if len(pcm) % 2 != 0:
        pcm = pcm[:-1]
    write_wav(args.output, bytes(pcm), sample_rate)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
