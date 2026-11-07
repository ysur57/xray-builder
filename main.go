package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"xray-builder/models"
	"xray-builder/qr"

	bashexecutor "xray-builder/domain/commands/bash"
	clientservice "xray-builder/domain/services/clients"
	linuxService "xray-builder/domain/services/osservice/linux"
	serverservice "xray-builder/domain/services/server"

	"github.com/alexflint/go-arg"
	"github.com/samber/lo"
)

func sudoMaybeRequired() {
	fmt.Println("You may need to run this command with sudo.")
}

func main() {
	var args models.Args
	argParser := arg.MustParse(&args)

	cmdExecutor := bashexecutor.New(args.Verbose)
	osService := linuxService.New(args.XrayConfigPath, args.XrayKeypairPath, cmdExecutor)
	isSuperUser, err := osService.IsSuperUser()
	if err != nil {
		panic(err)
	}

	if args.Setup != nil {
		if !isSuperUser {
			sudoMaybeRequired()
		}

		Setup(osService, args.Setup)
		return
	}

	if args.UserList != nil {
		ListClients(osService)
		return
	}

	if lo.IsNotEmpty(args.User) {
		if args.Add != nil {
			if !isSuperUser {
				sudoMaybeRequired()
			}

			AddClient(osService, args.User)
			return
		}

		if args.Remove != nil {
			if !isSuperUser {
				sudoMaybeRequired()
			}

			RemoveClient(osService, args.User)
			return
		}

		if args.Disable != nil {
			if !isSuperUser {
				sudoMaybeRequired()
			}

			ToggleClientEnabled(osService, args.User, false)
			return
		}

		if args.Enable != nil {
			if !isSuperUser {
				sudoMaybeRequired()
			}

			ToggleClientEnabled(osService, args.User, true)
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
		panic(err)
	}

	client := serverService.GetUser(serverConfig, comment)
	if client == nil {
		log.Fatalln("user not found")
		return
	}

	keyPair, err := serverService.ReadKeyPair(osService.XrayKeypairPath)
	if err != nil {
		panic(err)
	}

	clientService := clientservice.New(osService)
	clientConfig, err := clientService.CreateClientConfig(serverConfig.ServerName(), client, keyPair, &serverConfig.FirstInbound().StreamSettings)
	if err != nil {
		panic(err)
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

func Setup(osService *linuxService.LinuxOsService, args *models.SetupArgs) {
	serverService := serverservice.New()

	cfg, err := serverService.ReadConfig("")
	if err != nil {
		panic(err)
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
	osService.WriteServerConfig(cfg)
	if err = osService.RestartXray(); err != nil {
		log.Fatalln("Xray restart failed. Please restart it manually.")
	}
}

func AddClient(osService *linuxService.LinuxOsService, comment string) {
	clientService := clientservice.New(osService)
	serverService := serverservice.New()
	serverConfig, err := serverService.ReadConfig(osService.XrayConfigPath)
	if err != nil {
		panic(err)
	}

	if serverService.GetUser(serverConfig, comment) != nil {
		log.Fatalln("user already exists")
	}

	network := serverConfig.FirstInbound().StreamSettings.Network
	client, err := clientService.CreateClient(comment, network)
	if err != nil {
		panic(err)
	}
	serverService.AppendClient(serverConfig, client)

	err = osService.WriteServerConfig(serverConfig)
	if err != nil {
		log.Fatalln(err)
	}

	if err = osService.RestartXray(); err != nil {
		log.Fatalln("Xray restart failed. Please restart it manually.")
	}

	fmt.Println("xray restarted")
}

func ListClients(osService *linuxService.LinuxOsService) {
	serverService := serverservice.New()
	cfg, err := serverService.ReadConfig(osService.XrayConfigPath)
	if err != nil {
		log.Fatalln(err)
	}

	users := serverService.GetUsers(cfg)

	comments := lo.Map(*users, func(u models.Client, _ int) string {
		return u.Comment
	})

	fmt.Println(strings.Join(comments, "\n"))
}

func ToggleClientEnabled(osService *linuxService.LinuxOsService, comment string, isEnabled bool) {
	serverService := serverservice.New()
	cfg, err := serverService.ReadConfig(osService.XrayConfigPath)
	if err != nil {
		panic(err)
	}

	user := serverService.ToggleUserEnabled(cfg, comment, isEnabled)
	if user == nil {
		fmt.Println("user not found")
		return
	}

	err = osService.WriteServerConfig(cfg)
	if err != nil {
		panic(err)
	}

	err = osService.RestartXray()
	if err != nil {
		log.Fatalln("Xray restart failed. Please restart it manually.")
	}
}

func RemoveClient(osService *linuxService.LinuxOsService, comment string) {
	serverService := serverservice.New()
	cfg, err := serverService.ReadConfig(osService.XrayConfigPath)
	if err != nil {
		panic(err)
	}

	user := serverService.RemoveUser(cfg, comment)
	if user == nil {
		fmt.Println("user not found")
		return
	}

	err = osService.WriteServerConfig(cfg)
	if err != nil {
		panic(err)
	}

	userJson, _ := json.MarshalIndent(user, "", "    ")
	fmt.Println(string(userJson))

	err = osService.RestartXray()
	if err != nil {
		log.Fatalln("Xray restart failed. Please restart it manually.")
	}
}
