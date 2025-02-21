package constants

import (
	"fmt"
	"os"
	"time"
)

type DBConfig struct {
	Host     string
	Port     string
	Name     string
	User     string
	Password string
}

var DB = DBConfig{
	Host:     os.Getenv("DB_HOST"),
	Port:     os.Getenv("DB_PORT"),
	Name:     os.Getenv("DB_NAME"),
	User:     os.Getenv("DB_USER"),
	Password: os.Getenv("DB_PASSWORD"),
}

var (
	DatabaseConnString = fmt.Sprintf("server=%s;user id=%s;password=%s;port=%s;database=%s;", DB.Host, DB.User, DB.Password, DB.Port, DB.Name)
	SMTPHost           = os.Getenv("MAIL_HOST")
	SMTPPort           = os.Getenv("MAIL_PORT")
	SMTPUser           = os.Getenv("MAIL_USER")
	SMTPPass           = os.Getenv("MAIL_PASS")
)

const (
	MaxRetries         = 3
	HTTPRequestTimeout = 30 * time.Second
)

const (
	Healthy = iota
	Escalation
	Acknowledged
	Degraded
	UnknownStatus
)
