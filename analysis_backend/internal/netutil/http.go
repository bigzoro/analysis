package netutil

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func GetJSON(ctx context.Context, u string, out any) error {
	req, _ := http.NewRequestWithContext(ctx, "GET", u, nil)
	req.Header.Set("User-Agent", "por-collector")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GET %s => %d: %s", u, resp.StatusCode, string(b))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func PostJSON(ctx context.Context, u string, body any, out any) error {
	bs, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, "POST", u, bytes.NewReader(bs))
	req.Header.Set("User-Agent", "por-collector")
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("POST %s => %d: %s", u, resp.StatusCode, string(b))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
