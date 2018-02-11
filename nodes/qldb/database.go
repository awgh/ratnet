package qldb

import (
	"database/sql"
	"log"
	"strconv"
)

func outboxBulkInsert(db *sql.DB, channelName string, timestamp int64, msgs [][]byte) {
	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err.Error())
	}
	args := make([]interface{}, 1+(2*len(msgs)))
	args[0] = channelName
	//args[1] = timestamp
	idx := 2                                                    // starting 1-based index for 2nd arg
	sql := "INSERT INTO outbox(channel, msg, timestamp) VALUES" //($1,$2, $3);
	for i, v := range msgs {
		//sql += "($1,$" + strconv.Itoa(i+3) + ", $2)"
		sql += "($1,$" + strconv.Itoa(idx) + ", $" + strconv.Itoa(idx+1) + ")"
		if i != len(msgs) {
			sql += ", "
		} else {
			sql += ";"
		}
		args[idx-1] = v // convert to 0-based index here
		args[idx] = timestamp
		timestamp += 1 // increment timestamp by one each message to simplify queueing
		idx += 2
	}
	_, err = tx.Exec(sql, args...)
	if err != nil {
		log.Fatal(err.Error())
	}
	err = tx.Commit()
	if err != nil {
		log.Fatal(err.Error())
	}
}

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
