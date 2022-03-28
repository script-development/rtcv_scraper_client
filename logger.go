package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

const logfileName = "scraper_client.log"

func openLogFile() (*os.File, error) {
	return os.OpenFile(logfileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
}

var truthyStringValues = map[string]bool{
	"1":    true,
	"t":    true,
	"true": true,
}

var logFileEnabled = truthyStringValues[strings.ToLower(os.Getenv("ENABLE_SCRAPER_CLIENT_LOG"))]

// DebugToLogfile appends a new debug line to the logfile
func DebugToLogfile(a ...interface{}) {
	if !logFileEnabled {
		return
	}

	f, err := openLogFile()
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return
		}
		_, err = os.Create(logfileName)
		if err != nil {
			return
		}
		f, err = openLogFile()
		if err != nil {
			return
		}
	}

	f.WriteString(fmt.Sprintln(a...))
	f.Close()
}
