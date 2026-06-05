package gateway

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
)

func TestWorkspaceIndexJobRequestPromotesStoredDocumentWithoutLeakingContent(t *testing.T) {
	storeDir := filepath.Join(t.TempDir(), "a21-workspace-documents")
	server := NewServerWithOptions(ServerOptions{
		WorkspaceDocumentStoreDir: storeDir,
		WorkspaceDocumentMaxBytes: 1 << 20,
	})
	handler := server.Handler()
	privateContent := "RAW_PRIVATE_INDEX_DOCUMENT_CONTENT_FOR_A21"
	body, contentType := multipartWorkspaceDocumentBody(t, map[string]string{
		"user_id":        "a21_user_index",
		"workspace_id":   "a21_workspace_index",
		"source_scope":   "personal",
		"document_label": "Index readiness pack",
		"content_type":   "text/plain",
		"trace_id":       "a21-trace-index-upload",
		"session_id":     "a21-session-index-upload",
		"device_id":      "stackchan-sim-001",
	}, "SECRET-index-source.txt", privateContent)
	uploadReq := httptest.NewRequest(http.MethodPost, "/v1/workspace-documents", body)
	uploadReq.Header.Set("Content-Type", contentType)
	uploadRec := httptest.NewRecorder()
	handler.ServeHTTP(uploadRec, uploadReq)
	if uploadRec.Code != http.StatusOK {
		t.Fatalf("upload status = %d, want 200: %s", uploadRec.Code, uploadRec.Body.String())
	}
	var uploadResp WorkspaceDocumentsResponse
	if err := json.Unmarshal(uploadRec.Body.Bytes(), &uploadResp); err != nil {
		t.Fatal(err)
	}
	document := uploadResp.Documents[0]

	indexReq := httptest.NewRequest(http.MethodPost, "/v1/workspace-index-jobs", bytes.NewBufferString(`{
		"document_id":"`+document.DocumentID+`",
		"trace_id":"a21-trace-index-request",
		"session_id":"a21-session-index-request",
		"device_id":"stackchan-sim-001"
	}`))
	indexRec := httptest.NewRecorder()
	handler.ServeHTTP(indexRec, indexReq)
	if indexRec.Code != http.StatusOK {
		t.Fatalf("index status = %d, want 200: %s", indexRec.Code, indexRec.Body.String())
	}
	var indexResp WorkspaceIndexJobsResponse
	if err := json.Unmarshal(indexRec.Body.Bytes(), &indexResp); err != nil {
		t.Fatal(err)
	}
	if indexResp.SchemaVersion != "a21.gateway.workspace_index_jobs.v1" || indexResp.Status != "indexing_requested_no_execute" || len(indexResp.Jobs) != 1 {
		t.Fatalf("index response = %+v", indexResp)
	}
	indexJob := indexResp.Jobs[0]
	if indexJob.IndexJobID == "" ||
		indexJob.DocumentID != document.DocumentID ||
		indexJob.JobID != document.JobID ||
		indexJob.SourceID != document.SourceID ||
		indexJob.DocumentHash != document.DocumentHash ||
		indexJob.StorageStatus != "stored_local" ||
		indexJob.Status != "indexing_requested_no_execute" ||
		indexJob.IndexStatus != "indexing_requested_no_execute" ||
		indexJob.AdapterContractVersion != ProfessionalAdapterContractVersion ||
		indexJob.V21ExecutionAllowed ||
		indexJob.ExecutionStarted {
		t.Fatalf("index job = %+v", indexJob)
	}
	if indexJob.Redaction.DocumentTextStored ||
		!indexJob.Redaction.DocumentBytesStored ||
		indexJob.Redaction.Base64PayloadStored ||
		indexJob.Redaction.ImportURLStored ||
		indexJob.Redaction.LocalPathStored ||
		indexJob.Redaction.CredentialValueStored ||
		indexJob.Redaction.ProviderOutputStored {
		t.Fatalf("index redaction = %+v", indexJob.Redaction)
	}

	jobReq := httptest.NewRequest(http.MethodGet, "/v1/workspace-upload-jobs?job_id="+url.QueryEscape(document.JobID), nil)
	jobRec := httptest.NewRecorder()
	handler.ServeHTTP(jobRec, jobReq)
	if jobRec.Code != http.StatusOK {
		t.Fatalf("job status = %d, want 200: %s", jobRec.Code, jobRec.Body.String())
	}
	var jobs WorkspaceUploadJobsResponse
	if err := json.Unmarshal(jobRec.Body.Bytes(), &jobs); err != nil {
		t.Fatal(err)
	}
	if len(jobs.Jobs) != 1 ||
		jobs.Jobs[0].Status != "indexing_requested_no_execute" ||
		jobs.Jobs[0].IndexStatus != "indexing_requested_no_execute" ||
		!jobs.Jobs[0].IndexingAPIReady ||
		jobs.Jobs[0].ExecutionStarted {
		t.Fatalf("linked upload job = %+v", jobs)
	}

	sourcesReq := httptest.NewRequest(http.MethodGet, "/v1/workspace-sources?workspace_id=a21_workspace_index&source_scope=personal", nil)
	sourcesRec := httptest.NewRecorder()
	handler.ServeHTTP(sourcesRec, sourcesReq)
	if sourcesRec.Code != http.StatusOK {
		t.Fatalf("sources status = %d, want 200: %s", sourcesRec.Code, sourcesRec.Body.String())
	}
	var sources WorkspaceSourcesResponse
	if err := json.Unmarshal(sourcesRec.Body.Bytes(), &sources); err != nil {
		t.Fatal(err)
	}
	if len(sources.Sources) != 1 ||
		sources.Sources[0].Readiness != "indexing_requested_no_execute" ||
		!sources.Sources[0].IndexingRequested ||
		!sources.Sources[0].StoredLocal ||
		sources.Sources[0].Searchable {
		t.Fatalf("index source = %+v", sources)
	}
	if sources.Summary.WorkspaceStatus != "indexing_requested_no_execute" ||
		sources.Summary.IndexingRequestedSources != 1 ||
		sources.Summary.IndexingRequestedSourceScopeCounts["personal"] != 1 ||
		sources.Summary.SearchableSourceScopeCounts["personal"] != 0 {
		t.Fatalf("index source summary = %+v", sources.Summary)
	}

	workspaceReq := httptest.NewRequest(http.MethodPost, "/v1/professional-workspace", bytes.NewBufferString(`{"user_id":"a21_user_index","workspace_id":"a21_workspace_index","query_scope":"personal_only"}`))
	workspaceRec := httptest.NewRecorder()
	handler.ServeHTTP(workspaceRec, workspaceReq)
	if workspaceRec.Code != http.StatusOK {
		t.Fatalf("workspace status = %d, want 200: %s", workspaceRec.Code, workspaceRec.Body.String())
	}
	var workspace ProfessionalWorkspaceResponse
	if err := json.Unmarshal(workspaceRec.Body.Bytes(), &workspace); err != nil {
		t.Fatal(err)
	}
	if workspace.WorkspaceStatus != "indexing_requested_no_execute" ||
		workspace.Runtime.QueryScopeReadiness != "indexing_requested_no_execute" ||
		!workspace.Runtime.IndexingAPIReady ||
		workspace.Runtime.SearchableSourceScopeCounts["personal"] != 0 ||
		workspace.Runtime.V21ExecutionAllowed {
		t.Fatalf("workspace = %+v runtime=%+v", workspace, workspace.Runtime)
	}

	getIndexReq := httptest.NewRequest(http.MethodGet, "/v1/workspace-index-jobs?document_id="+url.QueryEscape(document.DocumentID), nil)
	getIndexRec := httptest.NewRecorder()
	handler.ServeHTTP(getIndexRec, getIndexReq)
	if getIndexRec.Code != http.StatusOK || !strings.Contains(getIndexRec.Body.String(), indexJob.IndexJobID) {
		t.Fatalf("index get = %d %s", getIndexRec.Code, getIndexRec.Body.String())
	}
	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-index-request", nil)
	traceRec := httptest.NewRecorder()
	handler.ServeHTTP(traceRec, traceReq)
	for _, want := range []string{"workspace.index_job.requested_no_execute", "workspace.source.indexing_requested_no_execute"} {
		if !strings.Contains(traceRec.Body.String(), want) {
			t.Fatalf("trace missing %q: %s", want, traceRec.Body.String())
		}
	}
	for _, forbidden := range []string{privateContent, "SECRET-index-source", storeDir, "/Users/", "api_key", "bearer ", "provider output"} {
		combined := indexRec.Body.String() + jobRec.Body.String() + sourcesRec.Body.String() + workspaceRec.Body.String() + getIndexRec.Body.String() + traceRec.Body.String()
		if strings.Contains(strings.ToLower(combined), strings.ToLower(forbidden)) {
			t.Fatalf("index surfaces leaked %q", forbidden)
		}
	}
}

