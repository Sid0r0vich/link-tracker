package utils

import (
	"fmt"
	"net/http"
	"time"
)

func CheckLink(url string) (bool, error) {
	req, err := http.NewRequest("Get", url, nil)
	if err != nil {
		return false, fmt.Errorf("making request to bot: %w", err)
	}

	client := http.Client{Timeout: time.Second * 10}

	resp, err := client.Do(req)
	if err != nil {
		return false, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	return true, nil
}
