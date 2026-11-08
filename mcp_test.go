package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
	"time"
	"xray-builder/models"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func connectMCP(t *testing.T, run mcpCommand) *mcp.ClientSession {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := newMCPServer(run).Connect(ctx, serverTransport, nil)
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		session.Close()
		serverSession.Close()
		cancel()
	})
	return session
}

func callMCPTool(t *testing.T, session *mcp.ClientSession, name string, arguments map[string]any) *mcp.CallToolResult {
	t.Helper()
	result, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: name, Arguments: arguments})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("%s failed: %v", name, result.Content)
	}
	return result
}

func TestMCPTools(t *testing.T) {
	var calls [][]string
	session := connectMCP(t, func(_ context.Context, arguments ...string) ([]byte, error) {
		calls = append(calls, append([]string(nil), arguments...))
		if reflect.DeepEqual(arguments, []string{"users", "--json"}) {
			return []byte(`[{"id":"alice-id","name":"alice","enabled":true}]`), nil
		}
		return []byte("done\n"), nil
	})
	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(tools.Tools) != 7 {
		t.Fatalf("tool count = %d, want 7", len(tools.Tools))
	}
	for _, tool := range tools.Tools {
		readOnly := tool.Name == "list_users" || tool.Name == "share_user"
		if tool.InputSchema == nil || tool.OutputSchema == nil || tool.Annotations == nil || tool.Annotations.ReadOnlyHint != readOnly {
			t.Fatalf("missing schema or incorrect annotations for %s", tool.Name)
		}
	}
	result := callMCPTool(t, session, "list_users", map[string]any{})
	data, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var users mcpUsersOutput
	if err := json.Unmarshal(data, &users); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(users.Users, []mcpUser{{ID: "alice-id", Name: "alice", Enabled: true}}) {
		t.Fatalf("unexpected users: %+v", users)
	}
	for _, command := range []string{"add", "remove", "enable", "disable"} {
		user := "alice; echo literal"
		callMCPTool(t, session, command+"_user", map[string]any{"user": user, "dry_run": true, "no_restart": true})
		want := []string{"--dry-run", "--no-restart", command, "--user", user}
		if got := calls[len(calls)-1]; !reflect.DeepEqual(got, want) {
			t.Fatalf("arguments = %v, want %v", got, want)
		}
	}
	callMCPTool(t, session, "setup_server", map[string]any{"destination": "example.com", "dry_run": true})
	if got, want := calls[len(calls)-1], []string{"--dry-run", "setup", "--destination", "example.com"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("setup arguments = %v, want %v", got, want)
	}
	callMCPTool(t, session, "share_user", map[string]any{"user": "alice"})
	if got, want := calls[len(calls)-1], []string{"--user", "alice", "share", "--format", "link"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("share arguments = %v, want %v", got, want)
	}
	before := len(calls)
	for _, arguments := range []map[string]any{{}, {"user": ""}, {"user": "alice", "dry_run": "true"}} {
		result, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: "add_user", Arguments: arguments})
		if err == nil && !result.IsError {
			t.Fatalf("accepted invalid input: %v", arguments)
		}
	}
	if len(calls) != before {
		t.Fatal("invalid input invoked a command")
	}
}

func TestMCPStdio(t *testing.T) {
	directory := t.TempDir()
	executable := filepath.Join(directory, "xray-builder.exe")
	if output, err := exec.Command("go", "build", "-o", executable, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, output)
	}
	configPath := filepath.Join(directory, "server.json")
	config := []byte(`{"inbounds":[{"settings":{"clients":[{"id":"alice-id","email":"alice","shortId":"alice-short"}]},"streamSettings":{"realitySettings":{"shortIds":["alice-short"]}}}]}`)
	if err := os.WriteFile(configPath, config, 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil)
	session, err := client.Connect(ctx, &mcp.CommandTransport{Command: exec.Command(executable, "--config", configPath, "mcp")}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	callMCPTool(t, session, "list_users", map[string]any{})
	for _, action := range []string{"add", "remove", "enable", "disable"} {
		user := "alice"
		if action == "add" {
			user = "bob"
		}
		callMCPTool(t, session, action+"_user", map[string]any{"user": user, "dry_run": true})
		if got, err := os.ReadFile(configPath); err != nil || !bytes.Equal(got, config) {
			t.Fatalf("%s preview changed config: %v", action, err)
		}
	}
	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "remove_user", Arguments: map[string]any{"user": "missing"}})
	if err != nil || !result.IsError {
		t.Fatalf("expected tool error for missing user: result=%+v error=%v", result, err)
	}
	callMCPTool(t, session, "list_users", map[string]any{})
	for _, action := range []string{"disable", "enable", "remove"} {
		callMCPTool(t, session, action+"_user", map[string]any{"user": "alice-id", "no_restart": true})
		data, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatal(err)
		}
		var cfg models.ServerConfig
		if err := json.Unmarshal(data, &cfg); err != nil {
			t.Fatal(err)
		}
		wantShortIDs := []string{}
		if action == "enable" {
			wantShortIDs = []string{"alice-short"}
		}
		if got := cfg.FirstInbound().StreamSettings.RealitySettings.ShortIds; !reflect.DeepEqual(got, wantShortIDs) {
			t.Fatalf("%s: short IDs = %v, want %v", action, got, wantShortIDs)
		}
		if action == "remove" && len(cfg.FirstInbound().Settings.Clients) != 0 {
			t.Fatal("UUID removal left the client in the config")
		}
	}
	result = callMCPTool(t, session, "list_users", map[string]any{})
	data, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var empty mcpUsersOutput
	if err := json.Unmarshal(data, &empty); err != nil {
		t.Fatal(err)
	}
	if empty.Users == nil || len(empty.Users) != 0 {
		t.Fatalf("expected an empty user array, got %+v", empty.Users)
	}
}
