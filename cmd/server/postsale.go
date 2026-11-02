package main

import (
	"os"
	"strconv"

	"opdeals/internal/api"
	"opdeals/internal/conversations"
	"opdeals/internal/logx"
	"opdeals/internal/postsale"
	"opdeals/internal/whatsapp"
)

const minAgentTokenLen = 32

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func buildPostSale(conversationsPath, adminToken string, logger *logx.Logger) api.PostSale {
	log := logger.WithPrefix("POSVENDA")
	switch transport := os.Getenv("POSTSALE_TRANSPORT"); transport {
	case "", "cloud":
		return buildCloudPostSale(conversationsPath, log)
	case "openbot":
		return buildOpenBotPostSale(conversationsPath, adminToken, log)
	default:
		log.Printf("POSTSALE_TRANSPORT=%q desconhecido (use cloud ou openbot): pós-venda desligado", transport)
		return api.PostSale{}
	}
}

func buildOpenBotPostSale(conversationsPath, adminToken string, log *logx.Logger) api.PostSale {
	token := os.Getenv("POSTSALE_AGENT_TOKEN")
	if len(token) < minAgentTokenLen || token == adminToken {
		log.Printf("POSTSALE_AGENT_TOKEN precisa ter %d+ caracteres e ser diferente do ADMIN_TOKEN: pós-venda desligado", minAgentTokenLen)
		return api.PostSale{}
	}
	autoSend, _ := strconv.ParseBool(os.Getenv("POSTSALE_AUTOSEND"))
	ps := api.PostSale{
		Conversations: conversations.NewStore(conversationsPath),
		Outbox:        true,
		AgentToken:    token,
		AutoSend:      autoSend,
		NtfyTopic:     os.Getenv("NTFY_TOPIC"),
	}
	log.Printf("pós-venda pelo OpenBot (WhatsApp Web): ANTHROPIC_API_KEY e WHATSAPP_* são ignorados")
	if ps.NtfyTopic == "" {
		log.Printf("NTFY_TOPIC unset: sem push de rascunhos, handoffs e alerta de OpenBot parado")
	}
	return ps
}

func buildCloudPostSale(conversationsPath string, log *logx.Logger) api.PostSale {
	token := os.Getenv("WHATSAPP_TOKEN")
	phoneID := os.Getenv("WHATSAPP_PHONE_NUMBER_ID")
	if token == "" || phoneID == "" {
		log.Printf("WHATSAPP_TOKEN/WHATSAPP_PHONE_NUMBER_ID unset: pós-venda pelo WhatsApp desligado")
		return api.PostSale{}
	}
	autoSend, _ := strconv.ParseBool(os.Getenv("POSTSALE_AUTOSEND"))
	ps := api.PostSale{
		Conversations:    conversations.NewStore(conversationsPath),
		WhatsApp:         whatsapp.New(envOr("WHATSAPP_API_BASE", whatsapp.DefaultBase), envOr("WHATSAPP_API_VERSION", "v23.0"), phoneID, token),
		AppSecret:        os.Getenv("WHATSAPP_APP_SECRET"),
		VerifyToken:      os.Getenv("WHATSAPP_VERIFY_TOKEN"),
		OfferTemplate:    envOr("WHATSAPP_TEMPLATE_OFFER", "collecta_pos_venda"),
		TrackingTemplate: envOr("WHATSAPP_TEMPLATE_TRACKING", "collecta_rastreio"),
		TemplateLang:     envOr("WHATSAPP_TEMPLATE_LANG", "pt_BR"),
		AutoSend:         autoSend,
		NtfyTopic:        os.Getenv("NTFY_TOPIC"),
	}
	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		ps.Brain = postsale.NewClaude(key, os.Getenv("POSTSALE_MODEL"))
	} else {
		log.Printf("ANTHROPIC_API_KEY unset: ofertas saem com frase padrão e respostas vão direto para você")
	}
	if ps.AppSecret == "" {
		log.Printf("WHATSAPP_APP_SECRET unset: webhook do WhatsApp vai recusar tudo")
	}
	if ps.NtfyTopic == "" {
		log.Printf("NTFY_TOPIC unset: sem push de rascunhos e handoffs")
	}
	return ps
}
