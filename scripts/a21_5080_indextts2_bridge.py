#!/usr/bin/env python3
"""A21 local bridge for running warmed IndexTTS2 on the 5080 lab host.

The A21 Go `voice_clone_cli` wrapper passes text, reference audio, and
reference text as local files. This bridge copies those files to the 5080 host,
runs IndexTTS2 from the remote repository root, copies the generated WAV back
to A21's requested local output path, and prints only basename-level status.
"""

from __future__ import annotations

import argparse
import base64
import json
import os
import re
import secrets
import subprocess
import sys
import time
from pathlib import Path


REMOTE_RUNNER = r'''
from __future__ import annotations

import argparse
import json
from pathlib import Path

import torch
import torchaudio

from indextts.infer_v2 import IndexTTS2


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Run A21 remote IndexTTS2 inference")
    parser.add_argument("--text-file", required=True)
    parser.add_argument("--ref-audio", required=True)
    parser.add_argument("--output", required=True)
    parser.add_argument("--sample-rate", type=int, required=True)
    parser.add_argument("--model-dir", required=True)
    parser.add_argument("--use-cuda-kernel", action="store_true")
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    text = Path(args.text_file).read_text(encoding="utf-8").strip()
    if not text:
        raise RuntimeError("text is required")
    output = Path(args.output)
    output.parent.mkdir(parents=True, exist_ok=True)
    raw_output = output.with_name(output.stem + ".raw.wav")

    checkpoint_dir = Path(args.model_dir).resolve()
    tts = IndexTTS2(
        cfg_path=str(checkpoint_dir / "config.yaml"),
        model_dir=str(checkpoint_dir),
        use_cuda_kernel=args.use_cuda_kernel,
    )
    tts.infer(
        spk_audio_prompt=str(Path(args.ref_audio)),
        text=text,
        output_path=str(raw_output),
        verbose=False,
    )

    waveform, source_rate = torchaudio.load(str(raw_output))
    target_rate = args.sample_rate
    if source_rate != target_rate:
        waveform = torchaudio.functional.resample(waveform, source_rate, target_rate)
    waveform = torch.clamp(waveform, -1.0, 1.0)
    torchaudio.save(
        str(output),
        waveform.cpu(),
        target_rate,
        encoding="PCM_S",
        bits_per_sample=16,
    )

    duration_ms = waveform.shape[-1] / target_rate * 1000 if target_rate > 0 else 0
    print(json.dumps({
        "status": "passed",
        "sample_rate_hz": target_rate,
        "duration_ms": round(duration_ms, 3),
    }, ensure_ascii=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
'''


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Bridge A21 voice_clone_cli to 5080 IndexTTS2")
    parser.add_argument("--dry-run", action="store_true")
    parser.add_argument("--ssh-host", default=os.environ.get("A21_5080_SSH_HOST", ""))
    parser.add_argument("--ssh-key", default=os.environ.get("A21_5080_SSH_KEY", ""))
    parser.add_argument("--ssh-bin", default=os.environ.get("A21_5080_SSH_BIN", "ssh"))
    parser.add_argument("--scp-bin", default=os.environ.get("A21_5080_SCP_BIN", "scp"))
    parser.add_argument("--remote-repo", default=os.environ.get("A21_5080_INDEXTTS2_REPO", ""))
    parser.add_argument("--remote-model-dir", default=os.environ.get("A21_5080_INDEXTTS2_MODEL_DIR", ""))
    parser.add_argument("--remote-work-root", default=os.environ.get("A21_5080_WORK_ROOT", ""))
    parser.add_argument(
        "--remote-python",
        default=os.environ.get("A21_5080_INDEXTTS2_PYTHON", ".venv/Scripts/python.exe"),
    )
    parser.add_argument("--hf-endpoint", default=os.environ.get("HF_ENDPOINT", ""))
    parser.add_argument("--connect-timeout", default=os.environ.get("A21_5080_SSH_CONNECT_TIMEOUT", "20"))
    parser.add_argument("--keep-remote", action="store_true")
    parser.add_argument(
        "--use-cuda-kernel",
        action="store_true",
        default=os.environ.get("A21_5080_INDEXTTS2_USE_CUDA_KERNEL", "").lower()
        in {"1", "true", "yes"},
    )

    parser.add_argument("--text-file", required=True)
    parser.add_argument("--output", required=True)
    parser.add_argument("--sample-rate", type=int, required=True)
    parser.add_argument("--ref-audio", required=True)
    parser.add_argument("--ref-text-file", required=True)
    parser.add_argument("--model", default="index_tts2")
    parser.add_argument("--persona", default="a21_workmate")
    parser.add_argument("--style", default="workmate_warm")
    return parser.parse_args()


