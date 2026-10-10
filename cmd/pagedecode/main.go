package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"opdeals/internal/liga"
)

type result struct {
	Key      string              `json:"key"`
	Listings []liga.StoreListing `json:"listings"`
}

func main() {
	addr := flag.String("addr", "127.0.0.1:8765", "listen address")
	out := flag.String("out", "pages.json", "path to write the collected results")
	flag.Parse()

	var (
		mu      sync.Mutex
		results = map[string][]liga.StoreListing{}
		atlases = map[string][]byte{}
	)

	fetchAtlas := func(url string) []byte {
		mu.Lock()
		cached, ok := atlases[url]
		mu.Unlock()
		if ok {
			return cached
		}
		if len(url) > 1 && url[:2] == "//" {
			url = "https:" + url
		}
		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Get(url)
		if err != nil {
			return nil
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil
		}
		mu.Lock()
		atlases[url] = body
		mu.Unlock()
		return body
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/page", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Private-Network", "true")
		w.Header().Set("Access-Control-Max-Age", "86400")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		key := r.URL.Query().Get("key")
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		listings, err := liga.DecodePage(body, fetchAtlas)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		mu.Lock()
		results[key] = listings
		n := len(results)
		mu.Unlock()
		fmt.Fprintf(os.Stderr, "[%d] %s -> %d ofertas\n", n, key, len(listings))
		json.NewEncoder(w).Encode(map[string]int{"ofertas": len(listings)})
	})
	mux.HandleFunc("/save", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Private-Network", "true")
		mu.Lock()
		defer mu.Unlock()
		buf, err := json.MarshalIndent(results, "", " ")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := os.WriteFile(*out, buf, 0o644); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(os.Stderr, "gravado %d paginas em %s\n", len(results), *out)
		json.NewEncoder(w).Encode(map[string]int{"paginas": len(results)})
	})

	fmt.Fprintf(os.Stderr, "pagedecode ouvindo em %s\n", *addr)
	if err := http.ListenAndServe(*addr, mux); err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		os.Exit(1)
	}
}
