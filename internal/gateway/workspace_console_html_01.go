package gateway

const workspaceConsoleHTMLPart01 = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <link rel="icon" href="data:,">
  <title>A21 工作台控制台</title>
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
      <h1>A21 工作台控制台</h1>
      <div class="header-meta"><span class="status-dot"></span><span id="serviceStatus">网关契约</span></div>
    </header>
    <main>
      <section aria-label="工作台设置">
        <div class="panel-head">
          <h2>工作台</h2>
          <button class="secondary" id="refreshWorkspace">刷新</button>
        </div>
        <div class="panel-body">
          <div class="grid">
            <label>查询范围
              <select id="queryScope">
                <option value="public_only">仅公共</option>
                <option value="personal_only">仅个人</option>
                <option value="personal_plus_public">个人 + 公共</option>
              </select>
            </label>
            <label>文档标签
              <input id="documentLabel" value="A21 PRD pack" autocomplete="off">
            </label>
            <label>设备
              <input id="deviceId" value="stackchan-sim-001" autocomplete="off">
            </label>
          </div>
          <label>文档
            <input id="documentFile" type="file">
          </label>
          <div class="actions">
            <button id="uploadDocument">上传文档</button>
            <button id="requestIndex">请求索引</button>
            <button class="secondary" id="bindDevice">绑定设备</button>
            <button class="secondary" id="revokeDevice">撤销绑定</button>
            <button class="secondary" id="refreshConnectedDevice">连接设备</button>
            <button class="secondary" id="refreshDeviceBindings">设备绑定</button>
            <button class="secondary" id="deleteSource">删除来源</button>
            <button class="secondary" id="exportMetadata">导出元数据</button>
            <button class="secondary" id="refreshSources">来源列表</button>
            <button class="secondary" id="refreshReads">读取记录</button>
          </div>
          <div class="status-strip">
            <div class="metric"><span>存储</span><strong id="storageStatus">本地已存，待索引</strong></div>
            <div class="metric"><span>索引</span><strong id="indexStatus">未开始，不执行</strong></div>
            <div class="metric"><span>可检索</span><strong id="searchableStatus">否</strong></div>
            <div class="metric"><span>设备绑定</span><strong id="deviceBindingStatus">尚未配置绑定</strong></div>
          </div>
        </div>
      </section>

      <section aria-label="就绪状态">
        <div class="panel-head">
          <h2>就绪状态</h2>
          <div class="tagline">
            <span class="tag ready" id="workspaceStatus">契约就绪</span>
            <span class="tag warn" id="adapterStatus">V21 适配器契约 v2</span>
          </div>
        </div>
        <div class="panel-body">
          <div class="status-strip">
            <div class="metric"><span>用户</span><strong id="userId">a21_local_user</strong></div>
            <div class="metric"><span>工作台</span><strong id="workspaceId">a21_local_workspace</strong></div>
            <div class="metric"><span>来源</span><strong id="sourceCount">0</strong></div>
            <div class="metric"><span>设备</span><strong id="deviceBindingCount">0</strong></div>
            <div class="metric"><span>读取</span><strong id="readCount">0</strong></div>
          </div>
          <div class="grid">
            <label>读取记录 ID
              <input id="readRecordFilter" placeholder="record_id" autocomplete="off">
            </label>
            <label>追踪 ID
              <input id="readTraceFilter" placeholder="trace_id" autocomplete="off">
            </label>
            <label>会话 ID
              <input id="readSessionFilter" placeholder="session_id" autocomplete="off">
            </label>
            <div class="actions">
              <button class="secondary" id="clearReadFilters">清空筛选</button>
            </div>
          </div>
          <div class="stack">
            <div class="row-list" id="sourceList" aria-label="工作台来源"></div>
            <div class="row-list" id="readList" aria-label="专业读取记录"></div>
          </div>
        </div>
      </section>

      <section class="wide" aria-label="陪伴模式设置">
        <div class="panel-head">
          <h2>陪伴模式设置</h2>
          <div class="tagline">
            <span class="tag ready" id="roleplayPromptStatus">提示输入就绪=否</span>
            <span class="tag warn" id="roleplayPhysicalStatus">实体验收=否</span>
          </div>
        </div>
        <div class="panel-body">
          <div class="grid">
            <label>角色设定
              <select id="roleplayProfileSelect"></select>
            </label>
            <label>场景
              <select id="roleplayScenarioSelect"></select>
            </label>
            <label>声音档案
              <select id="roleplayVoiceSelect"></select>
            </label>
            <label>记忆提示
              <input id="roleplayMemoryHint" placeholder="本轮边界内提示" autocomplete="off">
            </label>
          </div>
          <div class="actions">
            <button id="saveRoleplaySetup">保存陪伴模式</button>
            <button class="secondary" id="clearRoleplayMemory">清空记忆</button>
          </div>
          <div class="status-strip">
            <div class="metric"><span>角色设定</span><strong id="roleplayProfileStatus">a21_roleplay_default</strong></div>
            <div class="metric"><span>场景</span><strong id="roleplayScenarioStatus">desk_mouthpiece</strong></div>
            <div class="metric"><span>声音档案</span><strong id="roleplayVoiceStatus">a21_voice_default_dashscope</strong></div>
            <div class="metric"><span>记忆</span><strong id="roleplayMemoryStatus">空 / 0</strong></div>
          </div>
        </div>
      </section>

      <section class="wide" aria-label="语音链路设置">
        <div class="panel-head">
          <h2>语音链路设置</h2>
          <div class="tagline">
            <span class="tag ready" id="voiceChainHotSwitchStatus">热切换=是</span>
            <span class="tag warn" id="voiceChainFindingStatus">StepFun 未选择</span>
          </div>
        </div>
        <div class="panel-body">
          <div class="grid">
            <label>链路模式
              <select id="voiceChainModeSelect"></select>
            </label>
            <label>语音识别档案
              <select id="voiceChainASRSelect"></select>
            </label>
            <label>大模型档案
              <select id="voiceChainLLMSelect"></select>
            </label>
            <label>实时通道
              <select id="voiceChainRealtimeSelect"></select>
            </label>
          </div>
          <div class="actions">
            <button id="saveVoiceChainSetup">保存语音链路</button>
          </div>
          <div class="status-strip">
            <div class="metric"><span>链路</span><strong id="voiceChainModeStatus">cascade</strong></div>
            <div class="metric"><span>ASR</span><strong id="voiceChainASRStatus">dashscope_qwen_asr_realtime</strong></div>
            <div class="metric"><span>LLM</span><strong id="voiceChainLLMStatus">stepfun</strong></div>
            <div class="metric"><span>生效 TTS</span><strong id="voiceChainTTSStatus">dashscope_qwen_tts_realtime</strong></div>
            <div class="metric"><span>实时通道</span><strong id="voiceChainRealtimeStatus">doubao_realtime</strong></div>
          </div>
        </div>
      </section>

      <section class="wide" aria-label="唤醒词设置">
        <div class="panel-head">
          <h2>唤醒词设置</h2>
          <div class="tagline">
            <span class="tag warn" id="wakeWordBuildStatus">需要构建=否</span>
            <span class="tag off" id="wakeWordHotSwapStatus">运行时热切换=否</span>
          </div>
        </div>
        <div class="panel-body">
          <div class="grid">
            <label>唤醒模式
              <select id="wakeWordModeSelect">
                <option value="builtin_xiaozhi">内置小智</option>
                <option value="custom_multinet">自定义 MultiNet</option>
              </select>
            </label>
            <label>目标唤醒词
              <input id="wakeWordPhrase" value="小阿二一" autocomplete="off">
            </label>
            <label>目标拼音
              <input id="wakeWordPinyin" value="xiao a er yi" autocomplete="off">
            </label>
            <label>阈值
              <input id="wakeWordThreshold" type="number" min="1" max="100" value="30">
            </label>
          </div>
          <div class="actions">
            <button id="saveWakeWordSetup">保存唤醒词</button>
            <button class="secondary" id="resetWakeWordSetup">恢复内置</button>
          </div>
          <div class="status-strip">
            <div class="metric"><span>当前唤醒词</span><strong id="wakeWordActiveStatus">你好小智</strong></div>
            <div class="metric"><span>运行时</span><strong id="wakeWordRuntimeStatus">active_builtin_model</strong></div>
            <div class="metric"><span>固件</span><strong id="wakeWordFirmwareStatus">builtin_active</strong></div>
            <div class="metric"><span>代码</span><strong id="wakeWordCodeStatus">无</strong></div>
          </div>
        </div>
      </section>

      <section class="wide" aria-label="语音探针">
        <div class="panel-head">
          <h2>语音探针</h2>
          <div class="tagline">
            <span class="tag ready" id="voiceProbeRouteStatus">路径=空闲</span>
            <span class="tag warn" id="voiceProbeTraceStatus">追踪=无</span>
          </div>
        </div>
        <div class="panel-body">
          <div class="grid">
            <label>探针模式
              <select id="voiceProbeModeSelect">
                <option value="roleplay">陪伴模式</option>
                <option value="professional">专业模式</option>
              </select>
            </label>
            <label>探针提示
              <input id="voiceProbeInput" placeholder="安全短提示" autocomplete="off">
            </label>
          </div>
          <div class="actions">
            <button id="runRoleplayProbe">运行陪伴探针</button>
            <button id="runProfessionalProbe">运行专业探针</button>
            <button class="secondary" id="refreshVoiceProbeTrace">追踪标记</button>
          </div>
          <div class="status-strip">
            <div class="metric"><span>角色设定</span><strong id="voiceProbeRoleplayStatus">无</strong></div>
            <div class="metric"><span>声音档案</span><strong id="voiceProbeVoiceStatus">无</strong></div>
            <div class="metric"><span>记忆</span><strong id="voiceProbeMemoryStatus">无</strong></div>
            <div class="metric"><span>专业模式</span><strong id="voiceProbeProfessionalStatus">空闲</strong></div>
          </div>
          <div class="stack">
            <div class="row-list" id="voiceProbeTraceList" aria-label="语音探针追踪标记"></div>
            <div class="row-list" id="voiceProbeReadList" aria-label="语音探针读取记录"></div>
          </div>
        </div>
      </section>

      <section class="wide" aria-label="机身预设">
        <div class="panel-head">
          <h2>机身预设</h2>
          <div class="tagline">
            <span class="tag ready" id="bodyPresetStatus">姿态=空闲</span>
            <span class="tag warn" id="bodyPresetPhysicalStatus">实体验收=否</span>
          </div>
        </div>
        <div class="panel-body">
          <div class="actions" id="bodyPresetActions">
            <button class="secondary" data-body-preset="ready">准备</button>
            <button class="secondary" data-body-preset="listening">聆听</button>
            <button class="secondary" data-body-preset="thinking">思考</button>
            <button class="secondary" data-body-preset="speaking">说话</button>
            <button data-body-preset="celebrate">庆祝</button>
            <button class="secondary" data-body-preset="reset_idle">复位</button>
            <button class="secondary" data-body-motion="look_up">抬头</button>
            <button class="secondary" data-body-motion="nod">点头</button>
            <button class="secondary" data-body-motion="shake">摇头</button>
            <button class="secondary" data-body-motion="dance">MCP 舞动</button>
            <button class="secondary" data-body-motion="stop">停止动作</button>
            <button class="secondary" id="refreshBodyPresetTrace">追踪标记</button>
          </div>
          <div class="status-strip">
            <div class="metric"><span>追踪</span><strong id="bodyPresetTraceStatus">追踪=无</strong></div>
            <div class="metric"><span>RGB 灯</span><strong id="bodyPresetLEDStatus">RGB=无</strong></div>
            <div class="metric"><span>头部</span><strong id="bodyPresetHeadStatus">姿态=无</strong></div>
            <div class="metric"><span>传输</span><strong id="bodyPresetTransportStatus">小智 MCP 序列</strong></div>
          </div>
          <div class="row-list" id="bodyPresetTraceList" aria-label="机身预设追踪标记"></div>
        </div>
      </section>

      <section class="wide" aria-label="硬件场景">
        <div class="panel-head">
          <h2>硬件场景</h2>
          <div class="tagline">
            <span class="tag ready" id="hardwareSceneStatus">场景=空闲</span>
            <span class="tag warn" id="hardwareScenePhysicalStatus">实体验收=否</span>
          </div>
        </div>
        <div class="panel-body">
          <div class="actions" id="hardwareSceneActions">
            <button data-hardware-scene="full_check">全量检查</button>
            <button class="secondary" data-hardware-scene="showtime">展示</button>
            <button class="secondary" data-hardware-scene="focus">专注</button>
            <button class="secondary" data-hardware-scene="reset">复位</button>
            <button class="secondary" id="acceptHardwareScenePhysical">确认可见全量检查</button>
            <button class="secondary" id="refreshHardwareSceneTrace">追踪标记</button>
          </div>
          <div class="status-strip">
            <div class="metric"><span>追踪</span><strong id="hardwareSceneTraceStatus">追踪=无</strong></div>
            <div class="metric"><span>屏幕</span><strong id="hardwareSceneScreenStatus">屏幕=无</strong></div>
            <div class="metric"><span>机身</span><strong id="hardwareSceneBodyStatus">机身=无</strong></div>
            <div class="metric"><span>步数</span><strong id="hardwareSceneStepStatus">步数=0</strong></div>
            <div class="metric"><span>传输</span><strong id="hardwareSceneTransportStatus">小智 MCP 序列</strong></div>
            <div class="metric"><span>验收</span><strong id="hardwareSceneAcceptanceStatus">等待操作员确认</strong></div>
          </div>
          <div class="row-list" id="hardwareSceneTraceList" aria-label="硬件场景追踪标记"></div>
        </div>
      </section>

      <section class="wide" aria-label="屏幕控制">
        <div class="panel-head">
          <h2>屏幕控制</h2>
          <div class="tagline">
            <span class="tag ready" id="hardwareScreenStatus">屏幕=空闲</span>
            <span class="tag warn" id="hardwareScreenPhysicalStatus">实体验收=否</span>
          </div>
        </div>
        <div class="panel-body">
          <div class="grid">
            <label>亮度
              <input id="screenBrightness" type="range" min="0" max="100" step="1" value="55">
            </label>
            <label>数值
              <input id="screenBrightnessValue" value="55" readonly>
            </label>
          </div>
          <div class="actions">
            <button id="applyScreenBrightness">应用亮度</button>
            <button class="secondary" data-screen-theme="light">浅色</button>
            <button class="secondary" data-screen-theme="dark">深色</button>
            <button class="secondary" data-screen-theme="auto">自动</button>
            <button class="secondary" id="runDeviceStatus">设备状态</button>
            <button class="secondary" id="runScreenInfo">屏幕信息</button>
            <button class="secondary" id="refreshMCPCapabilities">能力</button>
            <button class="secondary" id="refreshHardwareScreenTrace">追踪标记</button>
          </div>
          <div class="status-strip">
            <div class="metric"><span>追踪</span><strong id="hardwareScreenTraceStatus">追踪=无</strong></div>
            <div class="metric"><span>亮度</span><strong id="screenBrightnessStatus">亮度=55</strong></div>
            <div class="metric"><span>主题</span><strong id="screenThemeStatus">主题=无</strong></div>
            <div class="metric"><span>工具</span><strong id="screenToolStatus">工具=无</strong></div>
            <div class="metric"><span>能力</span><strong id="mcpCapabilitiesStatus">MCP=未知</strong></div>
          </div>
          <div class="row-list" id="hardwareScreenTraceList" aria-label="屏幕控制追踪标记"></div>
        </div>
      </section>

      <section class="wide" aria-label="官方动作">
        <div class="panel-head">
          <h2>官方动作</h2>
          <div class="tagline">
            <span class="tag ready" id="officialActionStatus">动作=空闲</span>
            <span class="tag warn" id="officialRelayStatus">中继=未知</span>
            <span class="tag warn" id="officialActionPhysicalStatus">实体验收=否</span>
          </div>
        </div>
        <div class="panel-body">
          <div class="grid">
            <label>俯仰角
              <input id="officialActionYAngle" type="range" min="5" max="85" step="1" value="38">
            </label>
            <label>角度
              <input id="officialActionYAngleValue" value="38" readonly>
            </label>
          </div>
          <div class="actions" id="officialActionControls">
            <button class="secondary" data-official-state="idle">待机</button>
            <button class="secondary" data-official-state="listening">聆听中</button>
            <button class="secondary" data-official-state="thinking">思考中</button>
            <button class="secondary" data-official-state="speaking">说话中</button>
            <button class="secondary" data-official-face="happy">开心</button>
            <button class="secondary" data-official-face="attentive">专注</button>
            <button class="secondary" data-official-motion="look_up">抬头</button>
            <button class="secondary" data-official-motion="nod">点头</button>
            <button class="secondary" data-official-motion="shake">摇头</button>
            <button data-official-motion="dance">跳舞</button>
            <button class="secondary" data-official-motion="stop">停止</button>
            <button class="secondary" id="refreshOfficialRelayStatus">中继状态</button>
            <button class="secondary" id="refreshOfficialActionTrace">追踪标记</button>
`
