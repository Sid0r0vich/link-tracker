package mocks

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"
)

func NewMockStackoverflowAPI(t *testing.T, cfg *ApiConfig, creationDate, lastActivity int64) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case cfg.ServerUrl + cfg.TimeoutPath:
			time.Sleep(cfg.Timeout)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		case cfg.ServerUrl + cfg.FailPath:
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{}`))
		case cfg.ServerUrl + cfg.OkPath:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"items":[
					{"last_activity_date":` + int64ToString(lastActivity) + `,"title":"title"}
				]
			}`))
		case cfg.ServerUrl + cfg.OkPath + "/answers":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"items":[
					{
						"last_activity_date":` + int64ToString(lastActivity) + `,
						"creation_date":` + int64ToString(creationDate) + `,
						"owner":{"display_name":"name"},
						"body":"` + cfg.Body + `"
					}
				]
			}`))
		case cfg.ServerUrl + cfg.OkPath + "/comments":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"items":[]}`))
		default:
			fmt.Fprintf(os.Stderr, "expected URL: %s.found URL: %s", cfg.ServerUrl, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func int64ToString(v int64) string {
	return strconv.FormatInt(v, 10)
}
