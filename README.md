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

Run the checks with:

```sh
go test ./...
go vet ./...
```
