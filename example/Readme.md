#Ratnet API Example Utility
This application's purpose is to showcase the capabilities of the ratnet network in a format that is simple for developers to understand so that they may esisaly use ratnet to develop their own applications. Each example usecase is coupled with the exact functions that an application would need to call to perform the same actions shown here.
## getting started
To get started, simply compile the example application found within this folder
```
cd $GOPATH/src/github.com/awgh/ratnet/example
go build
```
The example application takes 2 mandatory parameters. hostname and or port designation (such as ":8080" or "192.168.0.14:8080"), and a node type (ram means the node will only exist in memory; ql means the node will create a sqlite database from which it will operate from). The usage syntax of these parameters is shown below.
```
Usage: ./example <port> <ram|ql>
```

##peer to peer messaging
first, Bob starts a node.

```
$ ./example 127.0.0.1:31337 ram
start
```
then Alice starts a node.

```
$ ./example 127.0.0.1:12345 ram
start
```
Bob then adds Alice's node as a peer.
```
addpeer Alice true 127.0.0.1:12345
```
Alice then tells her node to display her content key so that she can share it with bob.
```
cid
ZSU6j+N/nInUuQTVMFOLW6TZgreGPEfyq96ZwK/EskU=
```
Bob then adds Alice as a contact.
```
addcontact Alice ZSU6j+N/nInUuQTVMFOLW6TZgreGPEfyq96ZwK/EskU=
```
now Bob can send Alice messages using her content key.
```
--- Bob's screen ---
send Alice this is a test message

--- Alice's screen ---
[content] this is a test
```
