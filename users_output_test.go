package main

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
	"xray-builder/models"
)

func TestWriteClients(t *testing.T) {
	cfg := &models.ServerConfig{Inbounds: []models.Inbound{{
		Settings: models.InboundSettings{Clients: []models.Client{
			{Id: "alice-id", Comment: "alice", ShortId: "alice-short"},
			{Id: "bob-id", Comment: "bob", ShortId: "bob-short"},
		}},
		StreamSettings: models.StreamSettingsObject{
			RealitySettings: models.RealitySettingsObject{ShortIds: []string{"alice-short"}},
		},
	}}}
	var output bytes.Buffer
	if err := writeClients(&output, cfg, false); err != nil {
		t.Fatal(err)
	}
	if got := output.String(); got != "alice\nbob\n" {
		t.Fatalf("text output = %q", got)
	}
	output.Reset()
	if err := writeClients(&output, cfg, true); err != nil {
		t.Fatal(err)
	}
	var got []map[string]any
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	want := []map[string]any{
		{"id": "alice-id", "name": "alice", "enabled": true},
		{"id": "bob-id", "name": "bob", "enabled": false},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("JSON output = %v, want %v", got, want)
	}
	output.Reset()
	cfg.FirstInbound().Settings.Clients = nil
	if err := writeClients(&output, cfg, true); err != nil {
		t.Fatal(err)
	}
	if got := output.String(); got != "[]\n" {
		t.Fatalf("empty JSON output = %q", got)
	}
	if err := writeClients(&output, &models.ServerConfig{}, true); err == nil {
		t.Fatal("expected an error for a config without inbounds")
	}
}
