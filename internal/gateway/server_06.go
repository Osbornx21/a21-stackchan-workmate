package gateway

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"a21.local/a21/internal/v21adapter"
)

func professionalWorkspaceRedaction() ProfessionalWorkspaceRedaction {
	return ProfessionalWorkspaceRedaction{
		DocumentTextStored:    false,
		QueryTextStored:       false,
		RetrievedTextStored:   false,
		FullURLStored:         false,
		LocalPathStored:       false,
		CredentialValueStored: false,
		ProviderOutputStored:  false,
		VoiceTranscriptStored: false,
	}
}

func decodeProfessionalQueryRequest(r *http.Request) (ProfessionalQueryRequest, error) {
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		return ProfessionalQueryRequest{}, fmt.Errorf("invalid json")
	}
	for key := range raw {
		if professionalQueryForbiddenKey(key) {
			return ProfessionalQueryRequest{}, fmt.Errorf("professional query request must not include document text, evidence bodies, provider output, URLs, paths, credentials, base64, transcript, or audio payload fields")
		}
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return ProfessionalQueryRequest{}, fmt.Errorf("invalid json")
	}
	var req ProfessionalQueryRequest
	if err := json.Unmarshal(encoded, &req); err != nil {
		return ProfessionalQueryRequest{}, fmt.Errorf("invalid json")
	}
	return req, nil
}

func professionalQueryForbiddenKey(key string) bool {
	key = strings.ToLower(strings.TrimSpace(key))
	if workspaceUploadJobForbiddenKey(key) {
		return true
	}
	for _, forbidden := range []string{
		"answer",
		"fast_answer",
		"screen_card",
		"screen_cards",
		"speech_block",
		"speech_blocks",
		"follow_up",
		"follow_ups",
		"quote",
		"quotes",
		"retrieval",
		"retrieved",
		"source_text",
		"source_body",
	} {
		if key == forbidden {
			return true
		}
	}
	return false
}

func professionalQueryUtterance(req ProfessionalQueryRequest) string {
	if strings.TrimSpace(req.Text) != "" {
		return strings.TrimSpace(req.Text)
	}
	return strings.TrimSpace(req.Utterance)
}

func (s *Server) startProfessionalReadRecord(request v21adapter.QueryRequest, utteranceBucket string) string {
	nowMS := s.now().UnixMilli()
	record := ProfessionalReadRecord{
		Status:          "started",
		TraceID:         safeOptionalWorkspaceLabel(request.TraceID),
		SessionID:       safeOptionalWorkspaceLabel(request.SessionID),
		DeviceID:        safeOptionalWorkspaceLabel(request.DeviceID),
		UserID:          defaultProfessionalLabel(request.UserID, v21adapter.DefaultUserID),
		WorkspaceID:     defaultProfessionalLabel(request.WorkspaceID, v21adapter.DefaultWorkspaceID),
		QueryScope:      defaultProfessionalQueryScope(request.QueryScope),
		PrivacyScope:    defaultProfessionalPrivacyScope(request.PrivacyScope),
		LatencyProfile:  defaultProfessionalLatencyProfile(request.LatencyProfile),
		AnswerStyle:     defaultProfessionalAnswerStyle(request.AnswerStyle),
		UtteranceBucket: defaultProfessionalUtteranceBucket(utteranceBucket),
		StartedAtMS:     nowMS,
		Redaction:       professionalWorkspaceRedaction(),
	}
	s.mu.Lock()
	s.professionalReadRecordSeq++
	record.RecordID = fmt.Sprintf("a21-professional-read-%06d", s.professionalReadRecordSeq)
	s.professionalReadRecords[record.RecordID] = record
	s.mu.Unlock()
	if record.TraceID != "" {
		s.recordTrace(record.TraceID, record.SessionID, record.DeviceID, "professional.read_record.started", nowMS)
	}
	return record.RecordID
}

