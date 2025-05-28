package middleware

import (
	"log"
	"net/http"
	"time"
)

type responseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func wrapResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w}
}

func (rw *responseWriter) Status() int {
	return rw.status
}

func (rw *responseWriter) WriteHeader(code int) {
	if rw.wroteHeader {
		return
	}
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
	rw.wroteHeader = true
}

// LoggingMiddleware logs HTTP request details
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := wrapResponseWriter(w)

		// Log request details
		log.Printf("-> %s %s %s\n", r.Method, r.RequestURI, r.RemoteAddr)

		// Call the next handler
		next.ServeHTTP(wrapped, r)

		// Log response details
		log.Printf("<- %s %s %d %s %v\n",
			r.Method,
			r.RequestURI,
			wrapped.status,
			r.RemoteAddr,
			time.Since(start),
		)
	})
}
