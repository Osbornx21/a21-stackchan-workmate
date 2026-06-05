package gateway

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
)

func (s *Server) createWorkspaceIndexJob(req WorkspaceIndexJobRequest) (WorkspaceIndexJobsResponse, error) {
	nowMS := s.now().UnixMilli()
	traceID := safeOptionalWorkspaceLabel(req.TraceID)
	sessionID := safeOptionalWorkspaceLabel(req.SessionID)
	deviceID := safeOptionalWorkspaceLabel(req.DeviceID)
	s.mu.Lock()
	document, job, _, err := s.resolveWorkspaceIndexTargetLocked(req)
	if err != nil {
		s.mu.Unlock()
		return WorkspaceIndexJobsResponse{}, err
	}
	if document.Status == "deleted" || job.Status == "deleted" || document.StorageStatus != "stored_local" || job.StorageStatus != "stored_local" {
		s.mu.Unlock()
		return WorkspaceIndexJobsResponse{}, fmt.Errorf("workspace document must be stored_local before indexing can be requested")
	}
	if path := s.workspaceDocumentPath(document); path == "" {
		s.mu.Unlock()
		return WorkspaceIndexJobsResponse{}, fmt.Errorf("workspace document local store metadata unavailable")
	} else if _, err := os.Stat(path); err != nil {
		s.mu.Unlock()
		return WorkspaceIndexJobsResponse{}, fmt.Errorf("workspace document local store file unavailable")
	}
	if traceID == "" {
		traceID = document.TraceID
	}
	if sessionID == "" {
		sessionID = document.SessionID
	}
	if deviceID == "" {
		deviceID = document.DeviceID
	}
	s.workspaceIndexJobSeq++
	indexJobID := fmt.Sprintf("a21-workspace-index-job-%06d", s.workspaceIndexJobSeq)
	finding := WorkspaceUploadJobFinding{
		Code:    "workspace_index_requested_no_execute",
		Message: "A21 recorded an indexing request for a stored-local document; parsing, embedding, upload, and V21 execution remain disabled",
	}
	document.Status = "indexing_requested_no_execute"
	document.IndexStatus = "indexing_requested_no_execute"
	document.Readiness = "indexing_requested_no_execute"
	document.UpdatedAtMS = nowMS
	document.TraceID = traceID
	document.SessionID = sessionID
	document.DeviceID = deviceID
	document.Findings = append(document.Findings, finding)
	job.Status = "indexing_requested_no_execute"
	job.IndexStatus = "indexing_requested_no_execute"
	job.IndexingAPIReady = true
	job.ExecutionStarted = false
	job.UpdatedAtMS = nowMS
	job.TraceID = traceID
	job.SessionID = sessionID
	job.DeviceID = deviceID
	job.Findings = append(job.Findings, finding)
	source := workspaceSourceFromJob(job, "indexing_requested_no_execute")
	indexJob := WorkspaceIndexJob{
		IndexJobID:             indexJobID,
		DocumentID:             document.DocumentID,
		SourceID:               document.SourceID,
		JobID:                  document.JobID,
		DocumentHash:           document.DocumentHash,
		StorageStatus:          document.StorageStatus,
		UserID:                 document.UserID,
		WorkspaceID:            document.WorkspaceID,
		SourceScope:            document.SourceScope,
		SourceKind:             document.SourceKind,
		DocumentLabel:          document.DocumentLabel,
		ContentType:            document.ContentType,
		SizeBytes:              document.SizeBytes,
		Status:                 "indexing_requested_no_execute",
		IndexStatus:            "indexing_requested_no_execute",
		AdapterContractVersion: ProfessionalAdapterContractVersion,
		CreatedAtMS:            nowMS,
		UpdatedAtMS:            nowMS,
		TraceID:                traceID,
		SessionID:              sessionID,
		DeviceID:               deviceID,
		V21ExecutionAllowed:    false,
		ExecutionStarted:       false,
		Redaction:              workspaceUploadJobStoredDocumentRedaction(),
		Findings:               []WorkspaceUploadJobFinding{finding},
	}
	s.workspaceDocuments[document.DocumentID] = document
	s.workspaceUploadJobs[job.JobID] = job
	s.workspaceSources[source.SourceID] = source
	s.workspaceIndexJobs[indexJobID] = indexJob
	s.mu.Unlock()
	if traceID != "" {
		s.recordTrace(traceID, sessionID, deviceID, "workspace.index_job.requested_no_execute", nowMS)
		s.recordTrace(traceID, sessionID, deviceID, "workspace.source.indexing_requested_no_execute", nowMS)
	}
	response, err := s.workspaceIndexJobsResponse(indexJobID, "", "", "", nil)
	if err != nil {
		return WorkspaceIndexJobsResponse{}, err
	}
	response.Status = "indexing_requested_no_execute"
	return response, nil
}