func (s *Server) completeProfessionalReadRecord(recordID string, response v21adapter.QueryResponse) {
	s.updateProfessionalReadRecord(recordID, "completed", "", safeProfessionalSourceScopeCounts(response.SourceScopeCounts), safeProfessionalWorkspaceStatus(response.WorkspaceStatus))
}

func (s *Server) failProfessionalReadRecord(recordID string, code string) {
	s.updateProfessionalReadRecord(recordID, "failed", safeProfessionalReadFailureCode(code), nil, "")
}

func (s *Server) updateProfessionalReadRecord(recordID string, status string, failureCode string, counts map[string]int, workspaceStatus string) {
	recordID = strings.TrimSpace(recordID)
	if recordID == "" {
		return
	}
	nowMS := s.now().UnixMilli()
	s.mu.Lock()
	record, ok := s.professionalReadRecords[recordID]
	if !ok {
		s.mu.Unlock()
		return
	}
	record.Status = status
	record.FailureCode = failureCode
	record.CompletedAtMS = nowMS
	record.SourceScopeCounts = counts
	record.WorkspaceStatus = workspaceStatus
	s.professionalReadRecords[recordID] = record
	s.mu.Unlock()
	if record.TraceID != "" {
		marker := "professional.read_record." + status
		s.recordTrace(record.TraceID, record.SessionID, record.DeviceID, marker, nowMS)
	}
}

func (s *Server) professionalReadRecordsResponse(recordID string, traceID string, sessionID string) ProfessionalReadRecordsResponse {
	records := s.professionalReadRecordsSnapshot(recordID, traceID, sessionID)
	status := "ok"
	if (strings.TrimSpace(recordID) != "" || strings.TrimSpace(traceID) != "" || strings.TrimSpace(sessionID) != "") && len(records) == 0 {
		status = "not_found"
	}
	return ProfessionalReadRecordsResponse{
		SchemaVersion: ProfessionalReadRecordsSchemaVersion,
		Service:       DeviceRegistryServiceName,
		Status:        status,
		Records:       records,
		Redaction:     professionalWorkspaceRedaction(),
	}
}

func (s *Server) professionalReadRecordsSnapshot(recordID string, traceID string, sessionID string) []ProfessionalReadRecord {
	recordID = strings.TrimSpace(recordID)
	traceID = strings.TrimSpace(traceID)
	sessionID = strings.TrimSpace(sessionID)
	s.mu.Lock()
	defer s.mu.Unlock()
	records := make([]ProfessionalReadRecord, 0, len(s.professionalReadRecords))
	keys := make([]string, 0, len(s.professionalReadRecords))
	if recordID != "" {
		keys = append(keys, recordID)
	} else {
		for key := range s.professionalReadRecords {
			keys = append(keys, key)
		}
		sort.Strings(keys)
	}
	for _, key := range keys {
		record, ok := s.professionalReadRecords[key]
		if !ok {
			continue
		}
		if traceID != "" && record.TraceID != traceID {
			continue
		}
		if sessionID != "" && record.SessionID != sessionID {
			continue
		}
		records = append(records, copyProfessionalReadRecord(record))
	}
	return records
}

func copyProfessionalReadRecord(record ProfessionalReadRecord) ProfessionalReadRecord {
	if len(record.SourceScopeCounts) > 0 {
		counts := make(map[string]int, len(record.SourceScopeCounts))
		for scope, count := range record.SourceScopeCounts {
			counts[scope] = count
		}
		record.SourceScopeCounts = counts
	}
	return record
}

func defaultProfessionalPrivacyScope(scope string) string {
	if strings.TrimSpace(scope) == "professional_only" {
		return "professional_only"
	}
	return "professional_only"
}

func defaultProfessionalLatencyProfile(profile string) string {
	switch strings.TrimSpace(profile) {
	case "fast_first":
		return "fast_first"
	default:
		return "fast_first"
	}
}

func defaultProfessionalAnswerStyle(style string) string {
	switch strings.TrimSpace(style) {
	case "voice_first_with_citations":
		return "voice_first_with_citations"
	default:
		return "voice_first_with_citations"
	}
}

