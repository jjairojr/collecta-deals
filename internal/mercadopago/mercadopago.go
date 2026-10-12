package mercadopago

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mercadopago/sdk-go/pkg/config"
	"github.com/mercadopago/sdk-go/pkg/payment"
	"github.com/mercadopago/sdk-go/pkg/preference"
	"github.com/mercadopago/sdk-go/pkg/webhook"
)

type Item struct {
	ID      string
	Title   string
	Qty     int
	UnitBRL float64
}

type Payer struct {
	Name  string
	Email string
	Phone string
	CPF   string
}

type PreferenceInput struct {
	OrderID         string
	Number          int
	Items           []Item
	Payer           Payer
	ReturnURL       string
	NotificationURL string
	ExpiresAt       time.Time
}

type Preference struct {
	ID        string
	InitPoint string
}

type Payment struct {
	ID                int64
	Status            string
	StatusDetail      string
	ExternalReference string
	AmountBRL         float64
	Method            string
}

type Gateway interface {
	CreatePreference(ctx context.Context, in PreferenceInput) (Preference, error)
	GetPayment(ctx context.Context, id int64) (Payment, error)
}

type Client struct {
	prefs    preference.Client
	payments payment.Client
}

func New(accessToken string) (*Client, error) {
	cfg, err := config.New(accessToken)
	if err != nil {
		return nil, err
	}
	return &Client{prefs: preference.NewClient(cfg), payments: payment.NewClient(cfg)}, nil
}

func (c *Client) CreatePreference(ctx context.Context, in PreferenceInput) (Preference, error) {
	items := make([]preference.ItemRequest, 0, len(in.Items))
	for _, it := range in.Items {
		items = append(items, preference.ItemRequest{
			ID:         it.ID,
			Title:      it.Title,
			Quantity:   it.Qty,
			CurrencyID: "BRL",
			UnitPrice:  it.UnitBRL,
			CategoryID: "others",
		})
	}
	first, last := splitName(in.Payer.Name)
	area, number := splitPhone(in.Payer.Phone)
	from := time.Now()
	to := in.ExpiresAt
	req := preference.Request{
		Items: items,
		Payer: &preference.PayerRequest{
			Name:    first,
			Surname: last,
			Email:   in.Payer.Email,
			Phone:   &preference.PhoneRequest{AreaCode: area, Number: number},
			Identification: &preference.IdentificationRequest{
				Type:   "CPF",
				Number: in.Payer.CPF,
			},
		},
		BackURLs: &preference.BackURLsRequest{
			Success: in.ReturnURL + "?r=success",
			Pending: in.ReturnURL + "?r=pending",
			Failure: in.ReturnURL + "?r=failure",
		},
		AutoReturn:          autoReturn(in.ReturnURL),
		NotificationURL:     in.NotificationURL,
		ExternalReference:   in.OrderID,
		StatementDescriptor: "COLLECTA",
		Expires:             true,
		ExpirationDateFrom:  &from,
		ExpirationDateTo:    &to,
		PaymentMethods: &preference.PaymentMethodsRequest{
			ExcludedPaymentTypes: []preference.ExcludedPaymentTypeRequest{{ID: "ticket"}},
			Installments:         12,
		},
		Metadata: map[string]any{"order_number": in.Number},
	}
	resp, err := c.prefs.Create(ctx, req)
	if err != nil {
		return Preference{}, fmt.Errorf("mercadopago preference: %w", err)
	}
	return Preference{ID: resp.ID, InitPoint: resp.InitPoint}, nil
}

func (c *Client) GetPayment(ctx context.Context, id int64) (Payment, error) {
	resp, err := c.payments.Get(ctx, int(id))
	if err != nil {
		return Payment{}, fmt.Errorf("mercadopago payment %d: %w", id, err)
	}
	return Payment{
		ID:                int64(resp.ID),
		Status:            resp.Status,
		StatusDetail:      resp.StatusDetail,
		ExternalReference: resp.ExternalReference,
		AmountBRL:         resp.TransactionAmount,
		Method:            resp.PaymentMethodID,
	}, nil
}

func ValidateSignature(xSignature, xRequestID, dataID, secret string) error {
	return webhook.ValidateSignature(xSignature, xRequestID, dataID, secret)
}

func autoReturn(returnURL string) string {
	if strings.HasPrefix(returnURL, "https://") {
		return "approved"
	}
	return ""
}

func splitName(full string) (string, string) {
	parts := strings.Fields(full)
	if len(parts) == 0 {
		return "", ""
	}
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], strings.Join(parts[1:], " ")
}

func splitPhone(digits string) (string, string) {
	if len(digits) < 10 {
		return "", digits
	}
	return digits[:2], digits[2:]
}