func (s *Server) resolveWorkspaceIndexTargetLocked(req WorkspaceIndexJobRequest) (WorkspaceDocument, WorkspaceUploadJob, WorkspaceSource, error) {
	documentID, err := normalizeWorkspaceLedgerID("document_id", req.DocumentID)
	if err != nil {
		return WorkspaceDocument{}, WorkspaceUploadJob{}, WorkspaceSource{}, err
	}
	jobID, err := normalizeWorkspaceLedgerID("job_id", req.JobID)
	if err != nil {
		return WorkspaceDocument{}, WorkspaceUploadJob{}, WorkspaceSource{}, err
	}
	sourceID, err := normalizeWorkspaceLedgerID("source_id", req.SourceID)
	if err != nil {
		return WorkspaceDocument{}, WorkspaceUploadJob{}, WorkspaceSource{}, err
	}
	if documentID == "" && jobID != "" {
		job, ok := s.workspaceUploadJobs[jobID]
		if !ok || strings.TrimSpace(job.DocumentID) == "" {
			return WorkspaceDocument{}, WorkspaceUploadJob{}, WorkspaceSource{}, fmt.Errorf("workspace stored document not found")
		}
		documentID = job.DocumentID
	}
	if documentID == "" && sourceID != "" {
		source, ok := s.workspaceSources[sourceID]
		if !ok || strings.TrimSpace(source.DocumentID) == "" {
			return WorkspaceDocument{}, WorkspaceUploadJob{}, WorkspaceSource{}, fmt.Errorf("workspace stored document not found")
		}
		documentID = source.DocumentID
	}
	if documentID == "" {
		return WorkspaceDocument{}, WorkspaceUploadJob{}, WorkspaceSource{}, fmt.Errorf("document_id, job_id, or source_id is required")
	}
	document, ok := s.workspaceDocuments[documentID]
	if !ok {
		return WorkspaceDocument{}, WorkspaceUploadJob{}, WorkspaceSource{}, fmt.Errorf("workspace stored document not found")
	}
	if jobID != "" && document.JobID != jobID {
		return WorkspaceDocument{}, WorkspaceUploadJob{}, WorkspaceSource{}, fmt.Errorf("workspace stored document not found")
	}
	if sourceID != "" && document.SourceID != sourceID {
		return WorkspaceDocument{}, WorkspaceUploadJob{}, WorkspaceSource{}, fmt.Errorf("workspace stored document not found")
	}
	job, ok := s.workspaceUploadJobs[document.JobID]
	if !ok {
		return WorkspaceDocument{}, WorkspaceUploadJob{}, WorkspaceSource{}, fmt.Errorf("workspace upload job not found")
	}
	source, ok := s.workspaceSources[document.SourceID]
	if !ok {
		source = workspaceSourceFromJob(job, workspaceSourceReadinessForJob(job))
	}
	return document, job, source, nil
}

