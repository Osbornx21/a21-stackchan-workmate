#!/usr/bin/env python3
"""Tests for the A21 5080 IndexTTS2 bridge wrapper."""

from __future__ import annotations

import json
import importlib.util
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path


SCRIPT = Path(__file__).with_name("a21_5080_indextts2_bridge.py")


def load_bridge_module():
    spec = importlib.util.spec_from_file_location("a21_5080_indextts2_bridge", SCRIPT)
    if spec is None or spec.loader is None:
        raise RuntimeError("bridge module is unavailable")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


class IndexTTS2BridgeDryRunTest(unittest.TestCase):
    def test_dry_run_uses_remote_output_and_redacts_inputs(self) -> None:
        with tempfile.TemporaryDirectory(prefix="a21-bridge-test-") as tmp:
            root = Path(tmp)
            text_file = root / "text.txt"
            ref_text_file = root / "reference.txt"
            ref_audio = root / "reference.wav"
            output = root / "a21-output.wav"
            key_path = root / "a21_5080_secret_ed25519"
            text_file.write_text("用户文本不能进入计划输出", encoding="utf-8")
            ref_text_file.write_text("参考文本不能进入计划输出", encoding="utf-8")
            ref_audio.write_bytes(b"RIFF-a21-reference")
            key_path.write_text("not-a-real-key", encoding="utf-8")

            result = subprocess.run(
                [
                    sys.executable,
                    str(SCRIPT),
                    "--dry-run",
                    "--ssh-host",
                    "21@192.168.1.6",
                    "--ssh-key",
                    str(key_path),
                    "--remote-repo",
                    "D:/a21-mainland-latency-lab/cache/git/index-tts",
                    "--remote-model-dir",
                    "D:/a21-model-cache/modelscope/IndexTeam/IndexTTS-2",
                    "--remote-work-root",
                    "D:/a21-mainland-latency-lab/tmp/a21-indextts2",
                    "--text-file",
                    str(text_file),
                    "--output",
                    str(output),
                    "--sample-rate",
                    "16000",
                    "--ref-audio",
                    str(ref_audio),
                    "--ref-text-file",
                    str(ref_text_file),
                    "--model",
                    "index_tts2",
                    "--persona",
                    "a21_workmate",
                    "--style",
                    "workmate_warm",
                ],
                check=True,
                text=True,
                capture_output=True,
            )

        rendered = result.stdout
        plan = json.loads(rendered)
        self.assertEqual(plan["status"], "dry_run")
        self.assertEqual(plan["local_output"], "a21-output.wav")
        self.assertEqual(plan["remote_repo"], "index-tts")
        self.assertEqual(plan["remote_model_dir"], "IndexTTS-2")
        self.assertEqual(plan["remote_output"], "a21-output.wav")
        self.assertIn("scp_inputs", plan["steps"])
        self.assertIn("run_indextts2_from_repo_root", plan["steps"])
        self.assertIn("scp_output_back", plan["steps"])
        for forbidden in (
            "用户文本",
            "参考文本",
            "a21_5080_secret_ed25519",
            "21@192.168.1.6",
            tmp,
            str(output),
        ):
            self.assertNotIn(forbidden, rendered)

    def test_command_failure_redacts_paths_and_command_output(self) -> None:
        with tempfile.TemporaryDirectory(prefix="a21-bridge-test-") as tmp:
            root = Path(tmp)
            fake = root / "fake_command.py"
            fake.write_text(
                "\n".join(
                    [
                        "import sys",
                        "print('stdout /Users/local/secret prompt text')",
                        "print('stderr D:/remote/secret key-token', file=sys.stderr)",
                        "sys.exit(7)",
                    ]
                ),
                encoding="utf-8",
            )
            bridge = load_bridge_module()
            with self.assertRaises(RuntimeError) as raised:
                bridge.run_checked([sys.executable, str(fake)], "fixture step")

        rendered = str(raised.exception)
        self.assertIn("fixture step failed", rendered)
        self.assertIn("exit=7", rendered)
        self.assertIn("[path]", rendered)
        for forbidden in (tmp, "/Users/local/secret", "D:/remote/secret", "key-token"):
            self.assertNotIn(forbidden, rendered)


if __name__ == "__main__":
    unittest.main()
