package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"a21.local/a21/internal/v21adapter"
)

type v21AdapterBridgeOptions struct {
	Addr         string
	V21URL       string
	CollectionID string
}

type v21CollectionView struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	ActiveReleaseID *string `json:"active_release_id,omitempty"`
}

type v21VoiceQueryResponse struct {
	QueryRunID        string                 `json:"query_run_id"`
	SpokenAnswer      string                 `json:"spoken_answer"`
	FullAnswer        string                 `json:"full_answer"`
	Confidence        float64                `json:"confidence"`
	Evidence          []v21VoiceEvidence     `json:"evidence"`
	LatencyMS         map[string]int64       `json:"latency_ms,omitempty"`
	ToolResults       map[string]interface{} `json:"tool_results,omitempty"`
	SourceScopeCounts map[string]int         `json:"source_scope_counts,omitempty"`
	WorkspaceStatus   string                 `json:"workspace_status,omitempty"`
}

type v21VoiceEvidence struct {
	ChunkID      string  `json:"chunk_id"`
	AnchorID     string  `json:"anchor_id"`
	VersionID    string  `json:"version_id"`
	SourceUnitID string  `json:"source_unit_id"`
	SourceLabel  string  `json:"source_label"`
	SourceScope  string  `json:"source_scope,omitempty"`
	Excerpt      string  `json:"excerpt"`
	Score        float64 `json:"score"`
}

type v21RetrievalQueryResponse struct {
	CollectionID string               `json:"collection_id"`
	Results      []v21RetrievalResult `json:"results"`
}

type v21RetrievalResult struct {
	ChunkID      string  `json:"chunk_id"`
	VersionID    string  `json:"version_id"`
	AnchorID     string  `json:"anchor_id"`
	SourceUnitID string  `json:"source_unit_id"`
	SourceLabel  string  `json:"source_label"`
	SourceScope  string  `json:"source_scope,omitempty"`
	Excerpt      string  `json:"excerpt"`
	Score        float64 `json:"score"`
}

type v21BridgeQueryError struct {
	Status      int    `json:"-"`
	Code        string `json:"code"`
	StatusClass string `json:"status_class,omitempty"`
}

func (e v21BridgeQueryError) Error() string {
	if e.Code == "" {
		return "v21 bridge query failed"
	}
	return "v21 bridge query failed: " + e.Code
}

func runV21AdapterBridge(args []string, stdout io.Writer, stderr io.Writer) int {
	options := v21AdapterBridgeOptions{
		Addr:         firstNonEmpty(strings.TrimSpace(os.Getenv("A21_V21_ADAPTER_ADDR")), "127.0.0.1:21121"),
		V21URL:       firstNonEmpty(strings.TrimSpace(os.Getenv("A21_V21_BACKEND_URL")), "http://127.0.0.1:18080"),
		CollectionID: strings.TrimSpace(os.Getenv("A21_V21_COLLECTION_ID")),
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Fprintln(stdout, "a21 v21-adapter-bridge [--addr 127.0.0.1:21121] [--v21-url http://127.0.0.1:18080] [--collection-id col_...]")
			return 0
		case "--addr":
			if !readStringOption(args, &i, stderr, "--addr", &options.Addr) {
				return 2
			}
		case "--v21-url":
			if !readStringOption(args, &i, stderr, "--v21-url", &options.V21URL) {
				return 2
			}
		case "--collection-id":
			if !readStringOption(args, &i, stderr, "--collection-id", &options.CollectionID) {
				return 2
			}
		default:
			fmt.Fprintf(stderr, "unknown v21-adapter-bridge option %q\n", args[i])
			return 2
		}
	}
	handler, err := newV21AdapterBridgeHandler(context.Background(), options)
	if err != nil {
		fmt.Fprintf(stderr, "v21 adapter bridge not ready: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "a21 v21 adapter bridge listening on %s -> %s\n", options.Addr, sanitizedV21BridgeURL(options.V21URL))
	if err := http.ListenAndServe(options.Addr, handler); err != nil {
		fmt.Fprintf(stderr, "v21 adapter bridge stopped: %v\n", err)
		return 1
	}
	return 0
}

func newV21AdapterBridgeHandler(ctx context.Context, options v21AdapterBridgeOptions) (http.Handler, error) {
	v21Base, err := normalizeV21BackendURL(options.V21URL)
	if err != nil {
		return nil, err
	}
	client := v21BridgeHTTPClient(8 * time.Second)
	collectionID := strings.TrimSpace(options.CollectionID)
	if collectionID == "" {
		collectionID, err = discoverV21ActiveCollection(ctx, client, v21Base)
		if err != nil {
			return nil, err
		}
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := probeV21BackendHealth(r.Context(), client, v21Base); err != nil {
			http.Error(w, "v21 backend unavailable", http.StatusServiceUnavailable)
			return
		}
		writeV21BridgeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "a21-v21-adapter-bridge"})
	})
	mux.HandleFunc(v21adapter.QueryPath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var request v21adapter.QueryRequest
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&request); err != nil {
			http.Error(w, "invalid query request", http.StatusBadRequest)
			return
		}
		if err := v21adapter.ValidateProfessionalQueryRequest(request); err != nil {
			http.Error(w, "invalid professional query request", http.StatusBadRequest)
			return
		}
		response, err := executeV21BridgeQuery(r.Context(), client, v21Base, collectionID, request)
		if err != nil {
			writeV21BridgeQueryError(w, err)
			return
		}
		writeV21BridgeJSON(w, http.StatusOK, response)
	})
	return mux, nil
}

