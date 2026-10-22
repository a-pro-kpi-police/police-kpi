package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
)

func logInfof(format string, args ...any) {
	_ = log.Output(2, "INFO "+fmt.Sprintf(format, args...))
}

func logDebugf(format string, args ...any) {
	if !debugLoggingEnabled() {
		return
	}
	_ = log.Output(2, "DEBUG "+fmt.Sprintf(format, args...))
}

func logErrorf(format string, args ...any) {
	_ = log.Output(2, "ERROR "+fmt.Sprintf(format, args...))
}

func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			value := recover()
			if value == nil {
				return
			}
			logErrorf("panic method=%s path=%s value=%v", r.Method, r.URL.Path, value)
			logDebugf("panic stack:\n%s", string(debug.Stack()))
			writeError(w, http.StatusInternalServerError, "Внутренняя ошибка сервера")
		}()
		next.ServeHTTP(w, r)
	})
}

func debugLoggingEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("KPI_DEBUG"))) {
	case "1", "true", "yes", "y", "on", "debug":
		return true
	default:
		return false
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status      int
	bytes       int
	wroteHeader bool
}

func newStatusRecorder(w http.ResponseWriter) *statusRecorder {
	return &statusRecorder{ResponseWriter: w, status: http.StatusOK}
}

func (r *statusRecorder) WriteHeader(status int) {
	if r.wroteHeader {
		return
	}
	r.status = status
	r.wroteHeader = true
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(data []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	n, err := r.ResponseWriter.Write(data)
	r.bytes += n
	return n, err
}

func (r *statusRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}
