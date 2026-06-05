package gateway

import (
	"fmt"
	"sort"
	"strings"

	"a21.local/a21/internal/v21adapter"
)

func (s *Server) applyWorkspaceDeviceBindingAction(req WorkspaceDeviceBindingRequest) (WorkspaceDeviceBindingsResponse, error) {
	action := strings.ToLower(strings.TrimSpace(req.Action))
	if action == "" {
		action = "revoke"
	}
	nowMS := s.now().UnixMilli()
	s.mu.Lock()
	bindingID, err := s.resolveWorkspaceDeviceBindingIDLocked(req)
	if err != nil {
		s.mu.Unlock()
		return WorkspaceDeviceBindingsResponse{}, err
	}
	binding, ok := s.workspaceDeviceBindings[bindingID]
	if !ok {
		s.mu.Unlock()
		return WorkspaceDeviceBindingsResponse{}, fmt.Errorf("workspace device binding not found")
	}
	switch action {
	case "revoke":
		binding.Status = "revoked"
		binding.ProfessionalAllowed = false
		binding.RevokedAtMS = nowMS
		binding.UpdatedAtMS = nowMS
		binding.Findings = append(binding.Findings, WorkspaceUploadJobFinding{
			Code:    "workspace_device_binding_revoked",
			Message: "A21 revoked professional workspace access for this device without requiring firmware or NVS changes",
		})
	case "restore":
		binding.Status = "bound"
		binding.ProfessionalAllowed = true
		binding.RevokedAtMS = 0
		binding.UpdatedAtMS = nowMS
		binding.Findings = append(binding.Findings, WorkspaceUploadJobFinding{
			Code:    "workspace_device_binding_restored",
			Message: "A21 restored metadata-only professional workspace access for this device",
		})
	case "delete":
		binding.Status = "deleted_metadata_only"
		binding.ProfessionalAllowed = false
		binding.RevokedAtMS = nowMS
		binding.UpdatedAtMS = nowMS
		binding.DeviceLabel = "deleted"
		binding.Findings = append(binding.Findings, WorkspaceUploadJobFinding{
			Code:    "workspace_device_binding_deleted",
			Message: "A21 retained only a redacted binding tombstone",
		})
	default:
		s.mu.Unlock()
		return WorkspaceDeviceBindingsResponse{}, fmt.Errorf("action must be revoke, restore, or delete")
	}
	s.workspaceDeviceBindings[bindingID] = binding
	s.mu.Unlock()
	if binding.TraceID != "" {
		s.recordTrace(binding.TraceID, binding.SessionID, binding.DeviceID, "workspace.device."+binding.Status, nowMS)
	}
	return s.workspaceDeviceBindingsResponse(bindingID, "", "", "", "", nil)
}

func (s *Server) resolveWorkspaceDeviceBindingIDLocked(req WorkspaceDeviceBindingRequest) (string, error) {
	if id := strings.ToLower(strings.TrimSpace(req.BindingID)); id != "" {
		if safeOptionalWorkspaceLabel(id) == "" {
			return "", fmt.Errorf("valid redacted binding_id is required")
		}
		return id, nil
	}
	deviceID := safeWorkspaceDeviceID(req.DeviceID)
	if deviceID == "" {
		return "", fmt.Errorf("binding_id or valid A21 device_id is required")
	}
	userID, workspaceID, _, err := s.resolveProfessionalWorkspaceNoLock(req.UserID, req.WorkspaceID, "")
	if err != nil {
		return "", err
	}
	for bindingID, binding := range s.workspaceDeviceBindings {
		if binding.DeviceID == deviceID && binding.UserID == userID && binding.WorkspaceID == workspaceID && binding.Status != "deleted_metadata_only" {
			return bindingID, nil
		}
	}
	return "", fmt.Errorf("workspace device binding not found")
}

