package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/amitschendel/curing/pkg/common"
)

type Server struct {
	port   int
	config *CommandConfig
}

func NewServer(port int, configPath string) (*Server, error) {
	config, err := LoadCommandConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load command config: %v", err)
	}
	return &Server{
		port:   port,
		config: config,
	}, nil
}

func (s *Server) Run() {
	http.HandleFunc("/commands", s.handleGetCommands)
	http.HandleFunc("/results", s.handleSendResults)

	addr := fmt.Sprintf(":%d", s.port)
	slog.Info("Starting HTTP server", "port", s.port)
	if err := http.ListenAndServe(addr, nil); err != nil {
		slog.Error("Failed to start HTTP server", "error", err)
		os.Exit(1)
	}
}

func (s *Server) handleGetCommands(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	var req common.Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("Failed to decode request", "error", err)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Type != common.GetCommands {
		http.Error(w, "Invalid request type", http.StatusBadRequest)
		return
	}

	commands := s.config.GetCommandsForClient(req.AgentID, req.Groups)
	slog.Info("Resolved commands for client", "agentID", req.AgentID, "groups", req.Groups, "commandCount", len(commands))

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(commands); err != nil {
		slog.Error("Failed to encode response", "error", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (s *Server) handleSendResults(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	var req common.Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("Failed to decode request", "error", err)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Type != common.SendResults {
		http.Error(w, "Invalid request type", http.StatusBadRequest)
		return
	}

	for _, result := range req.Results {
		slog.Info("Received result", "result", result.CommandID, "returnCode", result.ReturnCode)
		slog.Info("Output preview", "output", string(result.Output))
	}
	w.WriteHeader(http.StatusOK)
}