func (s *Server) workspaceIndexJobsResponse(indexJobID string, documentID string, jobID string, sourceID string, findings []WorkspaceUploadJobFinding) (WorkspaceIndexJobsResponse, error) {
	jobs, err := s.workspaceIndexJobsSnapshot(indexJobID, documentID, jobID, sourceID)
	if err != nil {
		return WorkspaceIndexJobsResponse{}, err
	}
	status := "ok"
	if (strings.TrimSpace(indexJobID) != "" || strings.TrimSpace(documentID) != "" || strings.TrimSpace(jobID) != "" || strings.TrimSpace(sourceID) != "") && len(jobs) == 0 {
		status = "not_found"
		findings = append(findings, WorkspaceUploadJobFinding{
			Code:    "workspace_index_job_not_found",
			Message: "A21 has no redacted workspace index job matching those filters",
		})
	} else if len(jobs) == 1 && strings.TrimSpace(indexJobID) != "" {
		status = jobs[0].Status
	}
	return WorkspaceIndexJobsResponse{
		SchemaVersion: WorkspaceIndexJobsSchemaVersion,
		Service:       DeviceRegistryServiceName,
		Status:        status,
		Jobs:          jobs,
		Redaction:     workspaceUploadJobRedactionForIndexJobs(jobs),
		Findings:      findings,
	}, nil
}

func (s *Server) workspaceIndexJobsSnapshot(indexJobID string, documentID string, jobID string, sourceID string) ([]WorkspaceIndexJob, error) {
	var err error
	if indexJobID, err = normalizeWorkspaceLedgerID("index_job_id", indexJobID); err != nil {
		return nil, err
	}
	if documentID, err = normalizeWorkspaceLedgerID("document_id", documentID); err != nil {
		return nil, err
	}
	if jobID, err = normalizeWorkspaceLedgerID("job_id", jobID); err != nil {
		return nil, err
	}
	if sourceID, err = normalizeWorkspaceLedgerID("source_id", sourceID); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	keys := make([]string, 0, len(s.workspaceIndexJobs))
	if indexJobID != "" {
		keys = append(keys, indexJobID)
	} else {
		for key := range s.workspaceIndexJobs {
			keys = append(keys, key)
		}
		sort.Strings(keys)
	}
	jobs := make([]WorkspaceIndexJob, 0, len(keys))
	for _, key := range keys {
		job, ok := s.workspaceIndexJobs[key]
		if !ok {
			continue
		}
		if documentID != "" && job.DocumentID != documentID {
			continue
		}
		if jobID != "" && job.JobID != jobID {
			continue
		}
		if sourceID != "" && job.SourceID != sourceID {
			continue
		}
		jobs = append(jobs, copyWorkspaceIndexJob(job))
	}
	return jobs, nil
}