func (s *Server) workspaceDeviceBindingsResponse(bindingID string, deviceID string, userID string, workspaceID string, status string, findings []WorkspaceUploadJobFinding) (WorkspaceDeviceBindingsResponse, error) {
	bindings, err := s.workspaceDeviceBindingsSnapshot(bindingID, deviceID, userID, workspaceID, status)
	if err != nil {
		return WorkspaceDeviceBindingsResponse{}, err
	}
	responseStatus := "ok"
	if (strings.TrimSpace(bindingID) != "" || strings.TrimSpace(deviceID) != "" || strings.TrimSpace(userID) != "" || strings.TrimSpace(workspaceID) != "" || strings.TrimSpace(status) != "") && len(bindings) == 0 {
		responseStatus = "not_found"
		findings = append(findings, WorkspaceUploadJobFinding{
			Code:    "workspace_device_binding_not_found",
			Message: "A21 has no redacted workspace device binding matching those filters",
		})
	} else if len(bindings) == 1 && strings.TrimSpace(bindingID) != "" {
		responseStatus = bindings[0].Status
	}
	return WorkspaceDeviceBindingsResponse{
		SchemaVersion: WorkspaceDeviceBindingsSchemaVersion,
		Service:       DeviceRegistryServiceName,
		Status:        responseStatus,
		Bindings:      bindings,
		Summary:       workspaceDeviceBindingSummary(bindings),
		Redaction:     workspaceDeviceBindingRedaction(),
		Findings:      findings,
	}, nil
}

