# What is Ratnet?
Ratnet is a library that allows applications to communicate using an [onion-routed](https://en.wikipedia.org/wiki/Onion_routing) and [flood-routed](https://en.wikipedia.org/wiki/Flooding_(computer_networking)) message bus.  All communications are encrypted end-to-end by the library itself.

Ratnet is completely modular, meaning that the interactions of all significant components of the system are defined with interfaces, and are therefore interchangeable.  Network transports, cryptosystems, connection policies, and the ratnet nodes themselves can all be customized and swapped around dynamically.

The Ratnet library provides at least two working implementations for each of these interfaces:

- Network Transports:  [HTTPS](https://godoc.org/github.com/awgh/ratnet/transports/https), [TLS](https://godoc.org/github.com/awgh/ratnet/transports/tls), and [UDP](https://godoc.org/github.com/awgh/ratnet/transports/udp) are provided
- Cryptosystems: [ECC](https://godoc.org/github.com/awgh/bencrypt/ecc) and [RSA](https://godoc.org/github.com/awgh/bencrypt/ecc) implementations are provided
- Connection Policies: [Server](https://godoc.org/github.com/awgh/ratnet/policy#Server), [Polling](https://godoc.org/github.com/awgh/ratnet/policy#Poll), and [P2P](https://godoc.org/github.com/awgh/ratnet/policy#P2P) are provided
- Nodes: [QL Database-Backed Node](https://godoc.org/github.com/awgh/ratnet/nodes/qldb), a [RAM-only Node](https://godoc.org/github.com/awgh/ratnet/nodes/ram), a [FS-backed Node](https://godoc.org/github.com/awgh/ratnet/nodes/fs), and an [Upper.io db Backed Node](https://godoc.org/github.com/awgh/ratnet/nodes/db) are provided.

It's also easy to implement your own replacement for any or all of these components.  Multiple transport modules can be used at once, and different cryptosystems can be used for the Onion-routing and for the content encryption, if desired.

Ratnet provides input and output channels for your application to send and receive binary messages in clear text, making it very easy to interact with.

## What's a Connection Policy?

You caught me, I made that term up.  In ratnet, *Transports* are responsible for physically making and receiving connections and that's it.  *Nodes* are basically message queues with some key management and special knowledge about when to encrypt things (and the *Cryptosystem* is the method they would use to do that).  But none of those things actually starts a connection and moves the data around.  That is the responsibility of the *Connection Policy*.  Think of it as a script that controls a Node and any number of different Transports.  

We provide two very simple connection policies:

1. [Server](https://godoc.org/github.com/awgh/ratnet/policy#Server) - This just opens up a port and listens on it.
2. [Polling](https://godoc.org/github.com/awgh/ratnet/policy#Poll) - After a delay, this will connect to every Peer and exchange messages.

In real-world usage, you're very likely to want to implement your own version of Polling (via the [Policy](https://github.com/awgh/ratnet/blob/master/api/policy.go) interface).  We will be doing a lot more development and experimentation with new policies (and transports!) in the future.


# Examples

## RatNet Shell Example

There is a standalone example application specifically designed to showcase the relationship between common use-cases and the Ratnet API available [here](https://github.com/awgh/ratnet/tree/master/example).

## Fully Working IRC-like Demo 

[Hushcom](https://github.com/awgh/hushcom) is a fully working demo app that implements IRC-like chat functionality with a client and a server.

The [hushcom client application](https://github.com/awgh/hushcom/blob/master/hushcom/main.go) is a good reference for how to set up a client using the Poll connection policy.

The [hushcomd server application](https://github.com/awgh/hushcom/blob/master/hushcom/main.go) is a good reference for how to set up a server using the Server connection policy.


## Making a Node

Make a QL-Database-Backed Node, which saves states to disk:
```go
	// QLDB Node Mode
	node := qldb.New(new(ecc.KeyPair), new(ecc.KeyPair))
	node.BootstrapDB(dbFile)
```

Or, make a RAM-Only Node, which won't write anything to the disk:
```go
	// RamNode Mode:
	node := ram.New(new(ecc.KeyPair), new(ecc.KeyPair))
```

The KeyPairs passed in as arguments are just used to determine which cryptosystem should be used for the Onion-Routing (first argument) and for the Content Encryption (second argument).  There is no requirement that the routing encryption be the same as the content encryption.  It's easy to use RSA for content and ECC for routing, for example, just change the second argument above to rsa.KeyPair. 

## Setup Transports and Policies 

```go
	transportPublic := https.New("cert.pem", "key.pem", node, true)
	transportAdmin := https.New("cert.pem", "key.pem", node, true)
	node.SetPolicy(
		policy.NewServer(transportPublic, listenPublic, false),
		policy.NewServer(transportAdmin, listenAdmin, true))

	log.Println("Public Server starting: ", listenPublic)
	log.Println("Control Server starting: ", listenAdmin)

	node.Start()
```	

## Handle messages coming from the network

```go	
	go func() {
		for {
			msg := <-node.Out()
			if err := HandleMsg(msg); err != nil {
				log.Println(err.Error())
			}
		}
	}()
```

## Send messages to the network

Blocking Send:
```go
	message := api.Msg{Name: "destname1", IsChan: false}
	message.Content = bytes.NewBufferString(testMessage1)
	node.In() <- message
```
	
Non-Blocking Send:
```go
        select {
		case node.In() <- message:
			//fmt.Println("sent message", msg)
		default:
			//fmt.Println("no message sent")
		}	
```

# Additional Documentation

- Overview Slide Deck from Toorcamp 2016 [here](https://github.com/awgh/ratnet/blob/master/docs/RatNet-Toorcamp16-v1.pdf).
- API Docs are available [here](https://godoc.org/github.com/awgh/ratnet/api).

# Related Projects

- [Bencrypt, crypto abstraction layer & utils](https://github.com/awgh/bencrypt)
- [Ratnet, onion-routed messaging system with pluggable transports](https://github.com/awgh/ratnet)
- [HushCom, simple IRC-like client & server](https://github.com/awgh/hushcom)

#Authors and Contributors

- awgh@awgh.org (@awgh)

- vyrus001@gmail.com (@vyrus001)
