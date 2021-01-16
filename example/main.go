package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/awgh/ratnet/transports/udp"

	"github.com/AlexsJones/cli/cli"
	"github.com/AlexsJones/cli/command"
	"github.com/awgh/ratnet/nodes/ram"
	"github.com/awgh/ratnet/policy"
)

func main() {
	checkIfErr := func(err error) bool {
		if err != nil {
			fmt.Println(err)
			return false
		}
		return true
	}

	checkIfArgsAreOK := func(min, max int, args []string) bool {
		if min >= 0 && len(args) < min {
			checkIfErr(errors.New("Not enough arguments"))
			return false
		}
		if max >= 0 && len(args) > max {
			checkIfErr(errors.New("Too many arguments"))
			return false
		}
		return true
	}

	// make a new ram node
	node := ram.New(nil, nil)

	// ... and route it's output to stdOut
	go func() {
		for {
			msg := <-node.Out()
			fmt.Println("[RX From", msg.Name, "]:", msg.Content.String())
		}
	}()

	// make a new command line interface
	cli := cli.NewCli()

	// start
	cli.AddCommand(command.Command{
		Name: "Start",
		Help: "Start Ratnet",
		Func: func(args []string) {
			checkIfErr(node.Start())
		},
	})

	// stop
	cli.AddCommand(command.Command{
		Name: "Stop",
		Help: "Stop Ratnet",
		Func: func(args []string) {
			node.Stop()
		},
	})

	// display content key
	cli.AddCommand(command.Command{
		Name: "CID",
		Help: "Display content key",
		Func: func(args []string) {
			key, err := node.CID()
			if checkIfErr(err) {
				fmt.Println(key.ToB64())
			}
		},
	})

	// display routing key
	cli.AddCommand(command.Command{
		Name: "ID",
		Help: "Display routing key",
		Func: func(args []string) {
			key, err := node.ID()
			if checkIfErr(err) {
				fmt.Println(key.ToB64())
			}
		},
	})

	// add contact
	cli.AddCommand(command.Command{
		Name: "AddContact",
		Help: "Add a contact (name, key string)",
		Func: func(args []string) {
			if checkIfArgsAreOK(2, 2, args) {
				checkIfErr(node.AddContact(args[0], args[1]))
			}
		},
	})

	// delete contact
	cli.AddCommand(command.Command{
		Name: "DelContact",
		Help: "Delete a contact (name string)",
		Func: func(args []string) {
			if checkIfArgsAreOK(1, 1, args) {
				checkIfErr(node.DeleteContact(args[0]))
			}
		},
	})

	// display contacts
	cli.AddCommand(command.Command{
		Name: "ShowContacts",
		Help: "Display a list of the current node contacts (name, key string)",
		Func: func(args []string) {
			contacts, err := node.GetContacts()
			if checkIfErr(err) {
				for index, contact := range contacts {
					fmt.Printf("%d) %s\t\t%s\n", index, contact.Name, contact.Pubkey)
				}
			}
		},
	})

	// add channel
	cli.AddCommand(command.Command{
		Name: "AddChannel",
		Help: "Add a channel (name, key string)",
		Func: func(args []string) {
			if checkIfArgsAreOK(2, 2, args) {
				checkIfErr(node.AddChannel(args[0], args[1]))
			}
		},
	})

	// delete channel
	cli.AddCommand(command.Command{
		Name: "DelChannel",
		Help: "Delete a channel (name string)",
		Func: func(args []string) {
			if checkIfArgsAreOK(1, 1, args) {
				checkIfErr(node.DeleteContact(args[0]))
			}
		},
	})

	// display channels
	cli.AddCommand(command.Command{
		Name: "ShowChannels",
		Help: "Display a list of the current node channels (name, key string)",
		Func: func(args []string) {
			channels, err := node.GetChannels()
			if checkIfErr(err) {
				for index, channel := range channels {
					fmt.Printf("%d) %s\t\t%s\n", index, channel.Name, channel.Pubkey)
				}
			}
		},
	})

	// add profile
	cli.AddCommand(command.Command{
		Name: "AddProfile",
		Help: "Add a profile (Name string, Enabled bool)",
		Func: func(args []string) {
			if checkIfArgsAreOK(2, 2, args) {
				enabled := false
				if strings.Contains(strings.ToLower(args[0]), "true") {
					enabled = true
				}
				checkIfErr(node.AddProfile(args[0], enabled))
			}
		},
	})

	// delete profile
	cli.AddCommand(command.Command{
		Name: "DelProfile",
		Help: "Delete a profile (name string)",
		Func: func(args []string) {
			if checkIfArgsAreOK(1, 1, args) {
				checkIfErr(node.DeleteProfile(args[0]))
			}
		},
	})

	// display profiles
	cli.AddCommand(command.Command{
		Name: "ShowProfiles",
		Help: "Display a list of the current node profiles (name, key string)",
		Func: func(args []string) {
			profiles, err := node.GetProfiles()
			if checkIfErr(err) {
				enabled := "False"
				for index, profile := range profiles {
					if profile.Enabled {
						enabled = "True"
					}
					fmt.Printf("%d) %s\t\t%s\t%s\n", index, profile.Name, profile.Pubkey, enabled)
				}
			}
		},
	})

	// load profile
	cli.AddCommand(command.Command{
		Name: "LoadProfile",
		Help: "Load a profile (name string)",
		Func: func(args []string) {
			if checkIfArgsAreOK(0, 0, args) {
				key, err := node.LoadProfile(args[0])
				if checkIfErr(err) {
					fmt.Println("Key of loaded profile:", key.ToB64())
				}
			}
		},
	})

	// add peer
	cli.AddCommand(command.Command{
		Name: "AddPeer",
		Help: "Add a peer (name string, enabled bool, uri string)",
		Func: func(args []string) {
			if checkIfArgsAreOK(3, 3, args) {
				enabled := false
				if strings.Contains(strings.ToLower(args[1]), "true") {
					enabled = true
				}
				checkIfErr(node.AddPeer(args[0], enabled, args[2]))
			}
		},
	})

	// delete peer
	cli.AddCommand(command.Command{
		Name: "DelPeer",
		Help: "Delete a peer (name string)",
		Func: func(args []string) {
			if checkIfArgsAreOK(1, 1, args) {
				checkIfErr(node.DeletePeer(args[0]))
			}
		},
	})

	// display peers
	cli.AddCommand(command.Command{
		Name: "ShowPeers",
		Help: "Display a list of the current node's peers (name, key string)",
		Func: func(args []string) {
			peers, err := node.GetPeers()
			if checkIfErr(err) {
				enabled := "False"
				for index, peer := range peers {
					if peer.Enabled {
						enabled = "True"
					}
					fmt.Printf("%d) %s\t\t%s\t%s\n", index, peer.Name, peer.URI, enabled)
				}
			}
		},
	})

	// send message
	cli.AddCommand(command.Command{
		Name: "SendMsg",
		Help: "Sends a message to a node (contact, message string)",
		Func: func(args []string) {
			if checkIfArgsAreOK(2, -1, args) {
				msg := strings.Join(args[1:], " ")
				contact, err := node.GetContact(args[0])
				if checkIfErr(err) {
					checkIfErr(node.Send(contact.Name, []byte(msg)))
				}
			}
		},
	})

	// send channel message
	cli.AddCommand(command.Command{
		Name: "SendChanMsg",
		Help: "Sends a message to a channel (channel, message string)",
		Func: func(args []string) {
			if checkIfArgsAreOK(2, -1, args) {
				msg := strings.Join(args[1:], " ")
				channel, err := node.GetChannel(args[0])
				if checkIfErr(err) {
					checkIfErr(node.SendChannel(channel.Name, []byte(msg)))
				}
			}
		},
	})

	// show config
	cli.AddCommand(command.Command{
		Name: "ShowCfg",
		Help: "Show the config file for this node",
		Func: func(args []string) {
			cfg, err := node.Export()
			if checkIfErr(err) {
				fmt.Println(string(cfg))
			}
		},
	})

	// load config
	cli.AddCommand(command.Command{
		Name: "LoadCfg",
		Help: "Load a config file into this node (cfgFilePath string)",
		Func: func(args []string) {
			if checkIfArgsAreOK(1, 1, args) {
				file, err := ioutil.ReadFile(args[0])
				if checkIfErr(err) {
					checkIfErr(node.Import(file))
				}
			}
		},
	})

	// set UDP client transport
	cli.AddCommand(command.Command{
		Name: "SetClientTransport",
		Help: "Enable a polling policy, UDP transport within this node",
		Func: func(args []string) {
			node.SetPolicy(policy.NewPoll(udp.New(node), node, 500, 0))
		},
	})

	// start UDP server transport
	cli.AddCommand(command.Command{
		Name: "SetServerTransport",
		Help: "Enable a server policy, UDP transport within this node (uri string)",
		Func: func(args []string) {
			if checkIfArgsAreOK(1, 1, args) {
				node.SetPolicy(policy.NewServer(udp.New(node), args[0], false))
			}
		},
	})

	// exit
	cli.AddCommand(command.Command{
		Name: "Exit",
		Help: "Exit the command prompt",
		Func: func(args []string) {
			os.Exit(0)
		},
	})

	// start the interface
	cli.Run()
}
