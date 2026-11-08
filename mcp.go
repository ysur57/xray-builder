package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"xray-builder/models"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type mcpCommand func(context.Context, ...string) ([]byte, error)

type mcpUserInput struct {
	User string `json:"user" jsonschema:"User UUID or comment"`
}

type mcpMutationInput struct {
	User      string `json:"user" jsonschema:"User UUID or comment; for add_user supply the new comment"`
	DryRun    bool   `json:"dry_run,omitempty" jsonschema:"Preview without writing files, generating credentials, or restarting Xray"`
	NoRestart bool   `json:"no_restart,omitempty" jsonschema:"Save changes without restarting Xray"`
}

type mcpSetupInput struct {
	Destination string `json:"destination" jsonschema:"Destination hostname without a scheme, port, or path"`
	DryRun      bool   `json:"dry_run,omitempty" jsonschema:"Preview without writing files or generating credentials"`
	NoRestart   bool   `json:"no_restart,omitempty" jsonschema:"Save configuration and keys without restarting Xray"`
}

type mcpMutationOutput struct {
	Action    string `json:"action"`
	DryRun    bool   `json:"dry_run"`
	Restarted bool   `json:"restarted"`
	Message   string `json:"message"`
}

type mcpUser struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

type mcpUsersOutput struct {
	Users []mcpUser `json:"users"`
}

type mcpShareOutput struct {
	Link string `json:"link"`
}

func runMCP(ctx context.Context, args *models.Args) error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	var mutex sync.Mutex
	run := func(ctx context.Context, arguments ...string) ([]byte, error) {
		mutex.Lock()
		defer mutex.Unlock()
		arguments = append([]string{"--config", args.XrayConfigPath, "--keypair", args.XrayKeypairPath}, arguments...)
		cmd := exec.CommandContext(ctx, executable, arguments...)
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("command failed: %w: %s", err, strings.TrimSpace(stderr.String()))
		}
		return stdout.Bytes(), nil
	}
	return newMCPServer(run).Run(ctx, &mcp.StdioTransport{})
}

func newMCPServer(run mcpCommand) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "xray-builder", Version: "1.0.0"}, nil)
	mcp.AddTool(server, &mcp.Tool{
		Name: "list_users", Description: "List user IDs, names, and enabled state without returning credentials.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, mcpUsersOutput, error) {
		data, err := run(ctx, "users", "--json")
		if err != nil {
			return nil, mcpUsersOutput{}, err
		}
		output := mcpUsersOutput{Users: []mcpUser{}}
		err = json.Unmarshal(data, &output.Users)
		return nil, output, err
	})
	mcp.AddTool(server, &mcp.Tool{
		Name: "share_user", Description: "Return a VLESS share link containing connection credentials for the selected user.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input mcpUserInput) (*mcp.CallToolResult, mcpShareOutput, error) {
		if strings.TrimSpace(input.User) == "" {
			return nil, mcpShareOutput{}, fmt.Errorf("user must not be empty")
		}
		data, err := run(ctx, "--user", input.User, "share", "--format", "link")
		return nil, mcpShareOutput{Link: strings.TrimSpace(string(data))}, err
	})
	destructive := true
	for _, operation := range []struct{ name, command, description string }{
		{"add_user", "add", "Add a user with a new UUID and short ID. Restarts Xray unless dry_run or no_restart is set."},
		{"remove_user", "remove", "Remove a user. Restarts Xray unless dry_run or no_restart is set."},
		{"enable_user", "enable", "Enable a user. Restarts Xray unless dry_run or no_restart is set."},
		{"disable_user", "disable", "Disable a user. Restarts Xray unless dry_run or no_restart is set."},
	} {
		mcp.AddTool(server, &mcp.Tool{
			Name: operation.name, Description: operation.description,
			Annotations: &mcp.ToolAnnotations{DestructiveHint: &destructive},
		}, func(ctx context.Context, _ *mcp.CallToolRequest, input mcpMutationInput) (*mcp.CallToolResult, mcpMutationOutput, error) {
			if strings.TrimSpace(input.User) == "" {
				return nil, mcpMutationOutput{}, fmt.Errorf("user must not be empty")
			}
			output, err := callMCPMutation(ctx, run, operation.command, []string{"--user", input.User}, input.DryRun, input.NoRestart)
			return nil, output, err
		})
	}
	mcp.AddTool(server, &mcp.Tool{
		Name: "setup_server", Description: "Replace the server configuration and generate new keys for an existing Xray installation. Restarts Xray unless dry_run or no_restart is set.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: &destructive},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input mcpSetupInput) (*mcp.CallToolResult, mcpMutationOutput, error) {
		if strings.TrimSpace(input.Destination) == "" {
			return nil, mcpMutationOutput{}, fmt.Errorf("destination must not be empty")
		}
		output, err := callMCPMutation(ctx, run, "setup", []string{"--destination", input.Destination}, input.DryRun, input.NoRestart)
		return nil, output, err
	})
	return server
}

func callMCPMutation(ctx context.Context, run mcpCommand, action string, arguments []string, dryRun, noRestart bool) (mcpMutationOutput, error) {
	flags := []string{}
	if dryRun {
		flags = append(flags, "--dry-run")
	}
	if noRestart {
		flags = append(flags, "--no-restart")
	}
	data, err := run(ctx, append(append(flags, action), arguments...)...)
	return mcpMutationOutput{
		Action: action, DryRun: dryRun, Restarted: err == nil && !dryRun && !noRestart,
		Message: strings.TrimSpace(string(data)),
	}, err
}
