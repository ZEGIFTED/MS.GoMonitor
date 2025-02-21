package utils

import (
	"log"
	"os"
)

var Logger *log.Logger
var CronLogger *log.Logger

func init() {
	Logger = log.New(os.Stdout, "MS-SVC_MONITOR: ", log.Ldate|log.Ltime|log.Lshortfile)

	CronLogger = log.New(os.Stdout, "MS-CRON_SVC_MONITOR: ", log.Ldate|log.Ltime|log.Lshortfile)
}
