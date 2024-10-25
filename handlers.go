package utils

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	log "github.com/fangdingjun/go-log/v5"
)

type loggingWriter struct {
	w           http.ResponseWriter
	size        int
	statusWrote bool
	status      int
}

// Header implements http.ResponseWriter.
func (w *loggingWriter) Header() http.Header {
	return w.w.Header()
}

// WriteHeader implements http.ResponseWriter.
func (w *loggingWriter) WriteHeader(statusCode int) {
	if w.statusWrote {
		return
	}
	w.statusWrote = true
	w.status = statusCode
	w.w.WriteHeader(statusCode)
}

func (w *loggingWriter) Write(buf []byte) (int, error) {
	if !w.statusWrote {
		w.WriteHeader(200)
	}
	w.size += len(buf)
	return w.w.Write(buf)
}

var _ http.ResponseWriter = &loggingWriter{}

func LoggingHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writer := &loggingWriter{w: w}
		t1 := time.Now()
		next.ServeHTTP(writer, r)
		t2 := time.Now()
		log.Infof("%s %s %s %d %d \"%s\" \"%s\" %s",
			r.Method, r.RequestURI, r.Proto,
			writer.status,
			writer.size,
			r.Referer(),
			r.UserAgent(),
			t2.Sub(t1))
	})
}

func RecoveryHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Errorf("%s, %s", err, debug.Stack())
				http.Error(w, "", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func AllowMethods(methods []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodOptions {
				w.Header().Add("Allow", fmt.Sprintf("OPTIONS, %s", strings.Join(methods, " ,")))
				w.WriteHeader(200)
				return
			}
			for _, method := range methods {
				if r.Method == method {
					next.ServeHTTP(w, r)
					return
				}
			}
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		})
	}
}