func TestWorkspaceDocumentUploadRejectsRawJSONAndUnsafeLabelsWithoutStoring(t *testing.T) {
	storeDir := filepath.Join(t.TempDir(), "a21-workspace-documents")
	server := NewServerWithOptions(ServerOptions{WorkspaceDocumentStoreDir: storeDir})
	handler := server.Handler()

	jsonReq := httptest.NewRequest(http.MethodPost, "/v1/workspace-documents", bytes.NewBufferString(`{"document_text":"RAW_PRIVATE_JSON_DOCUMENT","file_path":"/Users/private/secret.pdf"}`))
	jsonReq.Header.Set("Content-Type", "application/json")
	jsonRec := httptest.NewRecorder()
	handler.ServeHTTP(jsonRec, jsonReq)
	if jsonRec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("json status = %d, want 415: %s", jsonRec.Code, jsonRec.Body.String())
	}
	for _, forbidden := range []string{"RAW_PRIVATE_JSON_DOCUMENT", "/Users/private", "secret.pdf"} {
		if strings.Contains(jsonRec.Body.String(), forbidden) {
			t.Fatalf("json rejection leaked %q: %s", forbidden, jsonRec.Body.String())
		}
	}

	body, contentType := multipartWorkspaceDocumentBody(t, map[string]string{
		"workspace_id":   "a21_workspace_upload",
		"document_label": "https://secret.example/file.pdf",
		"source_scope":   "personal",
	}, "unsafe-secret.pdf", "RAW_PRIVATE_MULTIPART_DOCUMENT")
	unsafeReq := httptest.NewRequest(http.MethodPost, "/v1/workspace-documents", body)
	unsafeReq.Header.Set("Content-Type", contentType)
	unsafeRec := httptest.NewRecorder()
	handler.ServeHTTP(unsafeRec, unsafeReq)
	if unsafeRec.Code != http.StatusBadRequest {
		t.Fatalf("unsafe status = %d, want 400: %s", unsafeRec.Code, unsafeRec.Body.String())
	}
	for _, forbidden := range []string{"secret.example", "unsafe-secret.pdf", "RAW_PRIVATE_MULTIPART_DOCUMENT"} {
		if strings.Contains(unsafeRec.Body.String(), forbidden) {
			t.Fatalf("unsafe rejection leaked %q: %s", forbidden, unsafeRec.Body.String())
		}
	}
	if stored := readAllFilesUnder(t, storeDir); stored != "" {
		t.Fatalf("unsafe uploads stored bytes: %q", stored)
	}
}

