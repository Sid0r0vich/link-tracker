package mocks

import "time"

type ApiConfig struct {
	ServerUrl           string
	OkPath              string
	TimeoutPath         string
	FailPath            string
	Body                string
	Timeout             time.Duration
	ExpectedStatusCodes []int
	RequestCount        int
}
