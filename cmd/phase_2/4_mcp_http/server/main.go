package main

import (
	mcpModel "agent_study/pkg/mcp/model"
	mcpServer "agent_study/pkg/mcp/server"
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"
)

func main() {
	mc := mcpServer.NewServer()

	helloTool, err := mcpModel.NewTool("hello_world",
		"Say hello to someone",
		mcpModel.ToolParams(
			mcpModel.RequiredParam("name", "string", "The name of the person to say hello to"),
		),
		handleHello,
	)
	if err != nil {
		panic(err)
	}

	uuidTool, err := mcpModel.NewTool("generate_uuid",
		"Generate a new UUID",
		mcpModel.ToolParams(),
		handleUUID,
	)
	if err != nil {
		panic(err)
	}

	err = mc.RegisterTool(helloTool, uuidTool)
	if err != nil {
		panic(err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/mcp", mc.NewHttpHandler().ServeHTTP)

	server := http.Server{
		Addr:    "0.0.0.0:7888",
		Handler: mux,
	}

	if err = server.ListenAndServe(); err != nil {
		panic(err)
	}

}

func handleHello(ctx context.Context, arguments map[string]interface{}) (string, error) {
	return fmt.Sprintf("hello %s", arguments["name"]), nil
}

func handleUUID(ctx context.Context, arguments map[string]interface{}) (string, error) {
	return uuid.NewString(), nil
}
