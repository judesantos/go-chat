package log

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	helpers "yt/chatbot/lib/utils"
	"yt/chatbot/server/chat/config"

	"github.com/petermattis/goid"
	"github.com/sirupsen/logrus"
)

var (
	instance *Logger
	once     sync.Once
)

// LogFormat - logger service formatter
type LogFormat struct{}

func (m LogFormat) Format(entry *logrus.Entry) ([]byte, error) {
	return []byte(fmt.Sprintf(
			"[%s] %s %s\n",
			entry.Time.Format("2006-01-02 15:04:05.000"),
			entry.Level,
			entry.Message,
		)),
		nil
}

// Logger - Thread safe logging service client
type Logger struct {
	logfile        string
	output         string
	stdoutloglevel logrus.Level
	fileloglevel   logrus.Level

	fileLog *logrus.Logger
	stdLog  *logrus.Logger

	fileHdl *os.File

	// synchronization
	traceCh chan string
	debugCh chan string
	infoCh  chan string
	warnCh  chan string
	errorCh chan string
	panicCh chan string
	fatalCh chan string

	done chan struct{}
}

func getInstance() *Logger {
	once.Do(func() {
		instance = &Logger{
			logfile:        config.GetValue("LOG_FILE"),
			output:         config.GetValue("LOG_OUTPUT"),
			fileloglevel:   getLogLevel(config.GetValue("LOG_FILE_LEVEL")),
			stdoutloglevel: getLogLevel(config.GetValue("LOG_CONSOLE_LEVEL")),

			fileLog: nil,
			stdLog:  nil,

			traceCh: make(chan string),
			debugCh: make(chan string),
			infoCh:  make(chan string),
			warnCh:  make(chan string),
			errorCh: make(chan string),
			panicCh: make(chan string),
			fatalCh: make(chan string),

			done: make(chan struct{}),
		}
		instance.initialize()

		//timer := time.NewTimer(10 * time.Millisecond)
		//<-timer.C
	})

	return instance
}

// Get singleton logger
func GetLogger() *Logger {

	return getInstance()
}

func (m *Logger) start() {

	// Use channels to synchronize access to logger
	for {
		select {
		case <-m.done:
			return
		case msg := <-m.traceCh:
			if m.stdLog != nil && m.stdoutloglevel >= logrus.TraceLevel {
				m.stdLog.Trace(msg)
			}
			if m.fileLog != nil && m.fileloglevel >= logrus.TraceLevel {
				m.fileLog.Trace(msg)
			}
		case msg := <-m.debugCh:
			if m.stdLog != nil && m.stdoutloglevel >= logrus.DebugLevel {
				m.stdLog.Debug(msg)
			}
			if m.fileLog != nil && m.fileloglevel >= logrus.DebugLevel {
				m.fileLog.Debug(msg)
			}
		case msg := <-m.warnCh:
			if m.stdLog != nil && m.stdoutloglevel >= logrus.WarnLevel {
				m.stdLog.Warn(msg)
			}
			if m.fileLog != nil && m.fileloglevel >= logrus.WarnLevel {
				m.fileLog.Warn(msg)
			}
		case msg := <-m.infoCh:
			if m.stdLog != nil && m.stdoutloglevel >= logrus.InfoLevel {
				m.stdLog.Info(msg)
			}
			if m.fileLog != nil && m.fileloglevel >= logrus.InfoLevel {
				m.fileLog.Info(msg)
			}
		case msg := <-m.errorCh:
			if m.stdLog != nil && m.stdoutloglevel >= logrus.ErrorLevel {
				m.stdLog.Error(msg)
			}
			if m.fileLog != nil && m.fileloglevel >= logrus.ErrorLevel {
				m.fileLog.Error(msg)
			}
		case msg := <-m.fatalCh:
			if m.stdLog != nil && m.stdoutloglevel >= logrus.FatalLevel {
				m.stdLog.Fatal(msg)
			}
			if m.fileLog != nil && m.fileloglevel >= logrus.FatalLevel {
				m.fileLog.Fatal(msg)
			}
		case msg := <-m.panicCh:
			if m.stdLog != nil && m.stdoutloglevel >= logrus.PanicLevel {
				m.stdLog.Panic(msg)
			}
			if m.fileLog != nil && m.fileloglevel >= logrus.PanicLevel {
				m.fileLog.Panic(msg)
			}
		}
	}
}

