#!/usr/bin/env python3
import importlib.util
import json
import pathlib
import sys
import unittest


MODULE_PATH = pathlib.Path(__file__).with_name("a21_virtual_xiaozhi.py")
SPEC = importlib.util.spec_from_file_location("a21_virtual_xiaozhi", MODULE_PATH)
a21_virtual_xiaozhi = importlib.util.module_from_spec(SPEC)
sys.modules["a21_virtual_xiaozhi"] = a21_virtual_xiaozhi
SPEC.loader.exec_module(a21_virtual_xiaozhi)


class A21VirtualXiaozhiPayloadTests(unittest.TestCase):
    def test_stock_profile_hello_has_no_debug_features(self):
        payload = a21_virtual_xiaozhi.build_hello_payload(
            profile="xiaozhi",
            device_id="stackchan-harness-001",
            trace_id="a21-trace-harness",
            session_id="a21-session-harness",
            protocol_version=3,
        )

        self.assertEqual(payload["type"], "hello")
        self.assertEqual(payload["version"], 3)
        self.assertEqual(payload["transport"], "websocket")
        self.assertEqual(payload["device_id"], "stackchan-harness-001")
        self.assertEqual(payload["trace_id"], "a21-trace-harness")
        self.assertEqual(payload["session_id"], "a21-session-harness")
        self.assertEqual(payload["features"], {"mcp": True, "aec": True})
        self.assertEqual(
            payload["audio_params"],
            {
                "format": "opus",
                "sample_rate": 16000,
                "channels": 1,
                "frame_duration": 60,
                "binary_protocol_version": 3,
            },
        )
        rendered = json.dumps(payload, sort_keys=True)
        self.assertNotIn("device_events", rendered)
        self.assertNotIn("debug_metrics", rendered)
        self.assertNotIn("x21", rendered.lower())

    def test_debug_profile_keeps_extension_isolated(self):
        payload = a21_virtual_xiaozhi.build_hello_payload(
            profile="a21-debug",
            device_id="stackchan-debug-001",
            trace_id="a21-trace-debug",
            session_id="a21-session-debug",
            protocol_version=2,
        )

        self.assertEqual(
            payload["features"],
            {
                "mcp": True,
                "aec": True,
                "device_events": True,
                "debug_metrics": True,
            },
        )
        self.assertEqual(payload["audio_params"]["binary_protocol_version"], 2)

    def test_report_redacts_target_and_never_promotes_prd_acceptance(self):
        report = a21_virtual_xiaozhi.build_report(
            target_url="ws://127.0.0.1:21080/v1/xiaozhi?token=secret",
            profile="xiaozhi",
            device_id="stackchan-harness-001",
            trace_id="a21-trace-harness",
            session_id="a21-session-harness",
            protocol_version=1,
        )

        self.assertEqual(report["target"], "127.0.0.1:21080/v1/xiaozhi")
        self.assertFalse(report["prd_accepted"])
        rendered = json.dumps(report, sort_keys=True)
        self.assertNotIn("secret", rendered)
        self.assertNotIn("token", rendered)
        self.assertNotIn("/Users/", rendered)

    def test_expect_audio_uses_builtin_opus_fixture_without_local_path(self):
        data, label = a21_virtual_xiaozhi.resolve_opus_fixture(None, expect_audio=True)

        self.assertGreater(len(data), 0)
        self.assertEqual(label, "builtin_speech")
        self.assertNotIn("/", label)


if __name__ == "__main__":
    unittest.main()
