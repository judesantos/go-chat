package main

import (
	"database/sql"
	"fmt"
	"os"
	"yt/chat/lib/db"
)

func createDbTables(conn *sql.DB) error {

	sqlStmt := `CREATE TABLE IF NOT EXISTS channel (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) UNIQUE NOT NULL,
			private INT NULL,
			created TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`

	_, err := conn.Exec(sqlStmt)
	if err != nil {
		return fmt.Errorf("%q: %s", err.Error(), sqlStmt)
	}

	sqlStmt = `CREATE TABLE IF NOT EXISTS subscriber (
			id SERIAL PRIMARY KEY,
			name VARCHAR(50) UNIQUE NOT NULL,
			password VARCHAR(255) NOT NULL,
			email VARCHAR(100) UNIQUE NOT NULL,
			created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`

	_, err = conn.Exec(sqlStmt)
	if err != nil {
		return fmt.Errorf("%q: %s", err.Error(), sqlStmt)
	}

	sqlStmt = `CREATE TABLE IF NOT EXISTS transient (
			id SERIAL PRIMARY KEY,
			name VARCHAR(50) UNIQUE NOT NULL,
			email VARCHAR(100) UNIQUE NOT NULL,
			created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP
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
