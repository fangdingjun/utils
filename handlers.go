package utils

import (
	"crypto/sha256"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
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

// Write implements http.ResponseWriter.
func (w *loggingWriter) Write(buf []byte) (int, error) {
	if !w.statusWrote {
		w.WriteHeader(200)
	}
	w.size += len(buf)
	return w.w.Write(buf)
}

var _ http.ResponseWriter = &loggingWriter{}

// LoggingHandler is http server middleware to log http request
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

// RecoveryHandler is http server middleware to recover from panic and log the call stack
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

// AllowMethods is http server middleware to limit http request methods
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

// FileUploadConfig config the file upload handler
type FileUploadConfig struct {
	AllowExts   []string // allowed file to upload, example  .jpg, .png
	StoragePath string   // path to store the uploaded file

	// ResponseFunc a function to write the response to user
	ResponseFunc func(w http.ResponseWriter, filenames []string, err error)
}

func storeFile(hdr *multipart.FileHeader, cfg *FileUploadConfig) (string, error) {
	fp, err := hdr.Open()
	if err != nil {
		log.Errorln(err)
		return "", err
	}
	defer fp.Close()

	fpTmp, err := os.CreateTemp("", "tmp-upload-")
	if err != nil {
		log.Errorln(err)
		return "", err
	}
	tmpname := fpTmp.Name()

	defer func() {
		fpTmp.Close()
		os.Remove(tmpname)
	}()

	h := sha256.New()
	buf := make([]byte, 4096)

	for {
		n, err := fp.Read(buf)
		if err != nil {
			if err != io.EOF {
				log.Errorln(err)
				return "", err
			}
			break
		}
		_, err = fpTmp.Write(buf[:n])
		if err != nil {
			log.Errorln(err)
			return "", err
		}
		h.Write(buf[:n])
	}

	_dstName := fmt.Sprintf("upload_%x%s", h.Sum(nil), filepath.Ext(hdr.Filename))
	name := filepath.Join(cfg.StoragePath, _dstName)
	if _, err = os.Stat(name); err == nil {
		// file exists, ignore
		return name, nil
	}

	fp1, err := os.Create(name)
	if err != nil {
		log.Errorln(err)
		return "", err
	}
	defer fp1.Close()

	fpTmp.Seek(0, io.SeekStart)
	io.Copy(fp1, fpTmp)

	return name, nil
}

// FileUploadHandler is http handler to handler file upload
func FileUploadHandler(cfg FileUploadConfig) http.Handler {
	if cfg.ResponseFunc == nil {
		cfg.ResponseFunc = func(w http.ResponseWriter, a []string, err error) {
			if err != nil {
				http.Error(w, "", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(200)
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseMultipartForm(10 * 1024 * 1024)
		if err != nil {
			log.Errorln(err)
			cfg.ResponseFunc(w, nil, err)
			return
		}

		names := []string{}
		for name, hdrs := range r.MultipartForm.File {
			log.Debugf("name %s", name)
			for _, hdr := range hdrs {
				log.Debugf("filename %s", hdr.Filename)

				if len(cfg.AllowExts) > 0 {
					allow := false
					ext := filepath.Ext(hdr.Filename)
					for _, ext1 := range cfg.AllowExts {
						if ext1 == ext {
							allow = true
						}
					}
					if !allow {
						cfg.ResponseFunc(w, nil, fmt.Errorf("upload %s is not allowed", ext))
						return
					}
				}

				name, err := storeFile(hdr, &cfg)
				if err != nil {
					cfg.ResponseFunc(w, nil, err)
					return
				}
				names = append(names, name)
			}
		}
		cfg.ResponseFunc(w, names, nil)
		return
	})
}
