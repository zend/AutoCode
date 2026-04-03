package llm

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// TrafficLogger logs LLM requests and responses to a file
type TrafficLogger struct {
	file    *os.File
	mu      sync.Mutex
	enabled bool
}

var (
	globalLogger *TrafficLogger
	once         sync.Once
)

// InitTrafficLogger initializes the global traffic logger
func InitTrafficLogger(logPath string) error {
	var err error
	once.Do(func() {
		// Ensure directory exists
		dir := filepath.Dir(logPath)
		if dir != "" && dir != "." {
			os.MkdirAll(dir, 0755)
		}

		file, openErr := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if openErr != nil {
			err = openErr
			return
		}

		globalLogger = &TrafficLogger{
			file:    file,
			enabled: true,
		}
	})
	return err
}

// LogRequest logs a request to the traffic log
func LogRequest(provider, model string, req interface{}) {
	if globalLogger == nil || !globalLogger.enabled {
		return
	}

	globalLogger.mu.Lock()
	defer globalLogger.mu.Unlock()

	entry := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"type":      "request",
		"provider":  provider,
		"model":     model,
		"payload":   req,
	}

	data, _ := json.MarshalIndent(entry, "", "  ")
	globalLogger.file.WriteString(string(data) + "\n\n")
}

// LogResponse logs a response to the traffic log
func LogResponse(provider, model string, resp interface{}, duration time.Duration) {
	if globalLogger == nil || !globalLogger.enabled {
		return
	}

	globalLogger.mu.Lock()
	defer globalLogger.mu.Unlock()

	entry := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"type":      "response",
		"provider":  provider,
		"model":     model,
		"duration":  duration.String(),
		"payload":   resp,
	}

	data, _ := json.MarshalIndent(entry, "", "  ")
	globalLogger.file.WriteString(string(data) + "\n\n")
}

// LogStreamEvent logs a streaming event
func LogStreamEvent(provider, model string, event interface{}) {
	if globalLogger == nil || !globalLogger.enabled {
		return
	}

	globalLogger.mu.Lock()
	defer globalLogger.mu.Unlock()

	entry := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"type":      "stream_event",
		"provider":  provider,
		"model":     model,
		"payload":   event,
	}

	data, _ := json.MarshalIndent(entry, "", "  ")
	globalLogger.file.WriteString(string(data) + "\n")
}

// LogError logs an error
func LogError(provider, model string, err error) {
	if globalLogger == nil || !globalLogger.enabled {
		return
	}

	globalLogger.mu.Lock()
	defer globalLogger.mu.Unlock()

	entry := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"type":      "error",
		"provider":  provider,
		"model":     model,
		"error":     err.Error(),
	}

	data, _ := json.MarshalIndent(entry, "", "  ")
	globalLogger.file.WriteString(string(data) + "\n\n")
}

// Close closes the traffic logger
func CloseTrafficLogger() {
	if globalLogger != nil && globalLogger.file != nil {
		globalLogger.file.Close()
	}
}