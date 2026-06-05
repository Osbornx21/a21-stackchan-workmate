package gateway

const workspaceConsoleHTMLPart02 = `          </div>
          <div class="status-strip">
            <div class="metric"><span>追踪</span><strong id="officialActionTraceStatus">追踪=无</strong></div>
            <div class="metric"><span>事件</span><strong id="officialActionEventStatus">事件=无</strong></div>
            <div class="metric"><span>数据包</span><strong id="officialActionPacketStatus">数据包=0</strong></div>
            <div class="metric"><span>传输</span><strong id="officialActionTransportStatus">StackChan 官方 WebSocket</strong></div>
            <div class="metric"><span>硬件面</span><strong id="officialActionSurfaceStatus">硬件面=无</strong></div>
            <div class="metric"><span>中继</span><strong id="officialRelaySocketStatus">连接=否</strong></div>
            <div class="metric"><span>下一步</span><strong id="officialRelayNextStatus">下一步=连接官方 StackChan WebSocket</strong></div>
          </div>
          <div class="row-list" id="officialActionTraceList" aria-label="官方动作追踪标记"></div>
        </div>
      </section>

      <section class="wide" aria-label="模式边界">
        <div class="panel-head">
          <h2>模式边界</h2>
          <div class="tagline">
            <span class="tag ready">陪伴模式</span>
            <span class="tag warn">专业模式</span>
            <span class="tag off">V21 执行=否</span>
          </div>
        </div>
        <div class="panel-body">
          <div class="actions" id="modeRitualActions">
            <button class="secondary" data-mode-ritual="roleplay">运行陪伴切换仪式</button>
            <button data-mode-ritual="professional">运行专业切换仪式</button>
            <button class="secondary" id="acceptModeRitualPhysical">确认可见模式仪式</button>
          </div>
          <div class="mode-band">
            <div class="metric"><span>陪伴表情</span><strong id="roleplayExpression">仅计划，不发送</strong></div>
            <div class="metric"><span>专业提示</span><strong id="professionalCue">专业 / 检查中</strong></div>
            <div class="metric"><span>模式仪式</span><strong id="modeRitualStatus">仪式=空闲</strong></div>
            <div class="metric"><span>仪式追踪</span><strong id="modeRitualTraceStatus">追踪=无</strong></div>
            <div class="metric"><span>实体验收</span><strong id="modeRitualPhysicalStatus">实体验收=否</strong></div>
          </div>
          <div class="log" id="eventLog" aria-label="工作台事件日志">工作台控制台已就绪</div>
        </div>
      </section>

      <section class="wide" aria-label="硬件验收">
        <div class="panel-head">
          <h2>验收看板</h2>
          <div class="tagline">
            <span class="tag warn" id="hardwareAcceptanceStatus">等待实体验收</span>
            <span class="tag off" id="hardwareAcceptanceDeviceStatus">设备=未知</span>
          </div>
        </div>
        <div class="panel-body">
          <div class="actions">
            <button class="secondary" id="refreshHardwareAcceptance">刷新验收</button>
          </div>
          <div class="status-strip">
            <div class="metric"><span>设备</span><strong id="hardwareAcceptanceDevice">设备=无</strong></div>
            <div class="metric"><span>连接</span><strong id="hardwareAcceptanceConnection">缺失</strong></div>
            <div class="metric"><span>实体验收</span><strong id="hardwareAcceptancePhysical">实体验收=否</strong></div>
            <div class="metric"><span>下一步</span><strong id="hardwareAcceptanceNext">运行全量检查</strong></div>
          </div>
          <div class="row-list" id="hardwareAcceptanceItems" aria-label="硬件验收项"></div>
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
      devices: [],
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
      officialRelayStatus: null,
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
      refreshConnectedDevice: document.getElementById('refreshConnectedDevice'),
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
      refreshOfficialRelayStatus: document.getElementById('refreshOfficialRelayStatus'),
      refreshOfficialActionTrace: document.getElementById('refreshOfficialActionTrace'),
      officialActionStatus: document.getElementById('officialActionStatus'),
      officialRelayStatus: document.getElementById('officialRelayStatus'),
      officialActionPhysicalStatus: document.getElementById('officialActionPhysicalStatus'),
      officialActionTraceStatus: document.getElementById('officialActionTraceStatus'),
      officialActionEventStatus: document.getElementById('officialActionEventStatus'),
      officialActionPacketStatus: document.getElementById('officialActionPacketStatus'),
      officialActionTransportStatus: document.getElementById('officialActionTransportStatus'),
      officialActionSurfaceStatus: document.getElementById('officialActionSurfaceStatus'),
      officialRelaySocketStatus: document.getElementById('officialRelaySocketStatus'),
      officialRelayNextStatus: document.getElementById('officialRelayNextStatus'),
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
    const zhTerms = [
      ['Cascade ASR -> LLM -> TTS', '级联：语音识别 -> 大模型 -> 语音合成'],
      ['Realtime voice', '实时语音'],
      ['A21 desk workmate', '紫悦桌面伙伴'],
      ['Wry peer', '有梗同伴'],
      ['Calm anchor', '稳定锚点'],
      ['Desk mouthpiece', '桌面嘴替'],
      ['Boss challenge', '老板挑战'],
      ['Engineer pushback', '工程反推'],
      ['User complaint', '用户投诉'],
      ['Pre-meeting', '会前'],
      ['Post-meeting', '会后'],
      ['Late-night radio', '深夜电台'],
      ['A21 natural voice', '紫悦自然声音'],
      ['A21 cloned voice', '紫悦克隆声音'],
      ['CosyVoice clone', 'CosyVoice 克隆'],
      ['MiniMax clone', 'MiniMax 克隆'],
      ['Qwen ASR realtime', 'Qwen 实时语音识别'],
      ['Doubao ASR realtime', '豆包实时语音识别'],
      ['Sherpa streaming local', 'Sherpa 本地流式识别'],
      ['StepFun 8k fast', 'StepFun 8k 快速'],
      ['DashScope Qwen flash', 'DashScope Qwen 快速'],
      ['SiliconFlow Qwen', 'SiliconFlow Qwen'],
      ['DeepSeek fallback', 'DeepSeek 降级'],
      ['Local Ollama', '本地 Ollama'],
      ['Doubao speech-to-speech realtime', '豆包端到端实时语音'],
      ['OpenAI realtime', 'OpenAI 实时'],
      ['Doubao realtime TTS bridge', '豆包实时 TTS 桥'],
      ['Qwen Omni realtime', 'Qwen Omni 实时'],
      ['Mode ritual', '模式仪式'],
      ['Full check', '全量检查'],
      ['Power lifecycle', '电源生命周期'],
      ['run_mode_ritual', '运行模式仪式'],
      ['verify_no_cable_power_button_boot', '验证无插线电源键启动'],
      ['a21.v21_adapter_query.v2', 'V21 适配器契约 v2'],
      ['available', '可用'],
      ['planned', '计划中'],
      ['recommended', '推荐'],
      ['fallback', '降级'],
      ['mac_local', 'Mac 本地'],
      ['default', '默认'],
      ['opt_in', '需主动选择'],
      ['connected device', '连接设备'],
      ['device bindings', '设备绑定'],
      ['device bind', '设备绑定'],
      ['device revoke', '撤销设备'],
      ['No sources', '暂无来源'],
      ['stored_local intake empty', '本地存储入口为空'],
      ['no_execute', '不执行'],
      ['No read records', '暂无读取记录'],
      ['professional route idle', '专业路径空闲'],
      ['v21 off', 'V21 关闭'],
      ['No trace markers', '暂无追踪标记'],
      ['No body trace markers', '暂无机身追踪标记'],
      ['No scene trace markers', '暂无场景追踪标记'],
      ['No screen trace markers', '暂无屏幕追踪标记'],
      ['No official action trace markers', '暂无官方动作追踪标记'],
      ['No probe read records', '暂无探针读取记录'],
      ['No acceptance items', '暂无验收项'],
      ['device_missing', '设备缺失'],
      ['probe idle', '探针空闲'],
      ['preset idle', '姿态空闲'],
      ['scene idle', '场景空闲'],
      ['screen idle', '屏幕空闲'],
      ['official action idle', '官方动作空闲'],
      ['device missing', '设备缺失'],
      ['not_delivered', '未送达'],
      ['delivered', '已送达'],
      ['disconnected', '未连接'],
      ['connected', '已连接'],
      ['blocked', '已阻止'],
      ['status', '状态'],
      ['workspace ', '工作台 '],
      ['scope ', '范围 '],
      ['sources ', '来源 '],
      ['reads ', '读取 '],
      ['filtered', '已筛选'],
      ['upload ', '上传 '],
      ['index ', '索引 '],
      ['delete ', '删除 '],
      ['document file missing', '缺少文档文件'],
      ['stored document missing', '缺少已存文档'],
      ['delete source missing', '缺少可删除来源'],
      ['device bound', '设备已绑定'],
      ['device revoked', '设备已撤销'],
      ['roleplay ', '陪伴模式 '],
      ['voice chain ', '语音链路 '],
      ['mode ritual physical', '模式仪式实体验收'],
      ['mode ritual ', '模式仪式 '],
      ['wake word ', '唤醒词 '],
      ['probe roleplay', '陪伴探针'],
      ['probe professional', '专业探针'],
      ['probe trace', '探针追踪'],
      ['body preset', '机身预设'],
      ['body motion', '机身动作'],
      ['body trace', '机身追踪'],
      ['hardware scene acceptance', '硬件场景验收'],
      ['hardware scene trace', '硬件场景追踪'],
      ['hardware scene', '硬件场景'],
      ['hardware acceptance', '硬件验收'],
      ['screen brightness', '屏幕亮度'],
      ['screen theme', '屏幕主题'],
      ['screen info', '屏幕信息'],
      ['screen trace', '屏幕追踪'],
      ['screen ', '屏幕 '],
      ['device status', '设备状态'],
      ['mcp capabilities', 'MCP 能力'],
      ['official relay status', '官方中继状态'],
      ['official relay', '官方中继'],
      ['official action fallback', '官方动作降级'],
      ['official action trace', '官方动作追踪'],
      ['official action', '官方动作'],
      ['after ', '原因 '],
      ['saved', '已保存'],
      ['loaded', '已加载'],
      ['recorded', '已记录'],
      ['ok', '正常'],
      ['mode ritual', '模式仪式'],
      ['workspace console ready', '工作台控制台已就绪'],
      ['gateway contract ready', '网关契约就绪'],
      ['gateway unavailable', '网关不可用'],
      ['gateway contract', '网关契约'],
      ['contract_ready', '契约就绪'],
      ['stored_local pending', '本地已存，待索引'],
      ['stored_local_pending_index', '本地已存，待索引'],
      ['metadata_only', '仅元数据'],
      ['not_started_no_execute', '未开始，不执行'],
      ['indexing_requested_no_execute', '已请求索引，不执行'],
      ['deleted_metadata_only', '已删除，仅保留元数据'],
      ['deleted_no_execute', '已删除，不执行'],
      ['binding_not_configured', '尚未配置绑定'],
      ['open_until_binding_configured', '未配置前开放'],
      ['workspace_access_status', '工作台访问状态'],
      ['device_binding_policy', '设备绑定策略'],
      ['prompt_input_ready', '提示输入就绪'],
      ['physical_accepted', '实体验收'],
      ['hot_switch', '热切换'],
      ['stepfun_not_selected', 'StepFun 未选择'],
      ['build_required', '需要构建'],
      ['runtime_hot_swap', '运行时热切换'],
      ['route=', '路径='],
      ['trace=', '追踪='],
      ['events=', '事件='],
      ['device=', '设备='],
      ['preset=', '姿态='],
      ['motion=', '动作='],
      ['scene=', '场景='],
      ['screen=', '屏幕='],
      ['theme:', '主题:'],
      ['brightness:', '亮度:'],
      ['brightness=', '亮度='],
      ['body=', '机身='],
      ['rgb:', 'RGB:'],
      ['pitch:', '俯仰:'],
      ['yaw:', '偏航:'],
      ['speed:', '速度:'],
      ['pose=', '姿态='],
      ['steps=', '步数='],
      ['ritual=', '仪式='],
      ['action=', '动作='],
      ['relay=', '中继='],
      ['event=', '事件='],
      ['packets=', '数据包='],
      ['surfaces=', '硬件面='],
      ['connected=', '连接='],
      ['next=', '下一步='],
      ['tool=', '工具='],
      ['mcp=', 'MCP='],
      ['allowed=', '允许='],
      ['blocked=', '阻止='],
      ['offset_ms=', '偏移毫秒='],
      ['searchable=', '可检索='],
      ['fallback=', '降级='],
      ['professional_query', '专业查询'],
      ['fast_companion_hybrid', '快速陪伴混合链路'],
      ['local_audio', '本地音频'],
      ['fallback_delivered', '降级已送达'],
      ['operator_visible_accepted', '操作员已确认可见'],
      ['operator_pending', '等待操作员确认'],
      ['physical_pending', '等待实体验收'],
      ['machine_evidence_pending', '等待机器证据'],
      ['run_full_check', '运行全量检查'],
      ['connect_official_stackchan_ws', '连接官方 StackChan WebSocket'],
      ['xiaozhi_mcp_sequence', '小智 MCP 序列'],
      ['stackchan_official_ws', 'StackChan 官方 WebSocket'],
      ['active_builtin_model', '内置模型生效'],
      ['builtin_active', '内置生效'],
      ['builtin_xiaozhi', '内置小智'],
      ['custom_multinet', '自定义 MultiNet'],
      ['public_only', '仅公共'],
      ['personal_only', '仅个人'],
      ['personal_plus_public', '个人 + 公共'],
      ['roleplay', '陪伴模式'],
      ['professional', '专业模式'],
      ['ready', '准备'],
      ['listening', '聆听中'],
      ['thinking', '思考中'],
      ['speaking', '说话中'],
      ['celebrate', '庆祝'],
      ['reset_idle', '复位待机'],
      ['look_up', '抬头'],
      ['nod', '点头'],
      ['shake', '摇头'],
      ['dance', '跳舞'],
      ['stop', '停止'],
      ['full_check', '全量检查'],
      ['showtime', '展示'],
      ['focus', '专注'],
      ['reset', '复位'],
      ['light', '浅色'],
      ['dark', '深色'],
      ['auto', '自动'],
      ['idle', '空闲'],
      ['none', '无'],
      ['unknown', '未知'],
      ['missing', '缺失'],
      ['online', '在线'],
      ['offline', '离线'],
      ['bound', '已绑定'],
      ['revoked', '已撤销'],
      ['deleted', '已删除'],
      ['completed', '已完成'],
      ['started', '已开始'],
      ['pending', '等待中'],
      ['sent', '已发送'],
      ['accepted', '已接受'],
      ['empty', '空'],
      ['true', '是'],
      ['false', '否'],
      ['PRO / checking', '专业 / 检查中'],
      ['no_send_plan_only', '仅计划，不发送']
    ];
    function escapeRegExp(value) {
      return String(value).replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
    }
    function cnValue(value) {
      if (value == null || value === '') return '无';
      let text = String(value);
      zhTerms.forEach(([from, to]) => {
        if (/^[A-Za-z0-9_]+$/.test(from)) {
          const pattern = new RegExp('(^|[^A-Za-z0-9_])' + escapeRegExp(from) + '(?=$|[^A-Za-z0-9_])', 'g');
          text = text.replace(pattern, '$1' + to);
        } else {
          text = text.split(from).join(to);
        }
      });
      return text;
    }
    function translateStaticStatusText() {
      [
        ui.serviceStatus,
        ui.storageStatus,
        ui.indexStatus,
        ui.searchableStatus,
        ui.deviceBindingStatus,
        ui.workspaceStatus,
        ui.roleplayPromptStatus,
        ui.roleplayPhysicalStatus,
        ui.roleplayMemoryStatus,
        ui.voiceChainHotSwitchStatus,
        ui.voiceChainFindingStatus,
        ui.wakeWordBuildStatus,
        ui.wakeWordHotSwapStatus,
        ui.wakeWordRuntimeStatus,
        ui.wakeWordFirmwareStatus,
        ui.wakeWordCodeStatus,
        ui.voiceProbeRouteStatus,
        ui.voiceProbeTraceStatus,
        ui.voiceProbeRoleplayStatus,
        ui.voiceProbeVoiceStatus,
        ui.voiceProbeMemoryStatus,
        ui.voiceProbeProfessionalStatus,
        ui.bodyPresetStatus,
        ui.bodyPresetPhysicalStatus,
        ui.bodyPresetTraceStatus,
        ui.bodyPresetLEDStatus,
        ui.bodyPresetHeadStatus,
        ui.hardwareSceneStatus,
        ui.hardwareScenePhysicalStatus,
        ui.hardwareSceneTraceStatus,
        ui.hardwareSceneScreenStatus,
        ui.hardwareSceneBodyStatus,
        ui.hardwareSceneStepStatus,
        ui.hardwareSceneAcceptanceStatus,
        ui.hardwareScreenStatus,
        ui.hardwareScreenPhysicalStatus,
        ui.hardwareScreenTraceStatus,
        ui.screenBrightnessStatus,
        ui.screenThemeStatus,
        ui.screenToolStatus,
        ui.mcpCapabilitiesStatus,
        ui.officialActionStatus,
        ui.officialRelayStatus,
        ui.officialActionPhysicalStatus,
        ui.officialActionTraceStatus,
        ui.officialActionEventStatus,
        ui.officialActionPacketStatus,
        ui.officialActionSurfaceStatus,
        ui.officialRelaySocketStatus,
        ui.officialRelayNextStatus,
        ui.roleplayExpression,
        ui.professionalCue,
        ui.modeRitualStatus,
        ui.modeRitualTraceStatus,
        ui.modeRitualPhysicalStatus,
        ui.hardwareAcceptanceStatus,
        ui.hardwareAcceptanceDeviceStatus,
        ui.hardwareAcceptanceDevice,
        ui.hardwareAcceptanceConnection,
        ui.hardwareAcceptancePhysical,
        ui.hardwareAcceptanceNext
      ].forEach((el) => {
        if (el) el.textContent = cnValue(el.textContent);
      });
      document.querySelectorAll('.tagline .tag').forEach((el) => {
        el.textContent = cnValue(el.textContent);
      });
    }
    function log(message) {
      const line = new Date().toLocaleTimeString() + '  ' + cnValue(message);
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
      el.textContent = cnValue(value == null || value === '' ? 'none' : String(value));
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
    function preferredConnectedDevice(devices) {
      const list = Array.isArray(devices) ? devices : [];
      return list.find((device) => device.connection_status === 'online' && device.capabilities && device.capabilities.xiaozhi_profile === 'stock')
        || list.find((device) => device.connection_status === 'online')
        || list[0]
        || null;
    }
    function shouldAutoAdoptDeviceID() {
      const value = (ui.deviceId.value || '').trim();
      return !value || value === 'stackchan-sim-001';
    }
    async function refreshConnectedDevice() {
      const payload = await fetchJSON('/v1/devices', { cache: 'no-store' });
      const devices = (payload && payload.devices) || [];
      state.devices = devices;
      const device = preferredConnectedDevice(devices);
      if (device && device.device_id && shouldAutoAdoptDeviceID()) {
        ui.deviceId.value = device.device_id;
      }
      const selected = device && device.device_id ? device.device_id : currentDeviceID();
      const status = device && device.connection_status ? device.connection_status : 'missing';
      setText(ui.deviceBindingStatus, 'device=' + selected + ' / ' + status);
      log('connected device ' + selected + ' / ' + status);
      return payload;
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
      return cnValue((option.label || option.id || 'none') + (option.status ? ' [' + option.status + ']' : ''));
    }
    function setSelectOptions(select, options, selected) {
      const current = selected || select.value;
      select.textContent = '';
      (options || []).forEach((option) => {
        const item = document.createElement('option');
        item.value = option.id || '';
        item.textContent = optionLabel(option);
        if ((option.id || '') === current) item.selected = true;
`
