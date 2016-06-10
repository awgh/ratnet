package ratnet

import (
	"database/sql"
	"log"
	"time"
)

func transactExec(db *sql.DB, sql string, params ...interface{}) {
	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err.Error())
	}
	//log.Println(sql, params)
	_, err = tx.Exec(sql, params...)
	if err != nil {
		log.Fatal(err.Error())
	}
	err = tx.Commit()
	if err != nil {
		log.Fatal(err.Error())
	}
}
func transactQuery(db *sql.DB, sql string, params ...interface{}) *sql.Rows {
	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err.Error())
	}
	//log.Println(sql, params)
	r, err := tx.Query(sql, params...)
	if err != nil {
		log.Fatal(err.Error())
	}
	err = tx.Commit()
	if err != nil {
		log.Fatal(err.Error())
	}
	return r
}
func transactQueryRow(db *sql.DB, sql string, params ...interface{}) *sql.Row {
	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err.Error())
	}
	//log.Println(sql, params)
	r := tx.QueryRow(sql, params...)
	err = tx.Commit()
	if err != nil {
		log.Fatal(err.Error())
	}
	return r
}

func FlushOutbox(db *sql.DB, maxAgeSeconds int64) {
	c := time.Now().UnixNano()
	c = c - (maxAgeSeconds * 1000000000)
	sql := "DELETE FROM outbox WHERE timestamp < ($1);"

	// todo: below does not work on android/arm, investigate
	//sql := "DELETE FROM outbox WHERE since(timestamp) > duration(\"" +
	//	strconv.FormatInt(maxAgeSeconds, 10) + "s\");"
	//log.Println("Flushed Database (seconds): ", maxAgeSeconds)
	transactExec(db, sql, c)
}

var DB func() *sql.DB

func init() {
	DB = nil
}

// BootstrapDB - Initialize or open a database file
func BootstrapDB(database string) func() *sql.DB {

	if DB != nil {
		return DB
	}

	DB = func() *sql.DB {
		//log.Println("db: " + database)  //todo: why does this trigger so much?
		c, err := sql.Open("ql", database)
		if err != nil {
			log.Fatal("DB Init:")
			log.Fatal(err.Error())
		}
		return c
	}

	// One-time Initialization
	c := DB()
	defer c.Close()

	transactExec(c, `
		CREATE TABLE IF NOT EXISTS destinations ( 			
            name    string          NOT NULL,                                    
            cpubkey string          NOT NULL 
        );
		CREATE UNIQUE INDEX IF NOT EXISTS destid ON destinations (id());
	`)

	transactExec(c, `
		CREATE TABLE IF NOT EXISTS channels ( 			
            name    string          NOT NULL,                                    
            privkey string          NOT NULL 
        );
	`)

	transactExec(c, `
		CREATE TABLE IF NOT EXISTS config ( 
            name    string          NOT NULL,            
            value   string          NOT NULL 
        );`)

	/*  timestamp field must stay int64 and not time type,
	due to a unknown bug only on android/arm in cznic/ql via sql driver
	*/
	transactExec(c, `
		CREATE TABLE IF NOT EXISTS outbox (		    	    
			channel		string			DEFAULT "",  
		    msg         string          NOT NULL,
		    timestamp   int64           NOT NULL
		);`)

	transactExec(c, `
		CREATE TABLE IF NOT EXISTS servers (		    	    
			name		string			NOT NULL,  
		    uri         string          NOT NULL,
		    enabled		bool			NOT NULL,
		    pubkey		string			DEFAULT NULL
		);`)

	transactExec(c, `
		CREATE TABLE IF NOT EXISTS profiles (		    	    
            name    string          NOT NULL,                                    
            privkey string          NOT NULL,
            enabled bool          NOT NULL
		);`)

	// Content Key Setup
	r1 := transactQueryRow(c, "SELECT * FROM config WHERE name == `contentkey`;")
	var n, s string
	err := r1.Scan(&n, &s)

	if err == sql.ErrNoRows {
		contentCrypt.GenerateKey()
		bs := contentCrypt.B64fromPrivateKey()
		transactExec(c, "INSERT INTO config VALUES( `contentkey`, $1 );", bs)
	} else if err != nil {
		log.Fatal(err.Error())
	} else {
		err = contentCrypt.B64toPrivateKey(s)
		if err != nil {
			log.Fatal(err.Error())
		}
	}

	// Routing Key Setup
	r2 := transactQueryRow(c, "SELECT * FROM config WHERE name == `routingkey`;")
	err = r2.Scan(&n, &s)
	if err == sql.ErrNoRows {
		routingCrypt.GenerateKey()
		bs := routingCrypt.B64fromPrivateKey()
		transactExec(c, "INSERT INTO config VALUES( `routingkey`, $1 );", bs)
	} else if err != nil {
		log.Fatal(err.Error())
	} else {
		err = routingCrypt.B64toPrivateKey(s)
		if err != nil {
			log.Fatal(err.Error())
		}
	}

	refreshChannels(c)
	return DB
}
