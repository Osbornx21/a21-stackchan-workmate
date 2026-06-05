package gateway

const workspaceConsoleHTMLPart04 = `      setText(ui.officialActionSurfaceStatus, officialSurfaceText(payload.official_action_surfaces || {}));
      if (payload.last_trace_id) {
        setText(ui.officialActionTraceStatus, 'trace=' + payload.last_trace_id);
      }
      if (payload.last_event) {
        setText(ui.officialActionEventStatus, 'event=' + payload.last_event);
      }
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
    async function refreshOfficialRelayStatus() {
      const payload = await fetchJSON('/v1/stackchan/official/status?device_id=' + encodeURIComponent(currentDeviceID()), { cache: 'no-store' });
      renderOfficialRelayStatus(payload);
      log('official relay ' + (payload.next_action || 'status'));
      return payload;
    }
    async function runOfficialAction(kind, value) {
      const action = kind + '-' + value;
      const ids = nextOfficialActionIDs(action);
      const body = {
        device_id: currentDeviceID(),
        event: kind,
        trace_id: ids.trace_id,
        session_id: ids.session_id,
        allow_mcp_fallback: true
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
        try {
          await refreshOfficialRelayStatus();
        } catch (refreshErr) {
          log('official relay status ' + refreshErr.message);
        }
        await refreshOfficialActionTrace();
        log('official action ' + action + ' ' + (payload.status || 'sent'));
      } catch (err) {
        try {
          await runOfficialActionFallback(kind, value, err.message);
          try {
            await refreshOfficialRelayStatus();
          } catch (refreshErr) {
            log('official relay status ' + refreshErr.message);
          }
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
        throw new Error('请先运行模式仪式');
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
        connected_device_count: state.devices.length,
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
`