func (s *Server) createWorkspaceUploadJob(req WorkspaceUploadJobRequest) (WorkspaceUploadJobsResponse, error) {
	userID, workspaceID, _, err := s.resolveProfessionalWorkspace(ProfessionalWorkspaceSelectionRequest{
		UserID:      req.UserID,
		WorkspaceID: req.WorkspaceID,
	})
	if err != nil {
		return WorkspaceUploadJobsResponse{}, err
	}
	sourceScope := defaultWorkspaceSourceScope(req.SourceScope)
	if !validWorkspaceSourceScope(sourceScope) {
		return WorkspaceUploadJobsResponse{}, fmt.Errorf("source_scope must be personal or public")
	}
	sourceKind := defaultWorkspaceSourceKind(req.SourceKind)
	if !validWorkspaceSourceKind(sourceKind) {
		return WorkspaceUploadJobsResponse{}, fmt.Errorf("source_kind must be upload or import")
	}
	documentLabel := safeWorkspaceDocumentLabel(req.DocumentLabel)
	if documentLabel == "" {
		return WorkspaceUploadJobsResponse{}, fmt.Errorf("valid redacted document_label is required")
	}
	contentType := safeWorkspaceContentType(req.ContentType)
	if strings.TrimSpace(req.ContentType) != "" && contentType == "" {
		return WorkspaceUploadJobsResponse{}, fmt.Errorf("valid redacted content_type is required")
	}
	if req.SizeBytes < 0 {
		return WorkspaceUploadJobsResponse{}, fmt.Errorf("size_bytes must be non-negative")
	}
	nowMS := s.now().UnixMilli()
	traceID := safeOptionalWorkspaceLabel(req.TraceID)
	sessionID := safeOptionalWorkspaceLabel(req.SessionID)
	deviceID := safeOptionalWorkspaceLabel(req.DeviceID)
	s.mu.Lock()
	s.workspaceUploadJobSeq++
	jobID := fmt.Sprintf("a21-workspace-job-%06d", s.workspaceUploadJobSeq)
	s.workspaceSourceSeq++
	sourceID := fmt.Sprintf("a21-workspace-source-%06d", s.workspaceSourceSeq)
	job := WorkspaceUploadJob{
		JobID:            jobID,
		SourceID:         sourceID,
		UserID:           userID,
		WorkspaceID:      workspaceID,
		SourceScope:      sourceScope,
		SourceKind:       sourceKind,
		DocumentLabel:    documentLabel,
		ContentType:      contentType,
		SizeBytes:        req.SizeBytes,
		Status:           "accepted_no_execute",
		IndexStatus:      "not_started_no_execute",
		Attempt:          1,
		CreatedAtMS:      nowMS,
		UpdatedAtMS:      nowMS,
		TraceID:          traceID,
		SessionID:        sessionID,
		DeviceID:         deviceID,
		UploadAPIReady:   true,
		ImportAPIReady:   true,
		IndexingAPIReady: false,
		ExecutionStarted: false,
		RetryAllowed:     true,
		DeleteAllowed:    true,
		Redaction:        workspaceUploadJobRedaction(),
		Findings: []WorkspaceUploadJobFinding{{
			Code:    "workspace_job_no_execute",
			Message: "A21 accepted only redacted workspace job metadata; upload bytes and indexing execution are not implemented in this slice",
		}},
	}
	s.workspaceUploadJobs[jobID] = job
	s.workspaceSources[sourceID] = workspaceSourceFromJob(job, "metadata_only")
	s.mu.Unlock()
	if traceID != "" {
		s.recordTrace(traceID, sessionID, deviceID, "workspace.upload_job.accepted_no_execute", nowMS)
		s.recordTrace(traceID, sessionID, deviceID, "workspace.index_job.not_started_no_execute", nowMS)
		s.recordTrace(traceID, sessionID, deviceID, "workspace.source.created_metadata_only", nowMS)
	}
	return s.workspaceUploadJobsResponse(jobID, nil), nil
}