func TestWorkspaceIndexJobsRejectRawPayloadFields(t *testing.T) {
	server := NewServer()
	req := httptest.NewRequest(http.MethodPost, "/v1/workspace-index-jobs", bytes.NewBufferString(`{
		"document_id":"a21-workspace-document-000001",
		"document_text":"RAW_PRIVATE_INDEX_PAYLOAD",
		"file_path":"/Users/private/secret.pdf",
		"provider_output":"PRIVATE_PROVIDER_OUTPUT"
	}`))
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", rec.Code, rec.Body.String())
	}
	for _, forbidden := range []string{"RAW_PRIVATE_INDEX_PAYLOAD", "/Users/private", "secret.pdf", "PRIVATE_PROVIDER_OUTPUT"} {
		if strings.Contains(rec.Body.String(), forbidden) {
			t.Fatalf("rejection leaked %q: %s", forbidden, rec.Body.String())
		}
	}
}

func TestWorkspaceUploadJobsLifecycleIsNoExecuteAndRedacted(t *testing.T) {
	server := NewServer()
	handler := server.Handler()

	createReq := httptest.NewRequest(http.MethodPost, "/v1/workspace-upload-jobs", bytes.NewBufferString(`{
		"user_id":"a21_user_demo",
		"workspace_id":"a21_workspace_demo",
		"source_scope":"personal",
		"source_kind":"upload",
		"document_label":"PRD pack",
		"content_type":"application/pdf",
		"size_bytes":2048,
		"trace_id":"a21-trace-upload-job",
		"session_id":"a21-session-upload-job",
		"device_id":"stackchan-sim-001"
	}`))
	createRec := httptest.NewRecorder()
	handler.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusOK {
		t.Fatalf("create status = %d, want 200: %s", createRec.Code, createRec.Body.String())
	}
	var createResp WorkspaceUploadJobsResponse
	if err := json.Unmarshal(createRec.Body.Bytes(), &createResp); err != nil {
		t.Fatal(err)
	}
	if createResp.SchemaVersion != "a21.gateway.workspace_upload_jobs.v1" || len(createResp.Jobs) != 1 {
		t.Fatalf("create response = %+v", createResp)
	}
	job := createResp.Jobs[0]
	if job.JobID == "" ||
		job.UserID != "a21_user_demo" ||
		job.WorkspaceID != "a21_workspace_demo" ||
		job.SourceScope != "personal" ||
		job.SourceKind != "upload" ||
		job.DocumentLabel != "PRD pack" ||
		job.ContentType != "application/pdf" ||
		job.Status != "accepted_no_execute" ||
		job.IndexStatus != "not_started_no_execute" ||
		!job.UploadAPIReady ||
		!job.ImportAPIReady ||
		job.IndexingAPIReady ||
		job.ExecutionStarted ||
		!job.RetryAllowed ||
		!job.DeleteAllowed {
		t.Fatalf("job = %+v", job)
	}
	if job.Redaction.DocumentTextStored || job.Redaction.DocumentBytesStored || job.Redaction.Base64PayloadStored ||
		job.Redaction.ImportURLStored || job.Redaction.LocalPathStored || job.Redaction.CredentialValueStored ||
		job.Redaction.ProviderOutputStored {
		t.Fatalf("job redaction = %+v", job.Redaction)
	}

	failReq := httptest.NewRequest(http.MethodPut, "/v1/workspace-upload-jobs", bytes.NewBufferString(`{"job_id":"`+job.JobID+`","action":"mark_failed"}`))
	failRec := httptest.NewRecorder()
	handler.ServeHTTP(failRec, failReq)
	if failRec.Code != http.StatusOK {
		t.Fatalf("fail status = %d: %s", failRec.Code, failRec.Body.String())
	}
	if !strings.Contains(failRec.Body.String(), `"status":"failed"`) || !strings.Contains(failRec.Body.String(), `"index_status":"failed_no_execute"`) {
		t.Fatalf("fail response = %s", failRec.Body.String())
	}

	retryReq := httptest.NewRequest(http.MethodPut, "/v1/workspace-upload-jobs", bytes.NewBufferString(`{"job_id":"`+job.JobID+`","action":"retry"}`))
	retryRec := httptest.NewRecorder()
	handler.ServeHTTP(retryRec, retryReq)
	if retryRec.Code != http.StatusOK {
		t.Fatalf("retry status = %d: %s", retryRec.Code, retryRec.Body.String())
	}
	if !strings.Contains(retryRec.Body.String(), `"attempt":2`) || !strings.Contains(retryRec.Body.String(), `"status":"accepted_no_execute"`) {
		t.Fatalf("retry response = %s", retryRec.Body.String())
	}

	deleteReq := httptest.NewRequest(http.MethodPut, "/v1/workspace-upload-jobs", bytes.NewBufferString(`{"job_id":"`+job.JobID+`","action":"delete"}`))
	deleteRec := httptest.NewRecorder()
	handler.ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("delete status = %d: %s", deleteRec.Code, deleteRec.Body.String())
	}
	if !strings.Contains(deleteRec.Body.String(), `"status":"deleted"`) || !strings.Contains(deleteRec.Body.String(), `"document_label":"deleted"`) {
		t.Fatalf("delete response = %s", deleteRec.Body.String())
	}

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-upload-job", nil)
	traceRec := httptest.NewRecorder()
	handler.ServeHTTP(traceRec, traceReq)
	for _, want := range []string{"workspace.upload_job.accepted_no_execute", "workspace.index_job.not_started_no_execute", "workspace.upload_job.deleted"} {
		if !strings.Contains(traceRec.Body.String(), want) {
			t.Fatalf("trace missing %q: %s", want, traceRec.Body.String())
		}
	}
	for _, forbidden := range []string{"raw document", "http://", "https://", "/Users/", "secret", "api_key", "content_base64"} {
		if strings.Contains(strings.ToLower(createRec.Body.String()+failRec.Body.String()+retryRec.Body.String()+deleteRec.Body.String()+traceRec.Body.String()), strings.ToLower(forbidden)) {
			t.Fatalf("workspace upload job leaked %q", forbidden)
		}
	}
}