func (s *Server) workspaceDeviceBindingsSnapshot(bindingID string, deviceID string, userID string, workspaceID string, status string) ([]WorkspaceDeviceBinding, error) {
	bindingID = strings.ToLower(strings.TrimSpace(bindingID))
	rawDeviceID := strings.TrimSpace(deviceID)
	deviceID = safeWorkspaceDeviceID(deviceID)
	userID = strings.ToLower(strings.TrimSpace(userID))
	workspaceID = strings.ToLower(strings.TrimSpace(workspaceID))
	status = strings.ToLower(strings.TrimSpace(status))
	if bindingID != "" && safeOptionalWorkspaceLabel(bindingID) == "" {
		return nil, fmt.Errorf("valid redacted binding_id is required")
	}
	if rawDeviceID != "" && deviceID == "" {
		return nil, fmt.Errorf("valid A21 device_id is required")
	}
	if userID != "" && !validProfessionalLabel(userID) {
		return nil, fmt.Errorf("valid redacted user_id is required")
	}
	if workspaceID != "" && !validProfessionalLabel(workspaceID) {
		return nil, fmt.Errorf("valid redacted workspace_id is required")
	}
	if status != "" && !validWorkspaceDeviceBindingStatus(status) {
		return nil, fmt.Errorf("status must be bound, revoked, or deleted_metadata_only")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	keys := make([]string, 0, len(s.workspaceDeviceBindings))
	if bindingID != "" {
		keys = append(keys, bindingID)
	} else {
		for key := range s.workspaceDeviceBindings {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	bindings := make([]WorkspaceDeviceBinding, 0, len(keys))
	for _, key := range keys {
		binding, ok := s.workspaceDeviceBindings[key]
		if !ok {
			continue
		}
		if deviceID != "" && binding.DeviceID != deviceID {
			continue
		}
		if userID != "" && binding.UserID != userID {
			continue
		}
		if workspaceID != "" && binding.WorkspaceID != workspaceID {
			continue
		}
		if status != "" && binding.Status != status {
			continue
		}
		bindings = append(bindings, copyWorkspaceDeviceBinding(binding))
	}
	return bindings, nil
}

func (s *Server) workspaceDeviceBindingSummaryForWorkspace(userID string, workspaceID string) WorkspaceDeviceBindingSummary {
	bindings, err := s.workspaceDeviceBindingsSnapshot("", "", userID, workspaceID, "")
	if err != nil {
		return emptyWorkspaceDeviceBindingSummary()
	}
	return workspaceDeviceBindingSummary(bindings)
}

func workspaceDeviceBindingSummary(bindings []WorkspaceDeviceBinding) WorkspaceDeviceBindingSummary {
	summary := emptyWorkspaceDeviceBindingSummary()
	summary.TotalBindings = 0
	for _, binding := range bindings {
		summary.TotalBindings++
		switch binding.Status {
		case "bound":
			summary.ActiveBindings++
			if binding.ProfessionalAllowed {
				summary.ProfessionalAllowedDevices++
			}
		case "revoked":
			summary.RevokedBindings++
		case "deleted_metadata_only":
			summary.DeletedBindings++
		}
	}
	switch {
	case summary.ActiveBindings > 0:
		summary.DeviceBindingPolicy = "bound_devices_only"
		summary.WorkspaceAccessStatus = "bound_device_ready"
	case summary.TotalBindings > 0:
		summary.DeviceBindingPolicy = "bound_devices_only"
		summary.WorkspaceAccessStatus = "no_active_device_binding"
	default:
		summary.DeviceBindingPolicy = "open_until_binding_configured"
		summary.WorkspaceAccessStatus = "binding_not_configured"
	}
	return summary
}

func emptyWorkspaceDeviceBindingSummary() WorkspaceDeviceBindingSummary {
	return WorkspaceDeviceBindingSummary{
		DeviceBindingPolicy:   "open_until_binding_configured",
		WorkspaceAccessStatus: "binding_not_configured",
	}
}

func copyWorkspaceDeviceBinding(binding WorkspaceDeviceBinding) WorkspaceDeviceBinding {
	binding.AllowedQueryScopes = append([]string(nil), binding.AllowedQueryScopes...)
	binding.Findings = append([]WorkspaceUploadJobFinding(nil), binding.Findings...)
	return binding
}

func workspaceDeviceBindingRedaction() WorkspaceDeviceBindingRedaction {
	return WorkspaceDeviceBindingRedaction{
		PairingSecretStored:    false,
		DeviceCredentialStored: false,
		DocumentTextStored:     false,
		QueryTextStored:        false,
		RetrievedTextStored:    false,
		FullURLStored:          false,
		LocalPathStored:        false,
		CredentialValueStored:  false,
		ProviderOutputStored:   false,
		VoiceTranscriptStored:  false,
	}
}

func (s *Server) professionalDeviceBindingDecision(deviceID string, userID string, workspaceID string, queryScope string) professionalDeviceBindingDecision {
	deviceID = safeWorkspaceDeviceID(deviceID)
	userID = defaultProfessionalLabel(userID, v21adapter.DefaultUserID)
	workspaceID = defaultProfessionalLabel(workspaceID, v21adapter.DefaultWorkspaceID)
	queryScope = defaultProfessionalQueryScope(queryScope)
	s.mu.Lock()
	defer s.mu.Unlock()
	candidates := make([]WorkspaceDeviceBinding, 0)
	for _, binding := range s.workspaceDeviceBindings {
		if binding.UserID == userID && binding.WorkspaceID == workspaceID {
			candidates = append(candidates, binding)
		}
	}
	if len(candidates) == 0 {
		return professionalDeviceBindingDecision{
			Allowed:     true,
			Policy:      "open_until_binding_configured",
			Status:      "not_configured",
			TraceMarker: "professional.device_binding.open_until_configured",
		}
	}
	if deviceID == "" {
		return professionalDeviceBindingDecision{
			Allowed:     false,
			Policy:      "bound_devices_only",
			Status:      "device_id_invalid",
			FailureCode: "device_unbound",
			TraceMarker: "professional.device_binding.blocked.device_unbound",
		}
	}
	for _, binding := range candidates {
		if binding.DeviceID != deviceID {
			continue
		}
		if binding.Status == "deleted_metadata_only" {
			return professionalDeviceBindingDecision{
				Allowed:     false,
				Policy:      "bound_devices_only",
				BindingID:   binding.BindingID,
				Status:      "deleted_metadata_only",
				FailureCode: "device_unbound",
				TraceMarker: "professional.device_binding.blocked.device_unbound",
			}
		}
		if binding.Status == "revoked" || !binding.ProfessionalAllowed {
			return professionalDeviceBindingDecision{
				Allowed:     false,
				Policy:      "bound_devices_only",
				BindingID:   binding.BindingID,
				Status:      "revoked",
				FailureCode: "device_binding_revoked",
				TraceMarker: "professional.device_binding.blocked.revoked",
			}
		}
		if !workspaceBindingAllowsQueryScope(binding, queryScope) {
			return professionalDeviceBindingDecision{
				Allowed:     false,
				Policy:      "bound_devices_only",
				BindingID:   binding.BindingID,
				Status:      "scope_denied",
				FailureCode: "device_scope_denied",
				TraceMarker: "professional.device_binding.blocked.scope_denied",
			}
		}
		return professionalDeviceBindingDecision{
			Allowed:     true,
			Policy:      "bound_devices_only",
			BindingID:   binding.BindingID,
			Status:      "bound",
			TraceMarker: "professional.device_binding.bound",
		}
	}
	return professionalDeviceBindingDecision{
		Allowed:     false,
		Policy:      "bound_devices_only",
		Status:      "unbound",
		FailureCode: "device_unbound",
		TraceMarker: "professional.device_binding.blocked.device_unbound",
	}
}

func workspaceSourceFromJob(job WorkspaceUploadJob, readiness string) WorkspaceSource {
	readiness = defaultWorkspaceSourceReadiness(readiness)
	storedLocal := job.StorageStatus == "stored_local" && readiness != "deleted_metadata_only"
	redaction := job.Redaction
	return WorkspaceSource{
		SourceID:          job.SourceID,
		JobID:             job.JobID,
		DocumentID:        job.DocumentID,
		DocumentHash:      job.DocumentHash,
		StorageStatus:     job.StorageStatus,
		UserID:            job.UserID,
		WorkspaceID:       job.WorkspaceID,
		SourceScope:       job.SourceScope,
		SourceKind:        job.SourceKind,
		DocumentLabel:     job.DocumentLabel,
		ContentType:       job.ContentType,
		SizeBytes:         job.SizeBytes,
		Readiness:         readiness,
		IndexStatus:       job.IndexStatus,
		CreatedAtMS:       job.CreatedAtMS,
		UpdatedAtMS:       job.UpdatedAtMS,
		TraceID:           job.TraceID,
		SessionID:         job.SessionID,
		DeviceID:          job.DeviceID,
		MetadataOnly:      !storedLocal,
		StoredLocal:       storedLocal,
		IndexingRequested: readiness == "indexing_requested_no_execute",
		Searchable:        readiness == "searchable_metadata_only",
		Deleted:           readiness == "deleted_metadata_only",
		Redaction:         redaction,
		Findings:          append([]WorkspaceUploadJobFinding(nil), job.Findings...),
	}
}

func workspaceSourceReadinessForJob(job WorkspaceUploadJob) string {
	switch {
	case job.Status == "deleted":
		return "deleted_metadata_only"
	case job.Status == "failed":
		return "failed_metadata_only"
	case job.Status == "indexing_requested_no_execute" || job.IndexStatus == "indexing_requested_no_execute":
		return "indexing_requested_no_execute"
	case job.Status == "stored_local_pending_index":
		return "stored_local_pending_index"
	case job.IndexStatus == "searchable_metadata_only":
		return "searchable_metadata_only"
	default:
		return "metadata_only"
	}
}

func defaultWorkspaceSourceReadiness(readiness string) string {
	switch strings.TrimSpace(readiness) {
	case "metadata_only", "stored_local_pending_index", "indexing_requested_no_execute", "searchable_metadata_only", "failed_metadata_only", "deleted_metadata_only":
		return strings.TrimSpace(readiness)
	default:
		return "metadata_only"
	}
}

func copyWorkspaceSource(source WorkspaceSource) WorkspaceSource {
	source.Findings = append([]WorkspaceUploadJobFinding(nil), source.Findings...)
	return source
}

func workspaceSourceSummary(sources []WorkspaceSource) WorkspaceSourceSummary {
	summary := emptyWorkspaceSourceSummary()
	for _, source := range sources {
		summary.TotalSources++
		if source.MetadataOnly {
			summary.MetadataOnlySources++
		}
		if source.StoredLocal {
			summary.StoredLocalSources++
		}
		if source.Deleted {
			summary.DeletedSources++
			continue
		}
		if validWorkspaceSourceScope(source.SourceScope) {
			summary.SourceScopeCounts[source.SourceScope]++
		}
		if source.Searchable {
			summary.SearchableSources++
			if validWorkspaceSourceScope(source.SourceScope) {
				summary.SearchableSourceScopeCounts[source.SourceScope]++
			}
		}
		if source.StoredLocal && validWorkspaceSourceScope(source.SourceScope) {
			summary.StoredLocalSourceScopeCounts[source.SourceScope]++
		}
		if source.IndexingRequested {
			summary.IndexingRequestedSources++
			if validWorkspaceSourceScope(source.SourceScope) {
				summary.IndexingRequestedSourceScopeCounts[source.SourceScope]++
			}
		}
	}
	switch {
	case summary.SearchableSources > 0:
		summary.WorkspaceStatus = "searchable_metadata_only"
	case summary.IndexingRequestedSources > 0:
		summary.WorkspaceStatus = "indexing_requested_no_execute"
	case summary.TotalSources == summary.DeletedSources && summary.TotalSources > 0:
		summary.WorkspaceStatus = "deleted_metadata_only"
	case summary.StoredLocalSources > 0:
		summary.WorkspaceStatus = "stored_local_pending_index"
	case summary.TotalSources > 0:
		summary.WorkspaceStatus = "metadata_only"
	default:
		summary.WorkspaceStatus = "no_sources_metadata_only"
	}
	return summary
}

func emptyWorkspaceSourceSummary() WorkspaceSourceSummary {
	return WorkspaceSourceSummary{
		SourceScopeCounts:                  map[string]int{"public": 0, "personal": 0},
		StoredLocalSourceScopeCounts:       map[string]int{"public": 0, "personal": 0},
		IndexingRequestedSourceScopeCounts: map[string]int{"public": 0, "personal": 0},
		SearchableSourceScopeCounts:        map[string]int{"public": 0, "personal": 0},
		WorkspaceStatus:                    "no_sources_metadata_only",
	}
}

func workspaceQueryScopeReadiness(queryScope string, summary WorkspaceSourceSummary) string {
	switch defaultProfessionalQueryScope(queryScope) {
	case v21adapter.QueryScopePublic:
		return workspaceSingleScopeReadiness("public", summary)
	case v21adapter.QueryScopePersonal:
		return workspaceSingleScopeReadiness("personal", summary)
	case v21adapter.QueryScopeCombined:
		publicReady := summary.SearchableSourceScopeCounts["public"] > 0
		personalReady := summary.SearchableSourceScopeCounts["personal"] > 0
		publicIndexing := summary.IndexingRequestedSourceScopeCounts["public"] > 0
		personalIndexing := summary.IndexingRequestedSourceScopeCounts["personal"] > 0
		publicStored := summary.StoredLocalSourceScopeCounts["public"] > 0
		personalStored := summary.StoredLocalSourceScopeCounts["personal"] > 0
		switch {
		case publicReady && personalReady:
			return "combined_searchable_metadata_only"
		case publicReady || personalReady:
			return "partial_searchable_metadata_only"
		case publicIndexing || personalIndexing:
			return "indexing_requested_no_execute"
		case publicStored || personalStored:
			return "stored_local_pending_index"
		case summary.SourceScopeCounts["public"] > 0 || summary.SourceScopeCounts["personal"] > 0:
			return "metadata_only"
		default:
			return "no_sources_metadata_only"
		}
	default:
		return "no_sources_metadata_only"
	}
}

func workspaceSingleScopeReadiness(scope string, summary WorkspaceSourceSummary) string {
	switch {
	case summary.SearchableSourceScopeCounts[scope] > 0:
		return "searchable_metadata_only"
	case summary.IndexingRequestedSourceScopeCounts[scope] > 0:
		return "indexing_requested_no_execute"
	case summary.StoredLocalSourceScopeCounts[scope] > 0:
		return "stored_local_pending_index"
	case summary.SourceScopeCounts[scope] > 0:
		return "metadata_only"
	default:
		return "no_sources_metadata_only"
	}
}

func copyWorkspaceSourceCounts(counts map[string]int) map[string]int {
	if len(counts) == 0 {
		return nil
	}
	out := make(map[string]int, len(counts))
	for scope, count := range counts {
		out[scope] = count
	}
	return out
}

func copyWorkspaceIndexJob(job WorkspaceIndexJob) WorkspaceIndexJob {
	job.Findings = append([]WorkspaceUploadJobFinding(nil), job.Findings...)
	return job
}

func normalizeWorkspaceLedgerID(field string, value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "", nil
	}
	if len(value) > 96 {
		return "", fmt.Errorf("valid redacted %s is required", field)
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			continue
		}
		return "", fmt.Errorf("valid redacted %s is required", field)
	}
	return value, nil
}

func defaultWorkspaceSourceScope(scope string) string {
	scope = strings.ToLower(strings.TrimSpace(scope))
	if scope == "" {
		return "personal"
	}
	return scope
}

func validWorkspaceSourceScope(scope string) bool {
	switch strings.ToLower(strings.TrimSpace(scope)) {
	case "personal", "public":
		return true
	default:
		return false
	}
}

func defaultWorkspaceSourceKind(kind string) string {
	kind = strings.ToLower(strings.TrimSpace(kind))
	if kind == "" {
		return "upload"
	}
	return kind
}

func validWorkspaceSourceKind(kind string) bool {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "upload", "import":
		return true
	default:
		return false
	}
}