func (s *Server) applyWorkspaceUploadJobAction(req WorkspaceUploadJobRequest) (WorkspaceUploadJobsResponse, error) {
	jobID := strings.TrimSpace(req.JobID)
	if jobID == "" {
		return WorkspaceUploadJobsResponse{}, fmt.Errorf("job_id is required")
	}
	action := strings.ToLower(strings.TrimSpace(req.Action))
	if action == "" {
		action = "retry"
	}
	s.mu.Lock()
	job, ok := s.workspaceUploadJobs[jobID]
	if !ok {
		s.mu.Unlock()
		return WorkspaceUploadJobsResponse{}, fmt.Errorf("workspace upload job not found")
	}
	nowMS := s.now().UnixMilli()
	switch action {
	case "mark_failed", "fail":
		job.Status = "failed"
		job.IndexStatus = "failed_no_execute"
		job.UpdatedAtMS = nowMS
		job.RetryAllowed = true
		job.DeleteAllowed = true
		job.Findings = append(job.Findings, WorkspaceUploadJobFinding{
			Code:    "workspace_job_marked_failed",
			Message: "A21 marked the redacted workspace job failed without storing document text or bytes",
		})
	case "mark_searchable", "mark_indexed_metadata_only":
		job.Status = "accepted_no_execute"
		job.IndexStatus = "searchable_metadata_only"
		job.UpdatedAtMS = nowMS
		job.RetryAllowed = true
		job.DeleteAllowed = true
		job.Findings = append(job.Findings, WorkspaceUploadJobFinding{
			Code:    "workspace_source_searchable_metadata_only",
			Message: "A21 marked this redacted source searchable as metadata-only readiness; no indexing execution occurred",
		})
	case "retry":
		if job.DocumentID != "" && job.StorageStatus == "stored_local" {
			job.Status = "stored_local_pending_index"
		} else {
			job.Status = "accepted_no_execute"
		}
		job.IndexStatus = "not_started_no_execute"
		job.Attempt++
		job.UpdatedAtMS = nowMS
		job.RetryAllowed = true
		job.DeleteAllowed = true
		job.Findings = append(job.Findings, WorkspaceUploadJobFinding{
			Code:    "workspace_job_retry_no_execute",
			Message: "A21 accepted a retry request but did not execute upload or indexing",
		})
	case "delete":
		job.Status = "deleted"
		job.IndexStatus = "deleted_no_execute"
		job.DocumentLabel = "deleted"
		job.ContentType = ""
		job.SizeBytes = 0
		job.UpdatedAtMS = nowMS
		job.RetryAllowed = false
		job.DeleteAllowed = false
		if job.DocumentID != "" {
			s.deleteWorkspaceDocumentForJobLocked(&job, nowMS)
		}
		job.Findings = append(job.Findings, WorkspaceUploadJobFinding{
			Code:    "workspace_job_deleted",
			Message: "A21 retained only a redacted deletion tombstone",
		})
	default:
		s.mu.Unlock()
		return WorkspaceUploadJobsResponse{}, fmt.Errorf("action must be retry, mark_failed, mark_searchable, or delete")
	}
	s.workspaceUploadJobs[jobID] = job
	if job.SourceID != "" {
		s.workspaceSources[job.SourceID] = workspaceSourceFromJob(job, workspaceSourceReadinessForJob(job))
	}
	s.mu.Unlock()
	if job.TraceID != "" {
		s.recordTrace(job.TraceID, job.SessionID, job.DeviceID, "workspace.upload_job."+job.Status, nowMS)
		if job.IndexStatus == "searchable_metadata_only" {
			s.recordTrace(job.TraceID, job.SessionID, job.DeviceID, "workspace.index_job.searchable_metadata_only", nowMS)
		}
	}
	return s.workspaceUploadJobsResponse(jobID, nil), nil
}

func (s *Server) workspaceUploadJobsResponse(jobID string, findings []WorkspaceUploadJobFinding) WorkspaceUploadJobsResponse {
	jobs := s.workspaceUploadJobsSnapshot(jobID)
	status := "ok"
	if strings.TrimSpace(jobID) != "" && len(jobs) == 0 {
		status = "not_found"
		findings = append(findings, WorkspaceUploadJobFinding{
			Code:    "workspace_job_not_found",
			Message: "A21 has no redacted workspace job with that id",
		})
	}
	return WorkspaceUploadJobsResponse{
		SchemaVersion: WorkspaceUploadJobsSchemaVersion,
		Service:       DeviceRegistryServiceName,
		Status:        status,
		Jobs:          jobs,
		Redaction:     workspaceUploadJobRedactionForJobs(jobs),
		Findings:      findings,
	}
}

func (s *Server) workspaceUploadJobsSnapshot(jobID string) []WorkspaceUploadJob {
	jobID = strings.TrimSpace(jobID)
	s.mu.Lock()
	defer s.mu.Unlock()
	jobs := make([]WorkspaceUploadJob, 0, len(s.workspaceUploadJobs))
	if jobID != "" {
		if job, ok := s.workspaceUploadJobs[jobID]; ok {
			return []WorkspaceUploadJob{job}
		}
		return nil
	}
	keys := make([]string, 0, len(s.workspaceUploadJobs))
	for key := range s.workspaceUploadJobs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		jobs = append(jobs, s.workspaceUploadJobs[key])
	}
	return jobs
}

