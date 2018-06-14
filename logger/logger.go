package logger

import (
	"log"
	"os"
)

var StdLog *log.Logger
var ErrLog *log.Logger

func NewLogger(fileName string) *log.Logger {
	logFile, _ := os.OpenFile("/var/log/"+fileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	l := log.New(logFile, "", log.Ldate|log.Ltime)
	return l
}