func TestWorkspaceSourcesRegistryTracksMetadataReadiness(t *testing.T) {
	server := NewServer()
	handler := server.Handler()

	createReq := httptest.NewRequest(http.MethodPost, "/v1/workspace-upload-jobs", bytes.NewBufferString(`{
		"user_id":"a21_user_sources",
		"workspace_id":"a21_workspace_sources",
		"source_scope":"personal",
		"source_kind":"upload",
		"document_label":"source readiness pack",
		"content_type":"application/pdf",
		"size_bytes":4096,
		"trace_id":"a21-trace-workspace-source",
		"session_id":"a21-session-workspace-source",
		"device_id":"stackchan-sim-001"
	}`))
	createRec := httptest.NewRecorder()
	handler.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusOK {
		t.Fatalf("create status = %d, want 200: %s", createRec.Code, createRec.Body.String())
	}
	var createResp WorkspaceUploadJobsResponse
	if err := json.Unmarshal(createRec.Body.Bytes(), &createResp); err != nil {
		t.Fatal(err)
	}
	if len(createResp.Jobs) != 1 || createResp.Jobs[0].SourceID == "" {
		t.Fatalf("create jobs = %+v, want source_id", createResp.Jobs)
	}
	job := createResp.Jobs[0]

	sourcesReq := httptest.NewRequest(http.MethodGet, "/v1/workspace-sources?workspace_id=a21_workspace_sources&source_scope=personal", nil)
	sourcesRec := httptest.NewRecorder()
	handler.ServeHTTP(sourcesRec, sourcesReq)
	if sourcesRec.Code != http.StatusOK {
		t.Fatalf("sources status = %d, want 200: %s", sourcesRec.Code, sourcesRec.Body.String())
	}
	var sources WorkspaceSourcesResponse
	if err := json.Unmarshal(sourcesRec.Body.Bytes(), &sources); err != nil {
		t.Fatal(err)
	}
	if sources.SchemaVersion != "a21.gateway.workspace_sources.v1" || len(sources.Sources) != 1 {
		t.Fatalf("sources response = %+v", sources)
	}
	source := sources.Sources[0]
	if source.SourceID != job.SourceID ||
		source.JobID != job.JobID ||
		source.UserID != "a21_user_sources" ||
		source.WorkspaceID != "a21_workspace_sources" ||
		source.SourceScope != "personal" ||
		source.Readiness != "metadata_only" ||
		source.IndexStatus != "not_started_no_execute" ||
		!source.MetadataOnly ||
		source.Searchable {
		t.Fatalf("source = %+v", source)
	}
	if sources.Summary.SourceScopeCounts["personal"] != 1 || sources.Summary.SearchableSourceScopeCounts["personal"] != 0 {
		t.Fatalf("source summary = %+v", sources.Summary)
	}

	searchableReq := httptest.NewRequest(http.MethodPut, "/v1/workspace-upload-jobs", bytes.NewBufferString(`{"job_id":"`+job.JobID+`","action":"mark_searchable"}`))
	searchableRec := httptest.NewRecorder()
	handler.ServeHTTP(searchableRec, searchableReq)
	if searchableRec.Code != http.StatusOK {
		t.Fatalf("mark_searchable status = %d, want 200: %s", searchableRec.Code, searchableRec.Body.String())
	}

	sourcesRec = httptest.NewRecorder()
	handler.ServeHTTP(sourcesRec, sourcesReq)
	if err := json.Unmarshal(sourcesRec.Body.Bytes(), &sources); err != nil {
		t.Fatal(err)
	}
	if len(sources.Sources) != 1 || sources.Sources[0].Readiness != "searchable_metadata_only" || !sources.Sources[0].Searchable {
		t.Fatalf("searchable sources = %+v", sources)
	}
	if sources.Summary.SearchableSourceScopeCounts["personal"] != 1 || sources.Summary.WorkspaceStatus != "searchable_metadata_only" {
		t.Fatalf("searchable summary = %+v", sources.Summary)
	}

	workspaceReq := httptest.NewRequest(http.MethodPost, "/v1/professional-workspace", bytes.NewBufferString(`{"user_id":"a21_user_sources","workspace_id":"a21_workspace_sources","query_scope":"personal_only"}`))
	workspaceRec := httptest.NewRecorder()
	handler.ServeHTTP(workspaceRec, workspaceReq)
	if workspaceRec.Code != http.StatusOK {
		t.Fatalf("workspace status = %d, want 200: %s", workspaceRec.Code, workspaceRec.Body.String())
	}
	var workspace ProfessionalWorkspaceResponse
	if err := json.Unmarshal(workspaceRec.Body.Bytes(), &workspace); err != nil {
		t.Fatal(err)
	}
	if workspace.WorkspaceStatus != "searchable_metadata_only" ||
		workspace.Runtime.SourceScopeCounts["personal"] != 1 ||
		workspace.Runtime.SearchableSourceScopeCounts["personal"] != 1 ||
		workspace.Runtime.QueryScopeReadiness != "searchable_metadata_only" ||
		workspace.Runtime.V21ExecutionAllowed {
		t.Fatalf("workspace source readiness = %+v runtime=%+v", workspace, workspace.Runtime)
	}

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id=a21-trace-workspace-source", nil)
	traceRec := httptest.NewRecorder()
	handler.ServeHTTP(traceRec, traceReq)
	for _, want := range []string{"workspace.source.created_metadata_only", "workspace.index_job.searchable_metadata_only"} {
		if !strings.Contains(traceRec.Body.String(), want) {
			t.Fatalf("trace missing %q: %s", want, traceRec.Body.String())
		}
	}
	for _, forbidden := range []string{"raw document", "content_base64", "http://", "https://", "/Users/", "secret", "api_key"} {
		if strings.Contains(strings.ToLower(sourcesRec.Body.String()+workspaceRec.Body.String()+traceRec.Body.String()), strings.ToLower(forbidden)) {
			t.Fatalf("workspace source leaked %q", forbidden)
		}
	}
}

