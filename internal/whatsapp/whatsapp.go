package whatsapp

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const DefaultBase = "https://graph.facebook.com"

type Sender interface {
	SendTemplate(ctx context.Context, to, name, lang string, params []string) (string, error)
	SendText(ctx context.Context, to, body string) (string, error)
}

type Client struct {
	base          string
	version       string
	phoneNumberID string
	token         string
	http          *http.Client
}

func New(base, version, phoneNumberID, token string) *Client {
	return &Client{
		base:          strings.TrimRight(base, "/"),
		version:       version,
		phoneNumberID: phoneNumberID,
		token:         token,
		http:          &http.Client{Timeout: 20 * time.Second},
	}
}

type templateParam struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type templateComponent struct {
	Type       string          `json:"type"`
	Parameters []templateParam `json:"parameters"`
}

type templateBody struct {
	Name       string              `json:"name"`
	Language   map[string]string   `json:"language"`
	Components []templateComponent `json:"components,omitempty"`
}

type textBody struct {
	Body       string `json:"body"`
	PreviewURL bool   `json:"preview_url"`
}

type outbound struct {
	MessagingProduct string        `json:"messaging_product"`
	RecipientType    string        `json:"recipient_type"`
	To               string        `json:"to"`
	Type             string        `json:"type"`
	Template         *templateBody `json:"template,omitempty"`
	Text             *textBody     `json:"text,omitempty"`
}

func (c *Client) SendTemplate(ctx context.Context, to, name, lang string, params []string) (string, error) {
	tpl := &templateBody{Name: name, Language: map[string]string{"code": lang}}
	if len(params) > 0 {
		comp := templateComponent{Type: "body"}
		for _, p := range params {
			comp.Parameters = append(comp.Parameters, templateParam{Type: "text", Text: TemplateSafe(p)})
		}
		tpl.Components = []templateComponent{comp}
	}
	return c.send(ctx, outbound{MessagingProduct: "whatsapp", RecipientType: "individual", To: to, Type: "template", Template: tpl})
}

func (c *Client) SendText(ctx context.Context, to, body string) (string, error) {
	return c.send(ctx, outbound{MessagingProduct: "whatsapp", RecipientType: "individual", To: to, Type: "text", Text: &textBody{Body: body, PreviewURL: true}})
}

func (c *Client) send(ctx context.Context, msg outbound) (string, error) {
	payload, err := json.Marshal(msg)
	if err != nil {
		return "", err
	}
	u := fmt.Sprintf("%s/%s/%s/messages", c.base, c.version, c.phoneNumberID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	var out struct {
		Messages []struct {
			ID string `json:"id"`
		} `json:"messages"`
		Error *struct {
			Message string `json:"message"`
			Code    int    `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", fmt.Errorf("whatsapp HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if out.Error != nil {
		return "", fmt.Errorf("whatsapp %d: %s", out.Error.Code, out.Error.Message)
	}
	if resp.StatusCode >= 300 || len(out.Messages) == 0 {
		return "", fmt.Errorf("whatsapp HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return out.Messages[0].ID, nil
}

func TemplateSafe(s string) string {
	s = strings.NewReplacer("\r", " ", "\n", " ", "\t", " ").Replace(s)
	return strings.Join(strings.Fields(s), " ")
}

type Inbound struct {
	ID   string
	From string
	Name string
	Text string
	At   time.Time
}

type webhookPayload struct {
	Entry []struct {
		Changes []struct {
			Field string `json:"field"`
			Value struct {
				Contacts []struct {
					WaID    string `json:"wa_id"`
					Profile struct {
						Name string `json:"name"`
					} `json:"profile"`
				} `json:"contacts"`
				Messages []struct {
					ID        string `json:"id"`
					From      string `json:"from"`
					Timestamp string `json:"timestamp"`
					Type      string `json:"type"`
					Text      struct {
						Body string `json:"body"`
					} `json:"text"`
					Button struct {
						Text string `json:"text"`
					} `json:"button"`
					Interactive struct {
						ButtonReply struct {
							Title string `json:"title"`
						} `json:"button_reply"`
						ListReply struct {
							Title string `json:"title"`
						} `json:"list_reply"`
					} `json:"interactive"`
				} `json:"messages"`
			} `json:"value"`
		} `json:"changes"`
	} `json:"entry"`
}

func ParseWebhook(body []byte) ([]Inbound, error) {
	var p webhookPayload
	if err := json.Unmarshal(body, &p); err != nil {
		return nil, err
	}
	var out []Inbound
	for _, e := range p.Entry {
		for _, ch := range e.Changes {
			if ch.Field != "messages" {
				continue
			}
			names := make(map[string]string, len(ch.Value.Contacts))
			for _, c := range ch.Value.Contacts {
				names[c.WaID] = c.Profile.Name
			}
			for _, m := range ch.Value.Messages {
				in := Inbound{ID: m.ID, From: m.From, Name: names[m.From]}
				if sec, err := strconv.ParseInt(m.Timestamp, 10, 64); err == nil {
					in.At = time.Unix(sec, 0)
				}
				switch m.Type {
				case "text":
					in.Text = m.Text.Body
				case "button":
					in.Text = m.Button.Text
				case "interactive":
					in.Text = m.Interactive.ButtonReply.Title
					if in.Text == "" {
						in.Text = m.Interactive.ListReply.Title
					}
				default:
					in.Text = "[" + m.Type + "]"
				}
				out = append(out, in)
			}
		}
	}
	return out, nil
}

var ErrSignature = errors.New("invalid webhook signature")

func ValidateSignature(body []byte, header, appSecret string) error {
	sig, ok := strings.CutPrefix(header, "sha256=")
	if !ok || appSecret == "" {
		return ErrSignature
	}
	got, err := hex.DecodeString(sig)
	if err != nil {
		return ErrSignature
	}
	mac := hmac.New(sha256.New, []byte(appSecret))
	mac.Write(body)
	if !hmac.Equal(got, mac.Sum(nil)) {
		return ErrSignature
	}
	return nil
}

func digitsOnly(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func BrazilE164(phone string) string {
	d := digitsOnly(phone)
	if len(d) == 10 || len(d) == 11 {
		return "55" + d
	}
	return d
}

func PhoneKey(phone string) string {
	d := digitsOnly(phone)
	if strings.HasPrefix(d, "55") && len(d) >= 12 {
		d = d[2:]
	}
	if len(d) == 11 && d[2] == '9' {
		d = d[:2] + d[3:]
	}
	return d
}
