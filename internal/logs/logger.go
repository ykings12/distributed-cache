package logs

import (
	"sync"
	"time"
)

type Level string

const (
	INFO  Level = "INFO"
	WARN  Level = "WARN"
	ERROR Level = "ERROR"
	DEBUG Level = "DEBUG"
)

// levelPriority defines the priority of each log level
// higher value= more severe
var levelPriority = map[Level]int{
	DEBUG: 1,
	INFO:  2,
	WARN:  3,
	ERROR: 4,
}

type Entry struct {
	TimeStamp time.Time `json:"timestamp"`
	Level     Level     `json:"level"`
	Message   string    `json:"message"`
}

type Logger struct {
	mu      sync.Mutex
	entries []Entry
	maxSize int
	level   Level
}

// level: minimum log level to record(e.g., INFO, WARN, ERROR,DEBUG)
//
//maxsize:maximum number of log entries kept in memory
func NewLogger(maxSize int, level Level) *Logger {
	return &Logger{
		entries: make([]Entry, 0, maxSize),
		maxSize: maxSize,
		level:   level,
	}
}

// log is the internal logging function
// it applies level filtering and ring buffer behavior
func (l *Logger) log(level Level, msg string) {
	//filter logds below the current level
	if levelPriority[level] < levelPriority[l.level] {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.entries) >= l.maxSize {
		//remove oldest entry(ring behavior	)
		l.entries = l.entries[1:]
	}

	l.entries = append(l.entries, Entry{
		TimeStamp: time.Now(),
		Level:     level,
		Message:   msg,
	})
}

func (l *Logger) Debug(msg string) {
	l.log(DEBUG, msg)
}

func (l *Logger) Info(msg string) {
	l.log(INFO, msg)
}

func (l *Logger) Warn(msg string) {
	l.log(WARN, msg)
}

func (l *Logger) Error(msg string) {
	l.log(ERROR, msg)
}

func (l *Logger) GetLast(n int) []Entry {
	l.mu.Lock()
	defer l.mu.Unlock()

	if n > len(l.entries) {
		out := make([]Entry, len(l.entries))
		copy(out, l.entries)
		return out
	}

	start := len(l.entries) - n
	out := make([]Entry, n)
	copy(out, l.entries[start:])
	return out
}