func safeWorkspaceDocumentLabel(label string) string {
	label = strings.TrimSpace(label)
	if label == "" || len([]rune(label)) > 96 {
		return ""
	}
	lower := strings.ToLower(label)
	for _, forbidden := range []string{"http://", "https://", "/", "\\", "api_key", "secret", "token", "bearer "} {
		if strings.Contains(lower, forbidden) {
			return ""
		}
	}
	for _, r := range label {
		if r < 32 || r == 127 {
			return ""
		}
	}
	return label
}

func safeWorkspaceDeviceID(deviceID string) string {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" || len(deviceID) > 96 || !validA21DeviceID(deviceID) {
		return ""
	}
	lower := strings.ToLower(deviceID)
	for _, forbidden := range []string{"http://", "https://", "/", "\\", "secret", "token", "credential", "api_key", "bearer ", "sk-"} {
		if strings.Contains(lower, forbidden) {
			return ""
		}
	}
	for _, r := range deviceID {
		if r < 33 || r == 127 {
			return ""
		}
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == ':' || r == '.' {
			continue
		}
		return ""
	}
	return deviceID
}

func safeWorkspaceDeviceLabel(label string, fallback string) string {
	if strings.TrimSpace(label) == "" {
		return safeWorkspaceDeviceID(fallback)
	}
	return safeWorkspaceDocumentLabel(label)
}