func normalizeV21BackendURL(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("v21 backend URL must use http or https")
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("v21 backend URL host is required")
	}
	if parsed.User != nil {
		return "", fmt.Errorf("v21 backend URL must not include credentials")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}

func discoverV21ActiveCollection(ctx context.Context, client *http.Client, v21Base string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v21Base+"/api/v1/collections", nil)
	if err != nil {
		return "", err
	}
	setV21DevHeaders(req)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("v21 collections returned status %d", resp.StatusCode)
	}
	var collections []v21CollectionView
	if err := json.NewDecoder(resp.Body).Decode(&collections); err != nil {
		return "", err
	}
	for _, collection := range collections {
		if strings.TrimSpace(collection.ID) != "" && collection.ActiveReleaseID != nil && strings.TrimSpace(*collection.ActiveReleaseID) != "" {
			return collection.ID, nil
		}
	}
	return "", fmt.Errorf("v21 has no active collection for A21 adapter bridge")
}

func probeV21BackendHealth(ctx context.Context, client *http.Client, v21Base string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v21Base+"/api/v1/healthz", nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("v21 health returned status %d", resp.StatusCode)
	}
	return nil
}

func executeV21VoiceQuery(ctx context.Context, client *http.Client, v21Base string, collectionID string, request v21adapter.QueryRequest) (v21adapter.QueryResponse, error) {
	utterance := strings.TrimSpace(request.Utterance)
	if utterance == "" {
		return v21adapter.QueryResponse{}, fmt.Errorf("utterance is required")
	}
	body := map[string]interface{}{
		"device_id":        strings.TrimSpace(request.DeviceID),
		"user_id":          firstNonEmpty(strings.TrimSpace(request.UserID), v21adapter.DefaultUserID),
		"workspace_id":     firstNonEmpty(strings.TrimSpace(request.WorkspaceID), v21adapter.DefaultWorkspaceID),
		"query_scope":      firstNonEmpty(strings.TrimSpace(request.QueryScope), v21adapter.QueryScopePublic),
		"agent_id":         "a21",
		"session_id":       firstNonEmpty(strings.TrimSpace(request.SessionID), "a21-session-v21-adapter"),
		"turn_id":          firstNonEmpty(strings.TrimSpace(request.TraceID), "a21-turn-v21-adapter"),
		"trace_id":         firstNonEmpty(strings.TrimSpace(request.TraceID), "a21-trace-v21-adapter"),
		"collection_ids":   []string{collectionID},
		"question":         utterance,
		"mode":             "grounded_qa",
		"response_style":   "short_spoken",
		"max_spoken_chars": 180,
		"allow_style_wrap": false,
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return v21adapter.QueryResponse{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, v21Base+"/internal/v1/knowledge/voice-query", bytes.NewReader(encoded))
	if err != nil {
		return v21adapter.QueryResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	setV21DevHeaders(req)
	resp, err := client.Do(req)
	if err != nil {
		return v21adapter.QueryResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return v21adapter.QueryResponse{}, v21BridgeQueryError{
			Status:      v21BridgeAdapterStatus(resp.StatusCode),
			Code:        v21BridgeVoiceQueryErrorCode(resp.StatusCode),
			StatusClass: v21BridgeStatusClass(resp.StatusCode),
		}
	}
	var v21Response v21VoiceQueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&v21Response); err != nil {
		return v21adapter.QueryResponse{}, err
	}
	response := v21adapter.QueryResponse{
		TraceID:    firstNonEmpty(strings.TrimSpace(request.TraceID), v21Response.QueryRunID),
		FastAnswer: firstNonEmpty(strings.TrimSpace(v21Response.SpokenAnswer), strings.TrimSpace(v21Response.FullAnswer)),
		Confidence: v21Response.Confidence,
	}
	for _, evidence := range v21Response.Evidence {
		response.Evidence = append(response.Evidence, v21adapter.Evidence{
			Title:    firstNonEmpty(strings.TrimSpace(evidence.SourceLabel), strings.TrimSpace(evidence.SourceUnitID), strings.TrimSpace(evidence.ChunkID)),
			Type:     "v21_evidence",
			SourceID: firstNonEmpty(strings.TrimSpace(evidence.AnchorID), strings.TrimSpace(evidence.SourceUnitID), strings.TrimSpace(evidence.ChunkID)),
			Summary:  truncateV21BridgeRunes(strings.TrimSpace(evidence.Excerpt), 240),
		})
	}
	if response.FastAnswer != "" {
		response.SpeechBlocks = []string{response.FastAnswer}
	}
	if len(response.Evidence) > 0 {
		response.ScreenCards = []v21adapter.ScreenCard{{Label: "V21 Evidence", Text: response.Evidence[0].Title}}
	}
	response.FollowUps = buildV21BridgeFollowUps(response.Evidence)
	response.SourceScopeCounts = copyV21BridgeSourceScopeCounts(v21Response.SourceScopeCounts)
	response.WorkspaceStatus = v21BridgeWorkspaceStatus(v21Response.WorkspaceStatus)
	return response, nil
}

func executeV21BridgeQuery(ctx context.Context, client *http.Client, v21Base string, collectionID string, request v21adapter.QueryRequest) (v21adapter.QueryResponse, error) {
	if err := v21adapter.ValidateProfessionalQueryRequest(request); err != nil {
		return v21adapter.QueryResponse{}, err
	}
	response, err := executeV21VoiceQuery(ctx, client, v21Base, collectionID, request)
	if err == nil || !isV21BridgeNoEvidence(err) {
		return response, err
	}
	return executeV21RetrievalQuery(ctx, client, v21Base, collectionID, request)
}

func executeV21RetrievalQuery(ctx context.Context, client *http.Client, v21Base string, collectionID string, request v21adapter.QueryRequest) (v21adapter.QueryResponse, error) {
	utterance := strings.TrimSpace(request.Utterance)
	response, err := executeV21RetrievalQueryText(ctx, client, v21Base, collectionID, request, utterance)
	if err == nil || !isV21BridgeNoEvidence(err) {
		return response, err
	}
	for _, expanded := range v21BridgeASRQueryExpansions(utterance) {
		response, retryErr := executeV21RetrievalQueryText(ctx, client, v21Base, collectionID, request, expanded)
		if retryErr == nil {
			return response, nil
		}
		if !isV21BridgeNoEvidence(retryErr) {
			return v21adapter.QueryResponse{}, retryErr
		}
	}
	return response, err
}

func executeV21RetrievalQueryText(ctx context.Context, client *http.Client, v21Base string, collectionID string, request v21adapter.QueryRequest, query string) (v21adapter.QueryResponse, error) {
	body := map[string]interface{}{
		"query": strings.TrimSpace(query),
		"limit": 5,
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return v21adapter.QueryResponse{}, err
	}
	endpoint := v21Base + "/api/v1/collections/" + url.PathEscape(collectionID) + "/retrieval/query"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(encoded))
	if err != nil {
		return v21adapter.QueryResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	setV21DevHeaders(req)
	resp, err := client.Do(req)
	if err != nil {
		return v21adapter.QueryResponse{}, v21BridgeQueryError{
			Status: http.StatusBadGateway,
			Code:   "upstream_unavailable",
		}
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return v21adapter.QueryResponse{}, v21BridgeQueryError{
			Status:      http.StatusBadGateway,
			Code:        "upstream_status",
			StatusClass: v21BridgeStatusClass(resp.StatusCode),
		}
	}
	var retrieval v21RetrievalQueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&retrieval); err != nil {
		return v21adapter.QueryResponse{}, v21BridgeQueryError{
			Status: http.StatusBadGateway,
			Code:   "upstream_contract_invalid",
		}
	}
	if len(retrieval.Results) == 0 {
		return v21adapter.QueryResponse{}, v21BridgeQueryError{
			Status: http.StatusFailedDependency,
			Code:   "no_evidence",
		}
	}
	response := v21adapter.QueryResponse{
		TraceID:    firstNonEmpty(strings.TrimSpace(request.TraceID), "a21-trace-v21-retrieval"),
		FastAnswer: buildV21RetrievalFastAnswer(retrieval.Results),
		Confidence: v21RetrievalConfidence(retrieval.Results),
	}
	for _, result := range retrieval.Results {
		response.Evidence = append(response.Evidence, v21adapter.Evidence{
			Title:    firstNonEmpty(strings.TrimSpace(result.SourceLabel), strings.TrimSpace(result.SourceUnitID), strings.TrimSpace(result.ChunkID)),
			Type:     "v21_retrieval_evidence",
			SourceID: firstNonEmpty(strings.TrimSpace(result.AnchorID), strings.TrimSpace(result.SourceUnitID), strings.TrimSpace(result.ChunkID)),
			Summary:  truncateV21BridgeRunes(strings.TrimSpace(result.Excerpt), 240),
		})
	}
	if response.FastAnswer != "" {
		response.SpeechBlocks = []string{response.FastAnswer}
	}
	if len(response.Evidence) > 0 {
		response.ScreenCards = []v21adapter.ScreenCard{{Label: "V21 Evidence", Text: response.Evidence[0].Title}}
	}
	response.FollowUps = buildV21BridgeFollowUps(response.Evidence)
	response.SourceScopeCounts = v21BridgeSourceScopeCountsFromRetrieval(retrieval.Results)
	response.WorkspaceStatus = v21BridgeWorkspaceStatusForFallback(response.SourceScopeCounts)
	return response, nil
}

