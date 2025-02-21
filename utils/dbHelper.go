package utils

import (
	"database/sql"
	"fmt"
	"log"
)

type DBConfig struct {
	Host     string
	Port     string
	Name     string
	User     string
	Password string
}

func DatabaseConnection() *sql.DB {
	// Database configuration
	var DB = DBConfig{
		Host:     GetEnvWithDefault("DB_HOST", "localhost"),
		Port:     GetEnvWithDefault("DB_PORT", "1433"),
		Name:     GetEnvWithDefault("DB_NAME", "MS"),
		User:     GetEnvWithDefault("DB_USER", "sa"),
		Password: GetEnvWithDefault("DB_PASSWORD", "dbuser123$"),
	}

	connString := fmt.Sprintf("server=%s;user id=%s;password=%s;port=%s;database=%s;",
		DB.Host, DB.User, DB.Password, DB.Port, DB.Name)

	db, err := sql.Open("sqlserver", connString)
	if err != nil {
		log.Fatal("Error creating connection pool: ", err.Error())
	}

	// Check if the database is open and can be pinged
	err = db.Ping()
	if err != nil {
		log.Fatal("Error connecting to the database: ", err.Error())
	}

	return db
}
