package mocks

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func NewMockGithubAPI(t *testing.T, cfg *ApiConfig, createdAt time.Time) *httptest.Server {
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
			_, _ = w.Write([]byte(`{"description":"test repo"}`))
		case cfg.ServerUrl + cfg.OkPath + "/pulls":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[]`))
		case cfg.ServerUrl + cfg.OkPath + "/issues":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[
				{
					"created_at":"` + createdAt.Format(time.RFC3339) + `",
					"title":"title",
					"body":"` + cfg.Body + `",
					"user":{"login":"name"}
				}
			]`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}
