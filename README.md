# xray-builder

Configure an existing Xray installation and manage VLESS users on Linux. Requires Xray, OpenSSL, and systemd.

Build and install the binary and templates from this checkout:

```sh
sudo bash install.sh
```

The server configuration defaults to `/usr/local/etc/xray/config.json`, keys to `/usr/local/etc/xray/keypair.json`, and templates to `/usr/local/etc/xray-builder`. Use `--config` and `--keypair` to override the server file paths.

```sh
xray-builder --help
xray-builder users
sudo xray-builder --user alice add
sudo xray-builder --user alice disable
sudo xray-builder --user alice enable
xray-builder --user alice share --format link
xray-builder --user alice share --format json
sudo xray-builder --user alice remove
```

User selectors accept a UUID or comment. Adding a user treats `--user` as the new user's comment.

Preview a mutation without writing files, generating credentials, or restarting Xray:

```sh
xray-builder --dry-run --user alice add
xray-builder --dry-run --user alice remove
xray-builder --dry-run setup --destination example.com
```

Save changes without restarting the service, then restart once after a batch:

```sh
sudo xray-builder --no-restart --user alice disable
sudo xray-builder --no-restart --user bob disable
sudo systemctl restart xray
```

`--dry-run` and `--no-restart` apply to `setup`, `add`, `remove`, `enable`, and `disable`. Mutations restart Xray by default. Setup generates new keys and replaces the server configuration; it requires Xray to be installed already. Supply a hostname such as `example.com` as the destination.

Commands exit with `0` on success, `1` on operational failure, and `2` on invalid arguments. Diagnostics go to stderr.

## MCP

Start the MCP server over stdio with `xray-builder mcp`. Configure an MCP client with:

```json
{
  "mcpServers": {
    "xray-builder": {
      "command": "/usr/local/bin/xray-builder",
      "args": ["mcp"]
    }
  }
}
```

To target another config, set `args` to `["--config", "/path/to/config.json", "--keypair", "/path/to/keypair.json", "mcp"]`.

Tools: `list_users`, `share_user`, `add_user`, `remove_user`, `enable_user`, `disable_user`, and `setup_server`. Each tool exposes an input schema and structured results. `share_user` returns a link containing connection credentials. `setup_server` replaces configuration and keys.

All mutation tools accept `dry_run` and `no_restart`, both defaulting to `false`. For example, call `remove_user` with `{"user":"alice","dry_run":true}` to preview removal. These options belong to tool calls; do not pass the CLI mutation flags when starting the MCP server.

The server runs commands with its own account's file and service permissions and serializes operations within the session. Command failures return tool errors while keeping the MCP session alive.

## Development

Run the checks with:

```sh
go test ./...
go vet ./...
```