func TestWorkspaceSourcesLifecycleSyncsRetryFailAndDelete(t *testing.T) {
	server := NewServer()
	handler := server.Handler()

	createReq := httptest.NewRequest(http.MethodPost, "/v1/workspace-upload-jobs", bytes.NewBufferString(`{
		"workspace_id":"a21_workspace_lifecycle",
		"source_scope":"public",
		"source_kind":"import",
		"document_label":"public release note",
		"content_type":"text/markdown"
	}`))
	createRec := httptest.NewRecorder()
	handler.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusOK {
		t.Fatalf("create status = %d, want 200: %s", createRec.Code, createRec.Body.String())
	}
	var createResp WorkspaceUploadJobsResponse
	if err := json.Unmarshal(createRec.Body.Bytes(), &createResp); err != nil {
		t.Fatal(err)
	}
	job := createResp.Jobs[0]

	failReq := httptest.NewRequest(http.MethodPut, "/v1/workspace-upload-jobs", bytes.NewBufferString(`{"job_id":"`+job.JobID+`","action":"mark_failed"}`))
	failRec := httptest.NewRecorder()
	handler.ServeHTTP(failRec, failReq)
	if failRec.Code != http.StatusOK {
		t.Fatalf("fail status = %d: %s", failRec.Code, failRec.Body.String())
	}
	source := fetchSingleWorkspaceSource(t, handler, job.SourceID)
	if source.Readiness != "failed_metadata_only" || source.IndexStatus != "failed_no_execute" || source.Searchable {
		t.Fatalf("failed source = %+v", source)
	}

	retryReq := httptest.NewRequest(http.MethodPut, "/v1/workspace-upload-jobs", bytes.NewBufferString(`{"job_id":"`+job.JobID+`","action":"retry"}`))
	retryRec := httptest.NewRecorder()
	handler.ServeHTTP(retryRec, retryReq)
	if retryRec.Code != http.StatusOK {
		t.Fatalf("retry status = %d: %s", retryRec.Code, retryRec.Body.String())
	}
	source = fetchSingleWorkspaceSource(t, handler, job.SourceID)
	if source.Readiness != "metadata_only" || source.IndexStatus != "not_started_no_execute" || source.Searchable {
		t.Fatalf("retried source = %+v", source)
	}

	deleteReq := httptest.NewRequest(http.MethodPut, "/v1/workspace-upload-jobs", bytes.NewBufferString(`{"job_id":"`+job.JobID+`","action":"delete"}`))
	deleteRec := httptest.NewRecorder()
	handler.ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("delete status = %d: %s", deleteRec.Code, deleteRec.Body.String())
	}
	source = fetchSingleWorkspaceSource(t, handler, job.SourceID)
	if source.Readiness != "deleted_metadata_only" || source.DocumentLabel != "deleted" || source.ContentType != "" || source.SizeBytes != 0 || !source.Deleted {
		t.Fatalf("deleted source = %+v", source)
	}
	if strings.Contains(fetchWorkspaceSourcesBody(t, handler, job.SourceID), "public release note") {
		t.Fatal("deleted source retained original document label")
	}
}

func TestWorkspaceSourcesRejectUnsafeFilters(t *testing.T) {
	server := NewServer()
	req := httptest.NewRequest(http.MethodGet, "/v1/workspace-sources?workspace_id=https%3A%2F%2Fsecret.example%2Fraw&source_scope=personal", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", rec.Code, rec.Body.String())
	}
	for _, forbidden := range []string{"secret.example", "raw", "https://"} {
		if strings.Contains(strings.ToLower(rec.Body.String()), strings.ToLower(forbidden)) {
			t.Fatalf("unsafe filter rejection leaked %q: %s", forbidden, rec.Body.String())
		}
	}
}

func TestWorkspaceUploadJobsRejectRawPayloadFields(t *testing.T) {
	server := NewServer()
	req := httptest.NewRequest(http.MethodPost, "/v1/workspace-upload-jobs", bytes.NewBufferString(`{
		"workspace_id":"a21_workspace_demo",
		"document_label":"secret.pdf",
		"content_base64":"UkFX",
		"import_url":"https://secret.example/file.pdf"
	}`))
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", rec.Code, rec.Body.String())
	}
	for _, forbidden := range []string{"UkFX", "secret.example", "secret.pdf"} {
		if strings.Contains(rec.Body.String(), forbidden) {
			t.Fatalf("rejection leaked %q: %s", forbidden, rec.Body.String())
		}
	}
}

