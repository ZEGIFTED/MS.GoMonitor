package utils

import (
	"database/sql"
	"log"

	"github.com/ZEGIFTED/MS.GoMonitor/pkg/constants"
)

func DatabaseConnection() *sql.DB {
	// Database configuration
	connString := constants.DatabaseConnString

	db, err := sql.Open("sqlserver", connString)
	if err != nil {
		log.Println("Error creating connection pool: ", err.Error())
	}

	// Check if the database is open and can be pinged
	err = db.Ping()
	if err != nil {
		log.Printf("Error connecting to the database with Connection String %s. Error: %s", connString, err.Error())
	}

	return db
}
