package gateway

const simulatorHTMLPart01 = `<!doctype html>
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
    .registry, .audio-link, .latency-summary, .wake-word, .workspace-audit {
      border: 1px solid var(--line);
      border-radius: 8px;
      padding: 12px;
      background: #141a1d;
      min-width: 0;
    }
    .registry h2, .audio-link h2, .latency-summary h2, .wake-word h2, .workspace-audit h2 {
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
          <button id="workspaceDocumentUpload">Upload Doc</button>
          <button id="workspaceIndexRequest">Index Request</button>
          <button id="workspaceSourcesRefresh">Sources</button>
          <button id="professionalReadRecordsRefresh">Read Records</button>
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
            <option value="a21_voice_default_dashscope">紫悦自然声音</option>
          </select>
          <select id="roleplayProfile" aria-label="roleplay soul profile">
            <option value="a21_roleplay_default">紫悦桌面伙伴</option>
          </select>
          <select id="roleplayScenario" aria-label="roleplay scenario">
            <option value="desk_mouthpiece">desk_mouthpiece</option>
            <option value="boss_challenge">boss_challenge</option>
            <option value="engineer_pushback">engineer_pushback</option>
          </select>
          <input id="roleplayMemoryHint" placeholder="session hint" aria-label="roleplay memory hint">
          <button id="saveRoleplayMemory">Memory</button>
          <button id="clearRoleplayMemory">Clear Memory</button>
          <select id="professionalQueryScope" aria-label="professional query scope">
            <option value="public_only">public_only</option>
            <option value="personal_only">personal_only</option>
            <option value="personal_plus_public">personal_plus_public</option>
          </select>
          <input id="workspaceDocumentLabel" value="PRD pack" aria-label="workspace document label">
          <input id="workspaceDocumentFile" type="file" aria-label="workspace document file">
          <input id="utterance" value="先说，我在" aria-label="utterance">
        </div>
        <div class="readout">
          <div class="metric"><label>State</label><div id="state">idle</div></div>
          <div class="metric"><label>Mode</label><div id="modeReadout">roleplay</div></div>
          <div class="metric"><label>Voice</label><div id="voiceModeReadout">roleplay</div></div>
          <div class="metric"><label>Mode Cue</label><div id="modeRitualReadout">我在。你说，我先接住。</div></div>
          <div class="metric"><label>Gateway</label><div id="gatewayProfileReadout">mac_local</div></div>
          <div class="metric"><label>Cloud Voice</label><div id="cloudVoiceProfileReadout">a21_doubao_tts_realtime</div></div>
          <div class="metric"><label>Chain</label><div id="voiceChainModeReadout">cascade</div></div>
          <div class="metric"><label>ASR</label><div id="cascadeASRProfileReadout">dashscope_qwen_asr_realtime</div></div>
          <div class="metric"><label>LLM</label><div id="cascadeLLMProfileReadout">stepfun</div></div>
          <div class="metric"><label>TTS</label><div id="selectedTTSProfileReadout">dashscope_qwen_tts_realtime</div></div>
          <div class="metric"><label>Realtime</label><div id="realtimeProviderReadout">doubao_realtime</div></div>
          <div class="metric"><label>Voice Name</label><div id="voiceCloneProfileReadout">紫悦自然声音</div></div>
          <div class="metric"><label>Soul</label><div id="roleplaySoulReadout">紫悦桌面伙伴</div></div>
          <div class="metric"><label>Memory</label><div id="roleplayMemoryReadout">empty / 0</div></div>
          <div class="metric"><label>Expression</label><div id="roleplayExpressionReadout">no-send / 0</div></div>
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
        <section class="workspace-audit" aria-label="Workspace Audit">
          <h2>Workspace Audit</h2>
          <div class="registry-grid">
            <div class="metric"><label>Upload Job</label><div id="workspaceJobReadout">none</div></div>
            <div class="metric"><label>Index Job</label><div id="workspaceIndexReadout">none</div></div>
            <div class="metric"><label>Document</label><div id="workspaceDocumentReadout">none</div></div>
            <div class="metric"><label>Storage</label><div id="workspaceDocumentStorage">none</div></div>
            <div class="metric"><label>Source Count</label><div id="workspaceSourceCount">0</div></div>
            <div class="metric"><label>Source Ready</label><div id="workspaceSourceReadiness">none</div></div>
            <div class="metric"><label>Read Records</label><div id="professionalReadRecordCount">0</div></div>
            <div class="metric"><label>Read Status</label><div id="professionalReadRecordStatus">none</div></div>
            <div class="metric"><label>Read Scope</label><div id="professionalReadRecordScope">none</div></div>
            <div class="metric"><label>Sources</label><div id="professionalReadRecordSources">none</div></div>
            <div class="metric"><label>Workspace</label><div id="professionalReadRecordWorkspace">none</div></div>
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
            <div class="metric"><label>Role Soul</label><div id="registryRoleplayProfile">none</div></div>
            <div class="metric"><label>Scenario</label><div id="registryRoleplayScenario">none</div></div>
            <div class="metric"><label>Role Memory</label><div id="registryRoleplayMemory">empty / 0</div></div>
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
      modeRitualReadout: document.getElementById('modeRitualReadout'),
      gatewayProfileReadout: document.getElementById('gatewayProfileReadout'),
      cloudVoiceProfileReadout: document.getElementById('cloudVoiceProfileReadout'),
      voiceChainModeReadout: document.getElementById('voiceChainModeReadout'),
      cascadeASRProfileReadout: document.getElementById('cascadeASRProfileReadout'),
      cascadeLLMProfileReadout: document.getElementById('cascadeLLMProfileReadout'),
      selectedTTSProfileReadout: document.getElementById('selectedTTSProfileReadout'),
      realtimeProviderReadout: document.getElementById('realtimeProviderReadout'),
      voiceCloneProfileReadout: document.getElementById('voiceCloneProfileReadout'),
      roleplaySoulReadout: document.getElementById('roleplaySoulReadout'),
      roleplayMemoryReadout: document.getElementById('roleplayMemoryReadout'),
      roleplayExpressionReadout: document.getElementById('roleplayExpressionReadout'),
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
      workspaceJobReadout: document.getElementById('workspaceJobReadout'),
      workspaceIndexReadout: document.getElementById('workspaceIndexReadout'),
      workspaceDocumentReadout: document.getElementById('workspaceDocumentReadout'),
      workspaceDocumentStorage: document.getElementById('workspaceDocumentStorage'),
      workspaceSourceCount: document.getElementById('workspaceSourceCount'),
      workspaceSourceReadiness: document.getElementById('workspaceSourceReadiness'),
      professionalReadRecordCount: document.getElementById('professionalReadRecordCount'),
      professionalReadRecordStatus: document.getElementById('professionalReadRecordStatus'),
      professionalReadRecordScope: document.getElementById('professionalReadRecordScope'),
      professionalReadRecordSources: document.getElementById('professionalReadRecordSources'),
      professionalReadRecordWorkspace: document.getElementById('professionalReadRecordWorkspace'),
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
      registryRoleplayProfile: document.getElementById('registryRoleplayProfile'),
      registryRoleplayScenario: document.getElementById('registryRoleplayScenario'),
      registryRoleplayMemory: document.getElementById('registryRoleplayMemory'),
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
`