func defaultProfessionalUtteranceBucket(bucket string) string {
	switch strings.TrimSpace(bucket) {
	case "length_empty", "length_1_16", "length_17_64", "length_65_160", "length_gt_160":
		return strings.TrimSpace(bucket)
	default:
		return "length_empty"
	}
}

func safeProfessionalSourceScopeCounts(counts map[string]int) map[string]int {
	if len(counts) == 0 {
		return nil
	}
	out := make(map[string]int, len(counts))
	for scope, count := range counts {
		switch scope {
		case "public", "personal":
		default:
			return nil
		}
		if count < 0 {
			return nil
		}
		out[scope] = count
	}
	return out
}

func safeProfessionalWorkspaceStatus(status string) string {
	switch strings.TrimSpace(status) {
	case v21adapter.WorkspaceSearchable, "uploaded", "indexing", "failed", "unavailable":
		return strings.TrimSpace(status)
	default:
		return ""
	}
}

func professionalReadFailureCode(err error, queryCtx context.Context) string {
	if errors.Is(err, context.DeadlineExceeded) || (queryCtx != nil && errors.Is(queryCtx.Err(), context.DeadlineExceeded)) {
		return "timeout"
	}
	if class := v21adapter.QueryFailureClassOf(err); class != "" {
		return string(class)
	}
	if statusClass := v21adapter.QueryFailureStatusClassOf(err); statusClass != "" {
		switch statusClass {
		case "4xx", "5xx":
			return "upstream_" + statusClass
		}
	}
	return "query_error"
}

func safeProfessionalReadFailureCode(code string) string {
	switch strings.TrimSpace(code) {
	case "timeout", "contract_invalid", "upstream_status", "no_evidence", "adapter_status", "transport_error", "query_error", "suppressed", "upstream_4xx", "upstream_5xx", "device_unbound", "device_binding_revoked", "device_scope_denied":
		return strings.TrimSpace(code)
	default:
		return "query_error"
	}
}

var errWorkspaceDocumentTooLarge = errors.New("workspace document exceeds A21 local intake limit")

type WorkspaceDocumentUploadRequest struct {
	UserID        string
	WorkspaceID   string
	SourceScope   string
	DocumentLabel string
	ContentType   string
	TraceID       string
	SessionID     string
	DeviceID      string
}

func workspaceDocumentUploadRequestFromForm(r *http.Request) WorkspaceDocumentUploadRequest {
	return WorkspaceDocumentUploadRequest{
		UserID:        strings.TrimSpace(r.FormValue("user_id")),
		WorkspaceID:   strings.TrimSpace(r.FormValue("workspace_id")),
		SourceScope:   strings.TrimSpace(r.FormValue("source_scope")),
		DocumentLabel: strings.TrimSpace(r.FormValue("document_label")),
		ContentType:   strings.TrimSpace(r.FormValue("content_type")),
		TraceID:       strings.TrimSpace(r.FormValue("trace_id")),
		SessionID:     strings.TrimSpace(r.FormValue("session_id")),
		DeviceID:      strings.TrimSpace(r.FormValue("device_id")),
	}
}