def require_config(args: argparse.Namespace) -> None:
    missing = []
    for name in ("ssh_host", "ssh_key", "remote_repo", "remote_model_dir", "remote_work_root"):
        if not str(getattr(args, name)).strip():
            missing.append(name.replace("_", "-"))
    if missing:
        raise RuntimeError("bridge configuration is missing: " + ", ".join(missing))
    if args.sample_rate <= 0:
        raise RuntimeError("sample rate must be positive")
    for field in ("text_file", "ref_audio", "ref_text_file"):
        if not Path(getattr(args, field)).is_file():
            raise RuntimeError(field.replace("_", "-") + " is unavailable")


def basename(path: str) -> str:
    return Path(str(path).replace("\\", "/")).name


def normalize_remote_path(path: str) -> str:
    return str(path).replace("\\", "/").rstrip("/")


def remote_join(*parts: str) -> str:
    cleaned = [normalize_remote_path(part).strip("/") for part in parts if str(part).strip()]
    if not cleaned:
        return ""
    first = cleaned[0]
    rest = cleaned[1:]
    return "/".join([first, *rest])


def remote_spec(host: str, path: str) -> str:
    return f"{host}:{path}"


def ssh_base(args: argparse.Namespace) -> list[str]:
    return [
        args.ssh_bin,
        "-i",
        args.ssh_key,
        "-o",
        "BatchMode=yes",
        "-o",
        f"ConnectTimeout={args.connect_timeout}",
        args.ssh_host,
    ]


def scp_base(args: argparse.Namespace) -> list[str]:
    return [
        args.scp_bin,
        "-i",
        args.ssh_key,
        "-o",
        "BatchMode=yes",
        "-o",
        f"ConnectTimeout={args.connect_timeout}",
    ]


def powershell_quote(value: str) -> str:
    return "'" + value.replace("'", "''") + "'"


def encoded_powershell(script: str) -> str:
    return base64.b64encode(script.encode("utf-16le")).decode("ascii")


def run_checked(command: list[str], step: str) -> None:
    result = subprocess.run(command, text=True, capture_output=True)
    if result.returncode != 0:
        detail = redact_process_diagnostic(result.stderr or result.stdout)
        if detail:
            raise RuntimeError(f"{step} failed exit={result.returncode}: {detail}")
        raise RuntimeError(f"{step} failed exit={result.returncode}")


def redact_process_diagnostic(value: str) -> str:
    value = (value or "").strip()
    if not value:
        return ""
    value = value[-1200:]
    value = re.sub(r"(?i)(authorization|bearer)\s*[:= ]\s*\S+", r"\1 [secret]", value)
    value = re.sub(r"(?i)(sk|ak|key|token|secret)[-_A-Za-z0-9]*", "[secret]", value)
    value = re.sub(r"\b\d{1,3}(?:\.\d{1,3}){3}\b", "[host]", value)
    value = re.sub(r"[A-Za-z]:[\\/][^\s'\"<>|]+", "[path]", value)
    value = re.sub(r"/(?:Users|var|tmp|private|Volumes)/[^\s'\"<>|]+", "[path]", value)
    return value


def remote_shell(args: argparse.Namespace, script: str, step: str) -> None:
    command = [
        *ssh_base(args),
        "powershell",
        "-NoProfile",
        "-ExecutionPolicy",
        "Bypass",
        "-EncodedCommand",
        encoded_powershell(script),
    ]
    run_checked(command, step)


def build_plan(args: argparse.Namespace) -> dict[str, object]:
    run_id = "a21-indextts2-" + time.strftime("%Y%m%d-%H%M%S") + "-" + secrets.token_hex(4)
    remote_dir = remote_join(args.remote_work_root, run_id)
    return {
        "run_id": run_id,
        "remote_dir": remote_dir,
        "remote_text": remote_join(remote_dir, "a21-text.txt"),
        "remote_ref_audio": remote_join(remote_dir, basename(args.ref_audio)),
        "remote_ref_text": remote_join(remote_dir, "a21-reference.txt"),
        "remote_runner": remote_join(remote_dir, "a21_indextts2_runner.py"),
        "remote_output": remote_join(remote_dir, basename(args.output)),
    }


def dry_run_report(args: argparse.Namespace, plan: dict[str, object]) -> dict[str, object]:
    return {
        "status": "dry_run",
        "provider": "voice_clone_cli",
        "engine": "index_tts2_5080_bridge",
        "model": args.model,
        "voice_persona": args.persona,
        "style_profile": args.style,
        "local_output": basename(args.output),
        "remote_repo": basename(args.remote_repo),
        "remote_model_dir": basename(args.remote_model_dir),
        "remote_output": basename(str(plan["remote_output"])),
        "sample_rate_hz": args.sample_rate,
        "steps": [
            "mkdir_remote",
            "scp_inputs",
            "run_indextts2_from_repo_root",
            "scp_output_back",
        ],
    }


