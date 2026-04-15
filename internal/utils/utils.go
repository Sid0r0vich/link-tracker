package utils

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"
)

func CheckLink(url string) error {
	req, err := http.NewRequest("Get", url, nil)
	if err != nil {
		return fmt.Errorf("making request to bot: %w", err)
	}

	client := http.Client{Timeout: time.Second * 10}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("status code not found")
	}

	return nil
}

func CutDescription(description string, maxLen int) string {
	return description[:min(len(description), maxLen)]
}

func IsNetError(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, os.ErrDeadlineExceeded) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	return false
}