func TestWorkspaceDeviceBindingsGateProfessionalQueries(t *testing.T) {
	v21 := &countingV21Client{}
	server := NewServerWithOptions(ServerOptions{V21Client: v21})
	handler := server.Handler()

	workspaceReq := httptest.NewRequest(http.MethodPost, "/v1/professional-workspace", bytes.NewBufferString(`{"user_id":"a21_user_device","workspace_id":"a21_workspace_device","query_scope":"public_only"}`))
	workspaceRec := httptest.NewRecorder()
	handler.ServeHTTP(workspaceRec, workspaceReq)
	if workspaceRec.Code != http.StatusOK {
		t.Fatalf("workspace status = %d: %s", workspaceRec.Code, workspaceRec.Body.String())
	}

	bindReq := httptest.NewRequest(http.MethodPost, "/v1/workspace-device-bindings", bytes.NewBufferString(`{"device_id":"stackchan-bound-001","device_label":"desk unit","user_id":"a21_user_device","workspace_id":"a21_workspace_device","allowed_query_scopes":["public_only"],"trace_id":"a21-trace-bind-device","session_id":"a21-session-bind-device"}`))
	bindRec := httptest.NewRecorder()
	handler.ServeHTTP(bindRec, bindReq)
	if bindRec.Code != http.StatusOK {
		t.Fatalf("bind status = %d: %s", bindRec.Code, bindRec.Body.String())
	}
	var bindResponse WorkspaceDeviceBindingsResponse
	if err := json.Unmarshal(bindRec.Body.Bytes(), &bindResponse); err != nil {
		t.Fatal(err)
	}
	if bindResponse.SchemaVersion != WorkspaceDeviceBindingsSchemaVersion ||
		bindResponse.Status != "bound" ||
		len(bindResponse.Bindings) != 1 ||
		bindResponse.Bindings[0].Status != "bound" ||
		!bindResponse.Bindings[0].ProfessionalAllowed ||
		bindResponse.Summary.ActiveBindings != 1 ||
		bindResponse.Summary.DeviceBindingPolicy != "bound_devices_only" {
		t.Fatalf("binding response = %+v", bindResponse)
	}
	if bindResponse.Redaction.PairingSecretStored || bindResponse.Redaction.DeviceCredentialStored ||
		bindResponse.Redaction.CredentialValueStored || bindResponse.Redaction.ProviderOutputStored ||
		bindResponse.Redaction.DocumentTextStored || bindResponse.Redaction.QueryTextStored ||
		bindResponse.Redaction.VoiceTranscriptStored {
		t.Fatalf("binding redaction = %+v", bindResponse.Redaction)
	}
	bindingID := bindResponse.Bindings[0].BindingID

	allowedReq := httptest.NewRequest(http.MethodPost, "/v1/mock-turn", bytes.NewBufferString(`{"device_id":"stackchan-bound-001","text":"查一下公开资料","mode":"professional","trace_id":"a21-trace-bound-device","session_id":"a21-session-bound-device"}`))
	allowedRec := httptest.NewRecorder()
	handler.ServeHTTP(allowedRec, allowedReq)
	if allowedRec.Code != http.StatusOK {
		t.Fatalf("allowed turn status = %d: %s", allowedRec.Code, allowedRec.Body.String())
	}
	if v21.calls != 1 {
		t.Fatalf("v21 calls after bound device = %d, want 1", v21.calls)
	}
	assertProfessionalReadRecordStatus(t, handler, "a21-trace-bound-device", "completed", "")
	assertTraceContains(t, handler, "a21-trace-bound-device", "professional.device_binding.bound")

	unboundReq := httptest.NewRequest(http.MethodPost, "/v1/mock-turn", bytes.NewBufferString(`{"device_id":"stackchan-unbound-001","text":"RAW_PRIVATE_DEVICE_SCOPE_QUERY","mode":"professional","trace_id":"a21-trace-unbound-device","session_id":"a21-session-unbound-device"}`))
	unboundRec := httptest.NewRecorder()
	handler.ServeHTTP(unboundRec, unboundReq)
	if unboundRec.Code != http.StatusOK {
		t.Fatalf("unbound turn status = %d: %s", unboundRec.Code, unboundRec.Body.String())
	}
	if v21.calls != 1 {
		t.Fatalf("v21 calls after unbound device = %d, want still 1", v21.calls)
	}
	assertProfessionalReadRecordStatus(t, handler, "a21-trace-unbound-device", "failed", "device_unbound")
	assertTraceContains(t, handler, "a21-trace-unbound-device", "professional.device_binding.blocked.device_unbound")
	assertTraceOmits(t, handler, "a21-trace-unbound-device", "v21.query.start")

	revokeReq := httptest.NewRequest(http.MethodPut, "/v1/workspace-device-bindings", bytes.NewBufferString(fmt.Sprintf(`{"binding_id":%q,"action":"revoke"}`, bindingID)))
	revokeRec := httptest.NewRecorder()
	handler.ServeHTTP(revokeRec, revokeReq)
	if revokeRec.Code != http.StatusOK {
		t.Fatalf("revoke status = %d: %s", revokeRec.Code, revokeRec.Body.String())
	}
	if !bytes.Contains(revokeRec.Body.Bytes(), []byte(`"status":"revoked"`)) ||
		!bytes.Contains(revokeRec.Body.Bytes(), []byte(`"professional_allowed":false`)) {
		t.Fatalf("revoke response missing revoked metadata: %s", revokeRec.Body.String())
	}

	revokedReq := httptest.NewRequest(http.MethodPost, "/v1/mock-turn", bytes.NewBufferString(`{"device_id":"stackchan-bound-001","text":"RAW_PRIVATE_REVOKED_DEVICE_QUERY","mode":"professional","trace_id":"a21-trace-revoked-device","session_id":"a21-session-revoked-device"}`))
	revokedRec := httptest.NewRecorder()
	handler.ServeHTTP(revokedRec, revokedReq)
	if revokedRec.Code != http.StatusOK {
		t.Fatalf("revoked turn status = %d: %s", revokedRec.Code, revokedRec.Body.String())
	}
	if v21.calls != 1 {
		t.Fatalf("v21 calls after revoked device = %d, want still 1", v21.calls)
	}
	assertProfessionalReadRecordStatus(t, handler, "a21-trace-revoked-device", "failed", "device_binding_revoked")
	assertTraceContains(t, handler, "a21-trace-revoked-device", "professional.device_binding.blocked.revoked")
}

