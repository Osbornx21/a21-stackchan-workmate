package gateway

const workspaceConsoleHTMLPart05 = `        screen_control_action: (state.hardwareScreen && state.hardwareScreen.action) || '',
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
        official_relay_connected: !!(state.officialRelayStatus && state.officialRelayStatus.connected),
        official_relay_device_id: (state.officialRelayStatus && state.officialRelayStatus.official_device_id) || '',
        official_relay_transport: (state.officialRelayStatus && state.officialRelayStatus.delivered_transport) || '',
        official_relay_next_action: (state.officialRelayStatus && state.officialRelayStatus.next_action) || '',
        official_relay_last_event: (state.officialRelayStatus && state.officialRelayStatus.last_event) || '',
        official_relay_physical_accepted: !!(state.officialRelayStatus && state.officialRelayStatus.physical_accepted),
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
        await refreshConnectedDevice();
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
    ui.refreshConnectedDevice.addEventListener('click', () => refreshConnectedDevice().catch((err) => log('connected device ' + err.message)));
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
    ui.refreshOfficialRelayStatus.addEventListener('click', () => refreshOfficialRelayStatus().catch((err) => log('official relay status ' + err.message)));
    ui.refreshOfficialActionTrace.addEventListener('click', () => refreshOfficialActionTrace().catch((err) => log('official action trace ' + err.message)));
    ui.voiceProbeModeSelect.addEventListener('change', () => {
      if (ui.voiceProbeModeSelect.value === 'professional') {
        ui.voiceProbeInput.placeholder = '安全证据提示';
      } else {
        ui.voiceProbeInput.placeholder = '安全短提示';
      }
    });
    translateStaticStatusText();
    boot();
  </script>
</body>
</html>`