func copyV21BridgeSourceScopeCounts(counts map[string]int) map[string]int {
	if len(counts) == 0 {
		return nil
	}
	out := make(map[string]int, len(counts))
	for scope, count := range counts {
		if count < 0 {
			return nil
		}
		switch strings.TrimSpace(scope) {
		case "public", "personal":
			out[strings.TrimSpace(scope)] = count
		default:
			return nil
		}
	}
	return out
}

func v21BridgeSourceScopeCountsFromRetrieval(results []v21RetrievalResult) map[string]int {
	counts := make(map[string]int, 2)
	for _, result := range results {
		switch strings.TrimSpace(result.SourceScope) {
		case "public":
			counts["public"]++
		case "personal":
			counts["personal"]++
		}
	}
	if len(counts) == 0 {
		return nil
	}
	return counts
}

func v21BridgeWorkspaceStatus(status string) string {
	switch strings.TrimSpace(status) {
	case v21adapter.WorkspaceSearchable, v21adapter.WorkspaceScopePending, "uploaded", "indexing", "failed", "unavailable":
		return strings.TrimSpace(status)
	default:
		return ""
	}
}

func v21BridgeWorkspaceStatusForFallback(counts map[string]int) string {
	if len(counts) == 0 {
		return "unavailable"
	}
	return v21adapter.WorkspaceSearchable
}

