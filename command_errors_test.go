package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIErrorProcess(t *testing.T) {
	if os.Getenv("XRAY_BUILDER_CLI_TEST") != "1" {
		return
	}
	for i, value := range os.Args {
		if value == "--" {
			os.Args = append([]string{"xray-builder"}, os.Args[i+1:]...)
			main()
			os.Exit(0)
		}
	}
	t.Fatal("missing command arguments")
}

func TestCLIErrorExit(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "server.json")
	config := []byte(`{"inbounds":[{"settings":{"clients":[{"id":"alice-id","email":"alice","shortId":"alice-short"}]},"streamSettings":{"realitySettings":{"shortIds":["alice-short"]}}}]}`)
	if err := os.WriteFile(configPath, config, 0600); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		args []string
		code int
		text string
	}{
		{"no command", nil, 2, "a subcommand is required"},
		{"missing user", []string{"add"}, 2, "--user is required"},
		{"unknown flag", []string{"--unknown"}, 2, "unknown argument"},
		{"invalid format", []string{"--user", "alice", "share", "--format", "yaml"}, 2, "--format must be link, json, or qr"},
		{"preview read command", []string{"--dry-run", "users"}, 2, "require a modifying command"},
		{"no restart share", []string{"--no-restart", "--user", "alice", "share"}, 2, "require a modifying command"},
		{"missing config", []string{"--config", configPath + ".missing", "users"}, 1, ".missing"},
		{"missing remove user", []string{"--config", configPath, "--user", "missing", "remove"}, 1, "user not found"},
		{"missing enable user", []string{"--config", configPath, "--user", "missing", "enable"}, 1, "user not found"},
		{"missing disable user", []string{"--config", configPath, "--user", "missing", "disable"}, 1, "user not found"},
		{"duplicate user", []string{"--config", configPath, "--user", "alice", "add"}, 1, "user already exists"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			arguments := append([]string{"-test.run=^TestCLIErrorProcess$", "--"}, tc.args...)
			cmd := exec.Command(os.Args[0], arguments...)
			cmd.Env = append(os.Environ(), "XRAY_BUILDER_CLI_TEST=1")
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			err := cmd.Run()
			var exitErr *exec.ExitError
			if !errors.As(err, &exitErr) || exitErr.ExitCode() != tc.code {
				t.Fatalf("exit = %v, want %d; stderr: %s", err, tc.code, stderr.String())
			}
			if !strings.Contains(stderr.String(), tc.text) {
				t.Errorf("stderr = %q, want %q", stderr.String(), tc.text)
			}
			if stdout.Len() != 0 {
				t.Errorf("unexpected stdout: %q", stdout.String())
			}
			if strings.Contains(stderr.String(), "panic:") {
				t.Errorf("unexpected panic: %s", stderr.String())
			}
			got, err := os.ReadFile(configPath)
			if err != nil || !bytes.Equal(got, config) {
				t.Fatalf("config changed after failed command: %v", err)
			}
		})
	}
}
