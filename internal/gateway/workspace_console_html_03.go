package gateway

const workspaceConsoleHTMLPart03 = `        select.append(item);
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
      strong.textContent = cnValue(title || 'none');
      sub.textContent = cnValue(left || 'none');
      tag.textContent = cnValue(right || 'unknown');
      subRight.textContent = cnValue(detail || 'searchable=false');
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
        throw new Error('请先运行全量检查');
      }
      if ((state.hardwareScene.scene || '') !== 'full_check') {
        throw new Error('请先运行全量检查');
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
    function renderOfficialRelayStatus(payload) {
      payload = payload || {};
      const connected = !!payload.connected;
      state.officialRelayStatus = payload;
      setText(ui.officialRelayStatus, 'relay=' + (connected ? 'connected' : 'disconnected'));
      setText(ui.officialRelaySocketStatus, 'connected=' + String(connected) + (payload.official_device_id ? ' / ' + payload.official_device_id : ''));
      setText(ui.officialRelayNextStatus, 'next=' + (payload.next_action || 'connect_official_stackchan_ws'));
      setText(ui.officialActionPhysicalStatus, 'physical_accepted=' + String(!!payload.physical_accepted));
      setText(ui.officialActionTransportStatus, payload.delivered_transport || 'xiaozhi_mcp_fallback_available');
      setText(ui.officialActionPacketStatus, 'packets=' + (payload.last_packet_count || 0));
`
