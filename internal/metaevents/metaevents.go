// Package metaevents monta eventos da Conversions API do Meta e a lista de
// clientes, com os dados do cliente normalizados e embaralhados em SHA-256 como
// o Meta pede. Nada aqui guarda ou registra dado de cliente: a saída já sai com
// hash (menos o navegador e os cookies do Meta, que o Meta pede sem hash).
package metaevents

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"unicode"
)

const Currency = "BRL"

// Action sources aceitos pelo Meta que a loja usa.
const (
	SourceWebsite = "website"        // compra no site
	SourceStore   = "physical_store" // venda no balcão
	SourceOther   = "other"          // venda na Liga (loja de terceiro)
)

// UserData são os campos de cliente de um evento. Os de identidade vão com hash;
// client_user_agent, fbp e fbc vão sem hash, como o Meta pede.
type UserData struct {
	Email     []string `json:"em,omitempty"`
	Phone     []string `json:"ph,omitempty"`
	FirstName []string `json:"fn,omitempty"`
	LastName  []string `json:"ln,omitempty"`
	City      []string `json:"ct,omitempty"`
	State     []string `json:"st,omitempty"`
	Zip       []string `json:"zp,omitempty"`
	Country   []string `json:"country,omitempty"`
	UserAgent string   `json:"client_user_agent,omitempty"`
	FBP       string   `json:"fbp,omitempty"`
	FBC       string   `json:"fbc,omitempty"`
}

// Matchable diz se há ao menos um identificador da pessoa (e-mail, telefone ou
// cookie do Meta). Sem isso o evento não ensina nada ao Meta.
func (u UserData) Matchable() bool {
	return len(u.Email) > 0 || len(u.Phone) > 0 || u.FBP != "" || u.FBC != ""
}

type CustomData struct {
	Value    float64 `json:"value"`
	Currency string  `json:"currency"`
	OrderID  string  `json:"order_id,omitempty"`
	NumItems int     `json:"num_items,omitempty"`
}

type Event struct {
	EventName      string     `json:"event_name"`
	EventTime      int64      `json:"event_time"`
	EventID        string     `json:"event_id"`
	ActionSource   string     `json:"action_source"`
	EventSourceURL string     `json:"event_source_url,omitempty"`
	UserData       UserData   `json:"user_data"`
	CustomData     CustomData `json:"custom_data"`
}

// Hash é o SHA-256 em hexadecimal minúsculo de um valor já normalizado.
func Hash(v string) string {
	sum := sha256.Sum256([]byte(v))
	return hex.EncodeToString(sum[:])
}

func hashed(v string) []string {
	if v == "" {
		return nil
	}
	return []string{Hash(v)}
}

// Email normaliza: sem espaços nas pontas, minúsculo; vazio se não parece e-mail.
func Email(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	at := strings.LastIndexByte(s, '@')
	if at <= 0 || at == len(s)-1 || strings.ContainsAny(s, " \t") || !strings.Contains(s[at:], ".") {
		return ""
	}
	return s
}

// Phone normaliza um telefone brasileiro: só dígitos, sem zeros à esquerda, com
// o 55 na frente quando vier só com DDD. Vazio se não tem jeito de telefone.
func Phone(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	d := strings.TrimLeft(b.String(), "0")
	if len(d) == 10 || len(d) == 11 {
		d = "55" + d
	}
	if len(d) < 12 || len(d) > 13 {
		return ""
	}
	return d
}

var accents = strings.NewReplacer(
	"á", "a", "à", "a", "â", "a", "ã", "a", "ä", "a",
	"é", "e", "è", "e", "ê", "e", "ë", "e",
	"í", "i", "ì", "i", "î", "i", "ï", "i",
	"ó", "o", "ò", "o", "ô", "o", "õ", "o", "ö", "o",
	"ú", "u", "ù", "u", "û", "u", "ü", "u",
	"ç", "c", "ñ", "n",
)

// Plain deixa só letras e números, minúsculos, sem acento, sem espaço e sem
// pontuação: o formato do Meta para nome, cidade e estado.
func Plain(s string) string {
	s = accents.Replace(strings.ToLower(strings.TrimSpace(s)))
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// SplitName separa primeiro nome e sobrenome (o último pedaço).
func SplitName(full string) (first, last string) {
	parts := strings.Fields(full)
	switch len(parts) {
	case 0:
		return "", ""
	case 1:
		return Plain(parts[0]), ""
	}
	return Plain(parts[0]), Plain(parts[len(parts)-1])
}

// Digits é o CEP só com dígitos.
func Digits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// Person é o que se sabe de um comprador, em texto puro, antes do hash.
type Person struct {
	Name, Email, Phone, City, State, Zip string
}

// User devolve os campos do Meta com hash. O país é sempre Brasil.
func (p Person) User() UserData {
	first, last := SplitName(p.Name)
	st := Plain(p.State)
	if len(st) != 2 {
		st = ""
	}
	return UserData{
		Email:     hashed(Email(p.Email)),
		Phone:     hashed(Phone(p.Phone)),
		FirstName: hashed(first),
		LastName:  hashed(last),
		City:      hashed(Plain(p.City)),
		State:     hashed(st),
		Zip:       hashed(Digits(p.Zip)),
		Country:   hashed("br"),
	}
}

// CustomerSchema é a ordem das colunas da lista de clientes (Custom Audience).
var CustomerSchema = []string{"EMAIL", "PHONE", "FN", "LN", "CT", "ST", "ZIP", "COUNTRY"}

// Row é uma linha da lista de clientes, com hash, na ordem de CustomerSchema;
// vazio onde não se sabe. ok é falso quando não há e-mail nem telefone.
func (p Person) Row() (row []string, ok bool) {
	email, phone := Email(p.Email), Phone(p.Phone)
	if email == "" && phone == "" {
		return nil, false
	}
	first, last := SplitName(p.Name)
	st := Plain(p.State)
	if len(st) != 2 {
		st = ""
	}
	h := func(v string) string {
		if v == "" {
			return ""
		}
		return Hash(v)
	}
	return []string{h(email), h(phone), h(first), h(last), h(Plain(p.City)), h(st), h(Digits(p.Zip)), Hash("br")}, true
}