func TestWorkspaceDeviceBindingsRejectUnsafePayloadFields(t *testing.T) {
	server := NewServer()
	req := httptest.NewRequest(http.MethodPost, "/v1/workspace-device-bindings", bytes.NewBufferString(`{
		"device_id":"http://secret.example/device",
		"user_id":"a21_user_device",
		"workspace_id":"a21_workspace_device",
		"pairing_secret":"RAW_PAIRING_SECRET",
		"device_credential":"RAW_DEVICE_CREDENTIAL"
	}`))
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", rec.Code, rec.Body.String())
	}
	for _, forbidden := range []string{"RAW_PAIRING_SECRET", "RAW_DEVICE_CREDENTIAL", "secret.example"} {
		if strings.Contains(rec.Body.String(), forbidden) {
			t.Fatalf("unsafe binding rejection leaked %q: %s", forbidden, rec.Body.String())
		}
	}
}

func TestProfessionalQueryEndpointExecutesBoundWorkspaceAndRedactsLedger(t *testing.T) {
	v21 := &capturingV21Client{}
	server := NewServerWithOptions(ServerOptions{V21Client: v21})
	handler := server.Handler()

	bindReq := httptest.NewRequest(http.MethodPost, "/v1/workspace-device-bindings", bytes.NewBufferString(`{"device_id":"stackchan-web-001","user_id":"a21_user_web","workspace_id":"a21_workspace_web","allowed_query_scopes":["personal_plus_public"]}`))
	bindRec := httptest.NewRecorder()
	handler.ServeHTTP(bindRec, bindReq)
	if bindRec.Code != http.StatusOK {
		t.Fatalf("bind status = %d: %s", bindRec.Code, bindRec.Body.String())
	}

	body := bytes.NewBufferString(`{"device_id":"stackchan-web-001","user_id":"a21_user_web","workspace_id":"a21_workspace_web","query_scope":"personal_plus_public","text":"RAW_PRIVATE_QUERY_ENDPOINT","trace_id":"a21-trace-professional-query","session_id":"a21-session-professional-query"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/professional-query", body)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("query status = %d: %s", rec.Code, rec.Body.String())
	}
	var response ProfessionalQueryResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.SchemaVersion != ProfessionalQuerySchemaVersion ||
		response.Status != "completed" ||
		response.Route != "professional_query" ||
		response.TraceID != "a21-trace-professional-query" ||
		response.SessionID != "a21-session-professional-query" ||
		response.DeviceID != "stackchan-web-001" ||
		response.Workspace.UserID != "a21_user_web" ||
		response.Workspace.WorkspaceID != "a21_workspace_web" ||
		response.Workspace.QueryScope != "personal_plus_public" ||
		response.DeviceBinding.Status != "bound" ||
		response.ReadRecordID == "" ||
		response.ReadRecordStatus != "completed" ||
		response.Answer == nil ||
		response.Answer.Text == "" ||
		response.EvidenceReport == nil ||
		response.EvidenceReport.EvidenceCount != 1 ||
		len(response.Events) < 4 {
		t.Fatalf("professional query response = %+v", response)
	}
	if v21.request.Utterance != "RAW_PRIVATE_QUERY_ENDPOINT" ||
		v21.request.UserID != "a21_user_web" ||
		v21.request.WorkspaceID != "a21_workspace_web" ||
		v21.request.QueryScope != "personal_plus_public" ||
		v21.request.Mode != "professional" ||
		v21.request.PrivacyScope != "professional_only" {
		t.Fatalf("v21 request = %+v", v21.request)
	}
	if strings.Contains(rec.Body.String(), "RAW_PRIVATE_QUERY_ENDPOINT") {
		t.Fatalf("professional query response echoed raw query: %s", rec.Body.String())
	}
	if response.Redaction.QueryTextStored || response.Redaction.RetrievedTextStored ||
		response.Redaction.ProviderOutputStored || response.Redaction.DocumentTextStored ||
		response.Redaction.CredentialValueStored || response.Redaction.VoiceTranscriptStored {
		t.Fatalf("query redaction = %+v", response.Redaction)
	}
	assertProfessionalReadRecordStatus(t, handler, "a21-trace-professional-query", "completed", "")
	assertTraceContains(t, handler, "a21-trace-professional-query", "http.professional_query.received")
	assertTraceContains(t, handler, "a21-trace-professional-query", "professional.query.started")
	assertTraceContains(t, handler, "a21-trace-professional-query", "professional.device_binding.bound")
	assertTraceContains(t, handler, "a21-trace-professional-query", "v21.query.start")

	recordsReq := httptest.NewRequest(http.MethodGet, "/v1/professional-read-records?trace_id=a21-trace-professional-query", nil)
	recordsRec := httptest.NewRecorder()
	handler.ServeHTTP(recordsRec, recordsReq)
	if strings.Contains(recordsRec.Body.String(), "RAW_PRIVATE_QUERY_ENDPOINT") {
		t.Fatalf("read records leaked query text: %s", recordsRec.Body.String())
	}
}

func TestProfessionalQueryEndpointBlocksUnboundDeviceBeforeV21(t *testing.T) {
	v21 := &countingV21Client{}
	server := NewServerWithOptions(ServerOptions{V21Client: v21})
	handler := server.Handler()

	bindReq := httptest.NewRequest(http.MethodPost, "/v1/workspace-device-bindings", bytes.NewBufferString(`{"device_id":"stackchan-bound-web-001","user_id":"a21_user_web_guard","workspace_id":"a21_workspace_web_guard","allowed_query_scopes":["public_only"]}`))
	bindRec := httptest.NewRecorder()
	handler.ServeHTTP(bindRec, bindReq)
	if bindRec.Code != http.StatusOK {
		t.Fatalf("bind status = %d: %s", bindRec.Code, bindRec.Body.String())
	}

	body := bytes.NewBufferString(`{"device_id":"stackchan-unbound-web-001","user_id":"a21_user_web_guard","workspace_id":"a21_workspace_web_guard","query_scope":"public_only","text":"RAW_PRIVATE_UNBOUND_QUERY_ENDPOINT","trace_id":"a21-trace-professional-query-unbound","session_id":"a21-session-professional-query-unbound"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/professional-query", body)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("query status = %d: %s", rec.Code, rec.Body.String())
	}
	var response ProfessionalQueryResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Status != "failed" ||
		response.FailureCode != "device_unbound" ||
		response.ReadRecordStatus != "failed" ||
		response.DeviceBinding.Status != "unbound" ||
		response.DeviceBinding.FailureCode != "device_unbound" {
		t.Fatalf("blocked query response = %+v", response)
	}
	if v21.calls != 0 {
		t.Fatalf("v21 calls after unbound professional query = %d, want 0", v21.calls)
	}
	if strings.Contains(rec.Body.String(), "RAW_PRIVATE_UNBOUND_QUERY_ENDPOINT") {
		t.Fatalf("blocked query response echoed raw query: %s", rec.Body.String())
	}
	assertProfessionalReadRecordStatus(t, handler, "a21-trace-professional-query-unbound", "failed", "device_unbound")
	assertTraceContains(t, handler, "a21-trace-professional-query-unbound", "professional.device_binding.blocked.device_unbound")
	assertTraceOmits(t, handler, "a21-trace-professional-query-unbound", "v21.query.start")
}

func TestProfessionalQueryEndpointRejectsUnsafePayloadFields(t *testing.T) {
	server := NewServer()
	req := httptest.NewRequest(http.MethodPost, "/v1/professional-query", bytes.NewBufferString(`{
		"device_id":"stackchan-web-001",
		"text":"查一下证据",
		"document_text":"RAW_DOCUMENT_TEXT",
		"provider_output":"RAW_PROVIDER_OUTPUT",
		"screen_cards":[{"text":"RAW_CARD"}]
	}`))
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", rec.Code, rec.Body.String())
	}
	for _, forbidden := range []string{"RAW_DOCUMENT_TEXT", "RAW_PROVIDER_OUTPUT", "RAW_CARD"} {
		if strings.Contains(rec.Body.String(), forbidden) {
			t.Fatalf("unsafe query rejection leaked %q: %s", forbidden, rec.Body.String())
		}
	}
}