func normalizeWorkspaceBindingQueryScopes(scopes []string) ([]string, error) {
	if len(scopes) == 0 {
		return []string{v21adapter.QueryScopePublic, v21adapter.QueryScopePersonal, v21adapter.QueryScopeCombined}, nil
	}
	seen := make(map[string]bool, len(scopes))
	out := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		scope = strings.ToLower(strings.TrimSpace(scope))
		if !v21adapter.ValidQueryScope(scope) {
			return nil, fmt.Errorf("allowed_query_scopes must contain only public_only, personal_only, or personal_plus_public")
		}
		if seen[scope] {
			continue
		}
		seen[scope] = true
		out = append(out, scope)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("allowed_query_scopes must contain at least one query scope")
	}
	return out, nil
}

func workspaceBindingAllowsQueryScope(binding WorkspaceDeviceBinding, queryScope string) bool {
	queryScope = defaultProfessionalQueryScope(queryScope)
	for _, scope := range binding.AllowedQueryScopes {
		if scope == queryScope {
			return true
		}
	}
	return false
}

func validWorkspaceDeviceBindingStatus(status string) bool {
	switch strings.TrimSpace(status) {
	case "bound", "revoked", "deleted_metadata_only":
		return true
	default:
		return false
	}
}

func safeWorkspaceContentType(contentType string) string {
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	if contentType == "" {
		return ""
	}
	if len(contentType) > 80 {
		return ""
	}
	for _, r := range contentType {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '/' || r == '-' || r == '_' || r == '+' || r == '.' {
			continue
		}
		return ""
	}
	return contentType
}
