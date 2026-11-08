package models

import (
	"fmt"
	"strings"
)

type Args struct {
	MutationArgs

	XrayConfigPath  string `arg:"--config" default:"/usr/local/etc/xray/config.json"`
	XrayKeypairPath string `arg:"--keypair" default:"/usr/local/etc/xray/keypair.json"`
	Verbose         bool   `arg:"-v,--verbose"`

	UserList *UsersArgs `arg:"subcommand:users" help:"list users"`
	Setup    *SetupArgs `arg:"subcommand:setup" help:"install xray and generate configs"`

	User    string     `arg:"-u,--user" help:"Id or comment of the user."`
	Add     *struct{}  `arg:"subcommand:add" help:"add user to xray config"`
	Remove  *struct{}  `arg:"subcommand:remove"`
	Enable  *struct{}  `arg:"subcommand:enable"`
	Disable *struct{}  `arg:"subcommand:disable"`
	Share   *ShareArgs `arg:"subcommand:share"`
	MCP     *struct{}  `arg:"subcommand:mcp" help:"serve MCP tools over stdin and stdout"`
}

type UsersArgs struct {
	JSON bool `arg:"--json" help:"output user IDs, names, and enabled state as JSON"`
}

type ShareArgs struct {
	Format string `arg:"-f,--format" default:"link" help:"qr | json | link"`
}

type SetupArgs struct {
	Destination string `arg:"-d,--destination,required" placeholder:"rkn.gov.ru/" help:"destination and serverName in xray server config"`
}

func (a *Args) Validate() error {
	if a.Setup == nil && a.UserList == nil && a.Add == nil && a.Remove == nil && a.Enable == nil && a.Disable == nil && a.Share == nil && a.MCP == nil {
		return fmt.Errorf("a subcommand is required")
	}
	if (a.Add != nil || a.Remove != nil || a.Enable != nil || a.Disable != nil || a.Share != nil) && strings.TrimSpace(a.User) == "" {
		return fmt.Errorf("--user is required for this command")
	}
	if a.Share != nil && a.Share.Format != "link" && a.Share.Format != "json" && a.Share.Format != "qr" {
		return fmt.Errorf("--format must be link, json, or qr")
	}
	if (a.DryRun || a.NoRestart) && (a.UserList != nil || a.Share != nil || a.MCP != nil) {
		return fmt.Errorf("--dry-run and --no-restart require a modifying command")
	}
	return nil
}

func (Args) Description() string {
	return "Configure xray-core and manage users.\nhttps://github.com/ysur67/xray-builder\n"
}

type MutationArgs struct {
	DryRun    bool `arg:"--dry-run" help:"preview a change without writing files or restarting Xray"`
	NoRestart bool `arg:"--no-restart" help:"save changes without restarting Xray"`
}
