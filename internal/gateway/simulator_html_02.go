package gateway

const simulatorHTMLPart02 = `      latencyV21: document.getElementById('latencyV21'),
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
      roleplayProfile: document.getElementById('roleplayProfile'),
      roleplayScenario: document.getElementById('roleplayScenario'),
      roleplayMemoryHint: document.getElementById('roleplayMemoryHint'),
      saveRoleplayMemory: document.getElementById('saveRoleplayMemory'),
      clearRoleplayMemory: document.getElementById('clearRoleplayMemory'),
      professionalQueryScope: document.getElementById('professionalQueryScope'),
      workspaceDocumentLabel: document.getElementById('workspaceDocumentLabel'),
      workspaceDocumentFile: document.getElementById('workspaceDocumentFile'),
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
      playbackSources: [],
      lastWorkspaceDocumentId: '',
      lastWorkspaceJobId: ''
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
    function setVoiceMode(mode, ritual) {
      ui.voiceMode.value = mode || 'roleplay';
      ui.voiceModeReadout.textContent = ui.voiceMode.value;
      const cue = (ritual && ritual.cue_text) || (ui.voiceMode.value === 'professional' ? '我在查，先把证据和置信度拉出来。' : '我在。你说，我先接住。');
      const screen = (ritual && ritual.screen_label) || (ui.voiceMode.value === 'professional' ? 'PRO' : 'A21');
      ui.modeRitualReadout.textContent = screen + ' / ' + cue;
    }
    function setRoleplayProfile(catalog) {
      const selectedProfile = catalog.selected_roleplay_profile || 'a21_roleplay_default';
      const profiles = catalog.profiles || [];
      if (profiles.length) {
        ui.roleplayProfile.innerHTML = profiles.map((profile) => optionHTML(profile, selectedProfile)).join('');
      }
      ui.roleplayProfile.value = selectedProfile;
      const selectedProfileMeta = profiles.find((profile) => (profile.id || '') === selectedProfile) || {};
      ui.roleplaySoulReadout.textContent = selectedProfileMeta.label || selectedProfile;
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
      const memory = catalog.memory || {};
      const count = (memory.user_preference_count || 0) + (memory.session_memory_count || 0);
      ui.roleplayMemoryReadout.textContent = (memory.status || 'empty') + ' / ' + count;
      const expression = catalog.expression_plan || {};
      const policy = expression.delivery_policy || 'no_send_plan_only';
      ui.roleplayExpressionReadout.textContent = policy + ' / ' + (expression.action_count || 0) + ' actions / ' + (expression.packet_count || 0) + ' packets';
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
      if (payload.evidence || payload.screen_cards || payload.speech_blocks) {
        renderProfessionalEvidence(payload);
        refreshProfessionalReadRecords();
      }
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
    function renderWorkspaceJob(job) {
      if (!job || !job.job_id) {
        ui.workspaceJobReadout.textContent = 'none';
        return;
      }
      sim.lastWorkspaceJobId = job.job_id || sim.lastWorkspaceJobId;
      ui.workspaceJobReadout.textContent = job.job_id + ' / ' + (job.status || 'unknown') + ' / ' + (job.index_status || 'index unknown');
    }
    function renderWorkspaceIndexJob(job) {
      if (!job || !job.index_job_id) {
        ui.workspaceIndexReadout.textContent = 'none';
        return;
      }
      ui.workspaceIndexReadout.textContent = job.index_job_id + ' / ' + (job.status || 'unknown') + ' / ' + (job.index_status || 'index unknown');
    }
    function renderWorkspaceDocument(document) {
      if (!document || !document.document_id) {
        ui.workspaceDocumentReadout.textContent = 'none';
        ui.workspaceDocumentStorage.textContent = 'none';
        return;
      }
      sim.lastWorkspaceDocumentId = document.document_id || sim.lastWorkspaceDocumentId;
      sim.lastWorkspaceJobId = document.job_id || sim.lastWorkspaceJobId;
      const shortHash = (document.document_hash || '').replace('sha256:', '').slice(0, 10);
      ui.workspaceDocumentReadout.textContent = document.document_id + ' / ' + (document.size_bytes || 0) + ' bytes' + (shortHash ? ' / ' + shortHash : '');
      ui.workspaceDocumentStorage.textContent = (document.storage_status || 'unknown') + ' / ' + (document.readiness || 'pending');
    }
    function renderWorkspaceSources(payload) {
      const sources = (payload && payload.sources) || [];
      const summary = (payload && payload.summary) || {};
      ui.workspaceSourceCount.textContent = String(sources.length);
      ui.workspaceSourceReadiness.textContent = [
        summary.workspace_status,
        formatSourceCounts(summary.source_scope_counts),
        'indexing ' + formatSourceCounts(summary.indexing_requested_source_scope_counts),
        'searchable ' + formatSourceCounts(summary.searchable_source_scope_counts)
      ].filter(Boolean).join(' / ') || 'none';
    }
    function formatSourceCounts(counts) {
      counts = counts || {};
      const parts = [];
      if (typeof counts.public === 'number') parts.push('public ' + counts.public);
      if (typeof counts.personal === 'number') parts.push('personal ' + counts.personal);
      return parts.join(' / ') || 'none';
    }
    function renderProfessionalReadRecords(payload) {
      const records = (payload && payload.records) || [];
      ui.professionalReadRecordCount.textContent = String(records.length);
      if (!records.length) {
        ui.professionalReadRecordStatus.textContent = (payload && payload.status) || 'none';
        ui.professionalReadRecordScope.textContent = 'none';
        ui.professionalReadRecordSources.textContent = 'none';
        ui.professionalReadRecordWorkspace.textContent = 'none';
        return;
      }
      const record = records[records.length - 1] || {};
      ui.professionalReadRecordStatus.textContent = record.status || 'unknown';
      if (record.failure_code) {
        ui.professionalReadRecordStatus.textContent += ' / ' + record.failure_code;
      }
      ui.professionalReadRecordScope.textContent = [record.query_scope, record.utterance_bucket].filter(Boolean).join(' / ') || 'none';
      ui.professionalReadRecordSources.textContent = formatSourceCounts(record.source_scope_counts);
      ui.professionalReadRecordWorkspace.textContent = [record.workspace_status, record.privacy_scope].filter(Boolean).join(' / ') || 'none';
    }
    async function refreshProfessionalReadRecords() {
      try {
        const query = sim.traceId ? '?trace_id=' + encodeURIComponent(sim.traceId) : '';
        const response = await fetch('/v1/professional-read-records' + query, { cache: 'no-store' });
        if (!response.ok) {
          log('professional read records error ' + response.status);
          return;
        }
        renderProfessionalReadRecords(await response.json());
      } catch (err) {
        log('professional read records unavailable');
      }
    }
    async function refreshWorkspaceSources() {
      try {
        const response = await fetch('/v1/workspace-sources', { cache: 'no-store' });
        if (!response.ok) {
          log('workspace sources error ' + response.status);
          return;
        }
        renderWorkspaceSources(await response.json());
      } catch (err) {
        log('workspace sources unavailable');
      }
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
        ui.registryRoleplayProfile.textContent = device.current_roleplay_profile || 'none';
        ui.registryRoleplayScenario.textContent = device.current_roleplay_scenario || 'none';
        const memoryState = device.roleplay_memory_ready ? 'ready' : 'empty';
        ui.registryRoleplayMemory.textContent = memoryState + ' / ' + (device.roleplay_memory_hint_count || 0);
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
        setVoiceMode(catalog.selected_voice_mode || 'roleplay', catalog.selected_ritual || null);
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
        setVoiceMode(catalog.selected_voice_mode || ui.voiceMode.value, catalog.selected_ritual || null);
        refreshRegistry();
      } catch (err) {
        log('voice mode save unavailable');
      }
    }
    async function saveRoleplayProfile(options) {
      options = options || {};
      try {
        const body = {
          roleplay_profile: ui.roleplayProfile.value,
          scenario: ui.roleplayScenario.value,
          voice_clone_profile: ui.voiceCloneProfile.value
        };
        if (options.includeMemory) {
          const hint = ui.roleplayMemoryHint.value.trim();
          body.memory_hints = hint ? [hint] : [];
        }
        if (options.clearMemory) {
          body.clear_memory = true;
        }
        const response = await fetch('/v1/roleplay-profile', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(body)
        });
        if (!response.ok) {
          log('roleplay save failed ' + response.status);
          refreshRoleplayProfile();
          return;
        }
`
