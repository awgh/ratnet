#Ratnet API Example Utility
This application's purpose is to showcase the capabilities of the ratnet network in a format that is simple for developers to understand so that they may esisaly use ratnet to develop their own applications. Each example usecase is coupled with the exact functions that an application would need to call to perform the same actions shown here.
## getting started
To get started, simply compile the example application found within this folder
```
cd $GOPATH/src/github.com/awgh/ratnet/example
go build
```

##peer to peer messaging
First, Bob sets his node to utilize the UDP transport with a policy of "server".

```
>>> SetServerTransport :8000
```

Alice sets her node to utilize the UDP transport with a policy of "polling".

```
>>> SetClientTransport
```

Next, Alice adds Bob as a peer

```
AddPeer Bob True 127.0.0.1:8000
```

Then, Bob tells his node to display his content key so that she can share it with Alice.

```
>>> CID
SqRHK39CyU3P7q8nBGQyPaMS2d65FkWKFC9rY4LjjSI=
```

And, Alice then adds Bob as a contact.

```
AddContact Bob SqRHK39CyU3P7q8nBGQyPaMS2d65FkWKFC9rY4LjjSI=
```

Finaly, Alice And Bob start their respective nodes.
```
Start
```

At this point, Alice can send Bob messages.

--- Alice's screen ---
SendMsg Bob this is a test

--- Bob's screen ---
[RX From [content] ]: this is a test
```
