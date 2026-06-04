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
      grid-template-columns: repeat(4, minmax(0, 1fr));
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
      font-size: 14px;
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
          </div>
          <label>Document
            <input id="documentFile" type="file">
          </label>
          <div class="actions">
            <button id="uploadDocument">Upload</button>
            <button id="requestIndex">Request index</button>
            <button class="secondary" id="refreshSources">Sources</button>
            <button class="secondary" id="refreshReads">Read records</button>
          </div>
          <div class="status-strip">
            <div class="metric"><span>Storage</span><strong id="storageStatus">stored_local pending</strong></div>
            <div class="metric"><span>Index</span><strong id="indexStatus">not_started_no_execute</strong></div>
            <div class="metric"><span>Searchable</span><strong id="searchableStatus">false</strong></div>
            <div class="metric"><span>Hardware</span><strong id="physicalStatus">physical_accepted=false</strong></div>
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
            <div class="metric"><span>Reads</span><strong id="readCount">0</strong></div>
          </div>
          <div class="stack">
            <div class="row-list" id="sourceList" aria-label="Workspace sources"></div>
            <div class="row-list" id="readList" aria-label="Professional read records"></div>
          </div>
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
          <div class="mode-band">
            <div class="metric"><span>Roleplay expression</span><strong id="roleplayExpression">no_send_plan_only</strong></div>
            <div class="metric"><span>Professional cue</span><strong id="professionalCue">PRO / checking</strong></div>
          </div>
          <div class="log" id="eventLog" aria-label="Workspace event log">workspace console ready</div>
        </div>
      </section>
    </main>
  </div>
  <script>
    const state = { document: null, job: null, source: null, indexJob: null };
    const ui = {
      queryScope: document.getElementById('queryScope'),
      documentLabel: document.getElementById('documentLabel'),
      documentFile: document.getElementById('documentFile'),
      uploadDocument: document.getElementById('uploadDocument'),
      requestIndex: document.getElementById('requestIndex'),
      refreshWorkspace: document.getElementById('refreshWorkspace'),
      refreshSources: document.getElementById('refreshSources'),
      refreshReads: document.getElementById('refreshReads'),
      storageStatus: document.getElementById('storageStatus'),
      indexStatus: document.getElementById('indexStatus'),
      searchableStatus: document.getElementById('searchableStatus'),
      physicalStatus: document.getElementById('physicalStatus'),
      workspaceStatus: document.getElementById('workspaceStatus'),
      adapterStatus: document.getElementById('adapterStatus'),
      userId: document.getElementById('userId'),
      workspaceId: document.getElementById('workspaceId'),
      sourceCount: document.getElementById('sourceCount'),
      readCount: document.getElementById('readCount'),
      sourceList: document.getElementById('sourceList'),
      readList: document.getElementById('readList'),
      roleplayExpression: document.getElementById('roleplayExpression'),
      professionalCue: document.getElementById('professionalCue'),
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
    function row(title, left, right, tone) {
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
      subRight.textContent = 'searchable=false';
      a.append(strong, sub);
      b.append(tag, subRight);
      item.append(a, b);
      return item;
    }
    function renderSources(payload) {
      const sources = (payload && payload.sources) || [];
      ui.sourceList.textContent = '';
      setText(ui.sourceCount, sources.length);
      if (!sources.length) {
        ui.sourceList.append(row('No sources', 'stored_local intake empty', 'no_execute', 'warn'));
        return;
      }
      sources.forEach((source) => {
        const title = source.document_label || source.source_id;
        const left = [source.source_scope, source.readiness, source.index_status].filter(Boolean).join(' / ');
        const tone = source.readiness === 'indexing_requested_no_execute' ? 'warn' : 'ready';
        ui.sourceList.append(row(title, left, source.source_id, tone));
      });
    }
    function renderReadRecords(payload) {
      const records = (payload && payload.records) || [];
      ui.readList.textContent = '';
      setText(ui.readCount, records.length);
      if (!records.length) {
        ui.readList.append(row('No read records', 'professional route idle', 'v21 off', 'warn'));
        return;
      }
      records.forEach((record) => {
        const left = [record.query_scope, record.workspace_status, countsText(record.source_scope_counts)].filter(Boolean).join(' / ');
        const tone = record.status === 'completed' ? 'ready' : 'warn';
        ui.readList.append(row(record.record_id, left, record.status, tone));
      });
    }
    function setWorkspace(payload) {
      const runtime = (payload && payload.runtime) || {};
      setText(ui.userId, payload.selected_user_id || runtime.user_id);
      setText(ui.workspaceId, payload.selected_workspace_id || runtime.workspace_id);
      setText(ui.workspaceStatus, payload.workspace_status || runtime.workspace_status || 'contract_ready');
      setText(ui.adapterStatus, payload.adapter_contract_version || runtime.adapter_contract_version || 'a21.v21_adapter_query.v2');
      ui.queryScope.value = payload.selected_query_scope || runtime.query_scope || 'public_only';
      setText(ui.searchableStatus, String(!!runtime.v21_execution_allowed && runtime.workspace_status === 'searchable'));
    }
    function setRoleplay(payload) {
      const plan = (payload && payload.expression_plan) || {};
      const value = [plan.delivery_policy || 'no_send_plan_only', (plan.action_count || 0) + ' actions', (plan.packet_count || 0) + ' packets'].join(' / ');
      setText(ui.roleplayExpression, value);
    }
    function setVoiceModes(payload) {
      const ritual = (payload && payload.selected_ritual) || {};
      setText(ui.professionalCue, (ritual.screen_label || 'PRO') + ' / ' + (ritual.cue_text || 'checking'));
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
    async function refreshSources() {
      const payload = await fetchJSON('/v1/workspace-sources', { cache: 'no-store' });
      renderSources(payload);
      log('sources ' + (((payload.summary || {}).total_sources) || 0));
    }
    async function refreshReads() {
      const payload = await fetchJSON('/v1/professional-read-records', { cache: 'no-store' });
      renderReadRecords(payload);
      log('reads ' + ((payload.records || []).length));
    }
    async function refreshRoleplayAndModes() {
      const roleplay = await fetchJSON('/v1/roleplay-profile', { cache: 'no-store' });
      setRoleplay(roleplay);
      const modes = await fetchJSON('/v1/voice-modes', { cache: 'no-store' });
      setVoiceModes(modes);
    }
    async function boot() {
      try {
        await refreshWorkspace();
        await refreshSources();
        await refreshReads();
        await refreshRoleplayAndModes();
        setText(ui.serviceStatus, 'gateway contract ready');
      } catch (err) {
        setText(ui.serviceStatus, 'gateway unavailable');
        log('boot ' + err.message);
      }
    }
    ui.queryScope.addEventListener('change', () => saveScope().catch((err) => log('scope ' + err.message)));
    ui.refreshWorkspace.addEventListener('click', () => refreshWorkspace().catch((err) => log('workspace ' + err.message)));
    ui.refreshSources.addEventListener('click', () => refreshSources().catch((err) => log('sources ' + err.message)));
    ui.refreshReads.addEventListener('click', () => refreshReads().catch((err) => log('reads ' + err.message)));
    ui.uploadDocument.addEventListener('click', () => uploadDocument().catch((err) => log('upload ' + err.message)));
    ui.requestIndex.addEventListener('click', () => requestIndex().catch((err) => log('index ' + err.message)));
    boot();
  </script>
</body>
</html>`
