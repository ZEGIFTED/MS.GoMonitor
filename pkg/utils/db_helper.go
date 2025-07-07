package utils

import (
	"database/sql"
	"log/slog"

	"github.com/ZEGIFTED/MS.GoMonitor/pkg/constants"
	_ "github.com/lib/pq" // PostgreSQL driver
)

func DatabaseConnection() *sql.DB {
	// Database configuration
	connString := constants.DatabaseConnString

	db, err := sql.Open("postgres", connString)

	if err != nil {
		slog.Error("Error creating connection pool: ", "Error", err.Error())
	}

	// Check if the database is open and can be pinged
	err = db.Ping()
	if err != nil {
		slog.Error("Error connecting to the database with Connection String %s. Error: %s", connString, err.Error())
	}

	return db
}
