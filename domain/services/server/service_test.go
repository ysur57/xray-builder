package impl

import (
	"reflect"
	"testing"
	"xray-builder/internal"
	"xray-builder/models"
)

func TestRemoveUser(t *testing.T) {
	for _, selector := range []string{"alice-id", "alice", "missing"} {
		t.Run(selector, func(t *testing.T) {
			alice := models.Client{Id: "alice-id", Comment: "alice", ShortId: "alice-short"}
			bob := models.Client{Id: "bob-id", Comment: "bob", ShortId: "bob-short"}
			cfg := config()
			cfg.FirstInbound().Settings.Clients = []models.Client{alice, bob}
			cfg.FirstInbound().StreamSettings.RealitySettings.ShortIds = []string{alice.ShortId, bob.ShortId}

			removed := service().RemoveUser(cfg, selector)
			wantClients := []models.Client{bob}
			wantShortIds := []string{bob.ShortId}
			if selector == "missing" {
				if removed != nil {
					t.Fatalf("removed unexpected user: %+v", removed)
				}
				wantClients = []models.Client{alice, bob}
				wantShortIds = []string{alice.ShortId, bob.ShortId}
			} else if removed == nil || removed.Id != alice.Id {
				t.Fatalf("expected Alice to be removed, got %+v", removed)
			}
			if got := cfg.FirstInbound().Settings.Clients; !reflect.DeepEqual(got, wantClients) {
				t.Errorf("clients = %+v, want %+v", got, wantClients)
			}
			if got := cfg.FirstInbound().StreamSettings.RealitySettings.ShortIds; !reflect.DeepEqual(got, wantShortIds) {
				t.Errorf("short IDs = %v, want %v", got, wantShortIds)
			}
		})
	}
}

func config() *models.ServerConfig {
	result := models.ServerConfig{}
	internal.ReadJson(internal.RootDir()+"/configs/server.template.json", &result)
	return &result
}

func service() *ServerService {
	return &ServerService{}
}

func TestAppend(t *testing.T) {
	const ClientsCount = 5

	cfg := config()
	svc := service()

	for ind := 0; ind < ClientsCount; ind++ {
		client := models.Client{
			Id:      "asdfadsfd",
			Flow:    "asdfasdfdsa",
			ShortId: "jopajopajopa",
			Comment: "some-comment",
		}

		svc.AppendClient(cfg, &client)
	}
	if len(cfg.Inbounds[0].Settings.Clients) != ClientsCount {
		t.Errorf("Expected %v clients found %v", ClientsCount, len(cfg.Inbounds[0].Settings.Clients))
	}
}

func TestSetPrivateKey(t *testing.T) {
	cfg := config()
	svc := service()
	keyPair := models.KeyPair{
		Pub:     "pub",
		Private: "private",
	}
	svc.SetPrivateKey(cfg, &keyPair)
	if cfg.Inbounds[0].StreamSettings.RealitySettings.PrivateKey != keyPair.Private {
		t.Error("Expected private key to be set, got invalid value instead")
	}
}

func TestSetDestinationAddress(t *testing.T) {
	cfg := config()
	addr := "https://rkn.gov.ru/"
	svc := service()
	svc.SetDestinationAddress(cfg, addr)
	if cfg.Inbounds[0].StreamSettings.RealitySettings.Dest != addr+":443" {
		t.Error("Expected dest to be set, got value invalid instead")
	}
}
