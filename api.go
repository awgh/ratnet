package ratnet

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"log"
	"github.com/awgh/ratnet/modules"
	"strconv"
	"strings"
	"time"

	. "github.com/awgh/bencrypt"

	_ "github.com/cznic/ql/driver"
)

/*
// To install ql:
//force github.com/cznic/zappy to purego mode
//go get -tags purego github.com/cznic/ql  (or ql+cgo seems to work on arm now, too)

Control API Usage:
https://localhost:20001?{"Action":<ACTION>,"Args":[<ARGS>]}

ID Test:
https://localhost:20001/?{%22Action%22:%22ID%22,%22Args%22:[]}

JSON Actions:

action: "AddDest" 	args(2): "name", "cpubkey" b64-encoded PEM public key
- takes a destination name and content public key, adds to database

action: "AddChannel" 	args(2): "name", "cpubkey" b64-encoded PEM private key
- takes a channel/channel name and channel key pair, adds to database

action: "ID" 		args:
- returns the public content key of this node, as base64-encoded PEM

action: "Send" 		args(2): "name", "message" b64-encoded binary
- takes a destination name (same as AddDest name) and a base64-encoded message

action: "SendChannel" args(2): "channel", "message" b64-encoded binary
- takes a channel name (same as AddChannel name) and a base64-encoded message

action: "Receive" 		args(1): "message" b64-encoded PEM?
- takes a destination name (same as AddDest name) and a base64-encoded message

*/

var (
	contentCrypt  CryptoAPI
	routingCrypt  CryptoAPI
	channelCrypts map[string]string

	recentPageIdx = 1
	recentPage1   map[string]byte
	recentPage2   map[string]byte

	firstRun = true
)

func init() {
	if ECC_MODE {
		contentCrypt = new(ECC)
		routingCrypt = new(ECC)
	} else {
		contentCrypt = new(RSA)
		routingCrypt = new(RSA)
	}

	recentPage1 = make(map[string]byte)
	recentPage2 = make(map[string]byte)

	channelCrypts = make(map[string]string)
}

type ApiCall struct {
	Action string
	Args   []string
}

type dest struct {
	Id      int
	Name    string
	Cpubkey string
}

type ServerConf struct {
	Name    string
	Uri     string
	Enabled bool
}

func refreshChannels(c *sql.DB) {
	// refresh the channelCrypts array
	rc := transactQuery(c, "SELECT name,privkey FROM channels;")
	for rc.Next() {
		var n, s string
		rc.Scan(&n, &s)
		channelCrypts[n] = s
	}
}

func seenRecently(hdr []byte) bool {

	shdr := string(hdr)
	_, aok := recentPage1[shdr]
	_, bok := recentPage2[shdr]
	retval := aok || bok

	switch recentPageIdx {
	case 1:
		if len(recentPage1) >= 50 {
			if len(recentPage2) >= 50 {
				recentPage2 = nil
				recentPage2 = make(map[string]byte)
			}
			recentPageIdx = 2
			recentPage2[shdr] = 1
		} else {
			recentPage1[shdr] = 1
		}
	case 2:
		if len(recentPage2) >= 50 {
			if len(recentPage1) >= 50 {
				recentPage1 = nil
				recentPage1 = make(map[string]byte)
			}
			recentPageIdx = 1
			recentPage1[shdr] = 1
		} else {
			recentPage2[shdr] = 1
		}
	}
	return retval
}

