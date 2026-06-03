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
    body[data-state="local_fallback"] .face { background: #d7e7d2; outline: 5px solid rgba(130,198,143,0.42); }
    body[data-state="error"] .face { background: #edc8c8; }
    body[data-mode="professional"] .face { outline: 5px solid rgba(120,183,206,0.36); }
    body[data-mode="public"] .face { outline: 5px solid rgba(214,177,95,0.34); }
    body[data-mode="private"] .face { outline: 5px solid rgba(130,198,143,0.34); }
    body[data-mode="muted"] .face { filter: grayscale(0.45); }
    body[data-state="listening"] .eye { transform: scaleY(1.08); }
    .side {
      display: grid;
      grid-template-rows: auto auto auto auto auto auto auto auto 1fr;
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
      grid-template-columns: repeat(5, minmax(0, 1fr));
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
    .registry, .audio-link, .latency-summary, .wake-word {
      border: 1px solid var(--line);
      border-radius: 8px;
      padding: 12px;
      background: #141a1d;
      min-width: 0;
    }
    .registry h2, .audio-link h2, .latency-summary h2, .wake-word h2 {
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
          <button id="workspaceJob">Workspace Job</button>
        </div>
        <div class="fields">
          <select id="mode" aria-label="mode">
            <option value="roleplay">roleplay</option>
            <option value="professional">professional</option>
          </select>
          <select id="voiceMode" aria-label="voice mode">
            <option value="roleplay">roleplay</option>
            <option value="professional">professional</option>
          </select>
          <select id="gatewayProfile" aria-label="gateway profile">
            <option value="mac_local">mac_local</option>
            <option value="public_wss">public_wss</option>
          </select>
          <select id="cloudVoiceProfile" aria-label="cloud voice profile">
            <option value="a21_doubao_tts_realtime">a21_doubao_tts_realtime</option>
          </select>
          <select id="voiceChainMode" aria-label="voice chain mode">
            <option value="cascade">cascade</option>
            <option value="realtime">realtime</option>
          </select>
          <select id="cascadeASRProfile" aria-label="cascade ASR profile">
            <option value="dashscope_qwen_asr_realtime">Qwen ASR realtime</option>
          </select>
          <select id="cascadeLLMProfile" aria-label="cascade LLM profile">
            <option value="stepfun">StepFun 8k fast</option>
          </select>
          <select id="realtimeProvider" aria-label="realtime provider">
            <option value="doubao_realtime">Doubao realtime</option>
          </select>
          <select id="voiceCloneProfile" aria-label="voice clone profile">
            <option value="a21_voice_default_dashscope">A21 natural voice</option>
          </select>
          <select id="roleplayScenario" aria-label="roleplay scenario">
            <option value="desk_mouthpiece">desk_mouthpiece</option>
            <option value="boss_challenge">boss_challenge</option>
            <option value="engineer_pushback">engineer_pushback</option>
          </select>
          <select id="professionalQueryScope" aria-label="professional query scope">
            <option value="public_only">public_only</option>
            <option value="personal_only">personal_only</option>
            <option value="personal_plus_public">personal_plus_public</option>
          </select>
          <input id="workspaceDocumentLabel" value="PRD pack" aria-label="workspace document label">
          <input id="utterance" value="先说，我在" aria-label="utterance">
        </div>
        <div class="readout">
          <div class="metric"><label>State</label><div id="state">idle</div></div>
          <div class="metric"><label>Mode</label><div id="modeReadout">roleplay</div></div>
          <div class="metric"><label>Voice</label><div id="voiceModeReadout">roleplay</div></div>
          <div class="metric"><label>Gateway</label><div id="gatewayProfileReadout">mac_local</div></div>
          <div class="metric"><label>Cloud Voice</label><div id="cloudVoiceProfileReadout">a21_doubao_tts_realtime</div></div>
          <div class="metric"><label>Chain</label><div id="voiceChainModeReadout">cascade</div></div>
          <div class="metric"><label>ASR</label><div id="cascadeASRProfileReadout">dashscope_qwen_asr_realtime</div></div>
          <div class="metric"><label>LLM</label><div id="cascadeLLMProfileReadout">stepfun</div></div>
          <div class="metric"><label>TTS</label><div id="selectedTTSProfileReadout">dashscope_qwen_tts_realtime</div></div>
          <div class="metric"><label>Realtime</label><div id="realtimeProviderReadout">doubao_realtime</div></div>
          <div class="metric"><label>Voice Name</label><div id="voiceCloneProfileReadout">A21 natural voice</div></div>
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
        <section class="wake-word" aria-label="Wake Word">
          <h2>Wake Word</h2>
          <div class="fields">
            <select id="wakeWordMode" aria-label="wake word mode">
              <option value="builtin_xiaozhi">builtin_xiaozhi</option>
              <option value="custom_multinet">custom_multinet</option>
            </select>
            <input id="wakeWordPhrase" value="小阿二一" aria-label="wake word phrase">
            <input id="wakeWordPinyin" value="xiao a er yi" aria-label="wake word pinyin">
            <input id="wakeWordThreshold" type="number" min="1" max="100" value="35" aria-label="wake word threshold">
            <button id="saveWakeWord">Save</button>
            <button id="resetWakeWord">Reset</button>
            <button id="exportWakeWord">Export</button>
          </div>
          <div class="registry-grid" style="margin-top:10px;">
            <div class="metric"><label>Active</label><div id="wakeWordActive">你好小智</div></div>
            <div class="metric"><label>Runtime</label><div id="wakeWordStatus">loading</div></div>
            <div class="metric"><label>Build</label><div id="wakeWordBuild">unknown</div></div>
            <div class="metric"><label>Firmware</label><div id="wakeWordFirmwareStatus">builtin_active</div></div>
            <div class="metric"><label>Hot Swap</label><div id="wakeWordHotSwap">disabled</div></div>
            <div class="metric"><label>Code</label><div id="wakeWordCode">none</div></div>
          </div>
        </section>
        <section class="registry" aria-label="Device Registry">
          <h2>Device Registry</h2>
          <div class="registry-grid">
            <div class="metric"><label>Device</label><div id="registryDevice">none</div></div>
            <div class="metric"><label>Identity</label><div id="registryIdentity">none</div></div>
            <div class="metric"><label>Connection</label><div id="registryConnection">none</div></div>
            <div class="metric"><label>Mode</label><div id="registryMode">none</div></div>
            <div class="metric"><label>Voice</label><div id="registryVoiceMode">none</div></div>
            <div class="metric"><label>Chain</label><div id="registryVoiceChainMode">none</div></div>
            <div class="metric"><label>ASR</label><div id="registryASRProfile">none</div></div>
            <div class="metric"><label>LLM</label><div id="registryLLMProfile">none</div></div>
            <div class="metric"><label>TTS</label><div id="registryTTSProfile">none</div></div>
            <div class="metric"><label>Realtime</label><div id="registryRealtimeProvider">none</div></div>
            <div class="metric"><label>Voice Name</label><div id="registryVoiceCloneProfile">none</div></div>
            <div class="metric"><label>Cloud Voice</label><div id="registryCloudVoiceProfile">none</div></div>
            <div class="metric"><label>Expression</label><div id="registryExpression">none</div></div>
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
      voiceModeReadout: document.getElementById('voiceModeReadout'),
      gatewayProfileReadout: document.getElementById('gatewayProfileReadout'),
      cloudVoiceProfileReadout: document.getElementById('cloudVoiceProfileReadout'),
      voiceChainModeReadout: document.getElementById('voiceChainModeReadout'),
      cascadeASRProfileReadout: document.getElementById('cascadeASRProfileReadout'),
      cascadeLLMProfileReadout: document.getElementById('cascadeLLMProfileReadout'),
      selectedTTSProfileReadout: document.getElementById('selectedTTSProfileReadout'),
      realtimeProviderReadout: document.getElementById('realtimeProviderReadout'),
      voiceCloneProfileReadout: document.getElementById('voiceCloneProfileReadout'),
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
      registryConnection: document.getElementById('registryConnection'),
      registryMode: document.getElementById('registryMode'),
      registryVoiceMode: document.getElementById('registryVoiceMode'),
      registryVoiceChainMode: document.getElementById('registryVoiceChainMode'),
      registryASRProfile: document.getElementById('registryASRProfile'),
      registryLLMProfile: document.getElementById('registryLLMProfile'),
      registryTTSProfile: document.getElementById('registryTTSProfile'),
      registryRealtimeProvider: document.getElementById('registryRealtimeProvider'),
      registryVoiceCloneProfile: document.getElementById('registryVoiceCloneProfile'),
      registryCloudVoiceProfile: document.getElementById('registryCloudVoiceProfile'),
      registryExpression: document.getElementById('registryExpression'),
      registryFirmware: document.getElementById('registryFirmware'),
      registryCommit: document.getElementById('registryCommit'),
      wakeWordMode: document.getElementById('wakeWordMode'),
      wakeWordPhrase: document.getElementById('wakeWordPhrase'),
      wakeWordPinyin: document.getElementById('wakeWordPinyin'),
      wakeWordThreshold: document.getElementById('wakeWordThreshold'),
      resetWakeWord: document.getElementById('resetWakeWord'),
      exportWakeWord: document.getElementById('exportWakeWord'),
      wakeWordActive: document.getElementById('wakeWordActive'),
      wakeWordStatus: document.getElementById('wakeWordStatus'),
      wakeWordBuild: document.getElementById('wakeWordBuild'),
      wakeWordFirmwareStatus: document.getElementById('wakeWordFirmwareStatus'),
      wakeWordHotSwap: document.getElementById('wakeWordHotSwap'),
      wakeWordCode: document.getElementById('wakeWordCode'),
      waterfall: document.getElementById('waterfall'),
      latencyAudioPlayback: document.getElementById('latencyAudioPlayback'),
      latencyV21: document.getElementById('latencyV21'),
      latencyBargeIn: document.getElementById('latencyBargeIn'),
      latencyProviderFirstAudio: document.getElementById('latencyProviderFirstAudio'),
      professionalEvidence: document.getElementById('professionalEvidence'),
      log: document.getElementById('log'),
      mode: document.getElementById('mode'),
      voiceMode: document.getElementById('voiceMode'),
      gatewayProfile: document.getElementById('gatewayProfile'),
      cloudVoiceProfile: document.getElementById('cloudVoiceProfile'),
      voiceChainMode: document.getElementById('voiceChainMode'),
      cascadeASRProfile: document.getElementById('cascadeASRProfile'),
      cascadeLLMProfile: document.getElementById('cascadeLLMProfile'),
      realtimeProvider: document.getElementById('realtimeProvider'),
      voiceCloneProfile: document.getElementById('voiceCloneProfile'),
      roleplayScenario: document.getElementById('roleplayScenario'),
      professionalQueryScope: document.getElementById('professionalQueryScope'),
      workspaceDocumentLabel: document.getElementById('workspaceDocumentLabel'),
      utterance: document.getElementById('utterance')
    };
    let latestWakeWordConfig = null;
    const sim = {
      control: null,
      audio: null,
      seq: 1,
      traceId: '',
      sessionId: '',
      mode: 'dialogue',
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
      firmware_commit: '0000000',
      capabilities: {
        microphone: 'available',
        speaker: 'available',
        screen: 'available',
        screen_touch: 'available',
        top_touch: 'available',
        servo_y: 'available',
        servo_x: 'planned_continuous_rotation_axis',
        rgb: 'available',
        camera: 'planned_core_s3_camera',
        imu: 'planned_9_axis_imu',
        ambient_light: 'planned_ambient_light_sensor',
        proximity: 'planned_proximity_sensor',
        battery: 'planned_550mah_battery',
        nfc: 'planned_nfc',
        infrared: 'planned_infrared_tx_rx'
      }
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
      sim.mode = mode || 'roleplay';
      document.body.dataset.mode = sim.mode;
      ui.modeReadout.textContent = sim.mode;
      updateVisibilityBadges();
    }
    function setVoiceMode(mode) {
      ui.voiceMode.value = mode || 'roleplay';
      ui.voiceModeReadout.textContent = ui.voiceMode.value;
    }
    function setRoleplayProfile(catalog) {
      const selectedScenario = catalog.selected_scenario || 'desk_mouthpiece';
      const scenarios = catalog.scenarios || [];
      if (scenarios.length) {
        ui.roleplayScenario.innerHTML = scenarios.map((scenario) => optionHTML(scenario, selectedScenario)).join('');
      }
      ui.roleplayScenario.value = selectedScenario;
      if (catalog.selected_voice_clone_profile) {
        ui.voiceCloneProfile.value = catalog.selected_voice_clone_profile;
        ui.voiceCloneProfileReadout.textContent = catalog.selected_voice_clone_profile;
      }
    }
    function setProfessionalWorkspace(catalog) {
      const selectedScope = catalog.selected_query_scope || 'public_only';
      const scopes = catalog.query_scopes || [];
      if (scopes.length) {
        ui.professionalQueryScope.innerHTML = scopes.map((scope) => optionHTML(scope, selectedScope)).join('');
      }
      ui.professionalQueryScope.value = selectedScope;
    }
    function setGatewayProfile(profile) {
      ui.gatewayProfile.value = profile || 'mac_local';
      ui.gatewayProfileReadout.textContent = ui.gatewayProfile.value;
    }
    function setCloudVoiceProfile(profile) {
      ui.cloudVoiceProfile.value = profile || 'a21_doubao_tts_realtime';
      ui.cloudVoiceProfileReadout.textContent = ui.cloudVoiceProfile.value;
    }
    function setVoiceChainProfile(catalog) {
      const selectedMode = catalog.selected_voice_chain_mode || 'cascade';
      const selectedASR = catalog.selected_asr_profile || 'dashscope_qwen_asr_realtime';
      const selectedLLM = catalog.selected_llm_profile || 'stepfun';
      const selectedRealtime = catalog.selected_realtime_provider || 'doubao_realtime';
      const selectedVoice = catalog.selected_voice_clone_profile || 'a21_voice_default_dashscope';
      const selectedTTS = catalog.selected_tts_profile || catalog.fixed_tts_profile || 'dashscope_qwen_tts_realtime';
      const asrProfiles = (catalog.cascade && catalog.cascade.asr_profiles) || [];
      const llmProfiles = (catalog.cascade && catalog.cascade.llm_profiles) || [];
      const realtimeProviders = (catalog.realtime && catalog.realtime.providers) || [];
      const voices = catalog.voices || [];
      ui.voiceChainMode.value = selectedMode;
      ui.voiceChainModeReadout.textContent = selectedMode;
      if (asrProfiles.length) {
        ui.cascadeASRProfile.innerHTML = asrProfiles.map((profile) => optionHTML(profile, selectedASR)).join('');
      }
      if (llmProfiles.length) {
        ui.cascadeLLMProfile.innerHTML = llmProfiles.map((profile) => optionHTML(profile, selectedLLM)).join('');
      }
      if (realtimeProviders.length) {
        ui.realtimeProvider.innerHTML = realtimeProviders.map((profile) => optionHTML(profile, selectedRealtime)).join('');
      }
      if (voices.length) {
        ui.voiceCloneProfile.innerHTML = voices.map((voice) => optionHTML({
          id: voice.id,
          label: voice.label,
          status: voice.status,
          disabled: voice.status === 'planned'
        }, selectedVoice)).join('');
      }
      ui.cascadeASRProfile.value = selectedASR;
      ui.cascadeLLMProfile.value = selectedLLM;
      ui.realtimeProvider.value = selectedRealtime;
      ui.voiceCloneProfile.value = selectedVoice;
      ui.cascadeASRProfileReadout.textContent = selectedASR;
      ui.cascadeLLMProfileReadout.textContent = selectedLLM;
      ui.selectedTTSProfileReadout.textContent = selectedTTS;
      ui.realtimeProviderReadout.textContent = selectedRealtime;
      const selectedVoiceOption = voices.find((voice) => voice.id === selectedVoice);
      ui.voiceCloneProfileReadout.textContent = selectedVoiceOption ? selectedVoiceOption.label : selectedVoice;
    }
    function optionHTML(option, selected) {
      const id = escapeText(option.id || '');
      const label = escapeText(option.label || option.id || '');
      const status = option.status ? ' [' + escapeText(option.status) + ']' : '';
      const recommended = option.recommended ? ' *' : '';
      const disabled = option.disabled || option.status === 'planned' ? ' disabled' : '';
      const selectedAttr = (option.id || '') === selected ? ' selected' : '';
      return '<option value="' + id + '"' + disabled + selectedAttr + '>' + label + recommended + status + '</option>';
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
        const age = typeof device.device_age_ms === 'number' ? ' / ' + Math.round(device.device_age_ms / 1000) + 's' : '';
        ui.registryConnection.textContent = (device.connection_status || 'none') + age;
        ui.registryMode.textContent = device.current_mode || 'none';
        ui.registryVoiceMode.textContent = device.current_voice_mode || 'none';
        ui.registryVoiceChainMode.textContent = device.current_voice_chain_mode || 'none';
        ui.registryASRProfile.textContent = device.current_asr_profile || 'none';
        ui.registryLLMProfile.textContent = device.current_llm_profile || 'none';
        ui.registryTTSProfile.textContent = device.current_tts_profile || 'none';
        ui.registryRealtimeProvider.textContent = device.current_realtime_provider || 'none';
        ui.registryVoiceCloneProfile.textContent = device.current_voice_clone_profile || 'none';
        ui.registryCloudVoiceProfile.textContent = device.current_cloud_voice_profile || 'none';
        ui.registryExpression.textContent = device.current_expression || 'none';
        ui.registryFirmware.textContent = [firmware.id, firmware.version, firmware.board].filter(Boolean).join(' / ') || 'none';
        ui.registryCommit.textContent = firmware.commit || 'none';
      } catch (err) {
        log('device registry unavailable');
      }
    }
    async function refreshVoiceModes() {
      try {
        const response = await fetch('/v1/voice-modes', { cache: 'no-store' });
        if (!response.ok) {
          log('voice mode catalog error ' + response.status);
          return;
        }
        const catalog = await response.json();
        setVoiceMode(catalog.selected_voice_mode || 'roleplay');
      } catch (err) {
        log('voice mode catalog unavailable');
      }
    }
    async function refreshRoleplayProfile() {
      try {
        const response = await fetch('/v1/roleplay-profile', { cache: 'no-store' });
        if (!response.ok) {
          log('roleplay profile error ' + response.status);
          return;
        }
        setRoleplayProfile(await response.json());
      } catch (err) {
        log('roleplay profile unavailable');
      }
    }
    async function refreshProfessionalWorkspace() {
      try {
        const response = await fetch('/v1/professional-workspace', { cache: 'no-store' });
        if (!response.ok) {
          log('professional workspace error ' + response.status);
          return;
        }
        setProfessionalWorkspace(await response.json());
      } catch (err) {
        log('professional workspace unavailable');
      }
    }
    async function refreshGatewayProfiles() {
      try {
        const response = await fetch('/v1/gateway-profiles', { cache: 'no-store' });
        if (!response.ok) {
          log('gateway profile catalog error ' + response.status);
          return;
        }
        const catalog = await response.json();
        const selected = catalog.selected_gateway_profile || 'mac_local';
        const profiles = catalog.profiles || [];
        if (profiles.length) {
          ui.gatewayProfile.innerHTML = profiles.map((profile) => {
            const id = escapeText(profile.id || '');
            const status = profile.status || 'unknown';
            const disabled = status === 'available' ? '' : ' disabled';
            const selectedAttr = (profile.id || '') === selected ? ' selected' : '';
            return '<option value="' + id + '"' + disabled + selectedAttr + '>' + id + '</option>';
          }).join('');
        }
        setGatewayProfile(selected);
      } catch (err) {
        log('gateway profile catalog unavailable');
      }
    }
    async function refreshCloudVoiceProfiles() {
      try {
        const response = await fetch('/v1/cloud-voice-profiles', { cache: 'no-store' });
        if (!response.ok) {
          log('cloud voice catalog error ' + response.status);
          return;
        }
        const catalog = await response.json();
        const selected = catalog.selected_cloud_voice_profile || 'a21_doubao_tts_realtime';
        const profiles = catalog.profiles || [];
        if (profiles.length) {
          ui.cloudVoiceProfile.innerHTML = profiles.map((profile) => {
            const id = escapeText(profile.id || '');
            const status = profile.status || 'unknown';
            const selectedAttr = (profile.id || '') === selected ? ' selected' : '';
            return '<option value="' + id + '"' + selectedAttr + '>' + id + ' [' + escapeText(status) + ']</option>';
          }).join('');
        }
        setCloudVoiceProfile(selected);
      } catch (err) {
        log('cloud voice catalog unavailable');
      }
    }
    async function refreshVoiceChainProfiles() {
      try {
        const response = await fetch('/v1/voice-chain-profiles', { cache: 'no-store' });
        if (!response.ok) {
          log('voice chain catalog error ' + response.status);
          return;
        }
        setVoiceChainProfile(await response.json());
      } catch (err) {
        log('voice chain catalog unavailable');
      }
    }
    async function saveVoiceMode() {
      try {
        const response = await fetch('/v1/voice-modes', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ voice_mode: ui.voiceMode.value })
        });
        if (!response.ok) {
          log('voice mode save failed ' + response.status);
          return;
        }
        const catalog = await response.json();
        setVoiceMode(catalog.selected_voice_mode || ui.voiceMode.value);
        refreshRegistry();
      } catch (err) {
        log('voice mode save unavailable');
      }
    }
    async function saveRoleplayProfile() {
      try {
        const response = await fetch('/v1/roleplay-profile', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            scenario: ui.roleplayScenario.value,
            voice_clone_profile: ui.voiceCloneProfile.value
          })
        });
        if (!response.ok) {
          log('roleplay save failed ' + response.status);
          refreshRoleplayProfile();
          return;
        }
        setRoleplayProfile(await response.json());
        refreshVoiceChainProfiles();
        refreshRegistry();
      } catch (err) {
        log('roleplay save unavailable');
      }
    }
    async function saveProfessionalWorkspace() {
      try {
        const response = await fetch('/v1/professional-workspace', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ query_scope: ui.professionalQueryScope.value })
        });
        if (!response.ok) {
          log('professional workspace save failed ' + response.status);
          refreshProfessionalWorkspace();
          return;
        }
        setProfessionalWorkspace(await response.json());
      } catch (err) {
        log('professional workspace save unavailable');
      }
    }
    async function createWorkspaceUploadJob() {
      try {
        const queryScope = ui.professionalQueryScope.value;
        const response = await fetch('/v1/workspace-upload-jobs', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            workspace_id: 'a21_local_workspace',
            user_id: 'a21_local_user',
            source_scope: queryScope === 'public_only' ? 'public' : 'personal',
            source_kind: 'upload',
            document_label: ui.workspaceDocumentLabel.value || 'workspace note',
            content_type: 'application/octet-stream',
            trace_id: sim.traceId || '',
            session_id: sim.sessionId || '',
            device_id: deviceId()
          })
        });
        if (!response.ok) {
          log('workspace job failed ' + response.status);
          return;
        }
        const payload = await response.json();
        const job = (payload.jobs || [])[0] || {};
        log('workspace job ' + (job.job_id || 'accepted') + ' ' + (job.status || 'no_execute'));
      } catch (err) {
        log('workspace job unavailable');
      }
    }
    async function saveGatewayProfile() {
      try {
        const response = await fetch('/v1/gateway-profiles', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ gateway_profile: ui.gatewayProfile.value })
        });
        if (!response.ok) {
          log('gateway profile save failed ' + response.status);
          refreshGatewayProfiles();
          return;
        }
        const catalog = await response.json();
        setGatewayProfile(catalog.selected_gateway_profile || ui.gatewayProfile.value);
        refreshRegistry();
      } catch (err) {
        log('gateway profile save unavailable');
      }
    }
    async function saveCloudVoiceProfile() {
      try {
        const response = await fetch('/v1/cloud-voice-profiles', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ cloud_voice_profile: ui.cloudVoiceProfile.value })
        });
        if (!response.ok) {
          log('cloud voice save failed ' + response.status);
          refreshCloudVoiceProfiles();
          return;
        }
        const catalog = await response.json();
        setCloudVoiceProfile(catalog.selected_cloud_voice_profile || ui.cloudVoiceProfile.value);
        refreshRegistry();
      } catch (err) {
        log('cloud voice save unavailable');
      }
    }
    async function saveVoiceChainProfile() {
      try {
        const response = await fetch('/v1/voice-chain-profiles', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            voice_chain_mode: ui.voiceChainMode.value,
            asr_profile: ui.cascadeASRProfile.value,
            llm_profile: ui.cascadeLLMProfile.value,
            realtime_provider: ui.realtimeProvider.value,
            voice_clone_profile: ui.voiceCloneProfile.value
          })
        });
        if (!response.ok) {
          log('voice chain save failed ' + response.status);
          refreshVoiceChainProfiles();
          return;
        }
        setVoiceChainProfile(await response.json());
        refreshRegistry();
      } catch (err) {
        log('voice chain save unavailable');
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
    function renderWakeWordConfig(config) {
      latestWakeWordConfig = config;
      ui.wakeWordActive.textContent = config.active_phrase || 'none';
      ui.wakeWordStatus.textContent = config.runtime_status || 'unknown';
      ui.wakeWordBuild.textContent = config.firmware_build_required ? 'required' : 'not required';
      ui.wakeWordFirmwareStatus.textContent = config.firmware_status || (config.firmware_build_required ? 'custom_pending_firmware' : 'builtin_active');
      ui.wakeWordHotSwap.textContent = config.runtime_hot_swap_supported ? (config.custom_runtime_active ? 'custom active' : 'supported') : 'disabled';
      ui.wakeWordCode.textContent = config.code || 'none';
      if (config.mode) ui.wakeWordMode.value = config.mode;
      ui.wakeWordPhrase.value = config.desired_phrase || '';
      ui.wakeWordPinyin.value = config.desired_pinyin || '';
      if (config.threshold) ui.wakeWordThreshold.value = String(config.threshold);
    }
    async function refreshWakeWordConfig() {
      try {
        const response = await fetch('/v1/wake-word', { cache: 'no-store' });
        if (!response.ok) {
          ui.wakeWordStatus.textContent = 'error ' + response.status;
          return;
        }
        renderWakeWordConfig(await response.json());
      } catch (err) {
        ui.wakeWordStatus.textContent = 'unavailable';
      }
    }
    async function saveWakeWordConfig() {
      const body = {
        mode: ui.wakeWordMode.value,
        desired_phrase: ui.wakeWordPhrase.value,
        desired_pinyin: ui.wakeWordPinyin.value,
        threshold: Number(ui.wakeWordThreshold.value || 0)
      };
      try {
        const response = await fetch('/v1/wake-word', {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(body)
        });
        if (!response.ok) {
          ui.wakeWordStatus.textContent = 'rejected ' + response.status;
          return;
        }
        const config = await response.json();
        renderWakeWordConfig(config);
        log('wake word config saved ' + (config.runtime_status || 'unknown'));
      } catch (err) {
        ui.wakeWordStatus.textContent = 'unavailable';
      }
    }
    async function resetWakeWordConfig() {
      try {
        const response = await fetch('/v1/wake-word', {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ mode: 'builtin_xiaozhi', threshold: 30 })
        });
        if (!response.ok) {
          ui.wakeWordStatus.textContent = 'rejected ' + response.status;
          return;
        }
        const config = await response.json();
        renderWakeWordConfig(config);
        log('wake word config reset ' + (config.runtime_status || 'unknown'));
      } catch (err) {
        ui.wakeWordStatus.textContent = 'unavailable';
      }
    }
    async function exportWakeWordConfig() {
      if (!latestWakeWordConfig) {
        await refreshWakeWordConfig();
      }
      if (!latestWakeWordConfig) {
        ui.wakeWordStatus.textContent = 'unavailable';
        return;
      }
      const blob = new Blob([JSON.stringify(latestWakeWordConfig, null, 2) + '\n'], { type: 'application/json' });
      const url = URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = 'a21-wake-word-config.json';
      link.click();
      URL.revokeObjectURL(url);
      log('wake word config exported ' + (latestWakeWordConfig.runtime_status || 'unknown'));
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
    document.getElementById('workspaceJob').addEventListener('click', createWorkspaceUploadJob);
    ui.mode.addEventListener('change', () => setMode(ui.mode.value));
    document.getElementById('startMic').addEventListener('click', startMicrophoneStream);
    document.getElementById('stopMic').addEventListener('click', stopMicrophoneStream);
    document.getElementById('mockAudioBurst').addEventListener('click', sendMockAudioBurst);
    document.getElementById('saveWakeWord').addEventListener('click', saveWakeWordConfig);
    ui.resetWakeWord.addEventListener('click', resetWakeWordConfig);
    ui.exportWakeWord.addEventListener('click', exportWakeWordConfig);
    ui.voiceMode.addEventListener('change', saveVoiceMode);
    ui.roleplayScenario.addEventListener('change', saveRoleplayProfile);
    ui.professionalQueryScope.addEventListener('change', saveProfessionalWorkspace);
    ui.gatewayProfile.addEventListener('change', saveGatewayProfile);
    ui.cloudVoiceProfile.addEventListener('change', saveCloudVoiceProfile);
    ui.voiceChainMode.addEventListener('change', saveVoiceChainProfile);
    ui.cascadeASRProfile.addEventListener('change', saveVoiceChainProfile);
    ui.cascadeLLMProfile.addEventListener('change', saveVoiceChainProfile);
    ui.realtimeProvider.addEventListener('change', saveVoiceChainProfile);
    ui.voiceCloneProfile.addEventListener('change', () => {
      saveVoiceChainProfile();
      saveRoleplayProfile();
    });
    refreshRegistry();
    refreshVoiceModes();
    refreshRoleplayProfile();
    refreshProfessionalWorkspace();
    refreshGatewayProfiles();
    refreshCloudVoiceProfiles();
    refreshVoiceChainProfiles();
    refreshWakeWordConfig();
    setMode(ui.mode.value);
    updateVisibilityBadges();
  </script>
</body>
</html>`
