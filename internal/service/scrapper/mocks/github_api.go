package mocks

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func NewMockGithubAPI(t *testing.T, serverUrl string, createdAt time.Time, body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case serverUrl:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"description":"test repo"}`))
		case serverUrl + "/pulls":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[]`))
		case serverUrl + "/issues":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[
				{
					"created_at":"` + createdAt.Format(time.RFC3339) + `",
					"title":"title",
					"body":"` + body + `",
					"user":{"login":"name"}
				}
			]`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}