func (m *Logger) initialize() {

	// Get output methods
	log_outputs := strings.Split(m.output, ",")
	if len(log_outputs) == 0 || log_outputs[0] == "" {
		return
	}

	for _, log_destination := range log_outputs {

		if log_destination == "stdout" {

			m.stdLog = logrus.New()
			m.stdLog.SetOutput(os.Stdout)
			m.stdLog.SetLevel(m.stdoutloglevel)
			m.stdLog.SetFormatter(&LogFormat{})

		}

		if log_destination == "file" {

			filePath := m.logfile

			if filePath == "" {
				log.Fatal("Log file='" + filePath + "' not specified.")
				os.Exit(-1)
			}
			_, err := os.Stat(filePath)
			if err != nil {
				dir := filepath.Dir(filePath)
				// Create directory path recursively if it doesn't exist
				err = os.MkdirAll(dir, 0755)
				if err != nil {
					log.Fatal("failed to create log file directory path: " + err.Error())
					os.Exit(-1)
				}
				// Create empty file if it doesn't exist
				m.fileHdl, err = os.Create(filePath)
				if err != nil {
					log.Fatal("failed to create log file: ", err.Error())
					os.Exit(-1)
				}
			} else {
				// File logger
				m.fileHdl, err = os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
				if err != nil {
					log.Fatalf("Failed to open log file: %v", err)
					os.Exit(-1)
				}
			}

			m.fileLog = logrus.New()
			m.fileLog.SetOutput(m.fileHdl)
			m.fileLog.SetLevel(m.fileloglevel)
			m.fileLog.SetFormatter(&LogFormat{})
		}
	} // end for-loop

	go m.start()
}

func (m *Logger) Stop() {

	// Kill logger routine
	m.done <- struct{}{}

	if m.fileHdl != nil {
		m.fileHdl.Close()
	}
	close(m.traceCh)
	close(m.debugCh)
	close(m.infoCh)
	close(m.warnCh)
	close(m.errorCh)
	close(m.panicCh)
	close(m.fatalCh)
	close(m.done)

}

func (m *Logger) fmt(msg string) string {
	fn, f, line := helpers.GetCallerInfo(3)
	formatted := fmt.Sprintf("[w=%d] %s::%s(%d) %s", goid.Get(), fn, f, line, msg)
	return formatted
}

func (m *Logger) Debug(msg string) {
	m.debugCh <- m.fmt(msg)
}

func (m *Logger) Info(msg string) {
	m.infoCh <- m.fmt(msg)
}

func (m *Logger) Warn(msg string) {
	m.warnCh <- m.fmt(msg)
}

func (m *Logger) Error(msg string) {
	m.errorCh <- m.fmt(msg)
}

func (m *Logger) Fatal(msg string) {
	m.fatalCh <- m.fmt(msg)
}

func (m *Logger) Panic(msg string) {
	m.panicCh <- m.fmt(msg)
}

func (m *Logger) Trace(msg string) {
	m.traceCh <- m.fmt(msg)
}

func getLogLevel(level string) logrus.Level {
	var logLevel logrus.Level = 0
	switch level {
	case "panic":
		logLevel = logrus.PanicLevel
	case "error":
		logLevel = logrus.ErrorLevel
	case "warn":
	case "warning":
		logLevel = logrus.WarnLevel
	case "info":
		logLevel = logrus.InfoLevel
	case "debug":
		logLevel = logrus.DebugLevel
	case "trace":
		logLevel = logrus.TraceLevel
	default:
		logLevel = logrus.ErrorLevel
	}

	return logLevel
}
