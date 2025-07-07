package tools

import (
	"bytes"
	"errors"
	"net/http"
	"time"
)

const url = "http://127.0.0.1:18080/api/v1/app_notify"

func SendDbusMessage(msg string) error {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(msg)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "text/json")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.New("failed to send http message, status: " + resp.Status)
	}
	return nil
}
