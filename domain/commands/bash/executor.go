package commands

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"xray-builder/models"
)

const ShellPath = "/usr/bin/sh"

type BashCmdExecutor struct {
	verbose bool
}

func (b *BashCmdExecutor) GenerateKeyPair() (*models.KeyPair, error) {
	out, stderr, err := b.Shell("xray x25519")
	if err != nil {
		return nil, errors.New(stderr)
	}
	keyPair, err := fromStdOut(out)
	if err != nil {
		return nil, err
	}
	return keyPair, nil
}

func (b *BashCmdExecutor) GenerateShortId() (*string, error) {
	out, _, err := b.Shell("openssl rand -hex 8")
	if err != nil {
		return nil, err
	}
	result := strings.TrimSuffix(out, "\n")
	return &result, nil
}

func (b *BashCmdExecutor) GetServerAddr() (*string, error) {
	out, _, err := b.Shell("hostname -I")
	if err != nil {
		return nil, err
	}
	result := strings.Split(out, " ")[0]
	return &result, nil
}

func (b *BashCmdExecutor) RestartXray() error {
	_, _, err := b.Shell("systemctl restart xray")
	return err
}

func (b *BashCmdExecutor) Shell(command string) (string, string, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command(ShellPath, "-c", command)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	if b.verbose {
		fmt.Fprintf(os.Stderr, "command %q: %v\n", command, err)
	}

	return stdout.String(), stderr.String(), err
}

func New(verbose bool) *BashCmdExecutor {
	return &BashCmdExecutor{verbose}
}
