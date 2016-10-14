package qldb

// To install ql:
//force github.com/cznic/zappy to purego mode
//go get -tags purego github.com/cznic/ql  (or ql+cgo seems to work on arm now, too)

import (
	"database/sql"
	"log"
	"time"

	"github.com/awgh/bencrypt/bc"
	"github.com/awgh/ratnet/api"

	_ "github.com/cznic/ql/driver" // load the QL database driver
)

// Node : defines an instance of the API with a ql-DB backed Node
type Node struct {
	contentKey  bc.KeyPair
	routingKey  bc.KeyPair
	channelKeys map[string]bc.KeyPair

	recentPageIdx int
	recentPage1   map[string]byte
	recentPage2   map[string]byte

	policies  []api.Policy
	db        func() *sql.DB
	firstRun  bool
	isRunning bool

	debugMode bool

	// external data members
	in  chan api.Msg
	out chan api.Msg
	err chan api.Msg
}

// New : creates a new instance of API
func New(contentKey, routingKey bc.KeyPair) *Node {
	// create node
	node := new(Node)

	// init page maps
	node.recentPage1 = make(map[string]byte)
	node.recentPage2 = make(map[string]byte)

	// init channel key map
	node.channelKeys = make(map[string]bc.KeyPair)

	// set crypto modes
	node.contentKey = contentKey
	node.routingKey = routingKey

	// setup chans
	node.in = make(chan api.Msg)
	node.out = make(chan api.Msg)
	node.err = make(chan api.Msg)

	return node
}

// SetPolicy : set the array of Policy objects for this Node
func (node *Node) SetPolicy(policies ...api.Policy) {
	node.policies = policies
}

// FlushOutbox : Deletes outbound messages older than maxAgeSeconds seconds
func (node *Node) FlushOutbox(maxAgeSeconds int64) {
	c := time.Now().UnixNano()
	c = c - (maxAgeSeconds * 1000000000)
	sql := "DELETE FROM outbox WHERE timestamp < ($1);"

	// todo: below does not work on android/arm, investigate
	//sql := "DELETE FROM outbox WHERE since(timestamp) > duration(\"" +
	//	strconv.FormatInt(maxAgeSeconds, 10) + "s\");"
	//log.Println("Flushed Database (seconds): ", maxAgeSeconds)

	transactExec(node.db(), sql, c)
}

// BootstrapDB - Initialize or open a database file
func (node *Node) BootstrapDB(database string) func() *sql.DB {

	if node.db != nil {
		return node.db
	}

	node.db = func() *sql.DB {
		//log.Println("db: " + database)  //todo: why does this trigger so much?
		c, err := sql.Open("ql", database)
		if err != nil {
			log.Fatal("DB Error Opening: " + database + "\n  " + err.Error())
		}
		return c
	}

	// One-time Initialization
	c := node.db()
	defer c.Close()

	transactExec(c, `
		CREATE TABLE IF NOT EXISTS contacts (
			name	string	NOT NULL,
			cpubkey	string	NOT NULL
		);		
	`)
	//CREATE UNIQUE INDEX IF NOT EXISTS contactid ON contacts (id());

	transactExec(c, `
		CREATE TABLE IF NOT EXISTS channels ( 			
			name	string	NOT NULL,
			privkey	string	NOT NULL
		);
	`)

	transactExec(c, `
		CREATE TABLE IF NOT EXISTS config ( 
			name	string	NOT NULL,
			value	string	NOT NULL
		);
	`)

	/*  timestamp field must stay int64 and not time type,
	due to a unknown bug only on android/arm in cznic/ql via sql driver
	*/
	transactExec(c, `
		CREATE TABLE IF NOT EXISTS outbox (
			channel		string	DEFAULT "",
			msg			string	NOT NULL,
			timestamp	int64	NOT NULL
		);
	`)

	transactExec(c, `
		CREATE TABLE IF NOT EXISTS peers (
			name	string	NOT NULL,  
			uri		string	NOT NULL,
			enabled	bool	NOT NULL,
			pubkey	string	DEFAULT NULL
		);
	`)

	transactExec(c, `
		CREATE TABLE IF NOT EXISTS profiles (
			name	string	NOT NULL,
			privkey	string	NOT NULL,
			enabled	bool	NOT NULL
		);
	`)

	var n, s string

	// Content Key Setup
	// todo: content key needs to go away and be replaced by vectorized enabled profiles.
	r1 := transactQueryRow(c, "SELECT * FROM config WHERE name == `contentkey`;")
	err := r1.Scan(&n, &s)
	if err == sql.ErrNoRows {
		node.contentKey.GenerateKey()
		bs := node.contentKey.ToB64()
		transactExec(c, "INSERT INTO config VALUES( `contentkey`, $1 );", bs)
	} else if err != nil {
		log.Fatal(err.Error())
	} else {
		err = node.contentKey.FromB64(s)
		if err != nil {
			log.Fatal(err.Error())
		}
	}
	// Routing Key Setup
	r2 := transactQueryRow(c, "SELECT * FROM config WHERE name == `routingkey`;")
	if err := r2.Scan(&n, &s); err == sql.ErrNoRows {
		node.routingKey.GenerateKey()
		bs := node.routingKey.ToB64()
		transactExec(c, "INSERT INTO config VALUES( `routingkey`, $1 );", bs)
	} else if err != nil {
		log.Fatal(err.Error())
	} else {
		err = node.routingKey.FromB64(s)
		if err != nil {
			log.Fatal(err.Error())
		}
	}
	node.refreshChannels(c)
	return node.db
}

// Channels

// In : Returns the In channel of this node
func (node *Node) In() chan api.Msg {
	return node.in
}

// Out : Returns the In channel of this node
func (node *Node) Out() chan api.Msg {
	return node.out
}

// Err : Returns the In channel of this node
func (node *Node) Err() chan api.Msg {
	return node.err
}

// Debug

// GetDebug : Returns the debug mode status of this node
func (node *Node) GetDebug() bool {
	return node.debugMode
}

// SetDebug : Sets the debug mode status of this node
func (node *Node) SetDebug(mode bool) {
	node.debugMode = mode
}
