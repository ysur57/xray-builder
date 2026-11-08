package impl

import (
	"os"
	"path/filepath"
	"strings"
	"xray-builder/internal"
	"xray-builder/models"

	"github.com/samber/lo"
)

type ServerService struct {
	configsDirectory string
}

func (s *ServerService) ReadName() (string, error) {
	name, err := os.ReadFile(filepath.Join(s.configsDirectory, "name"))
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(name)), nil
}

func (s *ServerService) ReadConfig(path string) (*models.ServerConfig, error) {
	if path == "" {
		path = s.configsDirectory + "/server.template.json"
	}
	config := models.ServerConfig{}
	err := internal.ReadJson(path, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func (s *ServerService) AppendClient(
	cfg *models.ServerConfig,
	client *models.Client,
) {
	inbound := cfg.FirstInbound()
	inbound.Settings.Clients = append(inbound.Settings.Clients, *client)
	inbound.StreamSettings.RealitySettings.ShortIds = append(inbound.StreamSettings.RealitySettings.ShortIds, client.ShortId)
}

func (s *ServerService) SetPrivateKey(
	cfg *models.ServerConfig,
	keyPair *models.KeyPair,
) {
	cfg.FirstInbound().StreamSettings.RealitySettings.PrivateKey = keyPair.Private
}

func (s *ServerService) SetDestinationAddress(cfg *models.ServerConfig, addr string) {
	first := cfg.FirstInbound()
	first.StreamSettings.RealitySettings.Dest = addr + ":443"
	first.StreamSettings.RealitySettings.ServerNames = []string{addr}
}

func (s *ServerService) SetupServer(
	cfg *models.ServerConfig,
	keyPair *models.KeyPair,
	destination string,
) {
	s.SetPrivateKey(cfg, keyPair)
	s.SetDestinationAddress(cfg, destination)
}

func (s *ServerService) GetUsers(cfg *models.ServerConfig) *[]models.Client {
	return &cfg.FirstInbound().Settings.Clients
}

func (s *ServerService) GetUser(cfg *models.ServerConfig, userIdOrComment string) *models.Client {
	inbound := cfg.FirstInbound()

	client, isFound := lo.Find(inbound.Settings.Clients, func(user models.Client) bool {
		return user.Comment == userIdOrComment || user.Id == userIdOrComment
	})

	if !isFound {
		return nil
	}

	return &client
}

func (s *ServerService) RemoveUser(cfg *models.ServerConfig, userIdOrComment string) *models.Client {
	inbound := cfg.FirstInbound()
	user := s.GetUser(cfg, userIdOrComment)
	if user == nil {
		return nil
	}

	s.ToggleUserEnabled(cfg, user.Id, false)
	inbound.Settings.Clients = lo.Filter(inbound.Settings.Clients, func(c models.Client, _ int) bool {
		return c.Id != user.Id
	})

	return user
}

func (s *ServerService) ToggleUserEnabled(cfg *models.ServerConfig, userComment string, isEnabled bool) *models.Client {
	inbound := cfg.FirstInbound()

	shortIds := inbound.StreamSettings.RealitySettings.ShortIds
	user := s.GetUser(cfg, userComment)
	if user == nil {
		return nil
	}

	if isEnabled {
		if lo.Contains(shortIds, user.ShortId) {
			return user
		}

		inbound.StreamSettings.RealitySettings.ShortIds = append(shortIds, user.ShortId)
	} else {
		inbound.StreamSettings.RealitySettings.ShortIds = internal.Remove(shortIds, user.ShortId)
	}

	return user
}

func (s *ServerService) ReadKeyPair(path string) (*models.KeyPair, error) {
	var result models.KeyPair
	if err := internal.ReadJson(path, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func New() *ServerService {
	return &ServerService{
		configsDirectory: internal.ResolveConfigPath(),
	}
}