func v21BridgeAdapterStatus(status int) int {
	if status == http.StatusFailedDependency {
		return http.StatusFailedDependency
	}
	return http.StatusBadGateway
}

func v21BridgeVoiceQueryErrorCode(status int) string {
	if status == http.StatusFailedDependency || status == http.StatusNotFound {
		return "no_evidence"
	}
	return "upstream_status"
}

func isV21BridgeNoEvidence(err error) bool {
	var queryErr v21BridgeQueryError
	return errors.As(err, &queryErr) && queryErr.Code == "no_evidence"
}

func v21BridgeASRQueryExpansions(utterance string) []string {
	cleaned := strings.TrimSpace(utterance)
	if cleaned == "" {
		return nil
	}
	candidates := make([]string, 0, 2)
	if strings.Contains(cleaned, "儿童") || strings.Contains(cleaned, "童") {
		candidates = append(candidates, "儿童锁 车型 车门")
	}
	if strings.Contains(cleaned, "唤醒") || strings.Contains(cleaned, "语音") {
		candidates = append(candidates, "语音唤醒 误触发")
	}
	result := make([]string, 0, len(candidates))
	seen := map[string]bool{cleaned: true}
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" || seen[candidate] {
			continue
		}
		seen[candidate] = true
		result = append(result, candidate)
	}
	return result
}

