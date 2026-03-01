package release

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
	LogLevelFatal
)

type ReleaseStage int

const (
	StageAnalysis ReleaseStage = iota
	StageVersionDecision
	StageBuild
	StageTest
	StageDeploy
	StageRollback
	StageComplete
)

type LogEntry struct {
	Timestamp time.Time     `json:"timestamp"`
	Level     LogLevel      `json:"level"`
	Stage     ReleaseStage  `json:"stage"`
	Message   string        `json:"message"`
	Details   interface{}   `json:"details,omitempty"`
	Error     string        `json:"error,omitempty"`
	ReleaseID string        `json:"releaseId"`
	Duration  time.Duration `json:"duration,omitempty"`
}

type ReleaseLogger struct {
	mu           sync.Mutex
	file         *os.File
	config       LoggingConfig
	entries      []LogEntry
	currentStage ReleaseStage
	releaseID    string
	startTime    time.Time
}

type ReleaseError struct {
	Code        string
	Stage       ReleaseStage
	Message     string
	OriginalErr error
	Recoverable bool
	Timestamp   time.Time
}

func NewReleaseLogger(config LoggingConfig, releaseID string) (*ReleaseLogger, error) {
	logger := &ReleaseLogger{
		config:    config,
		entries:   make([]LogEntry, 0),
		releaseID: releaseID,
		startTime: time.Now(),
	}

	if config.OutputPath != "" {
		if err := logger.initFileOutput(); err != nil {
			return nil, fmt.Errorf("failed to initialize log file: %w", err)
		}
	}

	return logger, nil
}

func (l *ReleaseLogger) initFileOutput() error {
	dir := filepath.Dir(l.config.OutputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	file, err := os.OpenFile(l.config.OutputPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	l.file = file
	return nil
}

func (l *ReleaseLogger) SetStage(stage ReleaseStage) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.currentStage = stage
}

func (l *ReleaseLogger) Debug(message string, details interface{}) {
	l.log(LogLevelDebug, message, details, nil)
}

func (l *ReleaseLogger) Info(message string, details interface{}) {
	l.log(LogLevelInfo, message, details, nil)
}

func (l *ReleaseLogger) Warn(message string, details interface{}) {
	l.log(LogLevelWarn, message, details, nil)
}

func (l *ReleaseLogger) Error(message string, err error, details interface{}) {
	l.log(LogLevelError, message, details, err)
}

func (l *ReleaseLogger) Fatal(message string, err error, details interface{}) {
	l.log(LogLevelFatal, message, details, err)
}

func (l *ReleaseLogger) log(level LogLevel, message string, details interface{}, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if level < l.parseLevel(l.config.Level) {
		return
	}

	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Stage:     l.currentStage,
		Message:   message,
		Details:   details,
		ReleaseID: l.releaseID,
		Duration:  time.Since(l.startTime),
	}

	if err != nil {
		entry.Error = err.Error()
	}

	l.entries = append(l.entries, entry)

	l.writeEntry(entry)
}

func (l *ReleaseLogger) writeEntry(entry LogEntry) {
	output := l.formatEntry(entry)

	if l.file != nil {
		l.file.WriteString(output + "\n")
	}

	log.Print(output)
}

func (l *ReleaseLogger) formatEntry(entry LogEntry) string {
	levelStr := l.levelToString(entry.Level)
	stageStr := l.stageToString(entry.Stage)

	base := fmt.Sprintf("[%s] [%s] [%s] [Release:%s] %s",
		entry.Timestamp.Format("2006-01-02 15:04:05"),
		levelStr,
		stageStr,
		entry.ReleaseID,
		entry.Message,
	)

	if entry.Error != "" {
		base += fmt.Sprintf(" | Error: %s", entry.Error)
	}

	if entry.Duration > 0 {
		base += fmt.Sprintf(" | Duration: %v", entry.Duration)
	}

	return base
}

func (l *ReleaseLogger) parseLevel(level string) LogLevel {
	switch level {
	case "debug":
		return LogLevelDebug
	case "info":
		return LogLevelInfo
	case "warn":
		return LogLevelWarn
	case "error":
		return LogLevelError
	case "fatal":
		return LogLevelFatal
	default:
		return LogLevelInfo
	}
}

