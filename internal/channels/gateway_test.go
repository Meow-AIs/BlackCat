package channels

import (
	"context"
	"sync"
	"testing"
	"time"
)

// mockAdapter is a test adapter that satisfies the Adapter interface.
type mockAdapter struct {
	platform Platform
	incoming chan IncomingMessage
	sent     []OutgoingMessage
	mu       sync.Mutex
	started  bool
	stopped  bool
}

func newMockAdapter(p Platform) *mockAdapter {
	return &mockAdapter{
		platform: p,
		incoming: make(chan IncomingMessage, 10),
	}
}

func (m *mockAdapter) Platform() Platform { return m.platform }

func (m *mockAdapter) Start(_ context.Context) error {
	m.started = true
	return nil
}

func (m *mockAdapter) Stop(_ context.Context) error {
	m.stopped = true
	return nil
}

func (m *mockAdapter) Send(_ context.Context, msg OutgoingMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent = append(m.sent, msg)
	return nil
}

func (m *mockAdapter) Receive() <-chan IncomingMessage { return m.incoming }

func (m *mockAdapter) sentMessages() []OutgoingMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]OutgoingMessage, len(m.sent))
	copy(result, m.sent)
	return result
}

func TestGatewayRegister(t *testing.T) {
	gw := NewGateway(nil)
	gw.Register(newMockAdapter(PlatformTelegram))
	gw.Register(newMockAdapter(PlatformDiscord))
	gw.Register(newMockAdapter(PlatformSlack))
	gw.Register(newMockAdapter(PlatformWhatsApp))
	gw.Register(newMockAdapter(PlatformSignal))
	gw.Register(newMockAdapter(PlatformEmail))

	adapters := gw.Adapters()
	if len(adapters) != 6 {
		t.Errorf("expected 6 adapters, got %d", len(adapters))
	}
}

func TestGatewayStartStop(t *testing.T) {
	tg := newMockAdapter(PlatformTelegram)
	dc := newMockAdapter(PlatformDiscord)

	gw := NewGateway(nil)
	gw.Register(tg)
	gw.Register(dc)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := gw.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if !tg.started || !dc.started {
		t.Error("expected all adapters started")
	}

	gw.Stop(ctx)
	if !tg.stopped || !dc.stopped {
		t.Error("expected all adapters stopped")
	}
}

func TestGatewayMessageRouting(t *testing.T) {
	received := make(chan string, 1)

	handler := func(ctx context.Context, msg IncomingMessage) (string, error) {
		received <- msg.Text
		return "response: " + msg.Text, nil
	}

	tg := newMockAdapter(PlatformTelegram)
	gw := NewGateway(handler)
	gw.Register(tg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	gw.Start(ctx)

	// Simulate incoming message
	tg.incoming <- IncomingMessage{
		Platform:  PlatformTelegram,
		ChannelID: "123",
		UserID:    "456",
		Text:      "hello blackcat",
		Timestamp: time.Now().Unix(),
	}

	// Wait for handler to process
	select {
	case text := <-received:
		if text != "hello blackcat" {
			t.Errorf("expected 'hello blackcat', got %q", text)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for message")
	}

	// Wait for response to be sent
	time.Sleep(100 * time.Millisecond)
	sent := tg.sentMessages()
	if len(sent) == 0 {
		t.Fatal("expected at least 1 sent message")
	}
	if sent[0].Text != "response: hello blackcat" {
		t.Errorf("expected response, got %q", sent[0].Text)
	}
}

func TestRateLimiter(t *testing.T) {
	rl := NewRateLimiter(3, 60)

	if !rl.Allow("user1") {
		t.Error("1st should be allowed")
	}
	if !rl.Allow("user1") {
		t.Error("2nd should be allowed")
	}
	if !rl.Allow("user1") {
		t.Error("3rd should be allowed")
	}
	if rl.Allow("user1") {
		t.Error("4th should be rate limited")
	}

	// Different user should be fine
	if !rl.Allow("user2") {
		t.Error("different user should be allowed")
	}

	// Reset
	rl.Reset("user1")
	if !rl.Allow("user1") {
		t.Error("after reset should be allowed")
	}
}

func TestSessionManager(t *testing.T) {
	sm := NewSessionManager()

	if sm.Count() != 0 {
		t.Error("expected 0 sessions")
	}

	sm.Set("tg:123", "sess-1")
	sm.Set("dc:456", "sess-2")

	if sm.Count() != 2 {
		t.Errorf("expected 2 sessions, got %d", sm.Count())
	}

	if sm.Get("tg:123") != "sess-1" {
		t.Error("expected sess-1")
	}

	sm.Remove("tg:123")
	if sm.Get("tg:123") != "" {
		t.Error("expected empty after remove")
	}
}

func TestPairingManager(t *testing.T) {
	pm := NewPairingManager()

	code := pm.GenerateCode(PlatformTelegram, "user123")
	if len(code) != 6 {
		t.Errorf("expected 6-digit code, got %q", code)
	}

	platform, userID, ok := pm.ValidateCode(code)
	if !ok {
		t.Fatal("expected valid code")
	}
	if platform != PlatformTelegram {
		t.Errorf("expected telegram, got %s", platform)
	}
	if userID != "user123" {
		t.Errorf("expected user123, got %s", userID)
	}

	// Code should be one-time use
	_, _, ok = pm.ValidateCode(code)
	if ok {
		t.Error("code should be invalid after first use")
	}
}

func TestPairingManagerInvalidCode(t *testing.T) {
	pm := NewPairingManager()
	_, _, ok := pm.ValidateCode("999999")
	if ok {
		t.Error("expected invalid for unknown code")
	}
}

func TestGatewayRateLimitedMessage(t *testing.T) {
	handler := func(_ context.Context, _ IncomingMessage) (string, error) {
		return "ok", nil
	}

	tg := newMockAdapter(PlatformTelegram)
	gw := NewGateway(handler)
	gw.limiter = NewRateLimiter(1, 60) // only 1 msg allowed
	gw.Register(tg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	gw.Start(ctx)

	// Send 2 messages quickly
	for i := 0; i < 2; i++ {
		tg.incoming <- IncomingMessage{
			Platform: PlatformTelegram, ChannelID: "1",
			UserID: "user1", Text: "msg",
		}
	}

	time.Sleep(300 * time.Millisecond)
	sent := tg.sentMessages()

	// Should have 2 responses: 1 normal + 1 rate limit warning
	hasRateLimit := false
	for _, s := range sent {
		if s.Text == "Rate limited. Please wait before sending more messages." {
			hasRateLimit = true
		}
	}
	if !hasRateLimit {
		t.Error("expected rate limit message")
	}
}

func TestAllPlatformConstants(t *testing.T) {
	platforms := []Platform{
		PlatformTelegram, PlatformDiscord, PlatformSlack,
		PlatformWhatsApp, PlatformSignal, PlatformEmail, PlatformCLI,
	}

	for _, p := range platforms {
		if p == "" {
			t.Error("platform should not be empty")
		}
	}

	if len(platforms) != 7 {
		t.Errorf("expected 7 platforms, got %d", len(platforms))
	}
}
