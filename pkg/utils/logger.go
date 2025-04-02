package utils

import (
	"io"
	"log"
	"os"

	"github.com/ZEGIFTED/MS.GoMonitor/pkg/constants"
	"github.com/natefinch/lumberjack"
)

var Logger *log.Logger
var CronLogger *log.Logger

type MSSVCLogger struct{}

func init() {
	Logger = log.New(os.Stdout, "MS-SVC_MONITOR: ", log.Ldate|log.Ltime|log.Lshortfile)

	CronLogger = log.New(os.Stdout, "MS-CRON_SVC_MONITOR: ", log.Ldate|log.Ltime|log.Lshortfile)

	// Ensure the log directory exists
	err := os.MkdirAll(constants.LogPath, os.ModePerm)
	if err != nil {
		log.Fatalf("Failed to create log directory: %v", err)
	}

	// Configure the rotating log file
	LogFile := &lumberjack.Logger{
		Filename:   constants.LogFileName, // Custom log file path
		MaxSize:    10,                    // Max megabytes before rotation
		MaxBackups: 5,                     // Max number of old log files to retain
		MaxAge:     30,                    // Max days to retain old log files
		Compress:   true,                  // Compress old log files
	}

	// Create multi-writer to log to both file and console
	multiWriter := io.MultiWriter(os.Stdout, LogFile)

	// Direct logs to lumberjack and Console
	log.SetOutput(multiWriter)
	log.SetPrefix("MS-SVC_MONITOR: ")
	log.SetFlags(log.LstdFlags | log.Lshortfile) // Add timestamp and file info

	// Close the log file when done
	defer func(logFile *lumberjack.Logger) {
		err := logFile.Close()
		if err != nil {
			log.Fatalf("Failed to close log file: %v", err)
		}
	}(LogFile)
}

func (logger *MSSVCLogger) LogHttpErrors(route string, errorS error) {
	log.Panicf("API Route: %s. Error >>> %s", route, errorS.Error())
}
