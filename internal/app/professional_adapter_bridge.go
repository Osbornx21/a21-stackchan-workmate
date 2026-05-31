package app

import (
	"bytes"
	"context"
	"encoding/json"
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
	QueryRunID   string                 `json:"query_run_id"`
	SpokenAnswer string                 `json:"spoken_answer"`
	FullAnswer   string                 `json:"full_answer"`
	Confidence   float64                `json:"confidence"`
	Evidence     []v21VoiceEvidence     `json:"evidence"`
	LatencyMS    map[string]int64       `json:"latency_ms,omitempty"`
	ToolResults  map[string]interface{} `json:"tool_results,omitempty"`
}

type v21VoiceEvidence struct {
	ChunkID      string  `json:"chunk_id"`
	AnchorID     string  `json:"anchor_id"`
	VersionID    string  `json:"version_id"`
	SourceUnitID string  `json:"source_unit_id"`
	SourceLabel  string  `json:"source_label"`
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
	Excerpt      string  `json:"excerpt"`
	Score        float64 `json:"score"`
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
		response, err := executeV21RetrievalQuery(r.Context(), client, v21Base, collectionID, request)
		if err != nil {
			http.Error(w, "v21 query unavailable", http.StatusServiceUnavailable)
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
	body := map[string]interface{}{
		"workspace_id":     "ws_air",
		"agent_id":         "a21",
		"session_id":       firstNonEmpty(strings.TrimSpace(request.SessionID), "a21-session-v21-adapter"),
		"turn_id":          firstNonEmpty(strings.TrimSpace(request.TraceID), "a21-turn-v21-adapter"),
		"trace_id":         firstNonEmpty(strings.TrimSpace(request.TraceID), "a21-trace-v21-adapter"),
		"collection_ids":   []string{collectionID},
		"question":         strings.TrimSpace(request.Utterance),
		"mode":             "grounded_qa",
		"response_style":   "short_spoken",
		"max_spoken_chars": 180,
		"allow_style_wrap": false,
	}
	if strings.TrimSpace(request.Utterance) == "" {
		return v21adapter.QueryResponse{}, fmt.Errorf("utterance is required")
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
		return v21adapter.QueryResponse{}, fmt.Errorf("v21 voice query returned status %d", resp.StatusCode)
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
	return response, nil
}

func executeV21RetrievalQuery(ctx context.Context, client *http.Client, v21Base string, collectionID string, request v21adapter.QueryRequest) (v21adapter.QueryResponse, error) {
	if err := v21adapter.ValidateProfessionalQueryRequest(request); err != nil {
		return v21adapter.QueryResponse{}, err
	}
	utterance := strings.TrimSpace(request.Utterance)
	body := map[string]interface{}{
		"query": utterance,
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
		return v21adapter.QueryResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return v21adapter.QueryResponse{}, fmt.Errorf("v21 retrieval query returned status %d", resp.StatusCode)
	}
	var retrieval v21RetrievalQueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&retrieval); err != nil {
		return v21adapter.QueryResponse{}, err
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
	return response, nil
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
	return &http.Client{
		Timeout:   timeout,
		Transport: &http.Transport{Proxy: nil},
	}
}

func writeV21BridgeJSON(w http.ResponseWriter, status int, value interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
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