func Api(a *ApiCall, db func() *sql.DB, adminPrivs bool, params ...interface{}) ([]byte, error) {

	c := db()
	defer c.Close()
	channelSend := false

	l := func(...interface{}) {}
	/*
		if len(params) > 0 { //debug print mode
			l = func(p ...interface{}) { log.Println(params[0], p) }
		}*/

	//
	//
	switch a.Action {

	// Admin Actions
	case "CID", "GetChannels", "GetChannelPrivKey", "AddChannel", "DeleteChannel",
		"AddDest", "Destinations", "GetServer", "UpdateServer", "DeleteServer",
		"Send", "SendChannel", "GetProfiles", "LoadProfile", "UpdateProfile",
		"DeleteProfile":
		{
			if !adminPrivs {
				return nil, errors.New("Your privilege has been checked.")
			}
			switch a.Action {
			case "CID":
				{
					if len(a.Args) != 0 {
						return nil, errors.New("Invalid Argument Count")
					}
					return []byte(contentCrypt.B64fromPublicKey(contentCrypt.GetPubKey())), nil
				}
			case "GetChannels":
				{
					//log.Println("GetChannels")
					if len(a.Args) != 0 {
						return nil, errors.New("Invalid Argument Count")
					}
					r := transactQuery(c, "SELECT name FROM channels;")
					var channels []string
					for r.Next() {
						var p string
						r.Scan(&p)
						channels = append(channels, p)
					}
					o, err := json.Marshal(channels)
					if err != nil {
						log.Println(err.Error())
					}
					return o, nil
				}
			// Limited violation of KSM for channel keys only, DON'T EXPOSE THIS
			case "GetChannelPrivKey":
				{
					//log.Println("GetChannelPrivKey")
					if len(a.Args) != 1 {
						return nil, errors.New("Invalid Argument Count")
					}
					r := transactQueryRow(c, "SELECT privkey FROM channels WHERE name==$1;", a.Args[0])
					var privkey string
					err := r.Scan(&privkey)
					if err == sql.ErrNoRows {
						return []byte(""), nil
					} else if err != nil {
						return nil, err
					} else {
						return []byte(privkey), nil
					}
				}
			case "AddChannel":
				{
					if len(a.Args) != 2 {
						return nil, errors.New("Invalid Argument Count")
					}
					// todo: sanity check key via bencrypt

					tx, err := c.Begin()
					if err != nil {
						log.Fatal(err.Error())
					}
					_, err = tx.Exec("DELETE FROM channels WHERE name==$1;", a.Args[0])
					_, err = tx.Exec("INSERT INTO channels VALUES( $1, $2 )", a.Args[0], a.Args[1])
					if err != nil {
						log.Fatal(err.Error())
					}
					tx.Commit()

					refreshChannels(c)
					return []byte("OK"), nil

				}
			case "DeleteChannel":
				{
					//log.Println("DeleteChannel")
					if len(a.Args) != 1 {
						return nil, errors.New("Invalid Argument Count")
					}
					transactExec(c, "DELETE FROM channels WHERE name==$1;", a.Args[0])
					return []byte("OK"), nil
				}
			case "AddDest":
				{
					//log.Println("AddDest: ", a.Args[1])
					if len(a.Args) != 2 {
						return nil, errors.New("Invalid Argument Count")
					}
					_, err := contentCrypt.B64toPublicKey(a.Args[1])
					if err != nil {
						return nil, err
					}
					tx, err := c.Begin()
					if err != nil {
						log.Fatal(err.Error())
					}
					_, err = tx.Exec("DELETE FROM destinations WHERE name==$1;", a.Args[0])
					_, err = tx.Exec("INSERT INTO destinations VALUES( $1, $2 )", a.Args[0], a.Args[1])
					if err != nil {
						log.Fatal(err.Error())
					}
					tx.Commit()

					return []byte("OK"), nil
				}
			case "Destinations":
				{
					if len(a.Args) != 0 {
						return nil, errors.New("Invalid Argument Count")
					}
					s := transactQuery(c, "SELECT id(),name,cpubkey FROM destinations;")
					var dests []dest
					for s.Next() {
						var d dest
						s.Scan(&d.Id, &d.Name, &d.Cpubkey)
						dests = append(dests, d)
					}
					enc, err := json.Marshal(dests)
					if err != nil {
						log.Println(err.Error())
					}
					return enc, nil
				}
			case "GetServer":
				{
					//log.Println("GetServer")
					var r *sql.Rows
					switch len(a.Args) {
					case 0:
						r = transactQuery(c, "SELECT name,uri,enabled FROM servers;")
					case 1:
						r = transactQuery(c, "SELECT name,uri,enabled FROM servers WHERE name==$1;", a.Args[0])
					default:
						return nil, errors.New("Invalid Argument Count")
					}
					var servers []ServerConf
					for r.Next() {
						var s ServerConf
						r.Scan(&s.Name, &s.Uri, &s.Enabled)
						servers = append(servers, s)
					}
					o, err := json.Marshal(servers)
					if err != nil {
						log.Println(err.Error())
					}
					return o, nil
				}
			case "UpdateServer":
				{
					//log.Println("UpdateServer")
					if len(a.Args) != 3 {
						return nil, errors.New("Invalid Argument Count")
					}
					enabled, err := strconv.ParseBool(a.Args[2])
					if err != nil {
						return nil, errors.New("Invalid enabled Value")
					}
					r := transactQueryRow(c, "SELECT name FROM servers WHERE name==$1;", a.Args[0])
					var name string
					err = r.Scan(&name)
					if err == sql.ErrNoRows {
						//log.Println("-> New Server")
						transactExec(c, "INSERT INTO servers (name,uri,enabled) VALUES( $1, $2, $3 );",
							a.Args[0], a.Args[1], enabled)
					} else if err == nil {
						//log.Println("-> Update Server")
						transactExec(c, "UPDATE servers SET enabled=$1,uri=$2 WHERE name==$3;",
							enabled, a.Args[1], a.Args[0])
					} else {
						log.Println(err.Error())
						return nil, err
					}
					return []byte("OK"), nil
				}
			case "DeleteServer":
				{
					//log.Println("DeleteServer")
					if len(a.Args) != 1 {
						return nil, errors.New("Invalid Argument Count")
					}
					transactExec(c, "DELETE FROM servers WHERE name==$1;", a.Args[0])
					return []byte("OK"), nil
				}
			case "GetProfiles":
				{
					//log.Println("GetProfiles")
					if len(a.Args) != 0 {
						return nil, errors.New("Invalid Argument Count")
					}
					r := transactQuery(c, "SELECT name,enabled FROM profiles;")
					type Profile struct {
						Name    string
						Enabled bool
					}
					var profiles []Profile
					for r.Next() {
						var p Profile
						r.Scan(&p.Name, &p.Enabled)
						profiles = append(profiles, p)
					}
					if len(profiles) < 1 {
						return nil, nil
					}
					o, err := json.Marshal(profiles)
					if err != nil {
						log.Println(err.Error())
					}
					return o, nil
				}
			case "LoadProfile":
				{
					//log.Println("LoadProfile")
					if len(a.Args) != 1 {
						return nil, errors.New("Invalid Argument Count")
					}
					row := transactQueryRow(c, "SELECT privkey FROM profiles WHERE name==$1;", a.Args[0])
					var pk string
					row.Scan(&pk)
					profileCrypt := new(ECC)
					err := profileCrypt.B64toPrivateKey(pk)
					if err != nil {
						log.Println(err.Error())
						return nil, err
					}
					contentCrypt = profileCrypt
					log.Println("Profile Loaded: " + profileCrypt.B64fromPublicKey(profileCrypt.GetPubKey()))
					return []byte(profileCrypt.B64fromPublicKey(profileCrypt.GetPubKey())), nil
				}
			case "UpdateProfile":
				{
					//log.Println("UpdateProfile")
					if len(a.Args) != 2 {
						return nil, errors.New("Invalid Argument Count")
					}
					enabled, err := strconv.ParseBool(a.Args[1])
					if err != nil {
						return nil, errors.New("Invalid enabled Value")
					}
					r := transactQueryRow(c, "SELECT * FROM profiles WHERE name==$1;", a.Args[0])
					var name, key, al string
					err = r.Scan(&name, &key, &al)
					if err == sql.ErrNoRows {
						//log.Println("-> New Profile")
						// generate new profile keypair
						profileCrypt := new(ECC) //todo: RSA option?
						profileCrypt.GenerateKey()
						// insert new profile
						transactExec(c, "INSERT INTO profiles VALUES( $1, $2, $3 )",
							a.Args[0], profileCrypt.B64fromPrivateKey(), enabled)
					} else if err == nil {
						//log.Println("-> Update Profile")
						transactExec(c, "UPDATE profiles SET enabled=$1 WHERE name==$2;",
							enabled, a.Args[0])
					} else {
						log.Println(err.Error())
						return nil, err
					}
					return []byte("OK"), nil
				}
			case "DeleteProfile":
				{
					//log.Println("DeleteProfile")
					if len(a.Args) != 1 {
						return nil, errors.New("Invalid Argument Count")
					}
					transactExec(c, "DELETE FROM profiles WHERE name==$1;", a.Args[0])
					return []byte("OK"), nil
				}
			case "SendChannel":
				channelSend = true
				fallthrough
			case "Send":
				{
					l("Send* called")
					if len(a.Args) != 2 && len(a.Args) != 3 {
						return nil, errors.New("Invalid Argument Count")
					}
					var r1 *sql.Row
					var destkey interface{}
					var rxsum []byte
					var err error

					channelName := ""
					if channelSend {
						channelName = a.Args[0]
						if len(a.Args) == 3 { // hack todo: third argument is pubkey override
							destkey, err = contentCrypt.B64toPublicKey(a.Args[2])
							l("SendChannel key override: ", destkey)
							if err != nil {
								return nil, err
							}
						} else {
							b64priv, ok := channelCrypts[channelName]
							if !ok {
								return nil, errors.New("No public key for Channel")
							}
							crypt := new(ECC)
							err = crypt.B64toPrivateKey(b64priv)
							if err != nil {
								return nil, err
							}
							destkey = crypt.GetPubKey() //fnord - need to get this from channelKeys in JS
							l("SendChannel key from DB: ", destkey)
						}
						// prepend a uint16 of channel name length, little-endian
						var t uint16
						t = uint16(len(channelName))
						rxsum = []byte{byte(t >> 8), byte(t & 0xFF)}
						rxsum = append(rxsum, []byte(channelName)...)
					} else {
						var s string
						if len(a.Args) == 3 { // hack todo: third argument is pubkey override
							s = a.Args[2]
							l("Send key override invoked: ", s)
						} else {
							r1 = transactQueryRow(c, "SELECT cpubkey FROM destinations WHERE name==$1;", a.Args[0])
							err = r1.Scan(&s)
							if err == sql.ErrNoRows {
								return []byte("No such destination."), errors.New("Unknown Destination")
							} else if err != nil {
								return nil, err
							}
							l("SendChannel key from DB: ", s)
						}
						destkey, err = contentCrypt.B64toPublicKey(s)
						if err != nil {
							return nil, err
						}
						// prepend a uint16 zero, meaning channel name length is zero
						rxsum = []byte{0, 0}
					}
					l("Send using destkey: ", destkey)

					// append a nonce
					salt, err := GenerateRandomBytes(16)
					if err != nil {
						return nil, err
					}
					rxsum = append(rxsum, salt...)

					// append a hash of content public key so recepient will know it's for them
					dh, err := DestHash(destkey, salt)
					if err != nil {
						return nil, err
					}
					rxsum = append(rxsum, dh...)

					msg, err := base64.StdEncoding.DecodeString(a.Args[1])
					if err != nil {
						return nil, err
					}

					//todo: is this passing msg by reference or not???
					data, err := contentCrypt.EncryptMessage(msg, destkey)
					if err != nil {
						return nil, err
					}
					data = append(rxsum, data...)
					t := time.Now().UnixNano()
					d := base64.StdEncoding.EncodeToString(data)
					transactExec(c, "INSERT INTO outbox(channel, msg, timestamp) VALUES($1,$2, $3);",
						channelName, d, t)
					l("Sent at time: ", t, d)
					return []byte("OK"), nil
				}
			}
		}

	// Public Actions
	case "ID":
		{
			if len(a.Args) != 0 {
				return nil, errors.New("Invalid Argument Count")
			}
			return []byte(routingCrypt.B64fromPublicKey(routingCrypt.GetPubKey())), nil
		}
	// args: data blob from PickupMail
	case "DeliverMail":
		{
			l("DeliverMail:" + a.Args[0])
			if len(a.Args) != 1 {
				return nil, errors.New("Invalid Argument Count")
			}
			if len(a.Args[0]) < 1 { // todo: correct min length
				return nil, errors.New("DeliverMail called with no data.")
			}

			cipher, err := base64.StdEncoding.DecodeString(a.Args[0])
			if err != nil {
				log.Println("base64 decode failed")
				return nil, err
			}

			data, err := routingCrypt.DecryptMessage(cipher)
			if err != nil {
				log.Println("DeliverMail failed in routing decrypt.")
				return nil, err
			}
			r := bytes.NewBuffer(data)
			line, err := r.ReadString('\n')
			for err == nil {
				if len(line) < 16 { // aes.BlockSize == 16
					//todo: remove padding before here?
					// note: can't break here for some reason, messages get lost.
					//       keep reading until EOF and then clean up after.
					line, err = r.ReadString('\n')
					continue
				}
				l("_-------------------_")
				msg, err := base64.StdEncoding.DecodeString(line)
				if err != nil {
					l(err.Error())
					line, err = r.ReadString('\n')
					continue
				}
				checkMessageForMe := true
				l("Received Encrypted: ", line)
				// beginning uint16 of message is channel name length
				var channelLen uint16
				channelLen = (uint16(msg[0]) << 8) | uint16(msg[1])

				if len(msg) < int(channelLen)+2+16+16 { // uint16 + nonce + hash //todo
					log.Println("Incorrect channel name length")
					line, err = r.ReadString('\n')
					continue
				}

				var crypt CryptoAPI
				channelName := ""
				if channelLen == 0 { // private message (zero length channel)
					crypt = contentCrypt
				} else { // channel message
					channelName = string(msg[2 : 2+channelLen])
					l("Channel Message Received on: ", channelName)
					b64priv, ok := channelCrypts[channelName]
					if !ok { // we are not listening to this channel
						l("... but I'm not listening... ")
						checkMessageForMe = false
					} else {
						crypt = new(ECC)
						err = crypt.B64toPrivateKey(b64priv)
						if err != nil {
							l(err.Error())
							line, err = r.ReadString('\n')
							continue
						}
					}
				}
				msg = msg[2+channelLen:] //skip over the channel name
				forward := true

				// LOOP PREVENTION before handling or forwarding
				if seenRecently(msg[:16]) {
					l("Repeating message skipped early.", msg[:16])
					forward = false
					checkMessageForMe = false
				}
				//
				if checkMessageForMe {
					// check to see if this is a msg for me
					pubkey := crypt.GetPubKey()
					hash, err := DestHash(pubkey, msg[:16])
					if err != nil {
						l(err.Error())
						line, err = r.ReadString('\n')
						continue
					}
					if bytes.Equal(hash, msg[16:16+len(hash)]) {
						if channelLen == 0 {
							l("Private message for me!")
							forward = false
						} else {
							l("Channel message for me!")
						}
						clear, err := crypt.DecryptMessage(msg[16+len(hash):])
						if err != nil {
							l("Decryption Failure! ", err.Error())
							line, err = r.ReadString('\n')
							continue
						}
						var typeID uint16
						buf := bytes.NewBuffer(clear)
						binary.Read(buf, binary.BigEndian, &typeID)
						err = modules.Dispatch(typeID, clear[2:])
						if err != nil {
							l(err.Error())
							line, err = r.ReadString('\n')
							continue
						}
					} else {
						l("Message is not for me.", channelName, msg[:8])
						l("used pubkey: ", pubkey)
					}
				}
				if forward {
					// save message in my outbox, if not already present
					r1 := transactQueryRow(c, "SELECT channel FROM outbox WHERE channel==$1 AND msg==$2;", channelName, line)
					var rc string
					err := r1.Scan(&rc)
					if err == sql.ErrNoRows {
						// we don't have this yet, so add it
						t := time.Now().UnixNano()
						transactExec(c, "INSERT INTO outbox(channel,msg,timestamp) VALUES($1,$2,$3);",
							channelName, line, t)
						l("... forwarded message with time: ", t)
					} else if err != nil {
						l("FATAL SQL ERROR")
						return nil, err
					} else {
						l("...duplicate message ignored...")
					}
				}
				l("-___________________-")
				line, err = r.ReadString('\n')
			}
			if err != nil && err != io.EOF {
				return nil, err
			}
			return nil, nil
		}
	// args: rpubkey, lastTime, optional: channels*
	// note: all arguments from 3 on will be used as channels
	case "PickupMail":
		{
			//log.Println("PickupMail: " + a.Args[1])
			var channels []string
			wildcard := false
			argLen := len(a.Args)
			switch argLen {
			default:
				channels = make([]string, argLen-2)
				for i := 2; i < argLen; i++ {
					for _, char := range a.Args[1] {
						if strings.Index("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0987654321", string(char)) == -1 {
							return nil, errors.New("Invalid characters in channel name!")
						}
					}
					channels[i-2] = a.Args[i]
				}
			case 2:
				// if no channels are given, get everything
				wildcard = true
			case 1:
				fallthrough
			case 0:
				return nil, errors.New("Invalid Argument Count")
			}

			rpub, err := routingCrypt.B64toPublicKey(a.Args[0])
			if err != nil {
				return nil, err
			}
			i, err := strconv.ParseInt(a.Args[1], 10, 64)
			if err != nil {
				return nil, err
			}

			sqlq := "SELECT msg, timestamp FROM outbox"
			if i != 0 {
				sqlq += " WHERE (int64(" + strconv.FormatInt(i, 10) +
					") < timestamp)"
			}

			if !wildcard && len(channels) > 0 {
				// QL is broken?  couldn't make it work with prepared stmts
				if i != 0 {
					sqlq += " AND"
				} else {
					sqlq += " WHERE"
				}
				sqlq = sqlq + " channel IN( \"" + channels[0] + "\""
				for i := 1; i < len(channels); i++ {
					sqlq = sqlq + ",\"" + channels[i] + "\""
				}
				sqlq = sqlq + " )"
			}

			// todo:  ORDER BY breaks on android/arm and returns nothing without error, report to cznic
			//			sqlq = sqlq + " ORDER BY timestamp ASC;"
			sqlq = sqlq + ";"

			lastTime := i
			r := transactQuery(c, sqlq)
			buffer := new(bytes.Buffer)
			for r.Next() {
				var msg string
				var ts int64
				r.Scan(&msg, &ts)
				if ts > lastTime { // do this instead of ORDER BY, for android
					lastTime = ts
				}
				l("picked up [time,i,lastTime]: ", ts, i, lastTime)
				l("   data: ", msg)
				n, err := buffer.WriteString(msg + "\n")
				if err != nil {
					l("FATAL BUFFER WRITE FAIL 1")
					return nil, err
				} else if n != len(msg)+1 {
					l("FATAL BUFFER WRITE FAIL 2")
					return nil, errors.New("buffer WriteString failed")
				}
			}
			// Prepend the timestamp of last message read
			buf := new(bytes.Buffer)
			binary.Write(buf, binary.BigEndian, lastTime)

			if buffer.Len() > 0 {
				cipher, err := routingCrypt.EncryptMessage(buffer.Bytes(), rpub)
				if err != nil {
					l("FATAL ENCRYPT FAIL")
					return nil, err
				}
				return append(buf.Bytes(), base64.StdEncoding.EncodeToString(cipher)...), err
			}
			return buf.Bytes(), nil
		}
	}
	return nil, errors.New("Invalid API Action")
}
