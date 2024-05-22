package main

import (
	"database/sql"
	"fmt"
	"yt/chat/lib/db"
	"yt/chat/lib/utils/log"
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

	return nil
}

var logger = log.GetLogger()

func main() {

	// Setup database
	conn, err := db.GetConnection()

	if err != nil {

		logger.Error("Get DB connection failed: " + err.Error())
		return

	} else if err := createDbTables(conn); err != nil {

		logger.Error("Create tables failed: " + err.Error())

	} else {

		log.GetLogger().Info("Create tables success!")

	}

	conn.Close()
	logger.Stop()

}
