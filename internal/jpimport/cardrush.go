package jpimport

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"
)

type Item struct {
	Set    string `json:"set"`
	Number int    `json:"num"`
	Rarity string `json:"rarity"`
	Yen    int    `json:"yen"`
	Stock  int    `json:"stock"`
	NameJP string `json:"nameJP"`
	URL    string `json:"url"`
}

type Capture struct {
	CapturedAt time.Time `json:"capturedAt"`
	Items      []Item    `json:"items"`
}

func LoadCapture(path string) (Capture, time.Time, error) {
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return Capture{}, time.Time{}, nil
	}
	if err != nil {
		return Capture{}, time.Time{}, fmt.Errorf("cardrush stat: %w", err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return Capture{}, time.Time{}, fmt.Errorf("cardrush read: %w", err)
	}
	var c Capture
	if err := json.Unmarshal(body, &c); err != nil {
		return Capture{}, time.Time{}, fmt.Errorf("cardrush decode: %w", err)
	}
	return c, info.ModTime(), nil
}