func (l *ReleaseLogger) levelToString(level LogLevel) string {
	switch level {
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	case LogLevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

func (l *ReleaseLogger) stageToString(stage ReleaseStage) string {
	switch stage {
	case StageAnalysis:
		return "ANALYSIS"
	case StageVersionDecision:
		return "VERSION_DECISION"
	case StageBuild:
		return "BUILD"
	case StageTest:
		return "TEST"
	case StageDeploy:
		return "DEPLOY"
	case StageRollback:
		return "ROLLBACK"
	case StageComplete:
		return "COMPLETE"
	default:
		return "UNKNOWN"
	}
}

func (l *ReleaseLogger) GetEntries() []LogEntry {
	l.mu.Lock()
	defer l.mu.Unlock()

	entries := make([]LogEntry, len(l.entries))
	copy(entries, l.entries)
	return entries
}

func (l *ReleaseLogger) ExportJSON() ([]byte, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	entries := make([]map[string]interface{}, len(l.entries))
	for i, entry := range l.entries {
		entries[i] = map[string]interface{}{
			"timestamp": entry.Timestamp,
			"level":     l.levelToString(entry.Level),
			"stage":     l.stageToString(entry.Stage),
			"message":   entry.Message,
			"details":   entry.Details,
			"error":     entry.Error,
			"releaseId": entry.ReleaseID,
			"duration":  entry.Duration.String(),
		}
	}

	result := map[string]interface{}{
		"releaseId":     l.releaseID,
		"startTime":     l.startTime,
		"totalDuration": time.Since(l.startTime).String(),
		"entries":       entries,
	}

	return json.Marshal(result)
}

func (l *ReleaseLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

func NewReleaseError(code string, stage ReleaseStage, message string, originalErr error, recoverable bool) *ReleaseError {
	return &ReleaseError{
		Code:        code,
		Stage:       stage,
		Message:     message,
		OriginalErr: originalErr,
		Recoverable: recoverable,
		Timestamp:   time.Now(),
	}
}

func (e *ReleaseError) Error() string {
	if e.OriginalErr != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.OriginalErr)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *ReleaseError) Unwrap() error {
	return e.OriginalErr
}

func (e *ReleaseError) IsRecoverable() bool {
	return e.Recoverable
}

type ErrorHandler struct {
	logger   *ReleaseLogger
	handlers map[string]func(*ReleaseError) error
}

func NewErrorHandler(logger *ReleaseLogger) *ErrorHandler {
	return &ErrorHandler{
		logger:   logger,
		handlers: make(map[string]func(*ReleaseError) error),
	}
}

func (h *ErrorHandler) RegisterHandler(code string, handler func(*ReleaseError) error) {
	h.handlers[code] = handler
}

func (h *ErrorHandler) Handle(err *ReleaseError) error {
	h.logger.Error(err.Message, err, map[string]interface{}{
		"code":        err.Code,
		"stage":       err.Stage,
		"recoverable": err.Recoverable,
	})

	if handler, exists := h.handlers[err.Code]; exists {
		return handler(err)
	}

	if err.Recoverable {
		h.logger.Warn("Attempting automatic recovery", map[string]interface{}{
			"code": err.Code,
		})
		return nil
	}

	return err
}

func (h *ErrorHandler) WrapAndHandle(code string, stage ReleaseStage, message string, err error, recoverable bool) error {
	releaseErr := NewReleaseError(code, stage, message, err, recoverable)
	return h.Handle(releaseErr)
}

const (
	ErrCodeVersionParse     = "VERSION_PARSE_ERROR"
	ErrCodeBuildFailed      = "BUILD_FAILED"
	ErrCodeTestFailed       = "TEST_FAILED"
	ErrCodeDeployFailed     = "DEPLOY_FAILED"
	ErrCodeHealthCheck      = "HEALTH_CHECK_FAILED"
	ErrCodeRollbackFailed   = "ROLLBACK_FAILED"
	ErrCodeConfigInvalid    = "CONFIG_INVALID"
	ErrCodeGitOperation     = "GIT_OPERATION_ERROR"
	ErrCodeNetworkError     = "NETWORK_ERROR"
	ErrCodeTimeout          = "TIMEOUT_ERROR"
	ErrCodePermissionDenied = "PERMISSION_DENIED"
)
