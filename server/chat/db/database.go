package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

func GetConnection(source string) (*sql.DB, error) {
	return initDs(source)
}

func initDs(source string) (*sql.DB, error) {

	var conn *sql.DB
	filePath := source

	if filePath == "" {
		return nil, fmt.Errorf("Server db file='" + filePath + "' not specified.")
	}

	// Create file, path if not exists
	_, err := os.Stat(filePath)
	if err != nil {

		dir := filepath.Dir(filePath)
		// Create directory path recursively if it doesn't exist
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			return nil, fmt.Errorf("failed to create server file directory path: " + err.Error())
		}
		// Create empty file if it doesn't exist
		file, err := os.Create(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to create server file: " + err.Error())
		}
		defer file.Close()

		conn, err = sql.Open("sqlite3", filePath)
		if err != nil {
			return nil, err
		}

		createDbTables(conn)

	} else {

		conn, err = sql.Open("sqlite3", filePath)
		if err != nil {
			return nil, err
		}
	}

	return conn, nil
}

func createDbTables(conn *sql.DB) error {

	sqlStmt := `CREATE TABLE IF NOT EXISTS channel (
			id VARCHAR(255) NOT NULL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			private TINYINT NULL
		);`

	_, err := conn.Exec(sqlStmt)
	if err != nil {
		return fmt.Errorf("%q: %s", err.Error(), sqlStmt)
	}

	sqlStmt = `CREATE TABLE IF NOT EXISTS subscriber (
			id VARCHAR(255) NOT NULL PRIMARY KEY,
			name VARCHAR(255) NOT NULL
		);`

	_, err = conn.Exec(sqlStmt)
	if err != nil {
		return fmt.Errorf("%q: %s", err.Error(), sqlStmt)
	}

	return nil
}
