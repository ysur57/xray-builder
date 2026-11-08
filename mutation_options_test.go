package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"xray-builder/domain/services/osservice"
	"xray-builder/models"
)

type configServiceStub struct {
	osservice.OsService
	writes     int
	restarts   int
	writeErr   error
	restartErr error
}

func (s *configServiceStub) WriteServerConfig(*models.ServerConfig) error {
	s.writes++
	return s.writeErr
}

func (s *configServiceStub) RestartXray() error {
	s.restarts++
	return s.restartErr
}

func TestApplyServerConfig(t *testing.T) {
	failure := errors.New("operation failed")
	cases := []struct {
		name       string
		options    models.MutationArgs
		writeErr   error
		restartErr error
		writes     int
		restarts   int
		wantErr    error
	}{
		{name: "default", writes: 1, restarts: 1},
		{name: "dry run", options: models.MutationArgs{DryRun: true}},
		{name: "no restart", options: models.MutationArgs{NoRestart: true}, writes: 1},
		{name: "both flags", options: models.MutationArgs{DryRun: true, NoRestart: true}},
		{name: "write failure", writeErr: failure, writes: 1, wantErr: failure},
		{name: "restart failure", restartErr: failure, writes: 1, restarts: 1, wantErr: failure},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			service := &configServiceStub{writeErr: tc.writeErr, restartErr: tc.restartErr}
			err := applyServerConfig(service, &models.ServerConfig{}, &tc.options)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("error = %v, want %v", err, tc.wantErr)
			}
			if service.writes != tc.writes || service.restarts != tc.restarts {
				t.Fatalf("writes/restarts = %d/%d, want %d/%d", service.writes, service.restarts, tc.writes, tc.restarts)
			}
		})
	}
}

func TestMutationPreview(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "server.json")
	config := []byte(`{"inbounds":[{"settings":{"clients":[{"id":"alice-id","email":"alice","shortId":"alice-short"}]},"streamSettings":{"realitySettings":{"shortIds":["alice-short"]}}}]}`)
	if err := os.WriteFile(configPath, config, 0600); err != nil {
		t.Fatal(err)
	}
	for _, action := range []string{"add", "remove", "enable", "disable"} {
		t.Run(action, func(t *testing.T) {
			user := "alice"
			if action == "add" {
				user = "bob"
			}
			cmd := exec.Command(os.Args[0], "-test.run=^TestCLIErrorProcess$", "--", "--config", configPath, "--user", user, "--dry-run", action)
			cmd.Env = append(os.Environ(), "XRAY_BUILDER_CLI_TEST=1")
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			if err := cmd.Run(); err != nil {
				t.Fatalf("preview failed: %v; stderr: %s", err, stderr.String())
			}
			if !strings.Contains(stdout.String(), "would "+action+" user") || stderr.Len() != 0 {
				t.Fatalf("unexpected output: stdout=%q stderr=%q", stdout.String(), stderr.String())
			}
			got, err := os.ReadFile(configPath)
			if err != nil || !bytes.Equal(got, config) {
				t.Fatalf("preview changed config: %v", err)
			}
		})
	}
}
