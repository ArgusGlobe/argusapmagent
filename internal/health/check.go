package health

import (
	"fmt"
	"net/http"
	"time"
)

func Check(url string) error {
	client := &http.Client{Timeout: 2 * time.Second}
	res, err := client.Get(url)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("health endpoint returned %s", res.Status)
	}
	return nil
}
