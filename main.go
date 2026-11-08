package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"xray-builder/models"
	"xray-builder/qr"

	bashexecutor "xray-builder/domain/commands/bash"
	clientservice "xray-builder/domain/services/clients"
	"xray-builder/domain/services/osservice"
	linuxService "xray-builder/domain/services/osservice/linux"
	serverservice "xray-builder/domain/services/server"

	"github.com/alexflint/go-arg"
	"github.com/samber/lo"
)

func sudoMaybeRequired() {
	fmt.Fprintln(os.Stderr, "You may need to run this command with sudo.")
}

func failUsage(parser *arg.Parser, err error) {
	parser.WriteUsage(os.Stderr)
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(2)
}

func main() {
	var args models.Args
	argParser, err := arg.NewParser(arg.Config{}, &args)
	if err != nil {
		log.Fatalln(err)
	}
	if err = argParser.Parse(os.Args[1:]); err == arg.ErrHelp {
		argParser.WriteHelp(os.Stdout)
		return
	} else if err != nil {
		failUsage(argParser, err)
	}
	if err := args.Validate(); err != nil {
		failUsage(argParser, err)
	}

	cmdExecutor := bashexecutor.New(args.Verbose)
	osService := linuxService.New(args.XrayConfigPath, args.XrayKeypairPath, cmdExecutor)
	isSuperUser, err := osService.IsSuperUser()
	if err != nil {
		log.Fatalln(err)
	}

	if args.Setup != nil {
		if !isSuperUser && !args.DryRun {
			sudoMaybeRequired()
		}

		Setup(osService, args.Setup, &args.MutationArgs)
		return
	}

	if args.UserList != nil {
		ListClients(osService, args.UserList)
		return
	}

	if lo.IsNotEmpty(args.User) {
		if args.Add != nil {
			if !isSuperUser && !args.DryRun {
				sudoMaybeRequired()
			}

			AddClient(osService, args.User, &args.MutationArgs)
			return
		}

		if args.Remove != nil {
			if !isSuperUser && !args.DryRun {
				sudoMaybeRequired()
			}

			RemoveClient(osService, args.User, &args.MutationArgs)
			return
		}

		if args.Disable != nil {
			if !isSuperUser && !args.DryRun {
				sudoMaybeRequired()
			}

			ToggleClientEnabled(osService, args.User, false, &args.MutationArgs)
			return
		}

		if args.Enable != nil {
			if !isSuperUser && !args.DryRun {
				sudoMaybeRequired()
			}

			ToggleClientEnabled(osService, args.User, true, &args.MutationArgs)
			return
		}

		if args.Share != nil {
			Share(osService, args.User, args.Share)
			return
		}
	}

	argParser.WriteHelp(log.Writer())
}

func Share(osService *linuxService.LinuxOsService, comment string, args *models.ShareArgs) {
	serverService := serverservice.New()
	serverConfig, err := serverService.ReadConfig(osService.XrayConfigPath)

	if err != nil {
		log.Fatalln(err)
	}

	client := serverService.GetUser(serverConfig, comment)
	if client == nil {
		log.Fatalln("user not found")
		return
	}

	keyPair, err := serverService.ReadKeyPair(osService.XrayKeypairPath)
	if err != nil {
		log.Fatalln(err)
	}

	clientService := clientservice.New(osService)
	clientConfig, err := clientService.CreateClientConfig(serverConfig.ServerName(), client, keyPair, &serverConfig.FirstInbound().StreamSettings)
	if err != nil {
		log.Fatalln(err)
	}

	switch args.Format {
	case "link", "qr":
		name, err := serverService.ReadName()
		if err != nil {
			log.Fatalln("read share name:", err)
			return
		}
		link := clientConfig.FirstOutbound().ShareLink(client.ShortId)
		if name != "" {
			link.Fragment = name
		}
		if args.Format == "qr" {
			qr.RenderString(link.String(), false)
		} else {
			fmt.Println(link.String())
		}

	case "json":
		result, _ := json.MarshalIndent(clientConfig, "", "    ")
		fmt.Println(string(result))
	}
}

func Setup(osService *linuxService.LinuxOsService, args *models.SetupArgs, options *models.MutationArgs) {
	serverService := serverservice.New()

	cfg, err := serverService.ReadConfig("")
	if err != nil {
		log.Fatalln(err)
	}
	if options.DryRun {
		fmt.Printf("would configure Xray for destination %q\n", args.Destination)
		return
	}

	keyPair, err := osService.GenerateKeyPair()
	if err != nil {
		log.Fatalln("GenerateKeyPair", err)
	}

	err = osService.SaveKeyPair(keyPair)
	if err != nil {
		log.Fatalln("SaveKeyPair", err)
	}

	serverService.SetupServer(cfg, keyPair, args.Destination)
	if err = applyServerConfig(osService, cfg, options); err != nil {
		log.Fatalln(err)
	}
}

