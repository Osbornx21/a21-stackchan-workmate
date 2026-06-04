# A21 Voice Clone TTS Contract

Status: host-local integration contract.

A21 uses `voice_clone_cli` as the product-facing voice-clone TTS seam. The Gateway and voice pipeline do not import a model repository directly; they call an A21-owned wrapper command that can target a local model such as IndexTTS2, CosyVoice, F5-TTS, or GPT-SoVITS.

This capability is separate from the rejected old local contest voice. During
`T-PROVIDER-002`, Iflytek TTS may be selected as the immediate fast real-time
dialogue source, but `voice_clone_cli` remains the retained A21 path for cloned
or persona voices. CosyVoice is a valid future ordinary local fallback and
voice-clone model candidate after a dedicated bakeoff; it must not be silently
substituted for the active contest TTS without evidence.

## Model Priority

1. `index_tts2`: preferred A21 personality voice target because it exposes voice cloning plus style/emotion controls.
2. `cosyvoice3` or `cosyvoice2`: Chinese naturalness and streaming-oriented fallback.
3. `f5_tts`: quick zero-shot clone baseline for local bakeoff and fallback.
4. `gpt_sovits`: fine-tune lane when a short reference sample is not enough.

The active model is selected by `A21_VOICE_CLONE_MODEL`; model installation and licenses must be verified by the lab before promotion.

## Environment

```bash
A21_TTS_FAST_PROFILE=voice_clone_cli
A21_LOCAL_TTS_ENGINE=voice_clone_cli
A21_VOICE_CLONE_COMMAND=/path/to/a21-voice-clone-wrapper
A21_VOICE_CLONE_MODEL=index_tts2
A21_VOICE_CLONE_REF_AUDIO=/path/to/a21-reference.wav
A21_VOICE_CLONE_REF_TEXT_FILE=/path/to/a21-reference.txt
A21_VOICE_PERSONA=a21_workmate
A21_VOICE_STYLE=workmate_warm
```

`A21_VOICE_CLONE_CLI` is accepted as a compatibility alias for
`A21_VOICE_CLONE_COMMAND` when an existing deployment already uses that env
name. `A21_VOICE_CLONE_COMMAND` remains the canonical name and takes priority
when both are set. Reports must record only env-name presence and safe profile
IDs, never command values or reference text.

`A21_VOICE_CLONE_REF_TEXT` may be used instead of `A21_VOICE_CLONE_REF_TEXT_FILE` for short lab-only runs. Reports must never record the reference text.

## Wrapper Args

The wrapper command receives this normalized argument contract:

```bash
--text-file <temp-text-file>
--output <wav-output>
--sample-rate <hz>
--ref-audio <reference-wav>
--ref-text-file <temp-reference-text-file>
--model <safe-model-id>
--persona <safe-persona-id>
--style <safe-style-id>
```

The wrapper must write PCM16 mono WAV at the requested sample rate. A21 product downlink requests 48 kHz and chunks audio into 60 ms frames; CLI smokes default to 16 kHz unless overridden by the caller.

## Report Red Lines

Reports may include:

- `provider`, `engine`, `model`, `voice_persona`, `style_profile`
- reference audio basename only
- `text_bytes`, audio format, timing, aggregate PCM quality

Reports must not include:

- user text, prompt, ASR transcript, provider output, reasoning text
- reference transcript text
- full reference audio path
- API key, token, `Authorization`, `Bearer`
- raw PCM or `data_base64`

## Host Checks

```bash
go run ./cmd/a21 local-tts-smoke --engine voice_clone_cli --clone-command "$A21_VOICE_CLONE_COMMAND" --clone-ref-audio "$A21_VOICE_CLONE_REF_AUDIO" --output-dir reports
go run ./cmd/a21 local-voice-loopback --engine voice_clone_cli --repeat 3 --output-dir reports
go run ./cmd/a21 product-readiness --use-latest-reports --output-dir reports
```

These are host-local checks. They do not imply physical StackChan speaker acceptance.

## 5080 IndexTTS2 Bridge

When IndexTTS2 is warmed on the 5080 lab host, do not point
`A21_VOICE_CLONE_COMMAND` directly at a remote `ssh ... infer_v2.py` command.
A21 passes a local Mac `--output` path to the wrapper, and a remote process
cannot write that path. Use the local bridge instead:

```bash
export A21_VOICE_CLONE_COMMAND="python3 scripts/a21_5080_indextts2_bridge.py --ssh-host ${A21_5080_SSH_HOST} --ssh-key ${A21_5080_SSH_KEY} --remote-repo ${A21_5080_INDEXTTS2_REPO} --remote-model-dir ${A21_5080_INDEXTTS2_MODEL_DIR} --remote-work-root ${A21_5080_WORK_ROOT}"
```

The bridge copies A21's UTF-8 text file, reference audio, and reference text
file to a per-run remote directory, executes IndexTTS2 from the remote repo
root, loads checkpoints from `A21_5080_INDEXTTS2_MODEL_DIR`, copies the
generated WAV back to A21's local `--output`, and leaves `local-tts-smoke`
responsible for the redacted report and aggregate `audio_quality` block. Set
`HF_ENDPOINT` in the local bridge environment only when the lab host needs the
Hugging Face mirror.

Bridge dry run:

```bash
python3 scripts/a21_5080_indextts2_bridge.py --dry-run \
  --ssh-host "$A21_5080_SSH_HOST" \
  --ssh-key "$A21_5080_SSH_KEY" \
  --remote-repo "$A21_5080_INDEXTTS2_REPO" \
  --remote-model-dir "$A21_5080_INDEXTTS2_MODEL_DIR" \
  --remote-work-root "$A21_5080_WORK_ROOT" \
  --text-file /tmp/a21-text.txt \
  --ref-audio /tmp/a21-reference.wav \
  --ref-text-file /tmp/a21-reference.txt \
  --output /tmp/a21-output.wav \
  --sample-rate 16000
```

The dry run prints basename-only plan fields and does not execute SSH/SCP or
model inference.
