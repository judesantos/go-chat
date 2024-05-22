package db

import (
	"database/sql"
	"fmt"
	"yt/chat/lib/config"

	_ "github.com/jackc/pgx/v4/stdlib"
)

func GetConnection() (*sql.DB, error) {

	psqlinfo := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		config.GetValue("DB_USER"),
		config.GetValue("DB_PASSWORD"),
		config.GetValue("DB_HOST"),
		config.GetValue("DB_PORT"),
		config.GetValue("DB_NAME"),
	)

	return sql.Open("pgx", psqlinfo)
}
