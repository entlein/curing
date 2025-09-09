package server

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log/slog"
	"net"
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
	slog.Info("Starting server", "port", s.port)
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		slog.Error("Failed to start server", "error", err)
		os.Exit(1)
	}
	defer func(listener net.Listener) {
		_ = listener.Close()
	}(listener)

	for {
		conn, err := listener.Accept()
		if err != nil {
			slog.Error("Failed to accept the connection", "error", err)
			continue
		}
		go s.handleRequest(conn)
	}
}

// In server:
func (s *Server) handleRequest(conn net.Conn) {
	defer func(conn net.Conn) {
		_ = conn.Close()
	}(conn)

	decoder := gob.NewDecoder(conn)
	encoder := gob.NewEncoder(conn)

	r := &common.Request{}
	if err := decoder.Decode(r); err != nil {
		slog.Error("Failed to decode request", "error", err)
		return
	}
	slog.Info("Received request", "type", r.Type, "agentID", r.AgentID, "groups", r.Groups)

	switch r.Type {
	case common.GetCommands:
		commands := s.config.GetCommandsForClient(r.AgentID, r.Groups)
		slog.Info("Resolved commands for client", "agentID", r.AgentID, "groups", r.Groups, "commandCount", len(commands))

		slog.Info("About to encode commands", "commands", commands)

		// Try encoding to a buffer first to verify the data
		var buf bytes.Buffer
		tmpEncoder := gob.NewEncoder(&buf)
		if err := tmpEncoder.Encode(commands); err != nil {
			slog.Error("Failed to encode to buffer", "error", err)
			return
		}

		slog.Info("Successfully encoded to buffer", "size", buf.Len())

		if err := encoder.Encode(commands); err != nil {
			slog.Error("Failed to encode commands", "error", err)
			return
		}

		slog.Info("Successfully encoded to connection")
		// Ensure all data is written before closing
		if conn, ok := conn.(*net.TCPConn); ok {
			conn.CloseWrite()
		}

	case common.SendResults:
		for _, r := range r.Results {
			slog.Info("Received result", "result", r.CommandID, "returnCode", r.ReturnCode)
			slog.Info("Output preview", "output", string(r.Output))
		}

	default:
		slog.Error("Unknown request type", "type", r.Type)
	}
}
