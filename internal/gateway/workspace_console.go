package gateway

import (
	"fmt"
	"net/http"
)

func (s *Server) handleWorkspaceConsole(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprint(w, workspaceConsoleHTML)
}

const workspaceConsoleHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <link rel="icon" href="data:,">
  <title>A21 Workspace Console</title>
  <style>
    :root {
      color-scheme: light;
      --bg: #f6f7f8;
      --surface: #ffffff;
      --surface-2: #eef2f1;
      --line: #d7dedb;
      --text: #17201d;
      --muted: #65726e;
      --accent: #0f766e;
      --accent-2: #2f5f9a;
      --warn: #9a6a13;
      --danger: #9a3f3f;
      --shadow: 0 12px 32px rgba(18, 32, 28, 0.08);
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
    .shell {
      min-height: 100vh;
      display: grid;
      grid-template-rows: auto 1fr;
    }
    header {
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 18px;
      padding: 18px 24px;
      border-bottom: 1px solid var(--line);
      background: rgba(255,255,255,0.94);
      position: sticky;
      top: 0;
      z-index: 4;
      backdrop-filter: blur(12px);
    }
    h1 {
      margin: 0;
      font-size: 20px;
      line-height: 1.2;
      font-weight: 760;
    }
    .header-meta {
      display: flex;
      align-items: center;
      gap: 10px;
      color: var(--muted);
      font-size: 13px;
      white-space: nowrap;
    }
    .status-dot {
      width: 9px;
      height: 9px;
      border-radius: 999px;
      background: var(--accent);
    }
    main {
      display: grid;
      grid-template-columns: minmax(300px, 0.9fr) minmax(420px, 1.2fr);
      gap: 18px;
      width: min(1280px, calc(100vw - 32px));
      margin: 0 auto;
      padding: 18px 0 28px;
    }
    section {
      background: var(--surface);
      border: 1px solid var(--line);
      border-radius: 8px;
      box-shadow: var(--shadow);
      min-width: 0;
    }
    .panel-head {
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 12px;
      padding: 16px;
      border-bottom: 1px solid var(--line);
    }
    h2 {
      margin: 0;
      font-size: 15px;
      line-height: 1.25;
      font-weight: 730;
    }
    .panel-body {
      padding: 16px;
      display: grid;
      gap: 14px;
    }
    .grid {
      display: grid;
      grid-template-columns: repeat(2, minmax(0, 1fr));
      gap: 12px;
    }
    label {
      display: grid;
      gap: 6px;
      color: var(--muted);
      font-size: 12px;
      font-weight: 650;
    }
    input, select, button {
      font: inherit;
      letter-spacing: 0;
    }
    input, select {
      width: 100%;
      border: 1px solid var(--line);
      border-radius: 8px;
      background: #fff;
      color: var(--text);
      padding: 10px 11px;
      font-size: 14px;
      min-height: 40px;
    }
    input[type="file"] {
      padding: 7px;
      color: var(--muted);
    }
    input[type="range"] {
      min-height: 40px;
      padding: 0;
      accent-color: var(--accent);
    }
    input[type="file"]::file-selector-button {
      border: 1px solid var(--line);
      border-radius: 6px;
      background: var(--surface-2);
      color: var(--text);
      min-height: 28px;
      padding: 4px 9px;
      margin-right: 10px;
      font: inherit;
      font-size: 13px;
      font-weight: 660;
      cursor: pointer;
    }
    button {
      border: 1px solid transparent;
      border-radius: 8px;
      background: var(--text);
      color: #fff;
      min-height: 40px;
      padding: 9px 13px;
      font-size: 14px;
      font-weight: 680;
      cursor: pointer;
    }
    button.secondary {
      background: #fff;
      color: var(--text);
      border-color: var(--line);
    }
    button:disabled {
      cursor: not-allowed;
      opacity: 0.5;
    }
    .actions {
      display: flex;
      flex-wrap: wrap;
      gap: 10px;
    }
    .status-strip {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
      gap: 10px;
    }
    .metric {
      border: 1px solid var(--line);
      border-radius: 8px;
      background: var(--surface-2);
      padding: 10px;
      min-width: 0;
    }
    .metric span {
      display: block;
      color: var(--muted);
      font-size: 11px;
      font-weight: 700;
      text-transform: uppercase;
      margin-bottom: 5px;
    }
    .metric strong {
      display: block;
      font-size: 13px;
      font-weight: 720;
      line-height: 1.35;
      overflow-wrap: anywhere;
    }
    .stack {
      display: grid;
      gap: 12px;
    }
    .row-list {
      display: grid;
      gap: 10px;
    }
    .row {
      display: grid;
      grid-template-columns: minmax(0, 1.15fr) minmax(0, 0.85fr);
      gap: 12px;
      align-items: center;
      border: 1px solid var(--line);
      border-radius: 8px;
      padding: 11px;
      background: #fff;
    }
    .row-title {
      display: block;
      font-weight: 720;
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
    }
    .row-sub {
      display: block;
      color: var(--muted);
      font-size: 12px;
      margin-top: 3px;
      line-height: 1.35;
      overflow-wrap: anywhere;
    }
    .tagline {
      display: flex;
      flex-wrap: wrap;
      gap: 8px;
    }
    .tag {
      border: 1px solid var(--line);
      border-radius: 8px;
      padding: 6px 8px;
      font-size: 12px;
      color: var(--muted);
      background: #fff;
    }
    .tag.ready { color: var(--accent); border-color: rgba(15,118,110,0.35); }
    .tag.warn { color: var(--warn); border-color: rgba(154,106,19,0.35); }
    .tag.off { color: var(--danger); border-color: rgba(154,63,63,0.28); }
    .log {
      min-height: 112px;
      max-height: 180px;
      overflow: auto;
      border: 1px solid var(--line);
      border-radius: 8px;
      background: #0f1513;
      color: #dce7e2;
      padding: 10px;
      font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
      font-size: 12px;
      line-height: 1.5;
      white-space: pre-wrap;
    }
    .wide { grid-column: 1 / -1; }
    .mode-band {
      display: grid;
      grid-template-columns: repeat(2, minmax(0, 1fr));
      gap: 12px;
    }
    @media (max-width: 900px) {
      header { align-items: flex-start; flex-direction: column; }
      main { grid-template-columns: 1fr; width: min(100vw - 24px, 760px); }
      .status-strip, .grid, .mode-band { grid-template-columns: 1fr; }
      .row { grid-template-columns: 1fr; }
    }
  </style>
</head>
<body>
  <div class="shell" data-testid="workspace-console-root">
    <header>
      <h1>A21 Workspace Console</h1>
      <div class="header-meta"><span class="status-dot"></span><span id="serviceStatus">gateway contract</span></div>
    </header>
    <main>
      <section aria-label="Workspace setup">
        <div class="panel-head">
          <h2>Workspace</h2>
          <button class="secondary" id="refreshWorkspace">Refresh</button>
        </div>
        <div class="panel-body">
          <div class="grid">
            <label>Query scope
              <select id="queryScope">
                <option value="public_only">public_only</option>
                <option value="personal_only">personal_only</option>
                <option value="personal_plus_public">personal_plus_public</option>
              </select>
            </label>
            <label>Document label
              <input id="documentLabel" value="A21 PRD pack" autocomplete="off">
            </label>
            <label>Device
              <input id="deviceId" value="stackchan-sim-001" autocomplete="off">
            </label>
          </div>
          <label>Document
            <input id="documentFile" type="file">
          </label>
          <div class="actions">
            <button id="uploadDocument">Upload</button>
            <button id="requestIndex">Request index</button>
            <button class="secondary" id="bindDevice">Bind device</button>
            <button class="secondary" id="revokeDevice">Revoke device</button>
            <button class="secondary" id="refreshDeviceBindings">Device bindings</button>
            <button class="secondary" id="deleteSource">Delete source</button>
            <button class="secondary" id="exportMetadata">Export metadata</button>
            <button class="secondary" id="refreshSources">Sources</button>
            <button class="secondary" id="refreshReads">Read records</button>
          </div>
          <div class="status-strip">
            <div class="metric"><span>Storage</span><strong id="storageStatus">stored_local pending</strong></div>
            <div class="metric"><span>Index</span><strong id="indexStatus">not_started_no_execute</strong></div>
            <div class="metric"><span>Searchable</span><strong id="searchableStatus">false</strong></div>
            <div class="metric"><span>Device binding</span><strong id="deviceBindingStatus">binding_not_configured</strong></div>
          </div>
        </div>
      </section>

      <section aria-label="Readiness">
        <div class="panel-head">
          <h2>Readiness</h2>
          <div class="tagline">
            <span class="tag ready" id="workspaceStatus">contract_ready</span>
            <span class="tag warn" id="adapterStatus">a21.v21_adapter_query.v2</span>
          </div>
        </div>
        <div class="panel-body">
          <div class="status-strip">
            <div class="metric"><span>User</span><strong id="userId">a21_local_user</strong></div>
            <div class="metric"><span>Workspace</span><strong id="workspaceId">a21_local_workspace</strong></div>
            <div class="metric"><span>Sources</span><strong id="sourceCount">0</strong></div>
            <div class="metric"><span>Devices</span><strong id="deviceBindingCount">0</strong></div>
            <div class="metric"><span>Reads</span><strong id="readCount">0</strong></div>
          </div>
          <div class="grid">
            <label>Read record id
              <input id="readRecordFilter" placeholder="record_id" autocomplete="off">
            </label>
            <label>Read trace id
              <input id="readTraceFilter" placeholder="trace_id" autocomplete="off">
            </label>
            <label>Read session id
              <input id="readSessionFilter" placeholder="session_id" autocomplete="off">
            </label>
            <div class="actions">
              <button class="secondary" id="clearReadFilters">Clear filters</button>
            </div>
          </div>
          <div class="stack">
            <div class="row-list" id="sourceList" aria-label="Workspace sources"></div>
            <div class="row-list" id="readList" aria-label="Professional read records"></div>
          </div>
        </div>
      </section>

      <section class="wide" aria-label="Roleplay setup">
        <div class="panel-head">
          <h2>Roleplay Setup</h2>
          <div class="tagline">
            <span class="tag ready" id="roleplayPromptStatus">prompt_input_ready=false</span>
            <span class="tag warn" id="roleplayPhysicalStatus">physical_accepted=false</span>
          </div>
        </div>
        <div class="panel-body">
          <div class="grid">
            <label>Role soul
              <select id="roleplayProfileSelect"></select>
            </label>
            <label>Scenario
              <select id="roleplayScenarioSelect"></select>
            </label>
            <label>Voice profile
              <select id="roleplayVoiceSelect"></select>
            </label>
            <label>Memory hint
              <input id="roleplayMemoryHint" placeholder="bounded session hint" autocomplete="off">
            </label>
          </div>
          <div class="actions">
            <button id="saveRoleplaySetup">Save roleplay</button>
            <button class="secondary" id="clearRoleplayMemory">Clear memory</button>
          </div>
          <div class="status-strip">
            <div class="metric"><span>Role soul</span><strong id="roleplayProfileStatus">a21_roleplay_default</strong></div>
            <div class="metric"><span>Scenario</span><strong id="roleplayScenarioStatus">desk_mouthpiece</strong></div>
            <div class="metric"><span>Voice profile</span><strong id="roleplayVoiceStatus">a21_voice_default_dashscope</strong></div>
            <div class="metric"><span>Memory</span><strong id="roleplayMemoryStatus">empty / 0</strong></div>
          </div>
        </div>
      </section>

      <section class="wide" aria-label="Voice chain setup">
        <div class="panel-head">
          <h2>Voice Chain Setup</h2>
          <div class="tagline">
            <span class="tag ready" id="voiceChainHotSwitchStatus">hot_switch=true</span>
            <span class="tag warn" id="voiceChainFindingStatus">stepfun_not_selected</span>
          </div>
        </div>
        <div class="panel-body">
          <div class="grid">
            <label>Chain mode
              <select id="voiceChainModeSelect"></select>
            </label>
            <label>ASR profile
              <select id="voiceChainASRSelect"></select>
            </label>
            <label>LLM profile
              <select id="voiceChainLLMSelect"></select>
            </label>
            <label>Realtime provider
              <select id="voiceChainRealtimeSelect"></select>
            </label>
          </div>
          <div class="actions">
            <button id="saveVoiceChainSetup">Save voice chain</button>
          </div>
          <div class="status-strip">
            <div class="metric"><span>Chain</span><strong id="voiceChainModeStatus">cascade</strong></div>
            <div class="metric"><span>ASR</span><strong id="voiceChainASRStatus">dashscope_qwen_asr_realtime</strong></div>
            <div class="metric"><span>LLM</span><strong id="voiceChainLLMStatus">stepfun</strong></div>
            <div class="metric"><span>Effective TTS</span><strong id="voiceChainTTSStatus">dashscope_qwen_tts_realtime</strong></div>
            <div class="metric"><span>Realtime</span><strong id="voiceChainRealtimeStatus">doubao_realtime</strong></div>
          </div>
        </div>
      </section>

      <section class="wide" aria-label="Wake word setup">
        <div class="panel-head">
          <h2>Wake Word Setup</h2>
          <div class="tagline">
            <span class="tag warn" id="wakeWordBuildStatus">build_required=false</span>
            <span class="tag off" id="wakeWordHotSwapStatus">runtime_hot_swap=false</span>
          </div>
        </div>
        <div class="panel-body">
          <div class="grid">
            <label>Wake mode
              <select id="wakeWordModeSelect">
                <option value="builtin_xiaozhi">builtin_xiaozhi</option>
                <option value="custom_multinet">custom_multinet</option>
              </select>
            </label>
            <label>Desired phrase
              <input id="wakeWordPhrase" value="小阿二一" autocomplete="off">
            </label>
            <label>Desired pinyin
              <input id="wakeWordPinyin" value="xiao a er yi" autocomplete="off">
            </label>
            <label>Threshold
              <input id="wakeWordThreshold" type="number" min="1" max="100" value="30">
            </label>
          </div>
          <div class="actions">
            <button id="saveWakeWordSetup">Save wake word</button>
            <button class="secondary" id="resetWakeWordSetup">Reset builtin</button>
          </div>
          <div class="status-strip">
            <div class="metric"><span>Active phrase</span><strong id="wakeWordActiveStatus">你好小智</strong></div>
            <div class="metric"><span>Runtime</span><strong id="wakeWordRuntimeStatus">active_builtin_model</strong></div>
            <div class="metric"><span>Firmware</span><strong id="wakeWordFirmwareStatus">builtin_active</strong></div>
            <div class="metric"><span>Code</span><strong id="wakeWordCodeStatus">none</strong></div>
          </div>
        </div>
      </section>

      <section class="wide" aria-label="Voice probe">
        <div class="panel-head">
          <h2>Voice Probe</h2>
          <div class="tagline">
            <span class="tag ready" id="voiceProbeRouteStatus">route=idle</span>
            <span class="tag warn" id="voiceProbeTraceStatus">trace=none</span>
          </div>
        </div>
        <div class="panel-body">
          <div class="grid">
            <label>Probe mode
              <select id="voiceProbeModeSelect">
                <option value="roleplay">roleplay</option>
                <option value="professional">professional</option>
              </select>
            </label>
            <label>Probe cue
              <input id="voiceProbeInput" placeholder="safe short cue" autocomplete="off">
            </label>
          </div>
          <div class="actions">
            <button id="runRoleplayProbe">Run roleplay probe</button>
            <button id="runProfessionalProbe">Run professional probe</button>
            <button class="secondary" id="refreshVoiceProbeTrace">Trace markers</button>
          </div>
          <div class="status-strip">
            <div class="metric"><span>Role soul</span><strong id="voiceProbeRoleplayStatus">none</strong></div>
            <div class="metric"><span>Voice profile</span><strong id="voiceProbeVoiceStatus">none</strong></div>
            <div class="metric"><span>Memory</span><strong id="voiceProbeMemoryStatus">none</strong></div>
            <div class="metric"><span>Professional</span><strong id="voiceProbeProfessionalStatus">idle</strong></div>
          </div>
          <div class="stack">
            <div class="row-list" id="voiceProbeTraceList" aria-label="Voice probe trace markers"></div>
            <div class="row-list" id="voiceProbeReadList" aria-label="Voice probe read records"></div>
          </div>
        </div>
      </section>

      <section class="wide" aria-label="Body presets">
        <div class="panel-head">
          <h2>Body Presets</h2>
          <div class="tagline">
            <span class="tag ready" id="bodyPresetStatus">preset=idle</span>
            <span class="tag warn" id="bodyPresetPhysicalStatus">physical_accepted=false</span>
          </div>
        </div>
        <div class="panel-body">
          <div class="actions" id="bodyPresetActions">
            <button class="secondary" data-body-preset="ready">Ready</button>
            <button class="secondary" data-body-preset="listening">Listen</button>
            <button class="secondary" data-body-preset="thinking">Think</button>
            <button class="secondary" data-body-preset="speaking">Speak</button>
            <button data-body-preset="celebrate">Celebrate</button>
            <button class="secondary" data-body-preset="reset_idle">Reset</button>
            <button class="secondary" data-body-motion="look_up">Look up</button>
            <button class="secondary" data-body-motion="nod">Nod</button>
            <button class="secondary" data-body-motion="shake">Shake</button>
            <button class="secondary" data-body-motion="dance">MCP dance</button>
            <button class="secondary" data-body-motion="stop">Stop motion</button>
            <button class="secondary" id="refreshBodyPresetTrace">Trace markers</button>
          </div>
          <div class="status-strip">
            <div class="metric"><span>Trace</span><strong id="bodyPresetTraceStatus">trace=none</strong></div>
            <div class="metric"><span>LED</span><strong id="bodyPresetLEDStatus">rgb=none</strong></div>
            <div class="metric"><span>Head</span><strong id="bodyPresetHeadStatus">pose=none</strong></div>
            <div class="metric"><span>Transport</span><strong id="bodyPresetTransportStatus">xiaozhi_mcp_sequence</strong></div>
          </div>
          <div class="row-list" id="bodyPresetTraceList" aria-label="Body preset trace markers"></div>
        </div>
      </section>

      <section class="wide" aria-label="Hardware scenes">
        <div class="panel-head">
          <h2>Hardware Scenes</h2>
          <div class="tagline">
            <span class="tag ready" id="hardwareSceneStatus">scene=idle</span>
            <span class="tag warn" id="hardwareScenePhysicalStatus">physical_accepted=false</span>
          </div>
        </div>
        <div class="panel-body">
          <div class="actions" id="hardwareSceneActions">
            <button data-hardware-scene="full_check">Full Check</button>
            <button class="secondary" data-hardware-scene="showtime">Showtime</button>
            <button class="secondary" data-hardware-scene="focus">Focus</button>
            <button class="secondary" data-hardware-scene="reset">Reset</button>
            <button class="secondary" id="acceptHardwareScenePhysical">Accept Visible Full Check</button>
            <button class="secondary" id="refreshHardwareSceneTrace">Trace markers</button>
          </div>
          <div class="status-strip">
            <div class="metric"><span>Trace</span><strong id="hardwareSceneTraceStatus">trace=none</strong></div>
            <div class="metric"><span>Screen</span><strong id="hardwareSceneScreenStatus">screen=none</strong></div>
            <div class="metric"><span>Body</span><strong id="hardwareSceneBodyStatus">body=none</strong></div>
            <div class="metric"><span>Steps</span><strong id="hardwareSceneStepStatus">steps=0</strong></div>
            <div class="metric"><span>Transport</span><strong id="hardwareSceneTransportStatus">xiaozhi_mcp_sequence</strong></div>
            <div class="metric"><span>Acceptance</span><strong id="hardwareSceneAcceptanceStatus">operator_pending</strong></div>
          </div>
          <div class="row-list" id="hardwareSceneTraceList" aria-label="Hardware scene trace markers"></div>
        </div>
      </section>

      <section class="wide" aria-label="Hardware screen">
        <div class="panel-head">
          <h2>Hardware Screen</h2>
          <div class="tagline">
            <span class="tag ready" id="hardwareScreenStatus">screen=idle</span>
            <span class="tag warn" id="hardwareScreenPhysicalStatus">physical_accepted=false</span>
          </div>
        </div>
        <div class="panel-body">
          <div class="grid">
            <label>Brightness
              <input id="screenBrightness" type="range" min="0" max="100" step="1" value="55">
            </label>
            <label>Level
              <input id="screenBrightnessValue" value="55" readonly>
            </label>
          </div>
          <div class="actions">
            <button id="applyScreenBrightness">Apply brightness</button>
            <button class="secondary" data-screen-theme="light">Light</button>
            <button class="secondary" data-screen-theme="dark">Dark</button>
            <button class="secondary" data-screen-theme="auto">Auto</button>
            <button class="secondary" id="runDeviceStatus">Device status</button>
            <button class="secondary" id="runScreenInfo">Screen info</button>
            <button class="secondary" id="refreshMCPCapabilities">Capabilities</button>
            <button class="secondary" id="refreshHardwareScreenTrace">Trace markers</button>
          </div>
          <div class="status-strip">
            <div class="metric"><span>Trace</span><strong id="hardwareScreenTraceStatus">trace=none</strong></div>
            <div class="metric"><span>Brightness</span><strong id="screenBrightnessStatus">brightness=55</strong></div>
            <div class="metric"><span>Theme</span><strong id="screenThemeStatus">theme=none</strong></div>
            <div class="metric"><span>Tool</span><strong id="screenToolStatus">tool=none</strong></div>
            <div class="metric"><span>Capabilities</span><strong id="mcpCapabilitiesStatus">mcp=unknown</strong></div>
          </div>
          <div class="row-list" id="hardwareScreenTraceList" aria-label="Hardware screen trace markers"></div>
        </div>
      </section>

      <section class="wide" aria-label="Official actions">
        <div class="panel-head">
          <h2>Official Actions</h2>
          <div class="tagline">
            <span class="tag ready" id="officialActionStatus">action=idle</span>
            <span class="tag warn" id="officialActionPhysicalStatus">physical_accepted=false</span>
          </div>
        </div>
        <div class="panel-body">
          <div class="grid">
            <label>Pitch angle
              <input id="officialActionYAngle" type="range" min="5" max="85" step="1" value="38">
            </label>
            <label>Angle
              <input id="officialActionYAngleValue" value="38" readonly>
            </label>
          </div>
          <div class="actions" id="officialActionControls">
            <button class="secondary" data-official-state="idle">Idle</button>
            <button class="secondary" data-official-state="listening">Listening</button>
            <button class="secondary" data-official-state="thinking">Thinking</button>
            <button class="secondary" data-official-state="speaking">Speaking</button>
            <button class="secondary" data-official-face="happy">Happy</button>
            <button class="secondary" data-official-face="attentive">Attentive</button>
            <button class="secondary" data-official-motion="look_up">Look up</button>
            <button class="secondary" data-official-motion="nod">Nod</button>
            <button class="secondary" data-official-motion="shake">Shake</button>
            <button data-official-motion="dance">Dance</button>
            <button class="secondary" data-official-motion="stop">Stop</button>
            <button class="secondary" id="refreshOfficialActionTrace">Trace markers</button>
          </div>
          <div class="status-strip">
            <div class="metric"><span>Trace</span><strong id="officialActionTraceStatus">trace=none</strong></div>
            <div class="metric"><span>Event</span><strong id="officialActionEventStatus">event=none</strong></div>
            <div class="metric"><span>Packets</span><strong id="officialActionPacketStatus">packets=0</strong></div>
            <div class="metric"><span>Transport</span><strong id="officialActionTransportStatus">stackchan_official_ws</strong></div>
            <div class="metric"><span>Surfaces</span><strong id="officialActionSurfaceStatus">surfaces=none</strong></div>
          </div>
          <div class="row-list" id="officialActionTraceList" aria-label="Official action trace markers"></div>
        </div>
      </section>

      <section class="wide" aria-label="Mode boundary">
        <div class="panel-head">
          <h2>Mode Boundary</h2>
          <div class="tagline">
            <span class="tag ready">roleplay</span>
            <span class="tag warn">professional</span>
            <span class="tag off">v21_execution_allowed=false</span>
          </div>
        </div>
        <div class="panel-body">
          <div class="actions" id="modeRitualActions">
            <button class="secondary" data-mode-ritual="roleplay">Run Roleplay Ritual</button>
            <button data-mode-ritual="professional">Run Professional Ritual</button>
            <button class="secondary" id="acceptModeRitualPhysical">Accept Visible Mode Ritual</button>
          </div>
          <div class="mode-band">
            <div class="metric"><span>Roleplay expression</span><strong id="roleplayExpression">no_send_plan_only</strong></div>
            <div class="metric"><span>Professional cue</span><strong id="professionalCue">PRO / checking</strong></div>
            <div class="metric"><span>Mode ritual</span><strong id="modeRitualStatus">ritual=idle</strong></div>
            <div class="metric"><span>Ritual trace</span><strong id="modeRitualTraceStatus">trace=none</strong></div>
            <div class="metric"><span>Physical</span><strong id="modeRitualPhysicalStatus">physical_accepted=false</strong></div>
          </div>
          <div class="log" id="eventLog" aria-label="Workspace event log">workspace console ready</div>
        </div>
      </section>

      <section class="wide" aria-label="Hardware acceptance">
        <div class="panel-head">
          <h2>Acceptance Board</h2>
          <div class="tagline">
            <span class="tag warn" id="hardwareAcceptanceStatus">physical_pending</span>
            <span class="tag off" id="hardwareAcceptanceDeviceStatus">device=unknown</span>
          </div>
        </div>
        <div class="panel-body">
          <div class="actions">
            <button class="secondary" id="refreshHardwareAcceptance">Refresh Acceptance</button>
          </div>
          <div class="status-strip">
            <div class="metric"><span>Device</span><strong id="hardwareAcceptanceDevice">device=none</strong></div>
            <div class="metric"><span>Connection</span><strong id="hardwareAcceptanceConnection">missing</strong></div>
            <div class="metric"><span>Physical</span><strong id="hardwareAcceptancePhysical">physical_accepted=false</strong></div>
            <div class="metric"><span>Next</span><strong id="hardwareAcceptanceNext">run_full_check</strong></div>
          </div>
          <div class="row-list" id="hardwareAcceptanceItems" aria-label="Hardware acceptance items"></div>
        </div>
      </section>
    </main>
  </div>
  <script>
    const state = {
      document: null,
      job: null,
      source: null,
      indexJob: null,
      sources: [],
      deviceBindings: [],
      deviceBinding: null,
      readRecords: [],
      workspace: null,
      roleplay: null,
      voiceCatalog: null,
      wakeWord: null,
      voiceModes: null,
      voiceProbe: null,
      bodyPreset: null,
      bodyMotion: null,
      hardwareScene: null,
      hardwareAcceptance: null,
      hardwareScreen: null,
      officialAction: null,
      lastExport: null
    };
    const ui = {
      queryScope: document.getElementById('queryScope'),
      documentLabel: document.getElementById('documentLabel'),
      deviceId: document.getElementById('deviceId'),
      documentFile: document.getElementById('documentFile'),
      uploadDocument: document.getElementById('uploadDocument'),
      requestIndex: document.getElementById('requestIndex'),
      bindDevice: document.getElementById('bindDevice'),
      revokeDevice: document.getElementById('revokeDevice'),
      refreshDeviceBindings: document.getElementById('refreshDeviceBindings'),
      deleteSource: document.getElementById('deleteSource'),
      exportMetadata: document.getElementById('exportMetadata'),
      refreshWorkspace: document.getElementById('refreshWorkspace'),
      refreshSources: document.getElementById('refreshSources'),
      refreshReads: document.getElementById('refreshReads'),
      readRecordFilter: document.getElementById('readRecordFilter'),
      readTraceFilter: document.getElementById('readTraceFilter'),
      readSessionFilter: document.getElementById('readSessionFilter'),
      clearReadFilters: document.getElementById('clearReadFilters'),
      roleplayProfileSelect: document.getElementById('roleplayProfileSelect'),
      roleplayScenarioSelect: document.getElementById('roleplayScenarioSelect'),
      roleplayVoiceSelect: document.getElementById('roleplayVoiceSelect'),
      roleplayMemoryHint: document.getElementById('roleplayMemoryHint'),
      saveRoleplaySetup: document.getElementById('saveRoleplaySetup'),
      clearRoleplayMemory: document.getElementById('clearRoleplayMemory'),
      roleplayPromptStatus: document.getElementById('roleplayPromptStatus'),
      roleplayPhysicalStatus: document.getElementById('roleplayPhysicalStatus'),
      roleplayProfileStatus: document.getElementById('roleplayProfileStatus'),
      roleplayScenarioStatus: document.getElementById('roleplayScenarioStatus'),
      roleplayVoiceStatus: document.getElementById('roleplayVoiceStatus'),
      roleplayMemoryStatus: document.getElementById('roleplayMemoryStatus'),
      voiceChainModeSelect: document.getElementById('voiceChainModeSelect'),
      voiceChainASRSelect: document.getElementById('voiceChainASRSelect'),
      voiceChainLLMSelect: document.getElementById('voiceChainLLMSelect'),
      voiceChainRealtimeSelect: document.getElementById('voiceChainRealtimeSelect'),
      saveVoiceChainSetup: document.getElementById('saveVoiceChainSetup'),
      voiceChainHotSwitchStatus: document.getElementById('voiceChainHotSwitchStatus'),
      voiceChainFindingStatus: document.getElementById('voiceChainFindingStatus'),
      voiceChainModeStatus: document.getElementById('voiceChainModeStatus'),
      voiceChainASRStatus: document.getElementById('voiceChainASRStatus'),
      voiceChainLLMStatus: document.getElementById('voiceChainLLMStatus'),
      voiceChainTTSStatus: document.getElementById('voiceChainTTSStatus'),
      voiceChainRealtimeStatus: document.getElementById('voiceChainRealtimeStatus'),
      wakeWordModeSelect: document.getElementById('wakeWordModeSelect'),
      wakeWordPhrase: document.getElementById('wakeWordPhrase'),
      wakeWordPinyin: document.getElementById('wakeWordPinyin'),
      wakeWordThreshold: document.getElementById('wakeWordThreshold'),
      saveWakeWordSetup: document.getElementById('saveWakeWordSetup'),
      resetWakeWordSetup: document.getElementById('resetWakeWordSetup'),
      wakeWordBuildStatus: document.getElementById('wakeWordBuildStatus'),
      wakeWordHotSwapStatus: document.getElementById('wakeWordHotSwapStatus'),
      wakeWordActiveStatus: document.getElementById('wakeWordActiveStatus'),
      wakeWordRuntimeStatus: document.getElementById('wakeWordRuntimeStatus'),
      wakeWordFirmwareStatus: document.getElementById('wakeWordFirmwareStatus'),
      wakeWordCodeStatus: document.getElementById('wakeWordCodeStatus'),
      voiceProbeModeSelect: document.getElementById('voiceProbeModeSelect'),
      voiceProbeInput: document.getElementById('voiceProbeInput'),
      runRoleplayProbe: document.getElementById('runRoleplayProbe'),
      runProfessionalProbe: document.getElementById('runProfessionalProbe'),
      refreshVoiceProbeTrace: document.getElementById('refreshVoiceProbeTrace'),
      voiceProbeRouteStatus: document.getElementById('voiceProbeRouteStatus'),
      voiceProbeTraceStatus: document.getElementById('voiceProbeTraceStatus'),
      voiceProbeRoleplayStatus: document.getElementById('voiceProbeRoleplayStatus'),
      voiceProbeVoiceStatus: document.getElementById('voiceProbeVoiceStatus'),
      voiceProbeMemoryStatus: document.getElementById('voiceProbeMemoryStatus'),
      voiceProbeProfessionalStatus: document.getElementById('voiceProbeProfessionalStatus'),
      voiceProbeTraceList: document.getElementById('voiceProbeTraceList'),
      voiceProbeReadList: document.getElementById('voiceProbeReadList'),
      bodyPresetActions: document.getElementById('bodyPresetActions'),
      refreshBodyPresetTrace: document.getElementById('refreshBodyPresetTrace'),
      bodyPresetStatus: document.getElementById('bodyPresetStatus'),
      bodyPresetPhysicalStatus: document.getElementById('bodyPresetPhysicalStatus'),
      bodyPresetTraceStatus: document.getElementById('bodyPresetTraceStatus'),
      bodyPresetLEDStatus: document.getElementById('bodyPresetLEDStatus'),
      bodyPresetHeadStatus: document.getElementById('bodyPresetHeadStatus'),
      bodyPresetTransportStatus: document.getElementById('bodyPresetTransportStatus'),
      bodyPresetTraceList: document.getElementById('bodyPresetTraceList'),
      hardwareSceneActions: document.getElementById('hardwareSceneActions'),
      acceptHardwareScenePhysical: document.getElementById('acceptHardwareScenePhysical'),
      refreshHardwareSceneTrace: document.getElementById('refreshHardwareSceneTrace'),
      hardwareSceneStatus: document.getElementById('hardwareSceneStatus'),
      hardwareScenePhysicalStatus: document.getElementById('hardwareScenePhysicalStatus'),
      hardwareSceneTraceStatus: document.getElementById('hardwareSceneTraceStatus'),
      hardwareSceneScreenStatus: document.getElementById('hardwareSceneScreenStatus'),
      hardwareSceneBodyStatus: document.getElementById('hardwareSceneBodyStatus'),
      hardwareSceneStepStatus: document.getElementById('hardwareSceneStepStatus'),
      hardwareSceneTransportStatus: document.getElementById('hardwareSceneTransportStatus'),
      hardwareSceneAcceptanceStatus: document.getElementById('hardwareSceneAcceptanceStatus'),
      hardwareSceneTraceList: document.getElementById('hardwareSceneTraceList'),
      screenBrightness: document.getElementById('screenBrightness'),
      screenBrightnessValue: document.getElementById('screenBrightnessValue'),
      applyScreenBrightness: document.getElementById('applyScreenBrightness'),
      runDeviceStatus: document.getElementById('runDeviceStatus'),
      runScreenInfo: document.getElementById('runScreenInfo'),
      refreshMCPCapabilities: document.getElementById('refreshMCPCapabilities'),
      refreshHardwareScreenTrace: document.getElementById('refreshHardwareScreenTrace'),
      hardwareScreenStatus: document.getElementById('hardwareScreenStatus'),
      hardwareScreenPhysicalStatus: document.getElementById('hardwareScreenPhysicalStatus'),
      hardwareScreenTraceStatus: document.getElementById('hardwareScreenTraceStatus'),
      screenBrightnessStatus: document.getElementById('screenBrightnessStatus'),
      screenThemeStatus: document.getElementById('screenThemeStatus'),
      screenToolStatus: document.getElementById('screenToolStatus'),
      mcpCapabilitiesStatus: document.getElementById('mcpCapabilitiesStatus'),
      hardwareScreenTraceList: document.getElementById('hardwareScreenTraceList'),
      officialActionControls: document.getElementById('officialActionControls'),
      officialActionYAngle: document.getElementById('officialActionYAngle'),
      officialActionYAngleValue: document.getElementById('officialActionYAngleValue'),
      refreshOfficialActionTrace: document.getElementById('refreshOfficialActionTrace'),
      officialActionStatus: document.getElementById('officialActionStatus'),
      officialActionPhysicalStatus: document.getElementById('officialActionPhysicalStatus'),
      officialActionTraceStatus: document.getElementById('officialActionTraceStatus'),
      officialActionEventStatus: document.getElementById('officialActionEventStatus'),
      officialActionPacketStatus: document.getElementById('officialActionPacketStatus'),
      officialActionTransportStatus: document.getElementById('officialActionTransportStatus'),
      officialActionSurfaceStatus: document.getElementById('officialActionSurfaceStatus'),
      officialActionTraceList: document.getElementById('officialActionTraceList'),
      storageStatus: document.getElementById('storageStatus'),
      indexStatus: document.getElementById('indexStatus'),
      searchableStatus: document.getElementById('searchableStatus'),
      deviceBindingStatus: document.getElementById('deviceBindingStatus'),
      workspaceStatus: document.getElementById('workspaceStatus'),
      adapterStatus: document.getElementById('adapterStatus'),
      userId: document.getElementById('userId'),
      workspaceId: document.getElementById('workspaceId'),
      sourceCount: document.getElementById('sourceCount'),
      deviceBindingCount: document.getElementById('deviceBindingCount'),
      readCount: document.getElementById('readCount'),
      sourceList: document.getElementById('sourceList'),
      readList: document.getElementById('readList'),
      roleplayExpression: document.getElementById('roleplayExpression'),
      professionalCue: document.getElementById('professionalCue'),
      modeRitualActions: document.getElementById('modeRitualActions'),
      acceptModeRitualPhysical: document.getElementById('acceptModeRitualPhysical'),
      modeRitualStatus: document.getElementById('modeRitualStatus'),
      modeRitualTraceStatus: document.getElementById('modeRitualTraceStatus'),
      modeRitualPhysicalStatus: document.getElementById('modeRitualPhysicalStatus'),
      refreshHardwareAcceptance: document.getElementById('refreshHardwareAcceptance'),
      hardwareAcceptanceStatus: document.getElementById('hardwareAcceptanceStatus'),
      hardwareAcceptanceDeviceStatus: document.getElementById('hardwareAcceptanceDeviceStatus'),
      hardwareAcceptanceDevice: document.getElementById('hardwareAcceptanceDevice'),
      hardwareAcceptanceConnection: document.getElementById('hardwareAcceptanceConnection'),
      hardwareAcceptancePhysical: document.getElementById('hardwareAcceptancePhysical'),
      hardwareAcceptanceNext: document.getElementById('hardwareAcceptanceNext'),
      hardwareAcceptanceItems: document.getElementById('hardwareAcceptanceItems'),
      eventLog: document.getElementById('eventLog'),
      serviceStatus: document.getElementById('serviceStatus')
    };
    function log(message) {
      const line = new Date().toLocaleTimeString() + '  ' + message;
      ui.eventLog.textContent = (ui.eventLog.textContent ? ui.eventLog.textContent + '\n' : '') + line;
      ui.eventLog.scrollTop = ui.eventLog.scrollHeight;
    }
    async function fetchJSON(url, options) {
      const response = await fetch(url, options || { cache: 'no-store' });
      const text = await response.text();
      let payload = {};
      if (text) {
        try { payload = JSON.parse(text); } catch (_) { payload = { raw: text }; }
      }
      if (!response.ok) {
        throw new Error((payload && (payload.error || payload.status)) || ('http ' + response.status));
      }
      return payload;
    }
    function postJSON(url, body) {
      return fetchJSON(url, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body || {})
      });
    }
    function putJSON(url, body) {
      return fetchJSON(url, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body || {})
      });
    }
    function setText(el, value) {
      el.textContent = value == null || value === '' ? 'none' : String(value);
    }
    function sourceScopeForQueryScope(scope) {
      return scope === 'public_only' ? 'public' : 'personal';
    }
    function countsText(counts) {
      if (!counts) return 'none';
      const parts = Object.keys(counts).sort().map((key) => key + ':' + counts[key]);
      return parts.length ? parts.join(' / ') : 'none';
    }
    function currentDeviceID() {
      return (ui.deviceId.value || '').trim() || 'stackchan-sim-001';
    }
    function renderDeviceBindings(payload) {
      const bindings = (payload && payload.bindings) || [];
      const summary = (payload && payload.summary) || {};
      state.deviceBindings = bindings;
      const active = bindings.find((binding) => binding.status === 'bound') || bindings[0] || null;
      state.deviceBinding = active;
      setText(ui.deviceBindingCount, String(summary.active_bindings || 0) + ' / ' + String(summary.total_bindings || bindings.length));
      setText(ui.deviceBindingStatus, [summary.workspace_access_status || 'binding_not_configured', summary.device_binding_policy || 'open_until_binding_configured'].join(' / '));
      if (active && active.device_id) ui.deviceId.value = active.device_id;
    }
    function optionLabel(option) {
      return (option.label || option.id || 'none') + (option.status ? ' [' + option.status + ']' : '');
    }
    function setSelectOptions(select, options, selected) {
      const current = selected || select.value;
      select.textContent = '';
      (options || []).forEach((option) => {
        const item = document.createElement('option');
        item.value = option.id || '';
        item.textContent = optionLabel(option);
        if ((option.id || '') === current) item.selected = true;
        select.append(item);
      });
      if (current && Array.from(select.options).some((option) => option.value === current)) {
        select.value = current;
      }
    }
    function row(title, left, right, tone, detail) {
      const item = document.createElement('div');
      item.className = 'row';
      const a = document.createElement('div');
      const b = document.createElement('div');
      const strong = document.createElement('span');
      const sub = document.createElement('span');
      const tag = document.createElement('span');
      const subRight = document.createElement('span');
      strong.className = 'row-title';
      sub.className = 'row-sub';
      tag.className = 'tag ' + (tone || '');
      subRight.className = 'row-sub';
      strong.textContent = title || 'none';
      sub.textContent = left || 'none';
      tag.textContent = right || 'unknown';
      subRight.textContent = detail || 'searchable=false';
      a.append(strong, sub);
      b.append(tag, subRight);
      item.append(a, b);
      return item;
    }
    function currentSource() {
      if (state.source && state.source.source_id) {
        const match = state.sources.find((source) => source.source_id === state.source.source_id);
        if (match && !match.deleted) return match;
      }
      return state.sources.find((source) => !source.deleted) || state.sources[0] || null;
    }
    function renderSources(payload) {
      const sources = (payload && payload.sources) || [];
      state.sources = sources;
      ui.sourceList.textContent = '';
      setText(ui.sourceCount, sources.length);
      if (!sources.length) {
        ui.sourceList.append(row('No sources', 'stored_local intake empty', 'no_execute', 'warn'));
        return;
      }
      const active = currentSource();
      if (active) {
        state.source = active;
        if (!state.job || state.job.job_id !== active.job_id) {
          state.job = { job_id: active.job_id };
        }
        if (active.document_id && (!state.document || state.document.document_id !== active.document_id)) {
          state.document = {
            document_id: active.document_id,
            storage_status: active.storage_status,
            index_status: active.index_status
          };
        }
        setText(ui.storageStatus, active.deleted ? 'deleted_metadata_only' : active.storage_status || 'metadata_only');
        setText(ui.indexStatus, active.deleted ? 'deleted_no_execute' : active.index_status || active.readiness);
        setText(ui.searchableStatus, String(!!active.searchable));
      }
      sources.forEach((source) => {
        const title = source.document_label || source.source_id;
        const left = [source.source_scope, source.readiness, source.index_status].filter(Boolean).join(' / ');
        const tone = source.deleted ? 'off' : source.readiness === 'indexing_requested_no_execute' ? 'warn' : 'ready';
        const detail = source.deleted ? 'deleted_metadata_only' : 'searchable=' + String(!!source.searchable);
        ui.sourceList.append(row(title, left, source.source_id, tone, detail));
      });
    }
    function renderReadRecords(payload) {
      const records = (payload && payload.records) || [];
      state.readRecords = records;
      ui.readList.textContent = '';
      setText(ui.readCount, records.length);
      if (!records.length) {
        ui.readList.append(row('No read records', 'professional route idle', 'v21 off', 'warn'));
        return;
      }
      records.forEach((record) => {
        const left = [record.query_scope, record.workspace_status, countsText(record.source_scope_counts)].filter(Boolean).join(' / ');
        const tone = record.status === 'completed' ? 'ready' : 'warn';
        const detail = [record.trace_id, record.session_id].filter(Boolean).join(' / ');
        ui.readList.append(row(record.record_id, left, record.status, tone, detail));
      });
    }
    function safeControlStates(events) {
      return (events || []).map((event) => {
        const raw = event && event.payload;
        let payload = raw;
        if (typeof raw === 'string') {
          try { payload = JSON.parse(raw); } catch (_) { payload = {}; }
        }
        return (payload && (payload.state || payload.mode || payload.type)) || event.type || 'event';
      }).filter(Boolean);
    }
    function traceNameList(payload) {
      return ((payload && payload.events) || []).map((event) => event.name || '').filter(Boolean);
    }
    function renderProbeTrace(payload) {
      const events = (payload && payload.events) || [];
      const summary = (payload && payload.summary) || {};
      ui.voiceProbeTraceList.textContent = '';
      setText(ui.voiceProbeTraceStatus, 'trace=' + (payload && payload.trace_id ? payload.trace_id : 'none') + ' / events=' + (summary.event_count || events.length));
      if (!events.length) {
        ui.voiceProbeTraceList.append(row('No trace markers', 'probe idle', 'none', 'warn'));
        return;
      }
      events.slice(-8).forEach((event) => {
        ui.voiceProbeTraceList.append(row(event.name, 'offset_ms=' + String(event.offset_ms || 0), event.session_id || 'session', 'ready', event.device_id || 'device'));
      });
    }
    function renderProbeReadRecords(payload) {
      const records = (payload && payload.records) || [];
      ui.voiceProbeReadList.textContent = '';
      if (!records.length) {
        ui.voiceProbeReadList.append(row('No probe read records', 'professional route idle', 'none', 'warn'));
        return;
      }
      records.forEach((record) => {
        const left = [record.query_scope, record.workspace_status, countsText(record.source_scope_counts)].filter(Boolean).join(' / ');
        const tone = record.status === 'completed' ? 'ready' : 'warn';
        const detail = [record.utterance_bucket, record.failure_code].filter(Boolean).join(' / ');
        ui.voiceProbeReadList.append(row(record.record_id, left, record.status, tone, detail));
      });
    }
    function nextProbeIDs(mode) {
      const stamp = Date.now();
      return {
        trace_id: 'a21-trace-workspace-probe-' + mode + '-' + stamp,
        session_id: 'a21-session-workspace-probe-' + mode + '-' + stamp
      };
    }
    function nextModeRitualIDs(mode) {
      const stamp = Date.now();
      return {
        trace_id: 'a21-trace-workspace-mode-ritual-' + mode + '-' + stamp,
        session_id: 'a21-session-workspace-mode-ritual-' + mode + '-' + stamp
      };
    }
    function nextBodyPresetIDs(preset) {
      const stamp = Date.now();
      return {
        trace_id: 'a21-trace-workspace-body-' + preset + '-' + stamp,
        session_id: 'a21-session-workspace-body-' + preset + '-' + stamp
      };
    }
    function nextBodyMotionIDs(motion) {
      const stamp = Date.now();
      return {
        trace_id: 'a21-trace-workspace-body-motion-' + motion + '-' + stamp,
        session_id: 'a21-session-workspace-body-motion-' + motion + '-' + stamp
      };
    }
    function nextHardwareSceneIDs(scene) {
      const stamp = Date.now();
      return {
        trace_id: 'a21-trace-workspace-hardware-scene-' + scene + '-' + stamp,
        session_id: 'a21-session-workspace-hardware-scene-' + scene + '-' + stamp
      };
    }
    function nextHardwareScreenIDs(action) {
      const stamp = Date.now();
      return {
        trace_id: 'a21-trace-workspace-screen-' + action + '-' + stamp,
        session_id: 'a21-session-workspace-screen-' + action + '-' + stamp
      };
    }
    function nextOfficialActionIDs(action) {
      const stamp = Date.now();
      return {
        trace_id: 'a21-trace-workspace-official-' + action + '-' + stamp,
        session_id: 'a21-session-workspace-official-' + action + '-' + stamp
      };
    }
    function probeCue(mode) {
      const value = ui.voiceProbeInput.value.trim();
      if (value) return value;
      return mode === 'professional' ? 'check evidence' : 'stay with the messy boundary';
    }
    function setProbeRoute(label, payload) {
      const states = safeControlStates((payload && payload.events) || []);
      setText(ui.voiceProbeRouteStatus, 'route=' + label + ' / events=' + states.length);
      state.voiceProbe = Object.assign({}, state.voiceProbe || {}, {
        route: label,
        event_count: states.length,
        state_markers: states.slice(-4)
      });
    }
    function setProbeRoleplay(payload) {
      const runtime = (payload && payload.roleplay) || {};
      setText(ui.voiceProbeRoleplayStatus, [runtime.roleplay_profile, runtime.scenario, 'prompt=' + String(!!runtime.prompt_composed)].filter(Boolean).join(' / '));
      setText(ui.voiceProbeVoiceStatus, runtime.voice_clone_profile || ui.roleplayVoiceStatus.textContent);
      setText(ui.voiceProbeMemoryStatus, (runtime.memory_configured ? 'ready' : 'empty') + ' / ' + (runtime.memory_hint_count || 0));
      setText(ui.voiceProbeProfessionalStatus, 'idle');
      state.voiceProbe = Object.assign({}, state.voiceProbe || {}, {
        roleplay_profile: runtime.roleplay_profile || '',
        scenario: runtime.scenario || '',
        voice_profile: runtime.voice_clone_profile || '',
        memory_hint_count: runtime.memory_hint_count || 0,
        prompt_ready: !!runtime.prompt_composed
      });
    }
    function setProbeProfessional(payload, recordsPayload) {
      const records = (recordsPayload && recordsPayload.records) || [];
      const latest = records[records.length - 1] || {};
      setText(ui.voiceProbeProfessionalStatus, [latest.status || 'started', latest.query_scope || ui.queryScope.value, latest.workspace_status || latest.failure_code || 'pending'].filter(Boolean).join(' / '));
      setText(ui.voiceProbeRoleplayStatus, 'professional route');
      setText(ui.voiceProbeVoiceStatus, ui.voiceChainTTSStatus.textContent || 'none');
      setText(ui.voiceProbeMemoryStatus, 'not used');
      state.voiceProbe = Object.assign({}, state.voiceProbe || {}, {
        professional_status: latest.status || '',
        professional_query_scope: latest.query_scope || '',
        professional_workspace_status: latest.workspace_status || '',
        professional_failure_code: latest.failure_code || ''
      });
      setProbeRoute('professional_query', payload);
    }
    async function refreshVoiceProbeTrace() {
      if (!state.voiceProbe || !state.voiceProbe.trace_id) {
        renderProbeTrace({ trace_id: '', events: [], summary: { event_count: 0 } });
        return;
      }
      const payload = await fetchJSON('/v1/traces?trace_id=' + encodeURIComponent(state.voiceProbe.trace_id), { cache: 'no-store' });
      state.voiceProbe.trace_markers = traceNameList(payload);
      renderProbeTrace(payload);
      log('probe trace ' + ((payload.summary || {}).event_count || 0));
    }
    async function refreshVoiceProbeReads() {
      if (!state.voiceProbe || !state.voiceProbe.trace_id) {
        renderProbeReadRecords({ records: [] });
        return { records: [] };
      }
      const payload = await fetchJSON('/v1/professional-read-records?trace_id=' + encodeURIComponent(state.voiceProbe.trace_id), { cache: 'no-store' });
      renderProbeReadRecords(payload);
      return payload;
    }
    function bodyPresetArgs(response, marker) {
      const step = ((response && response.steps) || []).find((item) => item.marker === marker) || {};
      return step.arguments || {};
    }
    function lastBodyStepArgs(response, marker) {
      const steps = ((response && response.steps) || []).filter((item) => item.marker === marker);
      const step = steps[steps.length - 1] || {};
      return step.arguments || {};
    }
    function renderBodyPresetResponse(response) {
      const led = bodyPresetArgs(response, 'robot_led_color');
      const head = bodyPresetArgs(response, 'robot_head_angles_set');
      setText(ui.bodyPresetStatus, 'preset=' + ((response && response.preset) || 'idle'));
      setText(ui.bodyPresetPhysicalStatus, 'physical_accepted=' + String(!!(response && response.physical_accepted)));
      setText(ui.bodyPresetTraceStatus, 'trace=' + ((response && response.trace_id) || 'none'));
      setText(ui.bodyPresetTransportStatus, (response && response.delivered_transport) || 'xiaozhi_mcp_sequence');
      setText(ui.bodyPresetLEDStatus, 'rgb=' + [led.red, led.green, led.blue].map((value) => value == null ? 'none' : value).join('/'));
      setText(ui.bodyPresetHeadStatus, 'pose=' + ['yaw:' + (head.yaw == null ? 'none' : head.yaw), 'pitch:' + (head.pitch == null ? 'none' : head.pitch), 'speed:' + (head.speed == null ? 'none' : head.speed)].join(' / '));
    }
    function renderBodyPresetTrace(payload) {
      const events = (payload && payload.events) || [];
      const summary = (payload && payload.summary) || {};
      ui.bodyPresetTraceList.textContent = '';
      setText(ui.bodyPresetTraceStatus, 'trace=' + (payload && payload.trace_id ? payload.trace_id : 'none') + ' / events=' + (summary.event_count || events.length));
      if (!events.length) {
        ui.bodyPresetTraceList.append(row('No body trace markers', 'preset idle', 'none', 'warn'));
        return;
      }
      events.slice(-8).forEach((event) => {
        ui.bodyPresetTraceList.append(row(event.name, 'offset_ms=' + String(event.offset_ms || 0), event.session_id || 'session', 'ready', event.device_id || 'device'));
      });
    }
    async function refreshBodyPresetTrace() {
      if (!state.bodyPreset || !state.bodyPreset.trace_id) {
        renderBodyPresetTrace({ trace_id: '', events: [], summary: { event_count: 0 } });
        return;
      }
      const payload = await fetchJSON('/v1/traces?trace_id=' + encodeURIComponent(state.bodyPreset.trace_id), { cache: 'no-store' });
      state.bodyPreset.trace_markers = traceNameList(payload);
      renderBodyPresetTrace(payload);
      log('body trace ' + ((payload.summary || {}).event_count || 0));
    }
    async function runBodyPreset(preset) {
      const ids = nextBodyPresetIDs(preset);
      const payload = await postJSON('/v1/xiaozhi/body-preset', {
        device_id: currentDeviceID(),
        preset: preset,
        trace_id: ids.trace_id,
        session_id: ids.session_id
      });
      state.bodyPreset = {
        preset: payload.preset || preset,
        trace_id: payload.trace_id || ids.trace_id,
        session_id: payload.session_id || ids.session_id,
        status: payload.status || '',
        delivered_transport: payload.delivered_transport || '',
        physical_accepted: !!payload.physical_accepted,
        steps: payload.steps || []
      };
      renderBodyPresetResponse(payload);
      await refreshBodyPresetTrace();
      log('body preset ' + (payload.preset || preset) + ' ' + (payload.status || 'sent'));
      return payload;
    }
    async function runBodyMotion(motion) {
      const ids = nextBodyMotionIDs(motion);
      const payload = await postJSON('/v1/xiaozhi/body-motion', {
        device_id: currentDeviceID(),
        motion: motion,
        trace_id: ids.trace_id,
        session_id: ids.session_id
      });
      const led = lastBodyStepArgs(payload, 'robot_led_color');
      const head = lastBodyStepArgs(payload, 'robot_head_angles_set');
      state.bodyMotion = {
        motion: payload.motion || motion,
        trace_id: payload.trace_id || ids.trace_id,
        session_id: payload.session_id || ids.session_id,
        status: payload.status || '',
        delivered_transport: payload.delivered_transport || '',
        physical_accepted: !!payload.physical_accepted,
        steps: payload.steps || []
      };
      state.bodyPreset = Object.assign({}, state.bodyPreset || {}, {
        trace_id: state.bodyMotion.trace_id,
        session_id: state.bodyMotion.session_id
      });
      setText(ui.bodyPresetStatus, 'motion=' + (payload.motion || motion));
      setText(ui.bodyPresetPhysicalStatus, 'physical_accepted=' + String(!!payload.physical_accepted));
      setText(ui.bodyPresetTraceStatus, 'trace=' + (payload.trace_id || ids.trace_id));
      setText(ui.bodyPresetTransportStatus, payload.delivered_transport || 'xiaozhi_mcp_sequence');
      setText(ui.bodyPresetLEDStatus, 'rgb=' + [led.red, led.green, led.blue].map((value) => value == null ? 'none' : value).join('/'));
      setText(ui.bodyPresetHeadStatus, 'pose=' + ['yaw:' + (head.yaw == null ? 'none' : head.yaw), 'pitch:' + (head.pitch == null ? 'none' : head.pitch), 'speed:' + (head.speed == null ? 'none' : head.speed)].join(' / '));
      await refreshBodyPresetTrace();
      state.bodyMotion.trace_markers = (state.bodyPreset && state.bodyPreset.trace_markers) || [];
      log('body motion ' + (payload.motion || motion) + ' ' + (payload.status || 'sent'));
      return payload;
    }
    function renderHardwareSceneResponse(response) {
      const theme = lastBodyStepArgs(response, 'screen_theme');
      const brightness = lastBodyStepArgs(response, 'screen_brightness');
      const led = lastBodyStepArgs(response, 'robot_led_color');
      const head = lastBodyStepArgs(response, 'robot_head_angles_set');
      const scene = (response && response.scene) || 'idle';
      const steps = ((response && response.steps) || []).length;
      setText(ui.hardwareSceneStatus, 'scene=' + scene);
      setText(ui.hardwareScenePhysicalStatus, 'physical_accepted=' + String(!!(response && response.physical_accepted)));
      setText(ui.hardwareSceneTraceStatus, 'trace=' + ((response && response.trace_id) || 'none'));
      setText(ui.hardwareSceneTransportStatus, (response && response.delivered_transport) || 'xiaozhi_mcp_sequence');
      setText(ui.hardwareSceneScreenStatus, 'screen=' + ['theme:' + (theme.theme || 'none'), 'brightness:' + (brightness.brightness == null ? 'none' : brightness.brightness)].join(' / '));
      setText(ui.hardwareSceneBodyStatus, 'body=' + ['rgb:' + [led.red, led.green, led.blue].map((value) => value == null ? 'none' : value).join('/'), 'pitch:' + (head.pitch == null ? 'none' : head.pitch)].join(' / '));
      setText(ui.hardwareSceneStepStatus, 'steps=' + steps);
      setText(ui.hardwareSceneAcceptanceStatus, (response && response.physical_accepted) ? 'operator_visible_accepted' : 'operator_pending');
      setText(ui.bodyPresetStatus, 'scene=' + scene);
      setText(ui.bodyPresetPhysicalStatus, 'physical_accepted=' + String(!!(response && response.physical_accepted)));
      setText(ui.bodyPresetTraceStatus, 'trace=' + ((response && response.trace_id) || 'none'));
      setText(ui.bodyPresetTransportStatus, (response && response.delivered_transport) || 'xiaozhi_mcp_sequence');
      setText(ui.bodyPresetLEDStatus, 'rgb=' + [led.red, led.green, led.blue].map((value) => value == null ? 'none' : value).join('/'));
      setText(ui.bodyPresetHeadStatus, 'pose=' + ['yaw:' + (head.yaw == null ? 'none' : head.yaw), 'pitch:' + (head.pitch == null ? 'none' : head.pitch), 'speed:' + (head.speed == null ? 'none' : head.speed)].join(' / '));
      setText(ui.hardwareScreenStatus, 'scene=' + scene);
      setText(ui.hardwareScreenPhysicalStatus, 'physical_accepted=false');
      setText(ui.hardwareScreenTraceStatus, 'trace=' + ((response && response.trace_id) || 'none'));
      setText(ui.screenBrightnessStatus, 'brightness=' + (brightness.brightness == null ? ui.screenBrightness.value : brightness.brightness));
      setText(ui.screenThemeStatus, 'theme=' + (theme.theme || 'none'));
      setText(ui.screenToolStatus, 'body-scene');
    }
    function renderHardwareSceneAcceptance(response) {
      const accepted = !!(response && response.physical_accepted);
      const status = accepted ? ((response && response.status) || 'accepted') : 'operator_pending';
      setText(ui.hardwareScenePhysicalStatus, 'physical_accepted=' + String(accepted));
      setText(ui.bodyPresetPhysicalStatus, 'physical_accepted=' + String(accepted));
      setText(ui.hardwareScreenPhysicalStatus, 'physical_accepted=' + String(accepted));
      setText(ui.hardwareSceneAcceptanceStatus, status);
      if (response && response.trace_id) {
        setText(ui.hardwareSceneTraceStatus, 'trace=' + response.trace_id);
      }
    }
    function renderHardwareSceneTrace(payload) {
      const events = (payload && payload.events) || [];
      const summary = (payload && payload.summary) || {};
      ui.hardwareSceneTraceList.textContent = '';
      setText(ui.hardwareSceneTraceStatus, 'trace=' + (payload && payload.trace_id ? payload.trace_id : 'none') + ' / events=' + (summary.event_count || events.length));
      if (!events.length) {
        ui.hardwareSceneTraceList.append(row('No scene trace markers', 'scene idle', 'none', 'warn'));
        return;
      }
      events.slice(-10).forEach((event) => {
        ui.hardwareSceneTraceList.append(row(event.name, 'offset_ms=' + String(event.offset_ms || 0), event.session_id || 'session', 'ready', event.device_id || 'device'));
      });
    }
    async function refreshHardwareSceneTrace() {
      if (!state.hardwareScene || !state.hardwareScene.trace_id) {
        renderHardwareSceneTrace({ trace_id: '', events: [], summary: { event_count: 0 } });
        return;
      }
      const payload = await fetchJSON('/v1/traces?trace_id=' + encodeURIComponent(state.hardwareScene.trace_id), { cache: 'no-store' });
      state.hardwareScene.trace_markers = traceNameList(payload);
      renderHardwareSceneTrace(payload);
      log('hardware scene trace ' + ((payload.summary || {}).event_count || 0));
    }
    async function runHardwareScene(scene) {
      const ids = nextHardwareSceneIDs(scene);
      const payload = await postJSON('/v1/xiaozhi/body-scene', {
        device_id: currentDeviceID(),
        scene: scene,
        trace_id: ids.trace_id,
        session_id: ids.session_id
      });
      state.hardwareScene = {
        scene: payload.scene || scene,
        trace_id: payload.trace_id || ids.trace_id,
        session_id: payload.session_id || ids.session_id,
        status: payload.status || '',
        delivered_transport: payload.delivered_transport || '',
        physical_accepted: !!payload.physical_accepted,
        acceptance_status: 'operator_pending',
        steps: payload.steps || []
      };
      state.bodyPreset = Object.assign({}, state.bodyPreset || {}, {
        trace_id: state.hardwareScene.trace_id,
        session_id: state.hardwareScene.session_id,
        physical_accepted: state.hardwareScene.physical_accepted
      });
      state.hardwareScreen = Object.assign({}, state.hardwareScreen || {}, {
        action: 'body_scene_' + (payload.scene || scene),
        trace_id: state.hardwareScene.trace_id,
        session_id: state.hardwareScene.session_id,
        status: payload.status || '',
        delivered_transport: payload.delivered_transport || '',
        physical_accepted: false
      });
      renderHardwareSceneResponse(payload);
      await refreshHardwareSceneTrace();
      state.bodyPreset.trace_markers = (state.hardwareScene && state.hardwareScene.trace_markers) || [];
      state.hardwareScreen.trace_markers = (state.hardwareScene && state.hardwareScene.trace_markers) || [];
      await refreshHardwareAcceptance();
      log('hardware scene ' + (payload.scene || scene) + ' ' + (payload.status || 'sent'));
      return payload;
    }
    async function acceptHardwareScenePhysical() {
      if (!state.hardwareScene || !state.hardwareScene.trace_id || !state.hardwareScene.session_id) {
        throw new Error('run Full Check first');
      }
      if ((state.hardwareScene.scene || '') !== 'full_check') {
        throw new Error('run Full Check first');
      }
      const payload = await postJSON('/v1/xiaozhi/body-scene-acceptance', {
        device_id: currentDeviceID(),
        scene: state.hardwareScene.scene || 'full_check',
        trace_id: state.hardwareScene.trace_id,
        session_id: state.hardwareScene.session_id,
        screen_visible: true,
        rgb_visible: true,
        servo_visible: true,
        observer: 'operator'
      });
      state.hardwareScene.physical_accepted = !!payload.physical_accepted;
      state.hardwareScene.acceptance_status = payload.status || '';
      state.hardwareScene.accepted_surfaces = payload.accepted_surfaces || [];
      state.bodyPreset = Object.assign({}, state.bodyPreset || {}, {
        trace_id: state.hardwareScene.trace_id,
        session_id: state.hardwareScene.session_id,
        physical_accepted: state.hardwareScene.physical_accepted
      });
      state.hardwareScreen = Object.assign({}, state.hardwareScreen || {}, {
        trace_id: state.hardwareScene.trace_id,
        session_id: state.hardwareScene.session_id,
        physical_accepted: state.hardwareScene.physical_accepted
      });
      renderHardwareSceneAcceptance(payload);
      await refreshHardwareSceneTrace();
      await refreshHardwareAcceptance();
      log('hardware scene acceptance ' + (payload.status || 'accepted'));
      return payload;
    }
    function officialActionFallback(kind, value) {
      if (kind === 'motion') {
        return { type: 'body_motion', value: value };
      }
      if (kind === 'state') {
        const presets = {
          idle: 'reset_idle',
          listening: 'listening',
          thinking: 'thinking',
          speaking: 'speaking'
        };
        return presets[value] ? { type: 'body_preset', value: presets[value] } : null;
      }
      if (kind === 'face') {
        const presets = {
          happy: 'celebrate',
          attentive: 'listening'
        };
        return presets[value] ? { type: 'body_preset', value: presets[value] } : null;
      }
      return null;
    }
    async function runOfficialActionFallback(kind, value, blockedReason) {
      const fallback = officialActionFallback(kind, value);
      if (!fallback) {
        throw new Error(blockedReason);
      }
      const payload = fallback.type === 'body_motion'
        ? await runBodyMotion(fallback.value)
        : await runBodyPreset(fallback.value);
      state.officialAction = {
        action: kind + '-' + value,
        trace_id: payload.trace_id || '',
        session_id: payload.session_id || '',
        status: 'fallback_delivered',
        delivered_transport: payload.delivered_transport || 'xiaozhi_mcp_sequence',
        event: kind,
        value: value,
        packet_count: 0,
        official_action_physical_accepted: false,
        official_action_surfaces: {
          official_relay: 'blocked',
          fallback: fallback.type,
          fallback_value: fallback.value
        },
        blocked_reason: blockedReason,
        fallback: fallback.type + ':' + fallback.value
      };
      renderOfficialActionResponse({
        trace_id: state.officialAction.trace_id,
        status: 'fallback_delivered',
        delivered_transport: state.officialAction.delivered_transport,
        event: kind,
        value: value,
        packet_count: 0,
        official_action_physical_accepted: false,
        official_action_surfaces: state.officialAction.official_action_surfaces
      });
      setText(ui.officialActionStatus, 'fallback=' + fallback.type + ':' + fallback.value);
      await refreshOfficialActionTrace();
      log('official action fallback ' + fallback.type + ':' + fallback.value + ' after ' + blockedReason);
      return payload;
    }
    function renderHardwareScreenResponse(response, action) {
      const args = (response && response.arguments) || {};
      const tool = (response && response.tool_name) || 'none';
      const brightness = args.brightness == null ? ui.screenBrightness.value : args.brightness;
      const theme = args.theme || (state.hardwareScreen && state.hardwareScreen.theme) || 'none';
      setText(ui.hardwareScreenStatus, 'screen=' + (action || 'delivered'));
      setText(ui.hardwareScreenPhysicalStatus, 'physical_accepted=false');
      setText(ui.hardwareScreenTraceStatus, 'trace=' + ((response && response.trace_id) || 'none'));
      setText(ui.screenBrightnessStatus, 'brightness=' + brightness);
      setText(ui.screenThemeStatus, 'theme=' + theme);
      setText(ui.screenToolStatus, tool);
    }
    function renderHardwareScreenTrace(payload) {
      const events = (payload && payload.events) || [];
      const summary = (payload && payload.summary) || {};
      ui.hardwareScreenTraceList.textContent = '';
      setText(ui.hardwareScreenTraceStatus, 'trace=' + (payload && payload.trace_id ? payload.trace_id : 'none') + ' / events=' + (summary.event_count || events.length));
      if (!events.length) {
        ui.hardwareScreenTraceList.append(row('No screen trace markers', 'screen idle', 'none', 'warn'));
        return;
      }
      events.slice(-8).forEach((event) => {
        ui.hardwareScreenTraceList.append(row(event.name, 'offset_ms=' + String(event.offset_ms || 0), event.session_id || 'session', 'ready', event.device_id || 'device'));
      });
    }
    async function refreshHardwareScreenTrace() {
      if (!state.hardwareScreen || !state.hardwareScreen.trace_id) {
        renderHardwareScreenTrace({ trace_id: '', events: [], summary: { event_count: 0 } });
        return;
      }
      const payload = await fetchJSON('/v1/traces?trace_id=' + encodeURIComponent(state.hardwareScreen.trace_id), { cache: 'no-store' });
      state.hardwareScreen.trace_markers = traceNameList(payload);
      renderHardwareScreenTrace(payload);
      log('screen trace ' + ((payload.summary || {}).event_count || 0));
    }
    async function refreshMCPCapabilities() {
      const payload = await fetchJSON('/v1/xiaozhi/mcp-capabilities?device_id=' + encodeURIComponent(currentDeviceID()), { cache: 'no-store' });
      const tools = payload.allowed_tools || [];
      const blocked = payload.blocked_tool_classes || [];
      setText(ui.mcpCapabilitiesStatus, 'mcp=' + String(!!payload.mcp_advertised) + ' / allowed=' + tools.length + ' / blocked=' + blocked.length);
      state.hardwareScreen = Object.assign({}, state.hardwareScreen || {}, {
        capabilities_device_id: payload.device_id || currentDeviceID(),
        mcp_advertised: !!payload.mcp_advertised,
        allowed_tools: tools,
        blocked_tool_classes: blocked,
        physical_accepted: !!payload.physical_accepted
      });
      log('mcp capabilities ' + tools.length);
    }
    async function runHardwareScreenAction(action, value) {
      const ids = nextHardwareScreenIDs(action);
      const base = {
        device_id: currentDeviceID(),
        trace_id: ids.trace_id,
        session_id: ids.session_id
      };
      let payload;
      if (action === 'brightness') {
        payload = await postJSON('/v1/xiaozhi/screen-brightness', Object.assign({}, base, {
          brightness: Number(value == null ? ui.screenBrightness.value : value)
        }));
      } else if (action === 'theme') {
        payload = await postJSON('/v1/xiaozhi/screen-theme', Object.assign({}, base, {
          theme: String(value || 'dark')
        }));
      } else if (action === 'screen_info') {
        payload = await postJSON('/v1/xiaozhi/mcp-control', Object.assign({}, base, {
          tool_name: 'self.screen.get_info'
        }));
      } else {
        payload = await postJSON('/v1/xiaozhi/device-status', base);
      }
      state.hardwareScreen = Object.assign({}, state.hardwareScreen || {}, {
        action: action,
        trace_id: payload.trace_id || ids.trace_id,
        session_id: payload.session_id || ids.session_id,
        status: payload.status || '',
        delivered_transport: payload.delivered_transport || '',
        tool_name: payload.tool_name || '',
        mcp_id: payload.mcp_id || '',
        arguments: payload.arguments || {},
        physical_accepted: false
      });
      if (payload.arguments && payload.arguments.theme) {
        state.hardwareScreen.theme = payload.arguments.theme;
      }
      renderHardwareScreenResponse(payload, action);
      await refreshHardwareScreenTrace();
      log('screen ' + action + ' ' + (payload.status || 'sent'));
    }
    function officialSurfaceText(surfaces) {
      surfaces = surfaces || {};
      const keys = Object.keys(surfaces).sort();
      if (!keys.length) return 'surfaces=none';
      return keys.map((key) => key + ':' + surfaces[key]).join(' / ');
    }
    function renderOfficialActionResponse(response) {
      const surfaces = (response && response.official_action_surfaces) || {};
      setText(ui.officialActionStatus, 'action=' + ((response && response.status) || 'idle'));
      setText(ui.officialActionPhysicalStatus, 'physical_accepted=' + String(!!(response && response.official_action_physical_accepted)));
      setText(ui.officialActionTraceStatus, 'trace=' + ((response && response.trace_id) || 'none'));
      setText(ui.officialActionEventStatus, 'event=' + [response && response.event, response && response.value].filter(Boolean).join(':'));
      setText(ui.officialActionPacketStatus, 'packets=' + ((response && response.packet_count) || 0));
      setText(ui.officialActionTransportStatus, (response && response.delivered_transport) || 'stackchan_official_ws');
      setText(ui.officialActionSurfaceStatus, officialSurfaceText(surfaces));
    }
    function renderOfficialActionTrace(payload) {
      const events = (payload && payload.events) || [];
      const summary = (payload && payload.summary) || {};
      ui.officialActionTraceList.textContent = '';
      setText(ui.officialActionTraceStatus, 'trace=' + (payload && payload.trace_id ? payload.trace_id : 'none') + ' / events=' + (summary.event_count || events.length));
      if (!events.length) {
        ui.officialActionTraceList.append(row('No official action trace markers', 'official action idle', 'none', 'warn'));
        return;
      }
      events.slice(-8).forEach((event) => {
        ui.officialActionTraceList.append(row(event.name, 'offset_ms=' + String(event.offset_ms || 0), event.session_id || 'session', 'ready', event.device_id || 'device'));
      });
    }
    async function refreshOfficialActionTrace() {
      if (!state.officialAction || !state.officialAction.trace_id) {
        renderOfficialActionTrace({ trace_id: '', events: [], summary: { event_count: 0 } });
        return;
      }
      const payload = await fetchJSON('/v1/traces?trace_id=' + encodeURIComponent(state.officialAction.trace_id), { cache: 'no-store' });
      state.officialAction.trace_markers = traceNameList(payload);
      renderOfficialActionTrace(payload);
      log('official action trace ' + ((payload.summary || {}).event_count || 0));
    }
    async function runOfficialAction(kind, value) {
      const action = kind + '-' + value;
      const ids = nextOfficialActionIDs(action);
      const body = {
        device_id: currentDeviceID(),
        event: kind,
        trace_id: ids.trace_id,
        session_id: ids.session_id
      };
      if (kind === 'state') {
        body.state = value;
      } else if (kind === 'face') {
        body.emotion = value;
      } else {
        body.name = value;
        if (value === 'look_up') {
          body.y_angle = Number(ui.officialActionYAngle.value || 38);
        }
      }
      try {
        const payload = await postJSON('/v1/stackchan/official/control', body);
        state.officialAction = {
          action: action,
          trace_id: payload.trace_id || ids.trace_id,
          session_id: payload.session_id || ids.session_id,
          status: payload.status || '',
          delivered_transport: payload.delivered_transport || '',
          event: payload.event || kind,
          value: payload.value || value,
          packet_count: payload.packet_count || 0,
          official_action_physical_accepted: !!payload.official_action_physical_accepted,
          official_action_surfaces: payload.official_action_surfaces || {}
        };
        renderOfficialActionResponse(payload);
        await refreshOfficialActionTrace();
        log('official action ' + action + ' ' + (payload.status || 'sent'));
      } catch (err) {
        try {
          await runOfficialActionFallback(kind, value, err.message);
        } catch (fallbackErr) {
          state.officialAction = {
            action: action,
            trace_id: ids.trace_id,
            session_id: ids.session_id,
            status: 'blocked',
            error: fallbackErr.message,
            blocked_reason: err.message,
            event: kind,
            value: value,
            official_action_physical_accepted: false
          };
          setText(ui.officialActionStatus, 'blocked=' + fallbackErr.message);
          setText(ui.officialActionPhysicalStatus, 'physical_accepted=false');
          setText(ui.officialActionTraceStatus, 'trace=' + ids.trace_id);
          setText(ui.officialActionEventStatus, 'event=' + kind + ':' + value);
          setText(ui.officialActionPacketStatus, 'packets=0');
          setText(ui.officialActionTransportStatus, 'stackchan_official_ws');
          setText(ui.officialActionSurfaceStatus, 'surfaces=blocked');
          renderOfficialActionTrace({ trace_id: ids.trace_id, events: [], summary: { event_count: 0 } });
          log('official action ' + action + ' blocked ' + fallbackErr.message);
        }
      }
    }
    async function runRoleplayProbe() {
      const ids = nextProbeIDs('roleplay');
      const localBoundary = { asr_provider: 'workspace_probe', first_partial_ms: 32 };
      localBoundary['final_' + 'trans' + 'cript_chars'] = Math.max(1, probeCue('roleplay').length);
      state.voiceProbe = {
        mode: 'roleplay',
        trace_id: ids.trace_id,
        session_id: ids.session_id
      };
      const payload = await postJSON('/v1/fast-companion/turn', {
        device_id: 'stackchan-sim-001',
        mode: 'roleplay',
        trace_id: ids.trace_id,
        session_id: ids.session_id,
        local_audio: localBoundary
      });
      setProbeRoute(payload.route || 'fast_companion_hybrid', payload);
      setProbeRoleplay(payload);
      await refreshVoiceProbeTrace();
      renderProbeReadRecords({ records: [] });
      log('probe roleplay ' + (payload.status || 'ok'));
    }
    async function runProfessionalProbe() {
      const ids = nextProbeIDs('professional');
      state.voiceProbe = {
        mode: 'professional',
        trace_id: ids.trace_id,
        session_id: ids.session_id
      };
      const payload = await postJSON('/v1/professional-query', {
        device_id: currentDeviceID(),
        user_id: ui.userId.textContent || 'a21_local_user',
        workspace_id: ui.workspaceId.textContent || 'a21_local_workspace',
        query_scope: ui.queryScope.value,
        text: probeCue('professional'),
        trace_id: ids.trace_id,
        session_id: ids.session_id
      });
      setProbeRoute(payload.route || 'professional_query', payload);
      await refreshVoiceProbeTrace();
      const records = await refreshVoiceProbeReads();
      setProbeProfessional(payload, records);
      await refreshReads();
      log('probe professional ' + (((records.records || [])[0] || {}).status || 'recorded'));
    }
    function setWorkspace(payload) {
      state.workspace = payload || null;
      const runtime = (payload && payload.runtime) || {};
      setText(ui.userId, payload.selected_user_id || runtime.user_id);
      setText(ui.workspaceId, payload.selected_workspace_id || runtime.workspace_id);
      setText(ui.workspaceStatus, payload.workspace_status || runtime.workspace_status || 'contract_ready');
      setText(ui.adapterStatus, payload.adapter_contract_version || runtime.adapter_contract_version || 'a21.v21_adapter_query.v2');
      ui.queryScope.value = payload.selected_query_scope || runtime.query_scope || 'public_only';
      setText(ui.searchableStatus, String(!!runtime.v21_execution_allowed && runtime.workspace_status === 'searchable'));
      if (runtime.device_binding_summary) renderDeviceBindings({ bindings: state.deviceBindings, summary: runtime.device_binding_summary });
    }
    function setRoleplay(payload) {
      state.roleplay = payload || null;
      const runtime = (payload && payload.runtime) || {};
      const memory = (payload && payload.memory) || {};
      const plan = (payload && payload.expression_plan) || {};
      const selectedProfile = payload.selected_roleplay_profile || runtime.roleplay_profile || 'a21_roleplay_default';
      const selectedScenario = payload.selected_scenario || runtime.scenario || 'desk_mouthpiece';
      const selectedVoice = payload.selected_voice_clone_profile || runtime.voice_clone_profile || 'a21_voice_default_dashscope';
      if (payload && payload.profiles) setSelectOptions(ui.roleplayProfileSelect, payload.profiles, selectedProfile);
      if (payload && payload.scenarios) setSelectOptions(ui.roleplayScenarioSelect, payload.scenarios, selectedScenario);
      if (ui.roleplayVoiceSelect.options.length && selectedVoice) ui.roleplayVoiceSelect.value = selectedVoice;
      const value = [plan.delivery_policy || 'no_send_plan_only', (plan.action_count || 0) + ' actions', (plan.packet_count || 0) + ' packets'].join(' / ');
      setText(ui.roleplayExpression, value);
      setText(ui.roleplayProfileStatus, selectedProfile);
      setText(ui.roleplayScenarioStatus, selectedScenario);
      setText(ui.roleplayVoiceStatus, selectedVoice);
      setText(ui.roleplayMemoryStatus, (memory.status || (runtime.memory_configured ? 'ready' : 'empty')) + ' / ' + (runtime.memory_hint_count || 0));
      setText(ui.roleplayPromptStatus, 'prompt_input_ready=' + String(!!runtime.prompt_composed || !!runtime.soul_prompt_input_ready));
      setText(ui.roleplayPhysicalStatus, 'physical_accepted=' + String(!!plan.physical_accepted));
    }
    function setVoiceCatalog(payload) {
      state.voiceCatalog = payload || null;
      const cascade = (payload && payload.cascade) || {};
      const realtime = (payload && payload.realtime) || {};
      const selectedMode = (payload && payload.selected_voice_chain_mode) || 'cascade';
      const selectedASR = (payload && payload.selected_asr_profile) || 'dashscope_qwen_asr_realtime';
      const selectedLLM = (payload && payload.selected_llm_profile) || 'stepfun';
      const selectedRealtime = (payload && payload.selected_realtime_provider) || 'doubao_realtime';
      const selectedVoice = (payload && payload.selected_voice_clone_profile) ||
        ((state.roleplay || {}).selected_voice_clone_profile) ||
        'a21_voice_default_dashscope';
      setSelectOptions(ui.voiceChainModeSelect, [
        { id: 'cascade', label: 'Cascade ASR -> LLM -> TTS', status: 'default' },
        { id: 'realtime', label: 'Realtime voice', status: 'opt_in' }
      ], selectedMode);
      setSelectOptions(ui.voiceChainASRSelect, cascade.asr_profiles || [], selectedASR);
      setSelectOptions(ui.voiceChainLLMSelect, cascade.llm_profiles || [], selectedLLM);
      setSelectOptions(ui.voiceChainRealtimeSelect, realtime.providers || [], selectedRealtime);
      setSelectOptions(ui.roleplayVoiceSelect, (payload && payload.voices) || [], selectedVoice);
      setText(ui.roleplayVoiceStatus, selectedVoice);
      setText(ui.voiceChainModeStatus, selectedMode);
      setText(ui.voiceChainASRStatus, selectedASR);
      setText(ui.voiceChainLLMStatus, selectedLLM);
      setText(ui.voiceChainTTSStatus, (payload && payload.selected_tts_profile) || 'dashscope_qwen_tts_realtime');
      setText(ui.voiceChainRealtimeStatus, selectedRealtime);
      setText(ui.voiceChainHotSwitchStatus, 'hot_switch=' + String(!!(payload && payload.hot_switch)));
      const findings = (payload && payload.findings) || [];
      setText(ui.voiceChainFindingStatus, findings.length ? findings.join(' / ') : 'ready');
    }
    function setWakeWord(payload) {
      state.wakeWord = payload || null;
      const mode = (payload && payload.mode) || 'builtin_xiaozhi';
      ui.wakeWordModeSelect.value = mode;
      ui.wakeWordThreshold.value = String((payload && payload.threshold) || 30);
      if (payload && payload.desired_phrase) ui.wakeWordPhrase.value = payload.desired_phrase;
      if (payload && payload.desired_pinyin) ui.wakeWordPinyin.value = payload.desired_pinyin;
      setText(ui.wakeWordActiveStatus, (payload && payload.active_phrase) || '你好小智');
      setText(ui.wakeWordRuntimeStatus, (payload && payload.runtime_status) || 'active_builtin_model');
      setText(ui.wakeWordFirmwareStatus, (payload && payload.firmware_status) || 'builtin_active');
      setText(ui.wakeWordCodeStatus, (payload && payload.code) || 'none');
      setText(ui.wakeWordBuildStatus, 'build_required=' + String(!!(payload && payload.firmware_build_required)));
      setText(ui.wakeWordHotSwapStatus, 'runtime_hot_swap=' + String(!!(payload && payload.runtime_hot_swap_supported)));
    }
    function setVoiceModes(payload) {
      state.voiceModes = payload || null;
      const ritual = (payload && payload.selected_ritual) || {};
      setText(ui.professionalCue, (ritual.screen_label || 'PRO') + ' / ' + (ritual.cue_text || 'checking'));
    }
    function setModeRitual(payload) {
      state.modeRitual = payload || null;
      const mode = (payload && payload.selected_voice_mode) || 'idle';
      setText(ui.modeRitualStatus, 'ritual=' + mode + ' / ' + ((payload && payload.status) || 'idle'));
      setText(ui.modeRitualTraceStatus, 'trace=' + ((payload && payload.trace_id) || 'none'));
      setText(ui.modeRitualPhysicalStatus, 'physical_accepted=' + String(!!(payload && payload.physical_accepted)));
      if (payload && payload.screen_label) {
        setText(ui.professionalCue, payload.screen_label + ' / mode ritual');
      }
    }
    function renderHardwareAcceptance(payload) {
      state.hardwareAcceptance = payload || null;
      const status = (payload && payload.overall_status) || 'device_missing';
      const items = (payload && payload.items) || [];
      const pending = items.find((item) => item.next_action && item.next_action !== 'accepted') || {};
      setText(ui.hardwareAcceptanceStatus, status);
      setText(ui.hardwareAcceptanceDeviceStatus, 'device=' + ((payload && payload.device_id) || currentDeviceID()));
      setText(ui.hardwareAcceptanceDevice, 'device=' + ((payload && payload.device_id) || currentDeviceID()));
      setText(ui.hardwareAcceptanceConnection, (payload && payload.connection_status) || 'missing');
      setText(ui.hardwareAcceptancePhysical, 'physical_accepted=' + String(!!(payload && payload.physical_accepted)));
      setText(ui.hardwareAcceptanceNext, pending.next_action || 'accepted');
      ui.hardwareAcceptanceItems.textContent = '';
      if (!items.length) {
        ui.hardwareAcceptanceItems.append(row('No acceptance items', 'device missing', 'pending', 'warn'));
        return;
      }
      items.forEach((item) => {
        const tone = item.physical_accepted ? 'ready' : (item.delivery_status === 'delivered' ? 'warn' : 'off');
        const left = [item.delivery_status || 'not_delivered', item.trace_id || 'trace=none'].join(' / ');
        const right = item.physical_accepted ? 'accepted' : (item.next_action || 'pending');
        ui.hardwareAcceptanceItems.append(row(item.label || item.id, left, right, tone, item.acceptance_endpoint || ''));
      });
    }
    async function refreshHardwareAcceptance() {
      const payload = await fetchJSON('/v1/hardware-acceptance?device_id=' + encodeURIComponent(currentDeviceID()), { cache: 'no-store' });
      renderHardwareAcceptance(payload);
      log('hardware acceptance ' + ((payload && payload.overall_status) || 'missing'));
      return payload;
    }
    async function refreshWorkspace() {
      const payload = await fetchJSON('/v1/professional-workspace', { cache: 'no-store' });
      setWorkspace(payload);
      log('workspace ' + (payload.workspace_status || 'contract_ready'));
    }
    async function saveScope() {
      const payload = await postJSON('/v1/professional-workspace', { query_scope: ui.queryScope.value });
      setWorkspace(payload);
      await refreshSources();
      log('scope ' + ui.queryScope.value);
    }
    async function uploadDocument() {
      const file = (ui.documentFile.files || [])[0];
      if (!file) {
        log('document file missing');
        return;
      }
      const form = new FormData();
      form.append('user_id', ui.userId.textContent || 'a21_local_user');
      form.append('workspace_id', ui.workspaceId.textContent || 'a21_local_workspace');
      form.append('source_scope', sourceScopeForQueryScope(ui.queryScope.value));
      form.append('document_label', ui.documentLabel.value || 'A21 workspace document');
      form.append('content_type', file.type || 'application/octet-stream');
      form.append('device_id', currentDeviceID());
      form.append('file', file);
      const payload = await fetchJSON('/v1/workspace-documents', { method: 'POST', body: form });
      state.document = (payload.documents || [])[0] || null;
      state.job = (payload.jobs || [])[0] || null;
      state.source = (payload.sources || [])[0] || null;
      setText(ui.storageStatus, state.document && state.document.storage_status);
      setText(ui.indexStatus, state.document && state.document.index_status);
      renderSources({ sources: payload.sources || [] });
      log('upload ' + ((state.document && state.document.status) || payload.status || 'stored_local'));
      await refreshWorkspace();
    }
    async function requestIndex() {
      if (!state.document || !state.job || !state.source) {
        log('stored document missing');
        return;
      }
      const payload = await postJSON('/v1/workspace-index-jobs', {
        document_id: state.document.document_id,
        job_id: state.job.job_id,
        source_id: state.source.source_id
      });
      state.indexJob = (payload.index_jobs || [])[0] || null;
      setText(ui.indexStatus, (state.indexJob && state.indexJob.index_status) || payload.status);
      setText(ui.searchableStatus, 'false');
      log('index ' + (payload.status || 'indexing_requested_no_execute'));
      await refreshSources();
      await refreshWorkspace();
    }
    async function deleteSource() {
      const source = currentSource();
      const jobID = (source && source.job_id) || (state.job && state.job.job_id);
      if (!jobID) {
        log('delete source missing');
        return;
      }
      const payload = await putJSON('/v1/workspace-upload-jobs', { job_id: jobID, action: 'delete' });
      state.job = (payload.jobs || [])[0] || { job_id: jobID, status: 'deleted' };
      state.document = null;
      state.indexJob = null;
      setText(ui.storageStatus, 'deleted_metadata_only');
      setText(ui.indexStatus, 'deleted_no_execute');
      setText(ui.searchableStatus, 'false');
      log('delete deleted_metadata_only');
      await refreshSources();
      await refreshWorkspace();
    }
    async function refreshSources() {
      const payload = await fetchJSON('/v1/workspace-sources', { cache: 'no-store' });
      renderSources(payload);
      log('sources ' + (((payload.summary || {}).total_sources) || 0));
    }
    function deviceBindingQuery() {
      const params = new URLSearchParams();
      const userID = ui.userId.textContent || '';
      const workspaceID = ui.workspaceId.textContent || '';
      if (userID && userID !== 'none') params.set('user_id', userID);
      if (workspaceID && workspaceID !== 'none') params.set('workspace_id', workspaceID);
      const query = params.toString();
      return query ? '?' + query : '';
    }
    async function refreshDeviceBindings() {
      const payload = await fetchJSON('/v1/workspace-device-bindings' + deviceBindingQuery(), { cache: 'no-store' });
      renderDeviceBindings(payload);
      log('device bindings ' + (((payload.summary || {}).active_bindings) || 0));
    }
    async function bindDevice() {
      const payload = await postJSON('/v1/workspace-device-bindings', {
        device_id: currentDeviceID(),
        user_id: ui.userId.textContent || 'a21_local_user',
        workspace_id: ui.workspaceId.textContent || 'a21_local_workspace',
        allowed_query_scopes: [ui.queryScope.value]
      });
      renderDeviceBindings(payload);
      await refreshWorkspace();
      log('device bound ' + currentDeviceID());
    }
    async function revokeDevice() {
      const bindingID = state.deviceBinding && state.deviceBinding.binding_id;
      const body = bindingID ? { binding_id: bindingID, action: 'revoke' } : {
        device_id: currentDeviceID(),
        user_id: ui.userId.textContent || 'a21_local_user',
        workspace_id: ui.workspaceId.textContent || 'a21_local_workspace',
        action: 'revoke'
      };
      const payload = await putJSON('/v1/workspace-device-bindings', body);
      renderDeviceBindings(payload);
      await refreshWorkspace();
      log('device revoked ' + currentDeviceID());
    }
    function readRecordQuery() {
      const params = new URLSearchParams();
      const recordID = ui.readRecordFilter.value.trim();
      const traceID = ui.readTraceFilter.value.trim();
      const sessionID = ui.readSessionFilter.value.trim();
      if (recordID) params.set('record_id', recordID);
      if (traceID) params.set('trace_id', traceID);
      if (sessionID) params.set('session_id', sessionID);
      const query = params.toString();
      return query ? '?' + query : '';
    }
    async function refreshReads() {
      const query = readRecordQuery();
      const payload = await fetchJSON('/v1/professional-read-records' + query, { cache: 'no-store' });
      renderReadRecords(payload);
      log('reads ' + ((payload.records || []).length) + (query ? ' filtered' : ''));
    }
    async function saveRoleplaySetup(options) {
      options = options || {};
      const body = {
        roleplay_profile: ui.roleplayProfileSelect.value,
        scenario: ui.roleplayScenarioSelect.value,
        voice_clone_profile: ui.roleplayVoiceSelect.value
      };
      if (options.includeMemory) {
        const hint = ui.roleplayMemoryHint.value.trim();
        body.memory_hints = hint ? [hint] : [];
      }
      if (options.clearMemory) {
        body.clear_memory = true;
      }
      const payload = await postJSON('/v1/roleplay-profile', body);
      if (options.includeMemory || options.clearMemory) {
        ui.roleplayMemoryHint.value = '';
      }
      setRoleplay(payload);
      const voiceCatalog = await fetchJSON('/v1/voice-chain-profiles', { cache: 'no-store' });
      setVoiceCatalog(voiceCatalog);
      log('roleplay ' + (payload.selected_roleplay_profile || 'saved'));
    }
    async function saveVoiceChainSetup() {
      const payload = await postJSON('/v1/voice-chain-profiles', {
        voice_chain_mode: ui.voiceChainModeSelect.value,
        asr_profile: ui.voiceChainASRSelect.value,
        llm_profile: ui.voiceChainLLMSelect.value,
        realtime_provider: ui.voiceChainRealtimeSelect.value,
        voice_clone_profile: ui.roleplayVoiceSelect.value
      });
      setVoiceCatalog(payload);
      const roleplay = await fetchJSON('/v1/roleplay-profile', { cache: 'no-store' });
      setRoleplay(roleplay);
      log('voice chain ' + (payload.selected_voice_chain_mode || 'saved'));
    }
    async function runModeRitual(mode) {
      const ids = nextModeRitualIDs(mode);
      const payload = await postJSON('/v1/voice-mode-ritual', {
        device_id: currentDeviceID(),
        voice_mode: mode,
        trace_id: ids.trace_id,
        session_id: ids.session_id
      });
      setModeRitual(payload);
      const modes = await fetchJSON('/v1/voice-modes', { cache: 'no-store' });
      setVoiceModes(modes);
      await refreshHardwareAcceptance();
      log('mode ritual ' + ((payload && payload.selected_voice_mode) || mode) + ' ' + ((payload && payload.status) || 'sent'));
      return payload;
    }
    async function acceptModeRitualPhysical() {
      if (!state.modeRitual || !state.modeRitual.trace_id || !state.modeRitual.session_id) {
        throw new Error('run mode ritual first');
      }
      const mode = state.modeRitual.selected_voice_mode || 'roleplay';
      const payload = await postJSON('/v1/voice-mode-ritual-acceptance', {
        device_id: currentDeviceID(),
        voice_mode: mode,
        trace_id: state.modeRitual.trace_id,
        session_id: state.modeRitual.session_id,
        screen_visible: true,
        rgb_visible: true,
        servo_visible: true,
        observer: 'operator'
      });
      setModeRitual(Object.assign({}, state.modeRitual, {
        physical_accepted: !!payload.physical_accepted,
        acceptance_status: payload.status || '',
        accepted_surfaces: payload.accepted_surfaces || []
      }));
      await refreshHardwareAcceptance();
      log('mode ritual physical ' + mode + ' ' + (payload.status || 'accepted'));
      return payload;
    }
    function wakeWordRequestBody(mode) {
      const body = {
        mode: mode || ui.wakeWordModeSelect.value,
        threshold: Number(ui.wakeWordThreshold.value || 30)
      };
      if (body.mode === 'custom_multinet') {
        body.desired_phrase = ui.wakeWordPhrase.value.trim();
        body.desired_pinyin = ui.wakeWordPinyin.value.trim();
      }
      return body;
    }
    async function refreshWakeWord() {
      const payload = await fetchJSON('/v1/wake-word', { cache: 'no-store' });
      setWakeWord(payload);
      log('wake word ' + (payload.runtime_status || 'loaded'));
    }
    async function saveWakeWordSetup() {
      const payload = await putJSON('/v1/wake-word', wakeWordRequestBody());
      setWakeWord(payload);
      log('wake word ' + (payload.runtime_status || 'saved'));
    }
    async function resetWakeWordSetup() {
      const payload = await putJSON('/v1/wake-word', { mode: 'builtin_xiaozhi', threshold: 30 });
      setWakeWord(payload);
      log('wake word builtin_xiaozhi');
    }
    function clearReadFilters() {
      ui.readRecordFilter.value = '';
      ui.readTraceFilter.value = '';
      ui.readSessionFilter.value = '';
      refreshReads().catch((err) => log('reads ' + err.message));
    }
    function safeSource(source) {
      return {
        source_id: source.source_id || '',
        job_id: source.job_id || '',
        document_id: source.document_id || '',
        document_hash: source.document_hash || '',
        storage_status: source.storage_status || '',
        user_id: source.user_id || '',
        workspace_id: source.workspace_id || '',
        source_scope: source.source_scope || '',
        source_kind: source.source_kind || '',
        document_label: source.document_label || '',
        content_type: source.content_type || '',
        size_bytes: source.size_bytes || 0,
        readiness: source.readiness || '',
        index_status: source.index_status || '',
        created_at_ms: source.created_at_ms || 0,
        updated_at_ms: source.updated_at_ms || 0,
        trace_id: source.trace_id || '',
        session_id: source.session_id || '',
        device_id: source.device_id || '',
        metadata_only: !!source.metadata_only,
        stored_local: !!source.stored_local,
        indexing_requested: !!source.indexing_requested,
        searchable: !!source.searchable,
        deleted: !!source.deleted
      };
    }
    function safeReadRecord(record) {
      return {
        record_id: record.record_id || '',
        status: record.status || '',
        trace_id: record.trace_id || '',
        session_id: record.session_id || '',
        device_id: record.device_id || '',
        user_id: record.user_id || '',
        workspace_id: record.workspace_id || '',
        query_scope: record.query_scope || '',
        privacy_scope: record.privacy_scope || '',
        latency_profile: record.latency_profile || '',
        answer_style: record.answer_style || '',
        utterance_bucket: record.utterance_bucket || '',
        workspace_status: record.workspace_status || '',
        source_scope_counts: record.source_scope_counts || {},
        started_at_ms: record.started_at_ms || 0,
        completed_at_ms: record.completed_at_ms || 0,
        failure_code: record.failure_code || ''
      };
    }
    function safeDeviceBinding(binding) {
      return {
        binding_id: binding.binding_id || '',
        device_id: binding.device_id || '',
        device_label: binding.device_label || '',
        user_id: binding.user_id || '',
        workspace_id: binding.workspace_id || '',
        status: binding.status || '',
        access_scope: binding.access_scope || '',
        allowed_query_scopes: binding.allowed_query_scopes || [],
        professional_allowed: !!binding.professional_allowed,
        physical_accepted: !!binding.physical_accepted,
        created_at_ms: binding.created_at_ms || 0,
        updated_at_ms: binding.updated_at_ms || 0,
        revoked_at_ms: binding.revoked_at_ms || 0
      };
    }
    function exportMetadata() {
      const payload = {
        schema_version: 'a21.workspace_console_export.v1',
        generated_at_ms: Date.now(),
        user_id: ui.userId.textContent,
        workspace_id: ui.workspaceId.textContent,
        query_scope: ui.queryScope.value,
        workspace_status: ui.workspaceStatus.textContent,
        adapter_contract_version: ui.adapterStatus.textContent,
        source_count: state.sources.length,
        device_binding_count: state.deviceBindings.length,
        read_record_count: state.readRecords.length,
        sources: state.sources.map(safeSource),
        device_bindings: state.deviceBindings.map(safeDeviceBinding),
        read_records: state.readRecords.map(safeReadRecord),
        roleplay_profile: ui.roleplayProfileStatus.textContent,
        roleplay_scenario: ui.roleplayScenarioStatus.textContent,
        roleplay_voice_profile: ui.roleplayVoiceStatus.textContent,
        roleplay_memory_status: ui.roleplayMemoryStatus.textContent,
        roleplay_expression: ui.roleplayExpression.textContent,
        mode_ritual_mode: (state.modeRitual && state.modeRitual.selected_voice_mode) || '',
        mode_ritual_trace_id: (state.modeRitual && state.modeRitual.trace_id) || '',
        mode_ritual_session_id: (state.modeRitual && state.modeRitual.session_id) || '',
        mode_ritual_status: (state.modeRitual && state.modeRitual.status) || '',
        mode_ritual_transport: (state.modeRitual && state.modeRitual.delivered_transport) || '',
        mode_ritual_physical_accepted: !!(state.modeRitual && state.modeRitual.physical_accepted),
        voice_chain_mode: ui.voiceChainModeStatus.textContent,
        voice_chain_asr_profile: ui.voiceChainASRStatus.textContent,
        voice_chain_llm_profile: ui.voiceChainLLMStatus.textContent,
        voice_chain_tts_profile: ui.voiceChainTTSStatus.textContent,
        voice_chain_realtime_provider: ui.voiceChainRealtimeStatus.textContent,
        wake_word_mode: ui.wakeWordModeSelect.value,
        wake_word_active_phrase: ui.wakeWordActiveStatus.textContent,
        wake_word_runtime_status: ui.wakeWordRuntimeStatus.textContent,
        wake_word_firmware_status: ui.wakeWordFirmwareStatus.textContent,
        wake_word_code: ui.wakeWordCodeStatus.textContent,
        voice_probe_mode: (state.voiceProbe && state.voiceProbe.mode) || '',
        voice_probe_trace_id: (state.voiceProbe && state.voiceProbe.trace_id) || '',
        voice_probe_session_id: (state.voiceProbe && state.voiceProbe.session_id) || '',
        voice_probe_route: (state.voiceProbe && state.voiceProbe.route) || '',
        voice_probe_event_count: (state.voiceProbe && state.voiceProbe.event_count) || 0,
        voice_probe_trace_markers: (state.voiceProbe && state.voiceProbe.trace_markers) || [],
        voice_probe_roleplay_profile: (state.voiceProbe && state.voiceProbe.roleplay_profile) || '',
        voice_probe_voice_profile: (state.voiceProbe && state.voiceProbe.voice_profile) || '',
        voice_probe_professional_status: (state.voiceProbe && state.voiceProbe.professional_status) || '',
        body_preset: (state.bodyPreset && state.bodyPreset.preset) || '',
        body_preset_trace_id: (state.bodyPreset && state.bodyPreset.trace_id) || '',
        body_preset_session_id: (state.bodyPreset && state.bodyPreset.session_id) || '',
        body_preset_status: (state.bodyPreset && state.bodyPreset.status) || '',
        body_preset_transport: (state.bodyPreset && state.bodyPreset.delivered_transport) || '',
        body_preset_physical_accepted: !!(state.bodyPreset && state.bodyPreset.physical_accepted),
        body_preset_trace_markers: (state.bodyPreset && state.bodyPreset.trace_markers) || [],
        body_motion: (state.bodyMotion && state.bodyMotion.motion) || '',
        body_motion_trace_id: (state.bodyMotion && state.bodyMotion.trace_id) || '',
        body_motion_session_id: (state.bodyMotion && state.bodyMotion.session_id) || '',
        body_motion_status: (state.bodyMotion && state.bodyMotion.status) || '',
        body_motion_transport: (state.bodyMotion && state.bodyMotion.delivered_transport) || '',
        body_motion_physical_accepted: !!(state.bodyMotion && state.bodyMotion.physical_accepted),
        body_motion_trace_markers: (state.bodyMotion && state.bodyMotion.trace_markers) || [],
        hardware_scene: (state.hardwareScene && state.hardwareScene.scene) || '',
        hardware_scene_trace_id: (state.hardwareScene && state.hardwareScene.trace_id) || '',
        hardware_scene_session_id: (state.hardwareScene && state.hardwareScene.session_id) || '',
        hardware_scene_status: (state.hardwareScene && state.hardwareScene.status) || '',
        hardware_scene_transport: (state.hardwareScene && state.hardwareScene.delivered_transport) || '',
        hardware_scene_physical_accepted: !!(state.hardwareScene && state.hardwareScene.physical_accepted),
        hardware_scene_acceptance_status: (state.hardwareScene && state.hardwareScene.acceptance_status) || '',
        hardware_scene_accepted_surfaces: (state.hardwareScene && state.hardwareScene.accepted_surfaces) || [],
        hardware_scene_step_count: ((state.hardwareScene && state.hardwareScene.steps) || []).length,
        hardware_scene_trace_markers: (state.hardwareScene && state.hardwareScene.trace_markers) || [],
        hardware_acceptance_status: (state.hardwareAcceptance && state.hardwareAcceptance.overall_status) || '',
        hardware_acceptance_physical_accepted: !!(state.hardwareAcceptance && state.hardwareAcceptance.physical_accepted),
        hardware_acceptance_items: (state.hardwareAcceptance && state.hardwareAcceptance.items) || [],
        screen_control_action: (state.hardwareScreen && state.hardwareScreen.action) || '',
        screen_control_trace_id: (state.hardwareScreen && state.hardwareScreen.trace_id) || '',
        screen_control_session_id: (state.hardwareScreen && state.hardwareScreen.session_id) || '',
        screen_control_status: (state.hardwareScreen && state.hardwareScreen.status) || '',
        screen_control_tool: (state.hardwareScreen && state.hardwareScreen.tool_name) || '',
        screen_control_transport: (state.hardwareScreen && state.hardwareScreen.delivered_transport) || '',
        screen_control_physical_accepted: !!(state.hardwareScreen && state.hardwareScreen.physical_accepted),
        screen_control_trace_markers: (state.hardwareScreen && state.hardwareScreen.trace_markers) || [],
        screen_control_mcp_advertised: !!(state.hardwareScreen && state.hardwareScreen.mcp_advertised),
        official_action: (state.officialAction && state.officialAction.action) || '',
        official_action_trace_id: (state.officialAction && state.officialAction.trace_id) || '',
        official_action_session_id: (state.officialAction && state.officialAction.session_id) || '',
        official_action_status: (state.officialAction && state.officialAction.status) || '',
        official_action_transport: (state.officialAction && state.officialAction.delivered_transport) || '',
        official_action_event: (state.officialAction && state.officialAction.event) || '',
        official_action_value: (state.officialAction && state.officialAction.value) || '',
        official_action_packet_count: (state.officialAction && state.officialAction.packet_count) || 0,
        official_action_physical_accepted: !!(state.officialAction && state.officialAction.official_action_physical_accepted),
        official_action_surfaces: (state.officialAction && state.officialAction.official_action_surfaces) || {},
        official_action_blocked_reason: (state.officialAction && state.officialAction.blocked_reason) || '',
        official_action_fallback: (state.officialAction && state.officialAction.fallback) || '',
        official_action_trace_markers: (state.officialAction && state.officialAction.trace_markers) || [],
        professional_cue: ui.professionalCue.textContent,
        redaction: {
          raw_content_included: false,
          private_location_included: false,
          secret_values_included: false,
          provider_result_included: false,
          voice_data_included: false,
          evidence_body_included: false
        }
      };
      state.lastExport = payload;
      const blob = new Blob([JSON.stringify(payload, null, 2)], { type: 'application/json' });
      const objectURL = URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = objectURL;
      link.download = 'a21-workspace-metadata.json';
      document.body.append(link);
      link.click();
      link.remove();
      URL.revokeObjectURL(objectURL);
      log('export a21.workspace_console_export.v1');
    }
    async function refreshRoleplayAndModes() {
      const roleplay = await fetchJSON('/v1/roleplay-profile', { cache: 'no-store' });
      setRoleplay(roleplay);
      const voiceCatalog = await fetchJSON('/v1/voice-chain-profiles', { cache: 'no-store' });
      setVoiceCatalog(voiceCatalog);
      const modes = await fetchJSON('/v1/voice-modes', { cache: 'no-store' });
      setVoiceModes(modes);
    }
    async function boot() {
      try {
        await refreshWorkspace();
        await refreshDeviceBindings();
        await refreshSources();
        await refreshReads();
        await refreshRoleplayAndModes();
        await refreshWakeWord();
        renderProbeTrace({ trace_id: '', events: [], summary: { event_count: 0 } });
        renderProbeReadRecords({ records: [] });
        renderBodyPresetTrace({ trace_id: '', events: [], summary: { event_count: 0 } });
        renderHardwareSceneTrace({ trace_id: '', events: [], summary: { event_count: 0 } });
        renderHardwareScreenTrace({ trace_id: '', events: [], summary: { event_count: 0 } });
        renderOfficialActionTrace({ trace_id: '', events: [], summary: { event_count: 0 } });
        await refreshHardwareAcceptance();
        setText(ui.serviceStatus, 'gateway contract ready');
      } catch (err) {
        setText(ui.serviceStatus, 'gateway unavailable');
        log('boot ' + err.message);
      }
    }
    ui.queryScope.addEventListener('change', () => saveScope().catch((err) => log('scope ' + err.message)));
    ui.refreshWorkspace.addEventListener('click', () => refreshWorkspace().catch((err) => log('workspace ' + err.message)));
    ui.refreshSources.addEventListener('click', () => refreshSources().catch((err) => log('sources ' + err.message)));
    ui.refreshDeviceBindings.addEventListener('click', () => refreshDeviceBindings().catch((err) => log('device bindings ' + err.message)));
    ui.refreshReads.addEventListener('click', () => refreshReads().catch((err) => log('reads ' + err.message)));
    ui.uploadDocument.addEventListener('click', () => uploadDocument().catch((err) => log('upload ' + err.message)));
    ui.requestIndex.addEventListener('click', () => requestIndex().catch((err) => log('index ' + err.message)));
    ui.bindDevice.addEventListener('click', () => bindDevice().catch((err) => log('device bind ' + err.message)));
    ui.revokeDevice.addEventListener('click', () => revokeDevice().catch((err) => log('device revoke ' + err.message)));
    ui.deleteSource.addEventListener('click', () => deleteSource().catch((err) => log('delete ' + err.message)));
    ui.exportMetadata.addEventListener('click', exportMetadata);
    ui.clearReadFilters.addEventListener('click', clearReadFilters);
    ui.roleplayProfileSelect.addEventListener('change', () => saveRoleplaySetup().catch((err) => log('roleplay ' + err.message)));
    ui.roleplayScenarioSelect.addEventListener('change', () => saveRoleplaySetup().catch((err) => log('roleplay ' + err.message)));
    ui.roleplayVoiceSelect.addEventListener('change', () => saveRoleplaySetup().catch((err) => log('roleplay ' + err.message)));
    ui.saveRoleplaySetup.addEventListener('click', () => saveRoleplaySetup({ includeMemory: true }).catch((err) => log('roleplay ' + err.message)));
    ui.clearRoleplayMemory.addEventListener('click', () => saveRoleplaySetup({ clearMemory: true }).catch((err) => log('roleplay ' + err.message)));
    ui.saveVoiceChainSetup.addEventListener('click', () => saveVoiceChainSetup().catch((err) => log('voice chain ' + err.message)));
    ui.modeRitualActions.addEventListener('click', (event) => {
      const button = event.target.closest('[data-mode-ritual]');
      if (!button) return;
      runModeRitual(button.dataset.modeRitual).catch((err) => log('mode ritual ' + err.message));
    });
    ui.acceptModeRitualPhysical.addEventListener('click', () => acceptModeRitualPhysical().catch((err) => log('mode ritual physical ' + err.message)));
    ui.refreshHardwareAcceptance.addEventListener('click', () => refreshHardwareAcceptance().catch((err) => log('hardware acceptance ' + err.message)));
    ui.saveWakeWordSetup.addEventListener('click', () => saveWakeWordSetup().catch((err) => log('wake word ' + err.message)));
    ui.resetWakeWordSetup.addEventListener('click', () => resetWakeWordSetup().catch((err) => log('wake word ' + err.message)));
    ui.runRoleplayProbe.addEventListener('click', () => runRoleplayProbe().catch((err) => log('probe roleplay ' + err.message)));
    ui.runProfessionalProbe.addEventListener('click', () => runProfessionalProbe().catch((err) => log('probe professional ' + err.message)));
    ui.refreshVoiceProbeTrace.addEventListener('click', () => refreshVoiceProbeTrace().catch((err) => log('probe trace ' + err.message)));
    ui.bodyPresetActions.addEventListener('click', (event) => {
      const button = event.target.closest('[data-body-preset]');
      if (button) {
        runBodyPreset(button.dataset.bodyPreset).catch((err) => log('body preset ' + err.message));
        return;
      }
      const motionButton = event.target.closest('[data-body-motion]');
      if (motionButton) {
        runBodyMotion(motionButton.dataset.bodyMotion).catch((err) => log('body motion ' + err.message));
      }
    });
    ui.refreshBodyPresetTrace.addEventListener('click', () => refreshBodyPresetTrace().catch((err) => log('body trace ' + err.message)));
    ui.hardwareSceneActions.addEventListener('click', (event) => {
      const button = event.target.closest('[data-hardware-scene]');
      if (!button) return;
      runHardwareScene(button.dataset.hardwareScene).catch((err) => log('hardware scene ' + err.message));
    });
    ui.acceptHardwareScenePhysical.addEventListener('click', () => acceptHardwareScenePhysical().catch((err) => log('hardware scene acceptance ' + err.message)));
    ui.refreshHardwareSceneTrace.addEventListener('click', () => refreshHardwareSceneTrace().catch((err) => log('hardware scene trace ' + err.message)));
    ui.screenBrightness.addEventListener('input', () => {
      ui.screenBrightnessValue.value = ui.screenBrightness.value;
      setText(ui.screenBrightnessStatus, 'brightness=' + ui.screenBrightness.value);
    });
    ui.applyScreenBrightness.addEventListener('click', () => runHardwareScreenAction('brightness').catch((err) => log('screen brightness ' + err.message)));
    document.querySelectorAll('[data-screen-theme]').forEach((button) => {
      button.addEventListener('click', () => runHardwareScreenAction('theme', button.dataset.screenTheme).catch((err) => log('screen theme ' + err.message)));
    });
    ui.runDeviceStatus.addEventListener('click', () => runHardwareScreenAction('device_status').catch((err) => log('device status ' + err.message)));
    ui.runScreenInfo.addEventListener('click', () => runHardwareScreenAction('screen_info').catch((err) => log('screen info ' + err.message)));
    ui.refreshMCPCapabilities.addEventListener('click', () => refreshMCPCapabilities().catch((err) => log('mcp capabilities ' + err.message)));
    ui.refreshHardwareScreenTrace.addEventListener('click', () => refreshHardwareScreenTrace().catch((err) => log('screen trace ' + err.message)));
    ui.officialActionYAngle.addEventListener('input', () => {
      ui.officialActionYAngleValue.value = ui.officialActionYAngle.value;
    });
    ui.officialActionControls.addEventListener('click', (event) => {
      const button = event.target.closest('[data-official-state],[data-official-face],[data-official-motion]');
      if (!button) return;
      if (button.dataset.officialState) {
        runOfficialAction('state', button.dataset.officialState);
      } else if (button.dataset.officialFace) {
        runOfficialAction('face', button.dataset.officialFace);
      } else if (button.dataset.officialMotion) {
        runOfficialAction('motion', button.dataset.officialMotion);
      }
    });
    ui.refreshOfficialActionTrace.addEventListener('click', () => refreshOfficialActionTrace().catch((err) => log('official action trace ' + err.message)));
    ui.voiceProbeModeSelect.addEventListener('change', () => {
      if (ui.voiceProbeModeSelect.value === 'professional') {
        ui.voiceProbeInput.placeholder = 'safe evidence cue';
      } else {
        ui.voiceProbeInput.placeholder = 'safe short cue';
      }
    });
    boot();
  </script>
</body>
</html>`