func (s *Server) storeWorkspaceDocumentUpload(req WorkspaceDocumentUploadRequest, file multipart.File, header *multipart.FileHeader) (WorkspaceDocumentsResponse, error) {
	userID, workspaceID, _, err := s.resolveProfessionalWorkspace(ProfessionalWorkspaceSelectionRequest{
		UserID:      req.UserID,
		WorkspaceID: req.WorkspaceID,
	})
	if err != nil {
		return WorkspaceDocumentsResponse{}, err
	}
	sourceScope := defaultWorkspaceSourceScope(req.SourceScope)
	if !validWorkspaceSourceScope(sourceScope) {
		return WorkspaceDocumentsResponse{}, fmt.Errorf("source_scope must be personal or public")
	}
	documentLabel := safeWorkspaceDocumentLabel(req.DocumentLabel)
	if documentLabel == "" {
		return WorkspaceDocumentsResponse{}, fmt.Errorf("valid redacted document_label is required")
	}
	contentType := safeWorkspaceContentType(req.ContentType)
	if contentType == "" && header != nil {
		contentType = safeWorkspaceContentType(header.Header.Get("Content-Type"))
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	traceID := safeOptionalWorkspaceLabel(req.TraceID)
	sessionID := safeOptionalWorkspaceLabel(req.SessionID)
	deviceID := safeOptionalWorkspaceLabel(req.DeviceID)
	maxBytes := s.workspaceDocumentMaxBytes
	if maxBytes <= 0 {
		maxBytes = defaultWorkspaceDocumentMaxBytes
	}
	hash, sizeBytes, err := s.storeWorkspaceDocumentFile(userID, workspaceID, sourceScope, file, maxBytes)
	if err != nil {
		if errors.Is(err, errWorkspaceDocumentTooLarge) {
			return WorkspaceDocumentsResponse{}, err
		}
		return WorkspaceDocumentsResponse{}, fmt.Errorf("workspace document local storage unavailable")
	}
	nowMS := s.now().UnixMilli()
	s.mu.Lock()
	s.workspaceDocumentSeq++
	documentID := fmt.Sprintf("a21-workspace-document-%06d", s.workspaceDocumentSeq)
	s.workspaceUploadJobSeq++
	jobID := fmt.Sprintf("a21-workspace-job-%06d", s.workspaceUploadJobSeq)
	s.workspaceSourceSeq++
	sourceID := fmt.Sprintf("a21-workspace-source-%06d", s.workspaceSourceSeq)
	documentHash := "sha256:" + hash
	findings := []WorkspaceUploadJobFinding{{
		Code:    "workspace_document_stored_local_pending_index",
		Message: "A21 stored the upload in the local workspace intake store; indexing and V21 execution remain disabled",
	}}
	document := WorkspaceDocument{
		DocumentID:    documentID,
		SourceID:      sourceID,
		JobID:         jobID,
		UserID:        userID,
		WorkspaceID:   workspaceID,
		SourceScope:   sourceScope,
		SourceKind:    "upload",
		DocumentLabel: documentLabel,
		ContentType:   contentType,
		SizeBytes:     sizeBytes,
		DocumentHash:  documentHash,
		Status:        "stored_local_pending_index",
		StorageStatus: "stored_local",
		IndexStatus:   "not_started_no_execute",
		Readiness:     "stored_local_pending_index",
		CreatedAtMS:   nowMS,
		UpdatedAtMS:   nowMS,
		TraceID:       traceID,
		SessionID:     sessionID,
		DeviceID:      deviceID,
		Redaction:     workspaceDocumentStoredRedaction(),
		Findings:      append([]WorkspaceUploadJobFinding(nil), findings...),
	}
	job := WorkspaceUploadJob{
		JobID:            jobID,
		SourceID:         sourceID,
		DocumentID:       documentID,
		DocumentHash:     documentHash,
		StorageStatus:    "stored_local",
		UserID:           userID,
		WorkspaceID:      workspaceID,
		SourceScope:      sourceScope,
		SourceKind:       "upload",
		DocumentLabel:    documentLabel,
		ContentType:      contentType,
		SizeBytes:        sizeBytes,
		Status:           "stored_local_pending_index",
		IndexStatus:      "not_started_no_execute",
		Attempt:          1,
		CreatedAtMS:      nowMS,
		UpdatedAtMS:      nowMS,
		TraceID:          traceID,
		SessionID:        sessionID,
		DeviceID:         deviceID,
		UploadAPIReady:   true,
		ImportAPIReady:   false,
		IndexingAPIReady: false,
		ExecutionStarted: false,
		RetryAllowed:     true,
		DeleteAllowed:    true,
		Redaction:        workspaceUploadJobStoredDocumentRedaction(),
		Findings:         append([]WorkspaceUploadJobFinding(nil), findings...),
	}
	source := workspaceSourceFromJob(job, "stored_local_pending_index")
	s.workspaceDocuments[documentID] = document
	s.workspaceUploadJobs[jobID] = job
	s.workspaceSources[sourceID] = source
	s.mu.Unlock()
	if traceID != "" {
		s.recordTrace(traceID, sessionID, deviceID, "workspace.document.stored_local", nowMS)
		s.recordTrace(traceID, sessionID, deviceID, "workspace.source.stored_local_pending_index", nowMS)
		s.recordTrace(traceID, sessionID, deviceID, "workspace.index_job.not_started_no_execute", nowMS)
	}
	return WorkspaceDocumentsResponse{
		SchemaVersion: WorkspaceDocumentsSchemaVersion,
		Service:       DeviceRegistryServiceName,
		Status:        "stored_local_pending_index",
		Documents:     []WorkspaceDocument{document},
		Jobs:          []WorkspaceUploadJob{job},
		Sources:       []WorkspaceSource{source},
		Redaction:     workspaceDocumentStoredRedaction(),
	}, nil
}

func (s *Server) storeWorkspaceDocumentFile(userID string, workspaceID string, sourceScope string, file multipart.File, maxBytes int64) (string, int64, error) {
	if maxBytes <= 0 {
		maxBytes = defaultWorkspaceDocumentMaxBytes
	}
	storeDir := strings.TrimSpace(s.workspaceDocumentStoreDir)
	if storeDir == "" {
		storeDir = workspaceDocumentStoreDir("")
	}
	targetDir := filepath.Join(storeDir, userID, workspaceID, sourceScope)
	if err := os.MkdirAll(targetDir, 0o700); err != nil {
		return "", 0, err
	}
	tmp, err := os.CreateTemp(targetDir, "a21-upload-*.tmp")
	if err != nil {
		return "", 0, err
	}
	tmpName := tmp.Name()
	defer func() {
		_ = os.Remove(tmpName)
	}()
	hasher := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(tmp, hasher), io.LimitReader(file, maxBytes+1))
	closeErr := tmp.Close()
	if copyErr != nil {
		return "", 0, copyErr
	}
	if closeErr != nil {
		return "", 0, closeErr
	}
	if written == 0 {
		return "", 0, fmt.Errorf("workspace document file is empty")
	}
	if written > maxBytes {
		return "", 0, errWorkspaceDocumentTooLarge
	}
	hash := fmt.Sprintf("%x", hasher.Sum(nil))
	finalPath := filepath.Join(targetDir, "sha256-"+hash+".bin")
	if err := os.Rename(tmpName, finalPath); err != nil {
		return "", 0, err
	}
	return hash, written, nil
}

