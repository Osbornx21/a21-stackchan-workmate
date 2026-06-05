package gateway

const simulatorHTMLPart03 = `        setRoleplayProfile(await response.json());
        if (options.includeMemory || options.clearMemory) {
          ui.roleplayMemoryHint.value = '';
        }
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
        renderWorkspaceJob(job);
        refreshWorkspaceSources();
        log('workspace job ' + (job.job_id || 'accepted') + ' ' + (job.status || 'no_execute'));
      } catch (err) {
        log('workspace job unavailable');
      }
    }
    async function uploadWorkspaceDocument() {
      const file = (ui.workspaceDocumentFile.files || [])[0];
      if (!file) {
        log('select a workspace document file first');
        return;
      }
      try {
        const queryScope = ui.professionalQueryScope.value;
        const form = new FormData();
        form.append('workspace_id', 'a21_local_workspace');
        form.append('user_id', 'a21_local_user');
        form.append('source_scope', queryScope === 'public_only' ? 'public' : 'personal');
        form.append('document_label', ui.workspaceDocumentLabel.value || 'workspace document');
        form.append('content_type', file.type || 'application/octet-stream');
        form.append('trace_id', sim.traceId || '');
        form.append('session_id', sim.sessionId || '');
        form.append('device_id', deviceId());
        form.append('file', file);
        const response = await fetch('/v1/workspace-documents', {
          method: 'POST',
          body: form
        });
        if (!response.ok) {
          log('workspace document upload failed ' + response.status);
          return;
        }
        const payload = await response.json();
        const document = (payload.documents || [])[0] || {};
        const job = (payload.jobs || [])[0] || {};
        renderWorkspaceDocument(document);
        renderWorkspaceJob(job);
        refreshWorkspaceSources();
        log('workspace document ' + (document.document_id || 'stored') + ' ' + (document.status || 'pending'));
      } catch (err) {
        log('workspace document upload unavailable');
      }
    }
    async function requestWorkspaceIndex() {
      const documentId = sim.lastWorkspaceDocumentId;
      const jobId = sim.lastWorkspaceJobId;
      if (!documentId && !jobId) {
        log('upload a workspace document before requesting index');
        return;
      }
      try {
        const body = {
          trace_id: sim.traceId || '',
          session_id: sim.sessionId || '',
          device_id: deviceId()
        };
        if (documentId) body.document_id = documentId;
        if (jobId) body.job_id = jobId;
        const response = await fetch('/v1/workspace-index-jobs', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(body)
        });
        if (!response.ok) {
          log('workspace index request failed ' + response.status);
          return;
        }
        const payload = await response.json();
        const indexJob = (payload.jobs || [])[0] || {};
        renderWorkspaceIndexJob(indexJob);
        refreshWorkspaceSources();
        refreshProfessionalWorkspace();
        log('workspace index ' + (indexJob.index_job_id || 'requested') + ' ' + (indexJob.status || 'no_execute'));
      } catch (err) {
        log('workspace index request unavailable');
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
    document.getElementById('workspaceDocumentUpload').addEventListener('click', uploadWorkspaceDocument);
    document.getElementById('workspaceIndexRequest').addEventListener('click', requestWorkspaceIndex);
    document.getElementById('workspaceSourcesRefresh').addEventListener('click', refreshWorkspaceSources);
    document.getElementById('professionalReadRecordsRefresh').addEventListener('click', refreshProfessionalReadRecords);
    ui.mode.addEventListener('change', () => setMode(ui.mode.value));
    document.getElementById('startMic').addEventListener('click', startMicrophoneStream);
    document.getElementById('stopMic').addEventListener('click', stopMicrophoneStream);
    document.getElementById('mockAudioBurst').addEventListener('click', sendMockAudioBurst);
    document.getElementById('saveWakeWord').addEventListener('click', saveWakeWordConfig);
    ui.resetWakeWord.addEventListener('click', resetWakeWordConfig);
    ui.exportWakeWord.addEventListener('click', exportWakeWordConfig);
    ui.voiceMode.addEventListener('change', saveVoiceMode);
    ui.roleplayProfile.addEventListener('change', saveRoleplayProfile);
    ui.roleplayScenario.addEventListener('change', saveRoleplayProfile);
    ui.saveRoleplayMemory.addEventListener('click', () => saveRoleplayProfile({ includeMemory: true }));
    ui.clearRoleplayMemory.addEventListener('click', () => saveRoleplayProfile({ clearMemory: true }));
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
    refreshWorkspaceSources();
    refreshProfessionalReadRecords();
    setMode(ui.mode.value);
    updateVisibilityBadges();
  </script>
</body>
</html>`
