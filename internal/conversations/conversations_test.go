package conversations

import (
	"testing"
	"time"
)

func TestCloseFailsQueuedMessages(t *testing.T) {
	now := time.Now()
	sent := NewMessage(RoleAssistant, "oi", MsgSent, now)
	queued := NewMessage(RoleOwner, "já vai", MsgQueued, now)
	queued.Kind = KindReply
	c := Conversation{State: StateActive, Messages: []Message{sent, queued}}
	c.Close(OutcomeExpired)
	if c.State != StateClosed || c.Outcome != OutcomeExpired {
		t.Fatalf("close = %s %s", c.State, c.Outcome)
	}
	if c.Messages[0].Status != MsgSent {
		t.Fatalf("sent message changed: %+v", c.Messages[0])
	}
	if c.Messages[1].Status != MsgFailed || c.Messages[1].Error == "" {
		t.Fatalf("queued message not failed: %+v", c.Messages[1])
	}
}

func TestQueuedOfferFindsOnlyOffers(t *testing.T) {
	reply := NewMessage(RoleAssistant, "resposta", MsgQueued, time.Now())
	reply.Kind = KindReply
	c := Conversation{Messages: []Message{reply}}
	if _, ok := c.QueuedOffer(); ok {
		t.Fatal("reply counted as offer")
	}
	offer := NewMessage(RoleAssistant, "oferta", MsgQueued, time.Now())
	offer.Kind = KindOffer
	c.Messages = append(c.Messages, offer)
	if i, ok := c.QueuedOffer(); !ok || i != 1 {
		t.Fatalf("QueuedOffer = %d %v", i, ok)
	}
}
