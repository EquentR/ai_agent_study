package main

import (
	"agent_study/pkg/mcp/model"
	"agent_study/pkg/mcp/server"
	"log"

	"github.com/google/uuid"
)

type uuidArgs struct{}

func main() {
	s := server.NewServer()

	uuidTool, err := model.NewTypedToolNoContext(
		"get_uuid",
		"Generate a random UUID",
		nil,
		func(args uuidArgs) (string, error) {
			_ = args
			return uuid.NewString(), nil
		},
	)
	if err != nil {
		log.Fatalf("failed to create get_uuid tool: %v", err)
	}

	if err := s.RegisterTool(uuidTool); err != nil {
		log.Fatalf("failed to register tools: %v", err)
	}

	if err := s.ServeStdio(); err != nil {
		log.Fatalf("stdio server exited with error: %v", err)
	}
}