func assertProfessionalReadRecordStatus(t *testing.T, handler http.Handler, traceID string, status string, failureCode string) {
	t.Helper()
	recordsReq := httptest.NewRequest(http.MethodGet, "/v1/professional-read-records?trace_id="+url.QueryEscape(traceID), nil)
	recordsRec := httptest.NewRecorder()
	handler.ServeHTTP(recordsRec, recordsReq)
	if recordsRec.Code != http.StatusOK {
		t.Fatalf("read-record status = %d: %s", recordsRec.Code, recordsRec.Body.String())
	}
	var response ProfessionalReadRecordsResponse
	if err := json.Unmarshal(recordsRec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Records) != 1 {
		t.Fatalf("read-record response = %+v", response)
	}
	record := response.Records[0]
	if record.Status != status || record.FailureCode != failureCode || record.TraceID != traceID {
		t.Fatalf("read record = %+v, want status=%s failure=%s trace=%s", record, status, failureCode, traceID)
	}
	for _, forbidden := range []string{"RAW_PRIVATE_DEVICE_SCOPE_QUERY", "RAW_PRIVATE_REVOKED_DEVICE_QUERY", "RAW_PAIRING_SECRET", "RAW_DEVICE_CREDENTIAL", "https://", "/Users/", "api_key", "token"} {
		if strings.Contains(strings.ToLower(recordsRec.Body.String()), strings.ToLower(forbidden)) {
			t.Fatalf("read records leaked %q: %s", forbidden, recordsRec.Body.String())
		}
	}
}

func assertTraceContains(t *testing.T, handler http.Handler, traceID string, marker string) {
	t.Helper()
	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id="+url.QueryEscape(traceID), nil)
	traceRec := httptest.NewRecorder()
	handler.ServeHTTP(traceRec, traceReq)
	if !strings.Contains(traceRec.Body.String(), marker) {
		t.Fatalf("trace %s missing %q: %s", traceID, marker, traceRec.Body.String())
	}
}

func assertTraceOmits(t *testing.T, handler http.Handler, traceID string, marker string) {
	t.Helper()
	traceReq := httptest.NewRequest(http.MethodGet, "/v1/traces?trace_id="+url.QueryEscape(traceID), nil)
	traceRec := httptest.NewRecorder()
	handler.ServeHTTP(traceRec, traceReq)
	if strings.Contains(traceRec.Body.String(), marker) {
		t.Fatalf("trace %s unexpectedly contained %q: %s", traceID, marker, traceRec.Body.String())
	}
}
