package main

import (
	"os"
	"strconv"
	"strings"

	"opdeals/internal/api"
	"opdeals/internal/checkoutfake"
	"opdeals/internal/logx"
	"opdeals/internal/mercadopago"
	"opdeals/internal/orders"
	"opdeals/internal/shipping"
)

func buildCheckout(ordersPath, storefrontToken string, logger *logx.Logger) (api.Checkout, error) {
	log := logger.WithPrefix("CHECKOUT")
	c := api.Checkout{
		Orders:          orders.NewStore(ordersPath, log),
		StorefrontToken: storefrontToken,
		WebhookSecret:   os.Getenv("MP_WEBHOOK_SECRET"),
		SiteURL:         os.Getenv("PUBLIC_SITE_URL"),
		APIURL:          os.Getenv("PUBLIC_API_URL"),
		FromCEP:         strings.TrimSpace(os.Getenv("SHIP_FROM_CEP")),
		PickupCity:      strings.TrimSpace(os.Getenv("PICKUP_CITY")),
		Log:             log,
	}
	if v := os.Getenv("FREE_SHIPPING_MIN_BRL"); v != "" {
		min, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return c, err
		}
		c.FreeShippingMin = min
	}
	fake, _ := strconv.ParseBool(os.Getenv("CHECKOUT_FAKE"))
	if token := os.Getenv("MP_ACCESS_TOKEN"); token != "" {
		gw, err := mercadopago.New(token)
		if err != nil {
			return c, err
		}
		c.Gateway = gw
	} else if fake {
		c.Gateway = checkoutfake.NewGateway()
		log.Printf("CHECKOUT_FAKE=1 without MP_ACCESS_TOKEN: every order is auto-approved by a local stub gateway — never in production")
	} else {
		log.Printf("MP_ACCESS_TOKEN unset: checkout disabled, storefront keeps the WhatsApp handoff only")
	}
	if token := os.Getenv("MELHORENVIO_TOKEN"); token != "" {
		base := shipping.ProductionBase
		if sandbox, _ := strconv.ParseBool(os.Getenv("MELHORENVIO_SANDBOX")); sandbox {
			base = shipping.SandboxBase
		}
		c.Quoter = shipping.NewMelhorEnvio(base, token, os.Getenv("MELHORENVIO_CONTACT"))
	} else if fake {
		c.Quoter = checkoutfake.Quoter{}
		log.Printf("CHECKOUT_FAKE=1 without MELHORENVIO_TOKEN: freight is made up — never in production")
	} else {
		log.Printf("MELHORENVIO_TOKEN unset: shipping quotes disabled")
	}
	if c.WebhookSecret == "" {
		log.Printf("MP_WEBHOOK_SECRET unset: webhook notifications will be rejected")
	}
	return c, nil
}
