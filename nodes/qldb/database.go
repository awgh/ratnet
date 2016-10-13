package qldb

import (
	"database/sql"
	"log"
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