func buildV21RetrievalFastAnswer(results []v21RetrievalResult) string {
	if len(results) == 0 {
		return "V21 当前没有找到可引用证据。"
	}
	label := strings.TrimSpace(results[0].SourceLabel)
	if label == "" {
		label = "已找到相关证据"
	}
	return truncateV21BridgeRunes("V21 找到证据："+label, 120)
}

func v21RetrievalConfidence(results []v21RetrievalResult) float64 {
	if len(results) == 0 {
		return 0
	}
	best := results[0].Score
	for _, result := range results[1:] {
		if result.Score > best {
			best = result.Score
		}
	}
	if best < 0 {
		return 0
	}
	if best > 1 {
		return 1
	}
	return best
}

func buildV21BridgeFollowUps(evidence []v21adapter.Evidence) []string {
	if len(evidence) == 0 {
		return []string{"要不要换一个专业问题继续查？"}
	}
	return []string{"要不要打开 V21 工作台查看证据详情？"}
}

func setV21DevHeaders(req *http.Request) {
	req.Header.Set("X-Dev-Role", "VIEWER")
	req.Header.Set("X-Dev-Actor-ID", "act_a21_v21_adapter")
	req.Header.Set("X-Dev-Workspace-ID", "ws_air")
}

func v21BridgeHTTPClient(timeout time.Duration) *http.Client {
	return &*a21DirectHTTPClient(timeout)
}

func writeV21BridgeJSON(w http.ResponseWriter, status int, value interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeV21BridgeQueryError(w http.ResponseWriter, err error) {
	bridgeErr, ok := err.(v21BridgeQueryError)
	if !ok {
		bridgeErr = v21BridgeQueryError{Status: http.StatusServiceUnavailable, Code: "query_unavailable"}
	}
	if bridgeErr.Status == 0 {
		bridgeErr.Status = http.StatusServiceUnavailable
	}
	if bridgeErr.StatusClass == "" {
		bridgeErr.StatusClass = v21BridgeStatusClass(bridgeErr.Status)
	}
	writeV21BridgeJSON(w, bridgeErr.Status, bridgeErr)
}

func v21BridgeStatusClass(status int) string {
	switch {
	case status >= 100 && status < 200:
		return "status_1xx"
	case status >= 200 && status < 300:
		return "status_2xx"
	case status >= 300 && status < 400:
		return "status_3xx"
	case status >= 400 && status < 500:
		return "status_4xx"
	case status >= 500 && status < 600:
		return "status_5xx"
	default:
		return ""
	}
}

func truncateV21BridgeRunes(value string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	if maxRunes <= 1 {
		return string(runes[:maxRunes])
	}
	return strings.TrimSpace(string(runes[:maxRunes-1])) + "..."
}

func sanitizedV21BridgeURL(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Host == "" {
		return "invalid_v21_backend"
	}
	return parsed.Scheme + "://" + parsed.Host
}
