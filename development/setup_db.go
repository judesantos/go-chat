package main

import (
	"database/sql"
	"fmt"
	"os"
	"yt/chat/lib/db"
)

func createDbTables(conn *sql.DB) error {

	sqlStmt := `CREATE TABLE IF NOT EXISTS channel (
			id VARCHAR(255) NOT NULL PRIMARY KEY,
			name VARCHAR(255) UNIQUE NOT NULL,
			private INT NULL
		);`

	_, err := conn.Exec(sqlStmt)
	if err != nil {
		return fmt.Errorf("%q: %s", err.Error(), sqlStmt)
	}

	sqlStmt = `CREATE TABLE IF NOT EXISTS subscriber (
			id VARCHAR(255) NOT NULL PRIMARY KEY,
			name VARCHAR(255) UNIQUE NOT NULL
		);`

	_, err = conn.Exec(sqlStmt)
	if err != nil {
		return fmt.Errorf("%q: %s", err.Error(), sqlStmt)
	}

	sqlStmt = `CREATE TABLE IF NOT EXISTS transient (
			id VARCHAR(255) NOT NULL PRIMARY KEY,
			name VARCHAR(255) UNIQUE NOT NULL
		);`

	_, err = conn.Exec(sqlStmt)
	if err != nil {
		return fmt.Errorf("%q: %s", err.Error(), sqlStmt)
	}

	return nil
}

func main() {

	// Setup database
	conn, err := db.GetConnection()
	defer func() {
		conn.Close()
	}()

	if err != nil {

		fmt.Println("Get DB connection failed: ", err.Error())
		os.Exit(-1)

	} else if err := createDbTables(conn); err != nil {

		fmt.Println("Create tables failed: " + err.Error())
		os.Exit(-2)

	} else {

		os.Exit(0)

	}

}
