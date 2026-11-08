package models

type Args struct {
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

func (Args) Description() string {
	return "Configure xray-core and manage users.\nhttps://github.com/ysur67/xray-builder\n"
}
