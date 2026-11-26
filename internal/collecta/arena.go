package collecta

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const arenaPrefix = "/store/arena/"

type Arena interface {
	Arena(ctx context.Context, method, path, rawQuery string, body []byte) (int, []byte, error)
}

func (c *HTTP) Arena(ctx context.Context, method, path, rawQuery string, body []byte) (int, []byte, error) {
	if !arenaPath(path) {
		return 0, nil, errors.New("collecta: arena path outside " + arenaPrefix)
	}
	target := c.base + path
	if rawQuery != "" {
		target += "?" + rawQuery
	}
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, target, reader)
	if err != nil {
		return 0, nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Store-Token", c.token)
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("collecta: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return 0, nil, fmt.Errorf("collecta: %w", err)
	}
	return resp.StatusCode, data, nil
}

func arenaPath(path string) bool {
	rest, ok := strings.CutPrefix(path, arenaPrefix)
	if !ok || rest == "" {
		return false
	}
	for _, segment := range strings.Split(rest, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return false
		}
	}
	return true
}