def create_remote_dir(args: argparse.Namespace, plan: dict[str, object]) -> None:
    script = (
        "$ProgressPreference='SilentlyContinue'\n"
        "$ErrorActionPreference='Stop'\n"
        f"New-Item -ItemType Directory -Force -Path {powershell_quote(str(plan['remote_dir']))} | Out-Null\n"
    )
    remote_shell(args, script, "remote setup")


def copy_inputs(args: argparse.Namespace, plan: dict[str, object]) -> None:
    pairs = (
        (args.text_file, str(plan["remote_text"])),
        (args.ref_audio, str(plan["remote_ref_audio"])),
        (args.ref_text_file, str(plan["remote_ref_text"])),
    )
    for local_path, remote_path in pairs:
        run_checked(
            [*scp_base(args), local_path, remote_spec(args.ssh_host, remote_path)],
            "input copy",
        )


def write_remote_runner(args: argparse.Namespace, plan: dict[str, object]) -> None:
    script = (
        "$ProgressPreference='SilentlyContinue'\n"
        "$ErrorActionPreference='Stop'\n"
        f"Set-Content -Path {powershell_quote(str(plan['remote_runner']))} "
        f"-Value {powershell_quote(REMOTE_RUNNER)} -Encoding UTF8\n"
    )
    remote_shell(args, script, "remote runner write")


def run_remote_indextts2(args: argparse.Namespace, plan: dict[str, object]) -> None:
    env_lines = [
        "$ProgressPreference='SilentlyContinue'",
        "$ErrorActionPreference='Stop'",
        "$env:PYTHONIOENCODING='utf-8'",
    ]
    if args.hf_endpoint:
        env_lines.append(f"$env:HF_ENDPOINT={powershell_quote(args.hf_endpoint)}")
    command_parts = [
        f"Set-Location -LiteralPath {powershell_quote(args.remote_repo)}",
        (
            f"& {powershell_quote(args.remote_python)} "
            f"{powershell_quote(str(plan['remote_runner']))} "
            f"--text-file {powershell_quote(str(plan['remote_text']))} "
            f"--ref-audio {powershell_quote(str(plan['remote_ref_audio']))} "
            f"--output {powershell_quote(str(plan['remote_output']))} "
            f"--model-dir {powershell_quote(args.remote_model_dir)} "
            f"--sample-rate {int(args.sample_rate)}"
            + (" --use-cuda-kernel" if args.use_cuda_kernel else "")
        ),
    ]
    remote_shell(args, "\n".join([*env_lines, *command_parts]) + "\n", "remote IndexTTS2 run")


def copy_output(args: argparse.Namespace, plan: dict[str, object]) -> None:
    output = Path(args.output)
    output.parent.mkdir(parents=True, exist_ok=True)
    run_checked(
        [*scp_base(args), remote_spec(args.ssh_host, str(plan["remote_output"])), str(output)],
        "output copy",
    )


def cleanup_remote(args: argparse.Namespace, plan: dict[str, object]) -> None:
    if args.keep_remote:
        return
    script = (
        "$ProgressPreference='SilentlyContinue'\n"
        "$ErrorActionPreference='SilentlyContinue'\n"
        f"Remove-Item -Recurse -Force -Path {powershell_quote(str(plan['remote_dir']))}\n"
    )
    try:
        remote_shell(args, script, "remote cleanup")
    except RuntimeError:
        pass


def run_bridge(args: argparse.Namespace, plan: dict[str, object]) -> None:
    create_remote_dir(args, plan)
    try:
        copy_inputs(args, plan)
        write_remote_runner(args, plan)
        run_remote_indextts2(args, plan)
        copy_output(args, plan)
    finally:
        cleanup_remote(args, plan)


def success_report(args: argparse.Namespace) -> dict[str, object]:
    output = Path(args.output)
    return {
        "status": "passed",
        "provider": "voice_clone_cli",
        "engine": "index_tts2_5080_bridge",
        "model": args.model,
        "voice_persona": args.persona,
        "style_profile": args.style,
        "local_output": output.name,
        "output_bytes": output.stat().st_size if output.exists() else 0,
    }


def main() -> int:
    args = parse_args()
    try:
        require_config(args)
        plan = build_plan(args)
        if args.dry_run:
            json.dump(dry_run_report(args, plan), sys.stdout, ensure_ascii=True)
            sys.stdout.write("\n")
            return 0
        run_bridge(args, plan)
        json.dump(success_report(args), sys.stdout, ensure_ascii=True)
        sys.stdout.write("\n")
        return 0
    except Exception as exc:
        print(f"a21 5080 IndexTTS2 bridge failed: {exc}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
