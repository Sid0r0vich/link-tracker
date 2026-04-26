package mocks

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
)

func NewMockStackoverflowAPI(t *testing.T, url string, creationDate, lastActivity int64, body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case url:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"items":[
					{"last_activity_date":` + int64ToString(lastActivity) + `,"title":"title"}
				]
			}`))
		case url + "/answers":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"items":[
					{
						"last_activity_date":` + int64ToString(lastActivity) + `,
						"creation_date":` + int64ToString(creationDate) + `,
						"owner":{"display_name":"name"},
						"body":"` + body + `"
					}
				]
			}`))
		case url + "/comments":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"items":[]}`))
		default:
			fmt.Fprintf(os.Stderr, "expected URL: %s.found URL: %s", url, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func int64ToString(v int64) string {
	return strconv.FormatInt(v, 10)
}