func (s *Server) deleteWorkspaceDocumentForJobLocked(job *WorkspaceUploadJob, nowMS int64) {
	if job == nil || strings.TrimSpace(job.DocumentID) == "" {
		return
	}
	document, ok := s.workspaceDocuments[job.DocumentID]
	if ok {
		if path := s.workspaceDocumentPath(document); path != "" {
			_ = os.Remove(path)
		}
		document.Status = "deleted"
		document.StorageStatus = "deleted_local"
		document.IndexStatus = "deleted_no_execute"
		document.Readiness = "deleted_metadata_only"
		document.DocumentLabel = "deleted"
		document.ContentType = ""
		document.SizeBytes = 0
		document.DocumentHash = ""
		document.UpdatedAtMS = nowMS
		document.Redaction = WorkspaceDocumentRedaction{}
		document.Findings = append(document.Findings, WorkspaceUploadJobFinding{
			Code:    "workspace_document_deleted",
			Message: "A21 deleted the local intake file and retained only a redacted tombstone",
		})
		s.workspaceDocuments[document.DocumentID] = document
		for indexJobID, indexJob := range s.workspaceIndexJobs {
			if indexJob.DocumentID != document.DocumentID {
				continue
			}
			indexJob.Status = "deleted_metadata_only"
			indexJob.IndexStatus = "deleted_no_execute"
			indexJob.StorageStatus = "deleted_local"
			indexJob.DocumentHash = ""
			indexJob.DocumentLabel = "deleted"
			indexJob.ContentType = ""
			indexJob.SizeBytes = 0
			indexJob.UpdatedAtMS = nowMS
			indexJob.Redaction = workspaceUploadJobRedaction()
			indexJob.Findings = append(indexJob.Findings, WorkspaceUploadJobFinding{
				Code:    "workspace_index_job_deleted",
				Message: "A21 retained only a redacted index request tombstone after local document deletion",
			})
			s.workspaceIndexJobs[indexJobID] = indexJob
		}
	}
	job.DocumentHash = ""
	job.StorageStatus = "deleted_local"
	job.Redaction = workspaceUploadJobRedaction()
}

