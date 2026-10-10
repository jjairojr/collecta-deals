package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"opdeals/internal/httpx"
	"opdeals/internal/logx"
	"opdeals/internal/tcgplayer"
)

func main() {
	ids := flag.String("ids", "", "comma-separated TCGplayer product ids")
	flag.Parse()

	var productIDs []int
	for _, s := range strings.Split(*ids, ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		id, err := strconv.Atoi(s)
		if err != nil {
			fmt.Fprintln(os.Stderr, "bad id:", s)
			os.Exit(2)
		}
		productIDs = append(productIDs, id)
	}

	client, err := httpx.New(30 * time.Second)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	tcg := tcgplayer.New(client, logx.New(os.Stderr))
	prices := tcg.LowestPrices(context.Background(), productIDs)

	out := make(map[string]tcgplayer.PriceInfo, len(prices))
	for id, p := range prices {
		out[strconv.Itoa(id)] = p
	}
	if err := json.NewEncoder(os.Stdout).Encode(out); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
