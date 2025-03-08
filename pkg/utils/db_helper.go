package utils

import (
	"database/sql"
	"github.com/ZEGIFTED/MS.GoMonitor/pkg/constants"
	"log"
)

func DatabaseConnection() *sql.DB {
	// Database configuration
	connString := constants.DatabaseConnString

	db, err := sql.Open("sqlserver", connString)
	if err != nil {
		log.Fatal("Error creating connection pool: ", err.Error())
	}

	// Check if the database is open and can be pinged
	err = db.Ping()
	if err != nil {
		log.Fatalf("Error connecting to the database with Connection String %s. Error: %s", connString, err.Error())
	}

	return db
}