func (s *Server) workspaceDocumentPath(document WorkspaceDocument) string {
	hash := strings.TrimPrefix(strings.TrimSpace(document.DocumentHash), "sha256:")
	if !validWorkspaceDocumentSHA256(hash) {
		return ""
	}
	storeDir := strings.TrimSpace(s.workspaceDocumentStoreDir)
	if storeDir == "" {
		storeDir = workspaceDocumentStoreDir("")
	}
	return filepath.Join(storeDir, document.UserID, document.WorkspaceID, document.SourceScope, "sha256-"+hash+".bin")
}

func validWorkspaceDocumentSHA256(hash string) bool {
	if len(hash) != 64 {
		return false
	}
	for _, r := range hash {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') {
			continue
		}
		return false
	}
	return true
}

func decodeWorkspaceUploadJobRequest(r *http.Request) (WorkspaceUploadJobRequest, error) {
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		return WorkspaceUploadJobRequest{}, fmt.Errorf("invalid json")
	}
	for key := range raw {
		if workspaceUploadJobForbiddenKey(key) {
			return WorkspaceUploadJobRequest{}, fmt.Errorf("workspace upload job request must not include raw document, URL, path, credential, or payload fields")
		}
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return WorkspaceUploadJobRequest{}, fmt.Errorf("invalid json")
	}
	var req WorkspaceUploadJobRequest
	if err := json.Unmarshal(encoded, &req); err != nil {
		return WorkspaceUploadJobRequest{}, fmt.Errorf("invalid json")
	}
	return req, nil
}

func workspaceUploadJobForbiddenKey(key string) bool {
	key = strings.ToLower(strings.TrimSpace(key))
	for _, forbidden := range []string{
		"content",
		"document_text",
		"raw_text",
		"text_body",
		"content_base64",
		"data_base64",
		"bytes",
		"file_path",
		"local_path",
		"import_url",
		"url",
		"credential",
		"credentials",
		"api_key",
		"token",
		"provider_output",
		"retrieved_text",
		"evidence",
		"transcript",
		"audio",
		"wav_path",
	} {
		if key == forbidden {
			return true
		}
	}
	for _, forbidden := range []string{"base64", "credential", "api_key"} {
		if strings.Contains(key, forbidden) {
			return true
		}
	}
	return false
}

func decodeWorkspaceIndexJobRequest(r *http.Request) (WorkspaceIndexJobRequest, error) {
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		return WorkspaceIndexJobRequest{}, fmt.Errorf("invalid json")
	}
	for key := range raw {
		if workspaceUploadJobForbiddenKey(key) {
			return WorkspaceIndexJobRequest{}, fmt.Errorf("workspace index job request must not include raw document, URL, path, credential, provider output, or payload fields")
		}
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return WorkspaceIndexJobRequest{}, fmt.Errorf("invalid json")
	}
	var req WorkspaceIndexJobRequest
	if err := json.Unmarshal(encoded, &req); err != nil {
		return WorkspaceIndexJobRequest{}, fmt.Errorf("invalid json")
	}
	return req, nil
}
