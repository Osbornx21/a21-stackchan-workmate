package gateway

import (
	"fmt"
	"net/http"
)

func (s *Server) handleSimulator(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprint(w, simulatorHTML)
}

const simulatorHTML = simulatorHTMLPart01 +
	simulatorHTMLPart02 +
	simulatorHTMLPart03