func AddClient(osService *linuxService.LinuxOsService, comment string, options *models.MutationArgs) {
	clientService := clientservice.New(osService)
	serverService := serverservice.New()
	serverConfig, err := serverService.ReadConfig(osService.XrayConfigPath)
	if err != nil {
		log.Fatalln(err)
	}

	if serverService.GetUser(serverConfig, comment) != nil {
		log.Fatalln("user already exists")
	}
	if options.DryRun {
		fmt.Printf("would add user %q\n", comment)
		return
	}

	network := serverConfig.FirstInbound().StreamSettings.Network
	client, err := clientService.CreateClient(comment, network)
	if err != nil {
		log.Fatalln(err)
	}
	serverService.AppendClient(serverConfig, client)

	err = applyServerConfig(osService, serverConfig, options)
	if err != nil {
		log.Fatalln(err)
	}

	if options.NoRestart {
		fmt.Println("config saved")
	} else {
		fmt.Println("xray restarted")
	}
}

func ListClients(osService *linuxService.LinuxOsService, args *models.UsersArgs) {
	serverService := serverservice.New()
	cfg, err := serverService.ReadConfig(osService.XrayConfigPath)
	if err != nil {
		log.Fatalln(err)
	}

	if err := writeClients(os.Stdout, cfg, args.JSON); err != nil {
		log.Fatalln(err)
	}
}

func writeClients(output io.Writer, cfg *models.ServerConfig, asJSON bool) error {
	if len(cfg.Inbounds) == 0 {
		return fmt.Errorf("server config has no inbounds")
	}
	users := cfg.FirstInbound().Settings.Clients
	if asJSON {
		type userInfo struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Enabled bool   `json:"enabled"`
		}
		result := make([]userInfo, 0, len(users))
		for _, user := range users {
			result = append(result, userInfo{
				ID:      user.Id,
				Name:    user.Comment,
				Enabled: lo.Contains(cfg.FirstInbound().StreamSettings.RealitySettings.ShortIds, user.ShortId),
			})
		}
		return json.NewEncoder(output).Encode(result)
	}

	comments := lo.Map(users, func(u models.Client, _ int) string { return u.Comment })
	_, err := fmt.Fprintln(output, strings.Join(comments, "\n"))
	return err
}

func ToggleClientEnabled(osService *linuxService.LinuxOsService, comment string, isEnabled bool, options *models.MutationArgs) {
	serverService := serverservice.New()
	cfg, err := serverService.ReadConfig(osService.XrayConfigPath)
	if err != nil {
		log.Fatalln(err)
	}

	user := serverService.ToggleUserEnabled(cfg, comment, isEnabled)
	if user == nil {
		log.Fatalln("user not found")
		return
	}
	if options.DryRun {
		action := "disable"
		if isEnabled {
			action = "enable"
		}
		fmt.Printf("would %s user %q\n", action, comment)
		return
	}

	err = applyServerConfig(osService, cfg, options)
	if err != nil {
		log.Fatalln(err)
	}
}

func RemoveClient(osService *linuxService.LinuxOsService, comment string, options *models.MutationArgs) {
	serverService := serverservice.New()
	cfg, err := serverService.ReadConfig(osService.XrayConfigPath)
	if err != nil {
		log.Fatalln(err)
	}

	user := serverService.RemoveUser(cfg, comment)
	if user == nil {
		log.Fatalln("user not found")
		return
	}
	if options.DryRun {
		fmt.Printf("would remove user %q\n", comment)
		return
	}

	err = applyServerConfig(osService, cfg, options)
	if err != nil {
		log.Fatalln(err)
	}

	userJson, _ := json.MarshalIndent(user, "", "    ")
	fmt.Println(string(userJson))
}

func applyServerConfig(service osservice.OsService, cfg *models.ServerConfig, options *models.MutationArgs) error {
	if options.DryRun {
		return nil
	}
	if err := service.WriteServerConfig(cfg); err != nil {
		return fmt.Errorf("write server config: %w", err)
	}
	if !options.NoRestart {
		if err := service.RestartXray(); err != nil {
			return fmt.Errorf("server config saved, but Xray restart failed: %w", err)
		}
	}
	return nil
}