func (s *Server) workspaceSourcesResponse(sourceID string, userID string, workspaceID string, sourceScope string) (WorkspaceSourcesResponse, error) {
	sources, err := s.workspaceSourcesSnapshot(sourceID, userID, workspaceID, sourceScope)
	if err != nil {
		return WorkspaceSourcesResponse{}, err
	}
	status := "ok"
	findings := []WorkspaceUploadJobFinding(nil)
	if (strings.TrimSpace(sourceID) != "" || strings.TrimSpace(userID) != "" || strings.TrimSpace(workspaceID) != "" || strings.TrimSpace(sourceScope) != "") && len(sources) == 0 {
		status = "not_found"
		findings = append(findings, WorkspaceUploadJobFinding{
			Code:    "workspace_source_not_found",
			Message: "A21 has no redacted workspace source matching those filters",
		})
	}
	return WorkspaceSourcesResponse{
		SchemaVersion: WorkspaceSourcesSchemaVersion,
		Service:       DeviceRegistryServiceName,
		Status:        status,
		Sources:       sources,
		Summary:       workspaceSourceSummary(sources),
		Redaction:     workspaceUploadJobRedactionForSources(sources),
		Findings:      findings,
	}, nil
}

func (s *Server) workspaceSourcesSnapshot(sourceID string, userID string, workspaceID string, sourceScope string) ([]WorkspaceSource, error) {
	sourceID = strings.TrimSpace(sourceID)
	userID = strings.ToLower(strings.TrimSpace(userID))
	workspaceID = strings.ToLower(strings.TrimSpace(workspaceID))
	sourceScope = strings.ToLower(strings.TrimSpace(sourceScope))
	if sourceID != "" && safeOptionalWorkspaceLabel(sourceID) == "" {
		return nil, fmt.Errorf("valid redacted source_id is required")
	}
	if userID != "" && !validProfessionalLabel(userID) {
		return nil, fmt.Errorf("valid redacted user_id is required")
	}
	if workspaceID != "" && !validProfessionalLabel(workspaceID) {
		return nil, fmt.Errorf("valid redacted workspace_id is required")
	}
	if sourceScope != "" && !validWorkspaceSourceScope(sourceScope) {
		return nil, fmt.Errorf("source_scope must be personal or public")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	sources := make([]WorkspaceSource, 0, len(s.workspaceSources))
	keys := make([]string, 0, len(s.workspaceSources))
	if sourceID != "" {
		keys = append(keys, sourceID)
	} else {
		for key := range s.workspaceSources {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	for _, key := range keys {
		source, ok := s.workspaceSources[key]
		if !ok {
			continue
		}
		if userID != "" && source.UserID != userID {
			continue
		}
		if workspaceID != "" && source.WorkspaceID != workspaceID {
			continue
		}
		if sourceScope != "" && source.SourceScope != sourceScope {
			continue
		}
		sources = append(sources, copyWorkspaceSource(source))
	}
	return sources, nil
}

func (s *Server) workspaceSourceSummaryForWorkspace(userID string, workspaceID string) WorkspaceSourceSummary {
	sources, err := s.workspaceSourcesSnapshot("", userID, workspaceID, "")
	if err != nil {
		return emptyWorkspaceSourceSummary()
	}
	return workspaceSourceSummary(sources)
}

func decodeWorkspaceDeviceBindingRequest(r *http.Request) (WorkspaceDeviceBindingRequest, error) {
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		return WorkspaceDeviceBindingRequest{}, fmt.Errorf("invalid json")
	}
	for key := range raw {
		if workspaceDeviceBindingForbiddenKey(key) {
			return WorkspaceDeviceBindingRequest{}, fmt.Errorf("workspace device binding request must not include pairing secrets, URLs, paths, credentials, document text, provider output, or audio")
		}
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return WorkspaceDeviceBindingRequest{}, fmt.Errorf("invalid json")
	}
	var req WorkspaceDeviceBindingRequest
	if err := json.Unmarshal(encoded, &req); err != nil {
		return WorkspaceDeviceBindingRequest{}, fmt.Errorf("invalid json")
	}
	return req, nil
}

func workspaceDeviceBindingForbiddenKey(key string) bool {
	key = strings.ToLower(strings.TrimSpace(key))
	if workspaceUploadJobForbiddenKey(key) {
		return true
	}
	for _, forbidden := range []string{
		"pairing_secret",
		"pairing_code",
		"device_secret",
		"device_credential",
		"wifi_password",
		"authorization",
		"cookie",
	} {
		if key == forbidden || strings.Contains(key, forbidden) {
			return true
		}
	}
	return false
}

func (s *Server) createWorkspaceDeviceBinding(req WorkspaceDeviceBindingRequest) (WorkspaceDeviceBindingsResponse, error) {
	userID, workspaceID, _, err := s.resolveProfessionalWorkspace(ProfessionalWorkspaceSelectionRequest{
		UserID:      req.UserID,
		WorkspaceID: req.WorkspaceID,
	})
	if err != nil {
		return WorkspaceDeviceBindingsResponse{}, err
	}
	deviceID := safeWorkspaceDeviceID(req.DeviceID)
	if deviceID == "" {
		return WorkspaceDeviceBindingsResponse{}, fmt.Errorf("valid A21 device_id is required")
	}
	deviceLabel := safeWorkspaceDeviceLabel(req.DeviceLabel, deviceID)
	if strings.TrimSpace(req.DeviceLabel) != "" && deviceLabel == "" {
		return WorkspaceDeviceBindingsResponse{}, fmt.Errorf("valid redacted device_label is required")
	}
	allowedScopes, err := normalizeWorkspaceBindingQueryScopes(req.AllowedQueryScopes)
	if err != nil {
		return WorkspaceDeviceBindingsResponse{}, err
	}
	traceID := safeOptionalWorkspaceLabel(req.TraceID)
	sessionID := safeOptionalWorkspaceLabel(req.SessionID)
	nowMS := s.now().UnixMilli()
	s.mu.Lock()
	for existingID, existing := range s.workspaceDeviceBindings {
		if existing.DeviceID == deviceID && existing.UserID == userID && existing.WorkspaceID == workspaceID && existing.Status != "deleted_metadata_only" {
			existing.DeviceLabel = deviceLabel
			existing.Status = "bound"
			existing.AccessScope = "professional_workspace"
			existing.AllowedQueryScopes = allowedScopes
			existing.ProfessionalAllowed = true
			existing.RevokedAtMS = 0
			existing.UpdatedAtMS = nowMS
			if traceID != "" {
				existing.TraceID = traceID
			}
			if sessionID != "" {
				existing.SessionID = sessionID
			}
			existing.Findings = append(existing.Findings, WorkspaceUploadJobFinding{
				Code:    "workspace_device_binding_updated",
				Message: "A21 refreshed an existing metadata-only device binding for this professional workspace",
			})
			s.workspaceDeviceBindings[existingID] = existing
			s.mu.Unlock()
			if traceID != "" {
				s.recordTrace(traceID, sessionID, deviceID, "workspace.device.bound", nowMS)
			}
			return s.workspaceDeviceBindingsResponse(existingID, "", "", "", "", nil)
		}
	}
	s.workspaceDeviceBindingSeq++
	bindingID := fmt.Sprintf("a21-workspace-device-binding-%06d", s.workspaceDeviceBindingSeq)
	binding := WorkspaceDeviceBinding{
		BindingID:           bindingID,
		DeviceID:            deviceID,
		DeviceLabel:         deviceLabel,
		UserID:              userID,
		WorkspaceID:         workspaceID,
		Status:              "bound",
		AccessScope:         "professional_workspace",
		AllowedQueryScopes:  allowedScopes,
		ProfessionalAllowed: true,
		PhysicalAccepted:    false,
		CreatedAtMS:         nowMS,
		UpdatedAtMS:         nowMS,
		TraceID:             traceID,
		SessionID:           sessionID,
		Redaction:           workspaceDeviceBindingRedaction(),
		Findings: []WorkspaceUploadJobFinding{{
			Code:    "workspace_device_bound",
			Message: "A21 bound this device to the selected professional workspace using metadata only; no pairing secret or provider credential is stored",
		}},
	}
	s.workspaceDeviceBindings[bindingID] = binding
	s.mu.Unlock()
	if traceID != "" {
		s.recordTrace(traceID, sessionID, deviceID, "workspace.device.bound", nowMS)
	}
	return s.workspaceDeviceBindingsResponse(bindingID, "", "", "", "", nil)
}
