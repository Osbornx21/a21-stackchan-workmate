package gateway

import (
	"fmt"
	"net/http"
)

func (s *Server) handleSimulator(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprint(w, simulatorHTML)
}

const simulatorHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <link rel="icon" href="data:,">
  <title>A21 Device Simulator</title>
  <style>
    :root {
      color-scheme: dark;
      --bg: #101417;
      --panel: #171d21;
      --panel-2: #1e262b;
      --line: #2d3941;
      --text: #edf4f4;
      --muted: #95a6aa;
      --green: #82c68f;
      --cyan: #78b7ce;
      --amber: #d6b15f;
      --red: #d97878;
      --ink: #070909;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      min-height: 100vh;
      background: var(--bg);
      color: var(--text);
      font-family: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
      letter-spacing: 0;
    }
    .app {
      min-height: 100vh;
      display: grid;
      grid-template-rows: auto 1fr;
    }
    header {
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 16px;
      padding: 18px 24px;
      border-bottom: 1px solid var(--line);
      background: #12181b;
    }
    h1 {
      margin: 0;
      font-size: 20px;
      font-weight: 720;
    }
    .status {
      display: flex;
      align-items: center;
      gap: 10px;
      color: var(--muted);
      font-size: 13px;
    }
    .dot {
      width: 10px;
      height: 10px;
      border-radius: 999px;
      background: var(--red);
    }
    body[data-connected="true"] .dot { background: var(--green); }
    main {
      display: grid;
      grid-template-columns: minmax(320px, 0.85fr) minmax(420px, 1.15fr);
      gap: 0;
      min-height: 0;
    }
    .stage {
      display: grid;
      place-items: center;
      padding: 32px;
      border-right: 1px solid var(--line);
      background: #111619;
    }
    .face-wrap {
      width: min(70vw, 360px);
      aspect-ratio: 1;
      display: grid;
      place-items: center;
    }
    .face {
      position: relative;
      width: 100%;
      height: 100%;
      border-radius: 32px;
      background: #d7efea;
      box-shadow: inset 0 -18px 0 rgba(0,0,0,0.08);
      transition: background 160ms ease, transform 160ms ease;
    }
    .eye {
      position: absolute;
      top: 32%;
      width: 52px;
      height: 58px;
      border-radius: 999px;
      background: var(--ink);
      transition: transform 160ms ease, height 160ms ease;
    }
    .eye.left { left: 26%; }
    .eye.right { right: 26%; }
    .mouth {
      position: absolute;
      left: 50%;
      bottom: 26%;
      width: 92px;
      height: 20px;
      border-radius: 0 0 999px 999px;
      border-bottom: 12px solid var(--ink);
      transform: translateX(-50%);
      transition: height 120ms ease, width 120ms ease, border-width 120ms ease;
    }
    body[data-state="thinking"] .face { background: #d8e4f4; transform: rotate(-2deg); }
    body[data-state="speaking"] .mouth { height: 56px; width: 72px; border-width: 22px; }
    body[data-state="interrupted"] .face { background: #f0dfc4; transform: rotate(0deg) scale(0.98); }
    body[data-state="interrupted"] .mouth { height: 8px; width: 70px; border-width: 5px; }
    body[data-state="error"] .face { background: #edc8c8; }
    body[data-mode="professional"] .face { outline: 5px solid rgba(120,183,206,0.36); }
    body[data-mode="public"] .face { outline: 5px solid rgba(214,177,95,0.34); }
    body[data-mode="private"] .face { outline: 5px solid rgba(130,198,143,0.34); }
    body[data-mode="muted"] .face { filter: grayscale(0.45); }
    body[data-state="listening"] .eye { transform: scaleY(1.08); }
    .side {
      display: grid;
      grid-template-rows: auto auto auto auto auto auto auto 1fr;
      gap: 18px;
      padding: 24px;
      min-width: 0;
      background: var(--panel);
    }
    .toolbar, .fields {
      display: flex;
      flex-wrap: wrap;
      gap: 10px;
      align-items: center;
    }
    button, select, input {
      height: 36px;
      border: 1px solid var(--line);
      background: var(--panel-2);
      color: var(--text);
      border-radius: 6px;
      padding: 0 12px;
      font: inherit;
      font-size: 14px;
    }
    button {
      cursor: pointer;
      min-width: 96px;
    }
    button.primary { border-color: #4d7d64; background: #203227; }
    button.warn { border-color: #796236; background: #302817; }
    input {
      flex: 1 1 280px;
      min-width: 180px;
    }
    .readout {
      display: grid;
      grid-template-columns: repeat(3, minmax(0, 1fr));
      gap: 10px;
    }
    .visibility {
      border: 1px solid var(--line);
      border-radius: 8px;
      padding: 12px;
      background: #11171a;
      min-width: 0;
    }
    .visibility h2 {
      margin: 0 0 10px;
      font-size: 13px;
      font-weight: 680;
    }
    .badge-row {
      display: grid;
      grid-template-columns: repeat(4, minmax(0, 1fr));
      gap: 10px;
    }
    .badge {
      min-height: 42px;
      border: 1px solid #324049;
      border-radius: 7px;
      display: grid;
      place-items: center;
      background: #151b1f;
      color: var(--muted);
      font-size: 12px;
      font-weight: 760;
      letter-spacing: 0;
    }
    .badge[data-active="true"] {
      border-color: #77b9c9;
      color: var(--text);
      background: #17252a;
    }
    .badge.private[data-active="true"] { border-color: #82c68f; background: #16231b; }
    .badge.public[data-active="true"] { border-color: #d6b15f; background: #2a2315; }
    .badge.muted[data-active="true"] { border-color: #95a6aa; background: #1c2225; }
    .registry, .audio-link, .latency-summary {
      border: 1px solid var(--line);
      border-radius: 8px;
      padding: 12px;
      background: #141a1d;
      min-width: 0;
    }
    .registry h2, .audio-link h2, .latency-summary h2 {
      margin: 0 0 10px;
      font-size: 13px;
      font-weight: 680;
    }
    .audio-controls {
      display: flex;
      flex-wrap: wrap;
      gap: 10px;
      align-items: center;
      margin-bottom: 10px;
    }
    .toggle {
      min-height: 36px;
      display: inline-flex;
      align-items: center;
      gap: 8px;
      padding: 0 10px;
      border: 1px solid var(--line);
      border-radius: 6px;
      background: #11171a;
      color: var(--muted);
      font-size: 13px;
    }
    .toggle input {
      width: 16px;
      height: 16px;
      min-width: 0;
      flex: 0 0 auto;
      accent-color: var(--cyan);
    }
    .registry-grid {
      display: grid;
      grid-template-columns: repeat(4, minmax(0, 1fr));
      gap: 10px;
    }
    .waterfall {
      border: 1px solid var(--line);
      border-radius: 8px;
      padding: 12px;
      background: #11171a;
      min-width: 0;
    }
    .waterfall h2 {
      margin: 0 0 10px;
      font-size: 13px;
      font-weight: 680;
    }
    .waterfall-list {
      display: grid;
      gap: 6px;
      font-size: 12px;
      color: #c8d4d6;
    }
    .waterfall-row {
      display: grid;
      grid-template-columns: 68px minmax(0, 1fr);
      gap: 8px;
      align-items: center;
    }
    .waterfall-name {
      overflow: hidden;
      white-space: nowrap;
      text-overflow: ellipsis;
    }
    .evidence {
      border: 1px solid var(--line);
      border-radius: 8px;
      padding: 12px;
      background: #141a1d;
      min-width: 0;
    }
    .evidence h2 {
      margin: 0 0 10px;
      font-size: 13px;
      font-weight: 680;
    }
    .evidence-list {
      display: grid;
      gap: 8px;
    }
    .evidence-card {
      border: 1px solid #314149;
      border-radius: 6px;
      padding: 9px;
      background: #101619;
      color: #d8e3e5;
      font-size: 12px;
      line-height: 1.45;
      min-width: 0;
    }
    .evidence-card strong {
      display: block;
      color: var(--text);
      font-size: 12px;
      margin-bottom: 3px;
    }
    .evidence-card span {
      display: block;
      color: var(--muted);
      overflow-wrap: anywhere;
    }
    .metric {
      border: 1px solid var(--line);
      border-radius: 8px;
      padding: 10px;
      background: #151b1f;
      min-width: 0;
    }
    .metric label {
      display: block;
      color: var(--muted);
      font-size: 11px;
      margin-bottom: 4px;
      text-transform: uppercase;
    }
    .metric div {
      overflow: hidden;
      white-space: nowrap;
      text-overflow: ellipsis;
      font-size: 13px;
    }
    pre {
      margin: 0;
      min-height: 220px;
      overflow: auto;
      border: 1px solid var(--line);
      border-radius: 8px;
      padding: 12px;
      background: #0c1012;
      color: #c8d4d6;
      font-size: 12px;
      line-height: 1.5;
      white-space: pre-wrap;
    }
    @media (max-width: 860px) {
      main { grid-template-columns: 1fr; }
      .stage { border-right: 0; border-bottom: 1px solid var(--line); }
      .readout { grid-template-columns: 1fr; }
      .registry-grid { grid-template-columns: 1fr; }
    }
  </style>
</head>
<body data-state="idle" data-connected="false">
  <div class="app" data-testid="simulator-root">
    <header>
      <h1>A21 Device Simulator</h1>
      <div class="status"><span class="dot"></span><span id="connection">DISCONNECTED</span></div>
    </header>
    <main>
      <section class="stage" aria-label="StackChan face">
        <div class="face-wrap">
          <div class="face" id="face">
            <div class="eye left"></div>
            <div class="eye right"></div>
            <div class="mouth"></div>
          </div>
        </div>
      </section>
      <section class="side">
        <div class="toolbar">
          <button class="primary" id="connect">Connect</button>
          <button id="disconnect">Disconnect</button>
          <button id="mockTurn">Mock Turn</button>
          <button class="warn" id="interrupt">Interrupt</button>
          <button id="audioFrame">Audio Frame</button>
        </div>
        <div class="fields">
          <select id="mode" aria-label="mode">
            <option value="workmate">workmate</option>
            <option value="companion">companion</option>
            <option value="co_creation">co_creation</option>
            <option value="roleplay">roleplay</option>
            <option value="professional">professional</option>
            <option value="focus">focus</option>
            <option value="public">public</option>
            <option value="private">private</option>
            <option value="muted">muted</option>
            <option value="local_fallback">local_fallback</option>
          </select>
          <input id="utterance" value="先说，我在" aria-label="utterance">
        </div>
        <div class="readout">
          <div class="metric"><label>State</label><div id="state">idle</div></div>
          <div class="metric"><label>Mode</label><div id="modeReadout">workmate</div></div>
          <div class="metric"><label>Trace</label><div id="trace">none</div></div>
        </div>
        <section class="visibility" aria-label="Office Visibility">
          <h2>Office Visibility</h2>
          <div class="badge-row">
            <div class="badge private" id="privacyBadge">PRIVATE</div>
            <div class="badge public" id="visibilityBadge">PUBLIC</div>
            <div class="badge" id="proBadge">PRO</div>
            <div class="badge muted" id="mutedBadge">MUTED</div>
          </div>
          <div class="badge-row" style="margin-top:10px;">
            <div class="badge" id="listeningBadge">LISTENING</div>
            <div class="metric"><label>Session</label><div id="session">none</div></div>
            <div class="metric"><label>Screen</label><div id="screenBadge">LOCAL</div></div>
            <div class="metric"><label>Output</label><div id="outputBadge">speaker</div></div>
          </div>
        </section>
        <section class="audio-link" aria-label="Audio Link">
          <h2>Audio Link</h2>
          <div class="audio-controls">
            <button id="startMic">Start Mic</button>
            <button id="stopMic">Stop Mic</button>
            <button id="mockAudioBurst">Mock Burst</button>
            <label class="toggle"><input id="mockPlayback" type="checkbox" checked> Playback</label>
          </div>
          <div class="registry-grid">
            <div class="metric"><label>Input</label><div id="audioInputState">idle</div></div>
            <div class="metric"><label>Frames</label><div id="audioFramesSent">0</div></div>
            <div class="metric"><label>RMS</label><div id="audioRms">0.000</div></div>
            <div class="metric"><label>Playback</label><div id="playbackState">stopped</div></div>
            <div class="metric"><label>Downlink</label><div id="playbackChunksReceived">0</div></div>
            <div class="metric"><label>Buffer</label><div id="playbackBufferedChunks">0</div></div>
            <div class="metric"><label>Stream</label><div id="playbackStream">none</div></div>
            <div class="metric"><label>Scheduled</label><div id="playbackScheduledChunks">0</div></div>
          </div>
        </section>
        <section class="registry" aria-label="Device Registry">
          <h2>Device Registry</h2>
          <div class="registry-grid">
            <div class="metric"><label>Device</label><div id="registryDevice">none</div></div>
            <div class="metric"><label>Identity</label><div id="registryIdentity">none</div></div>
            <div class="metric"><label>Firmware</label><div id="registryFirmware">none</div></div>
            <div class="metric"><label>Commit</label><div id="registryCommit">none</div></div>
          </div>
        </section>
        <section class="waterfall" aria-label="Latency Waterfall">
          <h2>Waterfall</h2>
          <div class="waterfall-list" id="waterfall">
            <div class="waterfall-row"><span>0 ms</span><span class="waterfall-name">none</span></div>
          </div>
        </section>
        <section class="latency-summary" aria-label="Latency Summary">
          <h2>Latency Summary</h2>
          <div class="registry-grid">
            <div class="metric"><label>Audio</label><div id="latencyAudioPlayback">n/a</div></div>
            <div class="metric"><label>V21</label><div id="latencyV21">n/a</div></div>
            <div class="metric"><label>Barge-in</label><div id="latencyBargeIn">n/a</div></div>
            <div class="metric"><label>Provider</label><div id="latencyProviderFirstAudio">n/a</div></div>
          </div>
        </section>
        <section class="evidence" aria-label="Professional Evidence">
          <h2>Professional Evidence</h2>
          <div class="evidence-list" id="professionalEvidence">
            <div class="evidence-card"><strong>none</strong><span>professional mode has not returned evidence yet</span></div>
          </div>
        </section>
        <pre id="log" aria-label="event log"></pre>
      </section>
    </main>
  </div>
  <script>
    const ui = {
      connection: document.getElementById('connection'),
      state: document.getElementById('state'),
      modeReadout: document.getElementById('modeReadout'),
      trace: document.getElementById('trace'),
      session: document.getElementById('session'),
      privacyBadge: document.getElementById('privacyBadge'),
      visibilityBadge: document.getElementById('visibilityBadge'),
      proBadge: document.getElementById('proBadge'),
      mutedBadge: document.getElementById('mutedBadge'),
      listeningBadge: document.getElementById('listeningBadge'),
      screenBadge: document.getElementById('screenBadge'),
      outputBadge: document.getElementById('outputBadge'),
      audioInputState: document.getElementById('audioInputState'),
      audioFramesSent: document.getElementById('audioFramesSent'),
      audioRms: document.getElementById('audioRms'),
      playbackState: document.getElementById('playbackState'),
      playbackChunksReceived: document.getElementById('playbackChunksReceived'),
      playbackBufferedChunks: document.getElementById('playbackBufferedChunks'),
      playbackStream: document.getElementById('playbackStream'),
      playbackScheduledChunks: document.getElementById('playbackScheduledChunks'),
      mockPlayback: document.getElementById('mockPlayback'),
      registryDevice: document.getElementById('registryDevice'),
      registryIdentity: document.getElementById('registryIdentity'),
      registryFirmware: document.getElementById('registryFirmware'),
      registryCommit: document.getElementById('registryCommit'),
      waterfall: document.getElementById('waterfall'),
      latencyAudioPlayback: document.getElementById('latencyAudioPlayback'),
      latencyV21: document.getElementById('latencyV21'),
      latencyBargeIn: document.getElementById('latencyBargeIn'),
      latencyProviderFirstAudio: document.getElementById('latencyProviderFirstAudio'),
      professionalEvidence: document.getElementById('professionalEvidence'),
      log: document.getElementById('log'),
      mode: document.getElementById('mode'),
      utterance: document.getElementById('utterance')
    };
    const sim = {
      control: null,
      audio: null,
      seq: 1,
      traceId: '',
      sessionId: '',
      mode: 'workmate',
      state: 'idle',
      audioFrames: 0,
      playbackChunks: 0,
      playbackBufferedChunks: 0,
      playbackScheduledChunks: 0,
      playbackStreamId: '',
      micStream: null,
      audioContext: null,
      analyser: null,
      micTimer: null,
      playbackContext: null,
      playbackNextAt: 0,
      playbackSources: []
    };
    const firmwareIdentity = {
      firmware_id: 'a21-stackchan',
      firmware_version: '0.1.0',
      firmware_board: 'm5stack-cores3',
      firmware_commit: '0000000'
    };

    function wsURL(path) {
      const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
      return proto + '//' + location.host + path;
    }
    function log(line) {
      const time = new Date().toLocaleTimeString();
      ui.log.textContent = '[' + time + '] ' + line + '\n' + ui.log.textContent;
    }
    function setConnected(connected) {
      document.body.dataset.connected = connected ? 'true' : 'false';
      ui.connection.textContent = connected ? 'CONNECTED' : 'DISCONNECTED';
      updateVisibilityBadges();
    }
    function setState(state) {
      sim.state = state || 'idle';
      document.body.dataset.state = sim.state;
      ui.state.textContent = sim.state;
      updateVisibilityBadges();
    }
    function setMode(mode) {
      sim.mode = mode || 'workmate';
      document.body.dataset.mode = sim.mode;
      ui.modeReadout.textContent = sim.mode;
      updateVisibilityBadges();
    }
    function rememberEnvelope(envelope) {
      if (envelope.trace_id) {
        sim.traceId = envelope.trace_id;
        ui.trace.textContent = envelope.trace_id;
      }
      if (envelope.session_id) {
        sim.sessionId = envelope.session_id;
        ui.session.textContent = envelope.session_id;
      }
    }
    function handleEnvelope(envelope) {
      rememberEnvelope(envelope);
      const payload = envelope.payload || {};
      if (envelope.kind === 'audio.playback.chunk') {
        handleAudioPlaybackChunk(payload);
        log(envelope.kind + ' seq=' + envelope.seq + ' stream=' + (payload.stream_id || 'n/a') + ' duration=' + (payload.duration_ms || 'n/a'));
        refreshWaterfall();
        return;
      }
      if (payload.state) setState(payload.state);
      if (payload.mode) setMode(payload.mode);
      updatePlaybackState(payload);
      if (payload.mode && payload.mode !== 'professional') clearProfessionalEvidence();
      if (payload.evidence || payload.screen_cards || payload.speech_blocks) renderProfessionalEvidence(payload);
      log(envelope.kind + ' seq=' + envelope.seq + ' state=' + (payload.state || 'n/a') + ' text=' + (payload.text || ''));
      refreshWaterfall();
    }
    function setBadge(element, active) {
      element.dataset.active = active ? 'true' : 'false';
    }
    function currentScreenLabel() {
      if (sim.mode === 'professional') return 'PRO';
      if (sim.mode === 'local_fallback') return 'LOCAL';
      if (sim.mode === 'public') return 'PUBLIC';
      if (sim.mode === 'private') return 'PRIVATE';
      if (sim.mode === 'muted') return 'MUTED';
      if (sim.state === 'listening') return 'LISTENING';
      return sim.mode.toUpperCase().replace(/_/g, ' ');
    }
    function updateVisibilityBadges() {
      setBadge(ui.privacyBadge, sim.mode === 'private');
      setBadge(ui.visibilityBadge, sim.mode === 'public');
      setBadge(ui.proBadge, sim.mode === 'professional');
      setBadge(ui.mutedBadge, sim.mode === 'muted');
      setBadge(ui.listeningBadge, sim.state === 'listening');
      ui.screenBadge.textContent = currentScreenLabel();
      ui.outputBadge.textContent = sim.mode === 'muted' ? 'muted' : 'speaker';
    }
    function updateAudioInput(state) {
      ui.audioInputState.textContent = state;
    }
    function updateAudioFrameStats(rms) {
      sim.audioFrames += 1;
      ui.audioFramesSent.textContent = String(sim.audioFrames);
      ui.audioRms.textContent = Number(rms || 0).toFixed(3);
    }
    function updatePlaybackState(payload) {
      if (!payload || !payload.state) return;
      if (payload.state === 'speaking') {
        ui.playbackState.textContent = payload.stream_id ? 'playing ' + payload.stream_id : 'playing';
        playMockPlaybackTick();
      } else if (payload.state === 'interrupted') {
        ui.playbackState.textContent = 'interrupted';
        clearPlaybackBuffer();
      } else if (payload.state === 'error') {
        ui.playbackState.textContent = 'error';
        clearPlaybackBuffer();
      } else if (payload.state === 'listening' && payload.text === 'audio frame accepted') {
        ui.playbackState.textContent = 'uplink ack';
      } else if (payload.state && payload.state !== 'speaking') {
        clearPlaybackBuffer();
      }
    }
    function handleAudioPlaybackChunk(payload) {
      if (!payload || !payload.stream_id) return;
      if (sim.playbackStreamId && sim.playbackStreamId !== payload.stream_id) {
        sim.playbackBufferedChunks = 0;
      }
      sim.playbackStreamId = payload.stream_id;
      sim.playbackChunks += 1;
      sim.playbackBufferedChunks = Math.min(sim.playbackBufferedChunks + 1, 8);
      ui.playbackChunksReceived.textContent = String(sim.playbackChunks);
      ui.playbackBufferedChunks.textContent = String(sim.playbackBufferedChunks);
      ui.playbackStream.textContent = payload.stream_id;
      ui.playbackState.textContent = 'buffered ' + payload.stream_id;
      schedulePCMPlayback(payload);
    }
    function clearPlaybackBuffer() {
      stopScheduledPlayback();
      sim.playbackBufferedChunks = 0;
      sim.playbackStreamId = '';
      sim.playbackNextAt = 0;
      ui.playbackBufferedChunks.textContent = '0';
      ui.playbackStream.textContent = 'none';
    }
    function decodePCM16Base64(dataBase64) {
      const binary = atob(dataBase64 || '');
      const sampleCount = Math.floor(binary.length / 2);
      const samples = new Float32Array(sampleCount);
      for (let i = 0; i < sampleCount; i++) {
        const lo = binary.charCodeAt(i * 2);
        const hi = binary.charCodeAt(i * 2 + 1);
        const value = (hi << 8) | lo;
        const signed = value >= 0x8000 ? value - 0x10000 : value;
        samples[i] = signed / 0x8000;
      }
      return samples;
    }
    function schedulePCMPlayback(payload) {
      if (!ui.mockPlayback.checked || !payload || payload.codec !== 'pcm_s16le' || payload.channels !== 1) return;
      const AudioCtor = window.AudioContext || window.webkitAudioContext;
      if (!AudioCtor) return;
      const context = sim.playbackContext || new AudioCtor();
      sim.playbackContext = context;
      const samples = decodePCM16Base64(payload.data_base64);
      if (!samples.length || !payload.sample_rate_hz) return;
      const buffer = context.createBuffer(1, samples.length, payload.sample_rate_hz);
      buffer.copyToChannel(samples, 0);
      const source = context.createBufferSource();
      source.buffer = buffer;
      source.connect(context.destination);
      const startAt = Math.max(context.currentTime + 0.01, sim.playbackNextAt || 0);
      sim.playbackSources.push(source);
      source.onended = () => {
        sim.playbackSources = sim.playbackSources.filter((item) => item !== source);
      };
      source.start(startAt);
      sim.playbackNextAt = startAt + buffer.duration;
      sim.playbackScheduledChunks += 1;
      ui.playbackScheduledChunks.textContent = String(sim.playbackScheduledChunks);
      ui.playbackState.textContent = 'scheduled ' + payload.stream_id;
    }
    function stopScheduledPlayback() {
      sim.playbackSources.forEach((source) => {
        try {
          source.stop();
        } catch (err) {
          // The source may already have ended; stopping is best-effort for barge-in.
        }
      });
      sim.playbackSources = [];
    }
    function playMockPlaybackTick() {
      if (!ui.mockPlayback.checked) return;
      const AudioCtor = window.AudioContext || window.webkitAudioContext;
      if (!AudioCtor) return;
      const context = sim.playbackContext || new AudioCtor();
      sim.playbackContext = context;
      const oscillator = context.createOscillator();
      const gain = context.createGain();
      oscillator.type = 'sine';
      oscillator.frequency.value = 440;
      gain.gain.setValueAtTime(0.0001, context.currentTime);
      gain.gain.exponentialRampToValueAtTime(0.035, context.currentTime + 0.012);
      gain.gain.exponentialRampToValueAtTime(0.0001, context.currentTime + 0.09);
      oscillator.connect(gain);
      gain.connect(context.destination);
      oscillator.start();
      oscillator.stop(context.currentTime + 0.1);
    }
    function escapeText(value) {
      return String(value || '').replace(/[&<>"']/g, (char) => ({
        '&': '&amp;',
        '<': '&lt;',
        '>': '&gt;',
        '"': '&quot;',
        "'": '&#39;'
      }[char]));
    }
    function clearProfessionalEvidence() {
      ui.professionalEvidence.innerHTML = '<div class="evidence-card"><strong>none</strong><span>professional mode has not returned evidence yet</span></div>';
    }
    function renderProfessionalEvidence(payload) {
      const cards = [];
      if (typeof payload.confidence === 'number') {
        cards.push('<div class="evidence-card"><strong>confidence</strong><span>' + Math.round(payload.confidence * 100) + '%</span></div>');
      }
      (payload.screen_cards || []).forEach((card) => {
        cards.push('<div class="evidence-card"><strong>' + escapeText(card.label) + '</strong><span>' + escapeText(card.text) + '</span></div>');
      });
      (payload.evidence || []).forEach((item) => {
        cards.push('<div class="evidence-card"><strong>' + escapeText(item.title || item.source_id) + '</strong><span>' + escapeText(item.summary || item.type) + '</span></div>');
      });
      if (payload.follow_ups && payload.follow_ups.length) {
        cards.push('<div class="evidence-card"><strong>follow-ups</strong><span>' + escapeText(payload.follow_ups.join(' / ')) + '</span></div>');
      }
      ui.professionalEvidence.innerHTML = cards.join('') || '<div class="evidence-card"><strong>professional</strong><span>No evidence returned.</span></div>';
    }
    async function refreshRegistry() {
      try {
        const response = await fetch('/v1/devices', { cache: 'no-store' });
        if (!response.ok) {
          log('device registry error ' + response.status);
          return;
        }
        const registry = await response.json();
        const device = (registry.devices || []).find((item) => item.device_id === 'stackchan-sim-001') || (registry.devices || [])[0];
        if (!device) return;
        const firmware = device.firmware || {};
        ui.registryDevice.textContent = device.device_id || 'none';
        ui.registryIdentity.textContent = device.identity_status || 'none';
        ui.registryFirmware.textContent = [firmware.id, firmware.version, firmware.board].filter(Boolean).join(' / ') || 'none';
        ui.registryCommit.textContent = firmware.commit || 'none';
      } catch (err) {
        log('device registry unavailable');
      }
    }
    async function refreshWaterfall() {
      if (!sim.traceId) return;
      try {
        const response = await fetch('/v1/traces?trace_id=' + encodeURIComponent(sim.traceId), { cache: 'no-store' });
        if (!response.ok) {
          log('trace waterfall error ' + response.status);
          return;
        }
        const trace = await response.json();
        const events = trace.events || [];
        if (!events.length) return;
        ui.waterfall.innerHTML = events.map((event) =>
          '<div class="waterfall-row"><span>' + event.offset_ms + ' ms</span><span class="waterfall-name">' + event.name + '</span></div>'
        ).join('');
        renderLatencySummary(trace.summary || {});
      } catch (err) {
        log('trace waterfall unavailable');
      }
    }
    function renderLatencySummary(summary) {
      const fmt = (value) => typeof value === 'number' ? value + ' ms' : 'n/a';
      ui.latencyAudioPlayback.textContent = fmt(summary.audio_frame_to_playback_ms);
      ui.latencyV21.textContent = fmt(summary.v21_query_first_result_ms);
      ui.latencyBargeIn.textContent = fmt(summary.barge_in_stop_ms);
      ui.latencyProviderFirstAudio.textContent = fmt(summary.provider_commit_to_first_audio_ms);
    }
    function connect() {
      if (sim.control && sim.control.readyState === WebSocket.OPEN) return;
      sim.control = new WebSocket(wsURL('/ws/control'));
      sim.audio = new WebSocket(wsURL('/ws/audio'));
      sim.control.onopen = () => { setConnected(true); log('control connected /ws/control'); };
      sim.control.onclose = () => { setConnected(false); log('control closed'); };
      sim.control.onerror = () => { setState('error'); log('control error'); };
      sim.control.onmessage = (event) => { handleEnvelope(JSON.parse(event.data)); refreshRegistry(); };
      sim.audio.onopen = () => log('audio connected /ws/audio');
      sim.audio.onmessage = (event) => handleEnvelope(JSON.parse(event.data));
      sim.audio.onerror = () => log('audio error');
      refreshRegistry();
      refreshWaterfall();
      setMode(ui.mode.value);
    }
    function disconnect() {
      stopMicrophoneStream();
      if (sim.control) sim.control.close();
      if (sim.audio) sim.audio.close();
      setConnected(false);
      setState('idle');
      setMode(ui.mode.value);
      ui.playbackState.textContent = 'stopped';
      clearPlaybackBuffer();
    }
    function sendDeviceEvent(eventName) {
      if (!sim.control || sim.control.readyState !== WebSocket.OPEN) {
        log('control socket is not connected');
        return;
      }
      const envelope = {
        protocol: 'a21.device.v1',
        device_id: 'stackchan-sim-001',
        kind: 'device.event',
        seq: sim.seq++,
        trace_id: sim.traceId,
        session_id: sim.sessionId,
        payload: Object.assign({ event: eventName, mode: ui.mode.value, text: ui.utterance.value }, firmwareIdentity)
      };
      sim.control.send(JSON.stringify(envelope));
      log('sent device event ' + eventName);
      refreshRegistry();
      refreshWaterfall();
    }
    function sendAudioFramePayload(dataBase64, durationMS, rms, source) {
      if (!sim.audio || sim.audio.readyState !== WebSocket.OPEN) {
        log('audio socket is not connected');
        return;
      }
      const endedAt = Date.now();
      const envelope = {
        protocol: 'a21.device.v1',
        device_id: 'stackchan-sim-001',
        kind: 'audio.frame',
        seq: sim.seq++,
        trace_id: sim.traceId,
        session_id: sim.sessionId,
        payload: {
          codec: 'pcm_s16le',
          sample_rate_hz: 16000,
          channels: 1,
          duration_ms: durationMS,
          capture_started_at_ms: endedAt - durationMS,
          capture_ended_at_ms: endedAt,
          data_base64: dataBase64
        }
      };
      sim.audio.send(JSON.stringify(envelope));
      updateAudioFrameStats(rms);
      log('sent ' + source + ' audio frame #' + sim.audioFrames);
      refreshWaterfall();
    }
    function sendAudioFrame() {
      sendAudioFramePayload('AAAA', 20, 0, 'mock');
    }
    function sendMockAudioBurst() {
      updateAudioInput('mock burst');
      for (let i = 0; i < 5; i++) {
        window.setTimeout(() => sendAudioFramePayload('AAAA', 20, 0, 'mock-burst'), i * 25);
      }
      window.setTimeout(() => updateAudioInput('idle'), 150);
    }
    function pcm16Base64FromFloat32(samples, count) {
      const length = count || samples.length;
      const bytes = new Uint8Array(length * 2);
      const view = new DataView(bytes.buffer);
      for (let i = 0; i < length; i++) {
        const value = Math.max(-1, Math.min(1, samples[i % samples.length] || 0));
        view.setInt16(i * 2, value < 0 ? value * 0x8000 : value * 0x7fff, true);
      }
      let binary = '';
      for (let i = 0; i < bytes.length; i++) binary += String.fromCharCode(bytes[i]);
      return btoa(binary);
    }
    async function startMicrophoneStream() {
      if (sim.micTimer) return;
      if (!sim.audio || sim.audio.readyState !== WebSocket.OPEN) {
        log('connect audio websocket before starting mic');
        updateAudioInput('needs connect');
        return;
      }
      if (!navigator.mediaDevices || !navigator.mediaDevices.getUserMedia) {
        updateAudioInput('unavailable');
        log('browser microphone API unavailable');
        return;
      }
      try {
        updateAudioInput('requesting');
        sim.micStream = await navigator.mediaDevices.getUserMedia({
          audio: { channelCount: 1, echoCancellation: true, noiseSuppression: true, autoGainControl: true },
          video: false
        });
        const AudioCtor = window.AudioContext || window.webkitAudioContext;
        sim.audioContext = new AudioCtor();
        const source = sim.audioContext.createMediaStreamSource(sim.micStream);
        sim.analyser = sim.audioContext.createAnalyser();
        sim.analyser.fftSize = 1024;
        source.connect(sim.analyser);
        const samples = new Float32Array(1024);
        sim.micTimer = window.setInterval(() => {
          if (!sim.analyser) return;
          sim.analyser.getFloatTimeDomainData(samples);
          let sum = 0;
          for (let i = 0; i < samples.length; i++) sum += samples[i] * samples[i];
          const rms = Math.sqrt(sum / samples.length);
          sendAudioFramePayload(pcm16Base64FromFloat32(samples, 640), 40, rms, 'mic');
        }, 40);
        updateAudioInput('streaming');
        log('microphone stream started');
      } catch (err) {
        updateAudioInput('blocked');
        log('microphone unavailable or permission denied');
      }
    }
    async function stopMicrophoneStream() {
      if (sim.micTimer) {
        window.clearInterval(sim.micTimer);
        sim.micTimer = null;
      }
      if (sim.micStream) {
        sim.micStream.getTracks().forEach((track) => track.stop());
        sim.micStream = null;
      }
      if (sim.audioContext) {
        await sim.audioContext.close();
        sim.audioContext = null;
      }
      sim.analyser = null;
      updateAudioInput('idle');
      log('microphone stream stopped');
    }
    document.getElementById('connect').addEventListener('click', connect);
    document.getElementById('disconnect').addEventListener('click', disconnect);
    document.getElementById('mockTurn').addEventListener('click', () => sendDeviceEvent('mock.turn'));
    document.getElementById('interrupt').addEventListener('click', () => sendDeviceEvent('interrupt'));
    document.getElementById('audioFrame').addEventListener('click', sendAudioFrame);
    ui.mode.addEventListener('change', () => setMode(ui.mode.value));
    document.getElementById('startMic').addEventListener('click', startMicrophoneStream);
    document.getElementById('stopMic').addEventListener('click', stopMicrophoneStream);
    document.getElementById('mockAudioBurst').addEventListener('click', sendMockAudioBurst);
    refreshRegistry();
    setMode(ui.mode.value);
    updateVisibilityBadges();
  </script>
</body>
</html>`
