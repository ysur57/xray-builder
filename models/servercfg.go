package models

import (
	"encoding/json"

	"github.com/google/uuid"
)

type LogObject struct {
	LogLevel string `json:"loglevel"`
}

type InboundSettings struct {
	Clients    []Client `json:"clients"`
	Decryption string   `json:"decryption"`
}

type Client struct {
	Id   string `json:"id"`
	Flow string `json:"flow,omitempty"`

	// Used to identify client in the `realitySettings.shortIds`.
	ShortId string `json:"shortId"`

	// Comment for the user.
	// Field name `email` is for compatibility with 3x-ui.
	Comment string `json:"email"`
}

type RealitySettingsObject struct {
	Show         bool     `json:"show"`
	Dest         string   `json:"dest"`
	Xver         int      `json:"xver"`
	ServerNames  []string `json:"serverNames"`
	PrivateKey   string   `json:"privateKey"`
	MinClientVer string   `json:"minClientVer"`
	MaxClientVer string   `json:"maxClientVer"`
	MaxTimeDiff  int      `json:"maxTimeDiff"`
	ShortIds     []string `json:"shortIds"`
}

type XhttpSettingsObject struct {
	Path string `json:"path"`
}

type StreamSettingsObject struct {
	Network         string                `json:"network"`
	Security        string                `json:"security"`
	RealitySettings RealitySettingsObject `json:"realitySettings"`
	XhttpSettings   *XhttpSettingsObject  `json:"xhttpSettings,omitempty"`
}

type SniffingObject struct {
	Enabled      bool     `json:"enabled"`
	DestOverride []string `json:"destOverride"`
}

type Inbound struct {
	Listen         string               `json:"listen"`
	Port           int                  `json:"port"`
	Protocol       string               `json:"protocol"`
	Settings       InboundSettings      `json:"settings"`
	StreamSettings StreamSettingsObject `json:"streamSettings"`
	Sniffing       SniffingObject       `json:"sniffing"`
}

type RoutingObject struct {
	DomainStrategy string `json:"domainStrategy"`
	// Kept as raw JSON so rule fields we don't model (protocol, port,
	// network, inboundTag, ...) aren't dropped when the config is rewritten.
	Rules []json.RawMessage `json:"rules"`
}

type Outbound struct {
	Protocol string `json:"protocol"`
	Tag      string `json:"tag"`
}

type PolicyObject struct {
	System struct {
		StatsInboundUplink    bool `json:"statsInboundUplink,omitempty"`
		StatsInboundDownlink  bool `json:"statsInboundDownlink,omitempty"`
		StatsOutboundUplink   bool `json:"statsOutboundUplink,omitempty"`
		StatsOutboundDownlink bool `json:"statsOutboundDownlink,omitempty"`
	} `json:"system,omitempty"`
	Levels struct {
		Num0 struct {
			StatsUserUplink   bool `json:"statsUserUplink,omitempty"`
			StatsUserDownlink bool `json:"statsUserDownlink,omitempty"`
			Handshake         int  `json:"handshake"`
			ConnIdle          int  `json:"connIdle"`
		} `json:"0"`
	} `json:"levels"`
}

type ServerConfig struct {
	Log       LogObject     `json:"log"`
	Routing   RoutingObject `json:"routing"`
	Outbounds []Outbound    `json:"outbounds"`
	Policy    PolicyObject  `json:"policy"`
	Inbounds  []Inbound     `json:"inbounds"`
	Stats     struct{}      `json:"stats"`
	Api       ApiConfig     `json:"api,omitempty"`
}

type ApiConfig struct {
	Tag      string   `json:"tag,omitempty"`
	Listen   string   `json:"listen,omitempty"`
	Services []string `json:"services,omitempty"`
}

func (s *ServerConfig) FirstInbound() *Inbound {
	return &s.Inbounds[0]
}

func (s *ServerConfig) ServerName() string {
	return s.FirstInbound().StreamSettings.RealitySettings.ServerNames[0]
}

func NewClient(shortId string, comment string, network string) *Client {
	flow := "xtls-rprx-vision"
	if network == "xhttp" {
		flow = ""
	}
	return &Client{
		Comment: comment,
		ShortId: shortId,
		Id:      uuid.New().String(),
		Flow:    flow,
	}
}
