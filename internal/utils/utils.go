package utils

import (
	"fmt"
	"net/http"
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
