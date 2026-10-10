package liga

import "image"

// DecodePage turns a fetched card page into its per-store listings, resolving the
// sprite atlases through fetch. It exists for callers that already hold the HTML
// (a browser session, a saved capture) and only need the decoding half.
func DecodePage(html []byte, fetch func(url string) []byte) ([]StoreListing, error) {
	var qty, price image.Image
	if u := quantAtlasURL(html); u != "" {
		if img, ok := decodeAtlas(fetch(u)); ok {
			qty = img
		}
	}
	if u := priceAtlasURL(html); u != "" {
		if img, ok := decodeAtlas(fetch(u)); ok {
			price = img
		}
	}
	return parseCardStock(html, qty, price)
}
