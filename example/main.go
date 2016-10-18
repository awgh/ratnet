package main

import (
	"github.com/awgh/bencrypt/ecc"
	"github.com/awgh/ratnet/api"
	"github.com/awgh/ratnet/nodes/qldb"
	"github.com/awgh/ratnet/nodes/ram"
	"github.com/awgh/ratnet/policy"
	"github.com/awgh/ratnet/transports/udp"

	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

var (
	// RATNET : holds a ptr to our core instance of ratnet
	RATNET api.Node

	// internal program stuff
	input *bufio.Reader
	err   error
)

func init() {
	input = bufio.NewReader(os.Stdin)
}

func main() {
	// check arguments
	if len(os.Args) < 3 {
		fmt.Println("Usage: " + os.Args[0] + " <[address]:port> <ram|ql> [debug]")
		os.Exit(0)
	}

	/*
		- create a new node
		(a ram node is a node that will exist only in memmory and not create a database)
		(a qldb node is a node that will create a ql database file in order to function and persist data over long periods and consistant shutdowns)
	*/
	if strings.ToLower(os.Args[2]) == "ram" {
		RATNET = ram.New(new(ecc.KeyPair), new(ecc.KeyPair))
	} else {
		RATNET = qldb.New(new(ecc.KeyPair), new(ecc.KeyPair))
		RATNET.(*qldb.Node).BootstrapDB(os.Args[0] + ".ql")
		defer os.Remove(os.Args[0] + ".ql")
		RATNET.(*qldb.Node).FlushOutbox(0)
	}

	/*
		- start the error server if appropiate
		(this loop will read from the node's err channel and replay it's contents over a TCP/IP connection)
		(a description of the message format is below):
		type ratnet/api.message struct {
			Name:		the error type (ERROR or DEBUG)
			Content:	the error data itself
			IsChan:		bool value that dictates whether or not this message is "fatal"
			PubKey:		[TODO]
		}
	*/
	if len(os.Args) > 3 {
		if strings.Contains(os.Args[3], "debug") {
			errListner, err := net.Listen("tcp", "localhost:9990")
			if err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}
			connection, err := errListner.Accept()
			if err != nil {
				fmt.Println(err.Error())
			}
			RATNET.SetDebug(true)
			go func() {
				for {
					// write the error content to the connected socket
					msg := <-RATNET.Err()
					_, err := connection.Write([]byte(msg.Content.String() + "\n"))
					if err != nil {
						fmt.Println(err.Error())
						os.Exit(1)
					}

					// if the error is a fatal error, exit the application
					if msg.IsChan {
						os.Exit(1)
					}
				}
			}()
		}
	}

	/*
		- create transports and policies to use them
		(a "transport" is a way of sending to, and or, reciving data from, a ratnet node)
		(a "policy" defines the way in which a ratnet node will interact with a transport)
		(in this case, we are using one "server" policy and one "poll" policy on the "udp" transport)
	*/
	transport := udp.New(RATNET)
	RATNET.SetPolicy(policy.NewServer(transport, os.Args[1], false), policy.NewPoll(transport, RATNET))

	/*
		- start the read loop
		(this loop will read from the node's output channel. This is how we recive messages from other nodes)
		(a description of the message format is below):
		type ratnet/api.message struct {
			Name:		the name of the "profile" this message is to
			Content:	the message data itself
			IsChan:		bool value that dictates whether or not this message is a "channel" message
			PubKey:		[]byte value that holds the public key for the key pair used to create this message
						(this is generally for use when the key used to encrypt the message exists outside the ratnet framework)
		}
	*/
	go func() {
		for {
			// read a message from the output channel
			msg := <-RATNET.Out()
			fmt.Println(msg.Name + " " + msg.Content.String())
		}
	}()

	/*
		- start the main loop
		(this loop contoniously reads user input and parses it into one of the commands isolated below as "case" values within the large switch statement)
	*/
mainLoop:
	for {
		// read user input
		command, err := input.ReadString(0x0a)
		if err != nil {
			fmt.Println(err.Error())
			continue
		}

		commandSegments := strings.Split(command, " ")
		for index, cmd := range commandSegments {
			commandSegments[index] = strings.TrimSpace(cmd)
		}
		switch strings.ToLower(commandSegments[0]) {

		/*
			### Admin functions ###
		*/

		case "cid":
			pubkey, err := RATNET.CID()
			if err != nil {
				fmt.Println(err.Error())
			}
			fmt.Println(pubkey.ToB64())

		case "getcontact":
			if len(commandSegments) < 1 {
				fmt.Println("Usage: GetContact(name)")
				continue
			}
			contact, err := RATNET.GetContact(commandSegments[1])
			if err != nil {
				fmt.Println(err.Error())
				continue
			}
			fmt.Println(contact.Name + " | " + contact.Pubkey)

		case "getcontacts":
			contacts, err := RATNET.GetContacts()
			if err != nil {
				fmt.Println(err.Error())
				continue
			}
			for index, contact := range contacts {
				fmt.Println(strconv.Itoa(index) + ") " + contact.Name + " | " + contact.Pubkey)
			}

		case "addcontact":
			if len(commandSegments) < 3 {
				fmt.Println("Usage: AddContact(name, key)")
				continue
			}
			if err := RATNET.AddContact(commandSegments[1], commandSegments[2]); err != nil {
				fmt.Println(err.Error())
				continue
			}

		case "delcontact":
			if len(commandSegments) < 2 {
				fmt.Println("Usage: DeleteContact(name)")
				continue
			}
			if err := RATNET.DeleteContact(commandSegments[1]); err != nil {
				fmt.Println(err.Error())
				continue
			}

		case "getchannel":
			if len(commandSegments) < 2 {
				fmt.Println("Usage: GetChannel(name)")
				continue
			}
			channel, err := RATNET.GetChannel(commandSegments[1])
			if err != nil {
				fmt.Println(err.Error())
				continue
			}
			fmt.Println(channel.Name + " | " + channel.Pubkey)

		case "getchannels":
			channels, err := RATNET.GetChannels()
			if err != nil {
				fmt.Println(err.Error())
				continue
			}
			for index, channel := range channels {
				fmt.Println(strconv.Itoa(index) + ") " + channel.Name + " | " + channel.Pubkey)
			}

		case "addchannel":
			if len(commandSegments) < 3 {
				fmt.Println("Usage: AddChannel(name, key)")
				continue
			}
			if err := RATNET.AddChannel(commandSegments[1], commandSegments[2]); err != nil {
				fmt.Println(err.Error())
			}

		case "delchannel":
			if len(commandSegments) < 2 {
				fmt.Println("Usage: DeleteChannel(name)")
				continue
			}
			if err := RATNET.DeleteChannel(commandSegments[1]); err != nil {
				fmt.Println(err.Error())
			}

		case "getprofile":
			if len(commandSegments) < 2 {
				fmt.Println("Usage: GetProfile(name)")
				continue
			}
			profile, err := RATNET.GetProfile(commandSegments[1])
			if err != nil {
				fmt.Println(err.Error())
				continue
			}
			fmt.Print(profile.Name + " | " + profile.Pubkey)
			if profile.Enabled {
				fmt.Println(" | Enabled")
			} else {
				fmt.Println("")
			}

		case "getprofiles":
			profiles, err := RATNET.GetProfiles()
			if err != nil {
				fmt.Println(err.Error())
				continue
			}
			for index, profile := range profiles {
				fmt.Print(strconv.Itoa(index) + ") " + profile.Name + " | " + profile.Pubkey)
				if profile.Enabled {
					fmt.Println(" | Enabled")
				} else {
					fmt.Println("")
				}
			}

		case "addprofile":
			if len(commandSegments) < 3 {
				fmt.Println("Usage: AddProfile(name, enabled? t/f)")
				continue
			}
			enabled := false
			if strings.ToLower(commandSegments[2]) == "true" {
				enabled = true
			}
			if err := RATNET.AddProfile(commandSegments[1], enabled); err != nil {
				fmt.Println(err.Error())
			}

		case "delprofile":
			if len(commandSegments) < 2 {
				fmt.Println("Usage: DeleteProfile(name)")
				continue
			}
			if err := RATNET.DeleteProfile(commandSegments[1]); err != nil {
				fmt.Println(err.Error())
			}

		case "loadprofile":
			if len(commandSegments) < 2 {
				fmt.Println("Usage: LoadProfile(name)")
				continue
			}
			profile, err := RATNET.LoadProfile(commandSegments[1])
			if err != nil {
				fmt.Println(err.Error())
				continue
			}
			fmt.Println("Key => " + profile.ToB64())

		case "getpeer":
			if len(commandSegments) < 2 {
				fmt.Println("Usage: GetPeer(name)")
				continue
			}
			peer, err := RATNET.GetPeer(commandSegments[1])
			if err != nil {
				fmt.Println(err.Error())
				continue
			}
			fmt.Print(peer.Name + " | " + peer.URI)
			if peer.Enabled {
				fmt.Println(" | Enabled")
			} else {
				fmt.Println("")
			}

		case "getpeers":
			peers, err := RATNET.GetPeers()
			if err != nil {
				fmt.Println(err.Error())
				continue
			}
			for index, peer := range peers {
				fmt.Print(strconv.Itoa(index) + ") " + peer.Name + " | " + peer.URI)
				if peer.Enabled {
					fmt.Println(" | Enabled")
				} else {
					fmt.Println("")
				}
			}

		case "addpeer":
			if len(commandSegments) < 4 {
				fmt.Println("Usage: AddPeer(name, enabled? t/f, uri)")
				continue
			}
			enabled := false
			if strings.ToLower(commandSegments[2]) == "true" {
				enabled = true
			}
			if err := RATNET.AddPeer(commandSegments[1], enabled, commandSegments[3]); err != nil {
				fmt.Println(err.Error())
			}

		case "delpeer":
			if len(commandSegments) < 2 {
				fmt.Println("Usage: DeletePeer(name)")
				continue
			}
			if err := RATNET.DeletePeer(commandSegments[1]); err != nil {
				fmt.Println(err.Error())
			}

		case "send":
			if len(commandSegments) < 3 {
				fmt.Println("Usage: Send(name, message)")
				continue
			}
			msg := strings.TrimSpace(strings.TrimSpace(command[len(commandSegments[0])+1+len(commandSegments[1])+1:]))
			if err := RATNET.Send(commandSegments[1], []byte(msg)); err != nil {
				fmt.Println(err.Error())
			}

		case "sendchannel":
			if len(commandSegments) < 3 {
				fmt.Println("Usage: SendChannel(channel, message)")
				continue
			}
			msg := strings.TrimSpace(strings.TrimSpace(command[len(commandSegments[0])+1+len(commandSegments[1])+1:]))
			if err := RATNET.SendChannel(commandSegments[1], []byte(msg)); err != nil {
				fmt.Println(err.Error())
			}

		case "start":
			if err := RATNET.Start(); err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}

		case "stop":
			RATNET.Stop()

		/*
			### Public functions ###
		*/

		case "id":
			key, err := RATNET.ID()
			if err != nil {
				fmt.Println(err.Error())
				continue
			}
			fmt.Println(key.ToB64())

		case "getdebug":
			fmt.Print("Debug mode for this node is ")
			if RATNET.GetDebug() {
				fmt.Println("on.")
			} else {
				fmt.Println("off.")
			}

		case "setdebug":
			if len(commandSegments) < 2 {
				fmt.Println("Usage: SetDebug(t/f)")
				continue
			}
			mode := false
			if strings.ToLower(commandSegments[1]) == "true" {
				mode = true
			}
			RATNET.SetDebug(mode)
		/*
			### non ratnet functions ###
		*/

		case "exit":
			fmt.Println("Are you sure you want to quit the application (y/n)?")
			resp, err := input.ReadString(0x0a)
			if err != nil {
				fmt.Println(err.Error())
				continue
			}
			if strings.Contains(resp, "y") {
				break mainLoop
			}

		default:
			fmt.Println("Command not found.")
		}
	}
}
