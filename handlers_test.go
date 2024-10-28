package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	log "github.com/fangdingjun/go-log/v5"
)

func TestUploadFile(t *testing.T) {
	handler := FileUploadHandler(FileUploadConfig{
		AllowExts:   []string{".txt"},
		StoragePath: "/tmp",
		ResponseFunc: func(w http.ResponseWriter, names []string, err error) {
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Add("content-type", "application/json")
			d1, _ := json.Marshal(map[string]interface{}{
				"imgs": names,
			})
			w.Write(d1)
		},
	})

	mux := http.NewServeMux()
	mux.Handle("/upload", handler)

	// start temp test server
	server := httptest.NewServer(mux)
	defer server.Close()

	// construct form data
	buf := new(bytes.Buffer)
	mpart := multipart.NewWriter(buf)
	w, err := mpart.CreateFormFile("file", "test.txt")
	if err != nil {
		t.Error(err)
		return
	}
	data := []byte("hello, world\nthis is log test\n")
	w.Write(data)
	mpart.Close()

	//log.Infof("%s", mpart.FormDataContentType())

	uri := fmt.Sprintf("%s/upload", server.URL)
	req, _ := http.NewRequest("POST", uri, buf)
	req.Header.Set("content-type", mpart.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("http error code %d", resp.StatusCode)
		return
	}
	res := map[string][]string{}
	json.NewDecoder(resp.Body).Decode(&res)
	log.Infof("%+v", res)
	fname := res["imgs"][0]

	// check
	d1, err := os.ReadFile(fname)
	if err != nil {
		t.Errorf("upload failed %s", err)
		return
	}
	os.Remove(fname)

	if bytes.Compare(d1, data) != 0 {
		t.Errorf("upload check failed")
		return
	}
}
