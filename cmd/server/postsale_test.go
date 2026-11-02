package main

import (
	"io"
	"path/filepath"
	"strings"
	"testing"

	"opdeals/internal/logx"
)

func setPostSaleEnv(t *testing.T, env map[string]string) {
	t.Helper()
	for _, key := range []string{"POSTSALE_TRANSPORT", "POSTSALE_AGENT_TOKEN", "ANTHROPIC_API_KEY", "WHATSAPP_TOKEN", "WHATSAPP_PHONE_NUMBER_ID", "NTFY_TOPIC"} {
		t.Setenv(key, env[key])
	}
}

func TestBuildPostSaleOpenBotIgnoresCloudAndClaude(t *testing.T) {
	setPostSaleEnv(t, map[string]string{
		"POSTSALE_TRANSPORT":       "openbot",
		"POSTSALE_AGENT_TOKEN":     strings.Repeat("a", 40),
		"ANTHROPIC_API_KEY":        "sk-test",
		"WHATSAPP_TOKEN":           "wa-token",
		"WHATSAPP_PHONE_NUMBER_ID": "123",
	})
	ps := buildPostSale(filepath.Join(t.TempDir(), "conversations.json"), "admin", logx.New(io.Discard))
	if !ps.Outbox || ps.Conversations == nil || ps.AgentToken != strings.Repeat("a", 40) {
		t.Fatalf("openbot post-sale not enabled: %+v", ps)
	}
	if ps.WhatsApp != nil || ps.Brain != nil {
		t.Fatal("openbot mode must not use the Cloud API nor call Claude")
	}
}

func TestBuildPostSaleOpenBotRejectsWeakToken(t *testing.T) {
	for name, token := range map[string]string{"short": "abc", "admin": strings.Repeat("x", 40)} {
		setPostSaleEnv(t, map[string]string{"POSTSALE_TRANSPORT": "openbot", "POSTSALE_AGENT_TOKEN": token})
		ps := buildPostSale(filepath.Join(t.TempDir(), "conversations.json"), strings.Repeat("x", 40), logx.New(io.Discard))
		if ps.Conversations != nil || ps.Outbox {
			t.Fatalf("%s token accepted", name)
		}
	}
}

func TestBuildPostSaleCloudStaysDefault(t *testing.T) {
	setPostSaleEnv(t, map[string]string{"WHATSAPP_TOKEN": "wa-token", "WHATSAPP_PHONE_NUMBER_ID": "123"})
	ps := buildPostSale(filepath.Join(t.TempDir(), "conversations.json"), "admin", logx.New(io.Discard))
	if ps.Outbox || ps.WhatsApp == nil || ps.Conversations == nil {
		t.Fatalf("cloud default broken: %+v", ps)
	}
	setPostSaleEnv(t, map[string]string{})
	if ps := buildPostSale(filepath.Join(t.TempDir(), "conversations.json"), "admin", logx.New(io.Discard)); ps.Conversations != nil {
		t.Fatal("post-sale enabled without any config")
	}
}

func TestBuildPostSaleUnknownTransportDisables(t *testing.T) {
	setPostSaleEnv(t, map[string]string{"POSTSALE_TRANSPORT": "telegram", "WHATSAPP_TOKEN": "wa-token", "WHATSAPP_PHONE_NUMBER_ID": "123"})
	if ps := buildPostSale(filepath.Join(t.TempDir(), "conversations.json"), "admin", logx.New(io.Discard)); ps.Conversations != nil {
		t.Fatal("unknown transport enabled post-sale")
	}
}
