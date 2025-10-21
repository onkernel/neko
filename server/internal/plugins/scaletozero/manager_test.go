package scaletozero

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/m1k1o/neko/server/internal/config"
	intsession "github.com/m1k1o/neko/server/internal/session"
	"github.com/m1k1o/neko/server/pkg/types"
)

func TestSingleSessionConnectDisconnectReconnect(t *testing.T) {
	sm := newSessionManager(t)
	m, fc := newPluginWithFakeCtrl(sm)

	require.NoError(t, m.Start())
	require.Equal(t, 0, fc.disableCalls)
	require.Equal(t, 0, fc.enableCalls)

	s, p := connect(t, sm, "1")
	require.Equal(t, 1, fc.disableCalls)
	require.Equal(t, 0, fc.enableCalls)

	s.DisconnectWebSocketPeer(p, true)
	require.Equal(t, 1, fc.disableCalls)
	require.Equal(t, 0, fc.enableCalls)

	// wait for an arbitrary fraction of the delay duration to mimic client behavior
	start := time.Now()
	time.Sleep(intsession.WS_DELAYED_DURATION / 10)

	// safeguard to prevent flake
	if time.Since(start) >= intsession.WS_DELAYED_DURATION {
		return
	}

	// connect and ensure no subsequent stz calls
	_, p2 := connect(t, sm, "1")
	require.Equal(t, 1, fc.disableCalls)
	require.Equal(t, 0, fc.enableCalls)
	_ = p2
}

func TestMultipleSessionsConnectDisconnect(t *testing.T) {
	sm := newSessionManager(t)
	m, fc := newPluginWithFakeCtrl(sm)
	require.NoError(t, m.Start())

	s1, p1 := connect(t, sm, "1")
	require.Equal(t, 1, fc.disableCalls)
	require.Equal(t, 0, fc.enableCalls)
	_, p2 := connect(t, sm, "2")
	require.Equal(t, 1, fc.disableCalls)
	require.Equal(t, 0, fc.enableCalls)

	// enable only after both sessions are disconnected
	s1.DisconnectWebSocketPeer(p1, false)
	require.Equal(t, 1, fc.disableCalls)
	require.Equal(t, 0, fc.enableCalls)
	s2, ok := sm.Get("2")
	require.True(t, ok)
	s2.DisconnectWebSocketPeer(p2, false)
	require.Equal(t, 1, fc.disableCalls)
	require.Equal(t, 1, fc.enableCalls)
}

func TestSingleSessionReplacementDoesNotDoubleDisable(t *testing.T) {
	sm := newSessionManager(t)
	m, fc := newPluginWithFakeCtrl(sm)
	require.NoError(t, m.Start())

	s, _ := connect(t, sm, "1")
	require.Equal(t, 1, fc.disableCalls)
	require.Equal(t, 0, fc.enableCalls)

	// replacement: connect again while connected; should not trigger another disable
	p2 := &mockWebsocketPeer{}
	s.ConnectWebSocketPeer(p2)
	require.Equal(t, 1, fc.disableCalls)
	require.Equal(t, 0, fc.enableCalls)
}

type mockScaleToZeroer struct {
	disableCalls int
	enableCalls  int
	disableErr   error
	enableErr    error
}

func (f *mockScaleToZeroer) Disable(ctx context.Context) error {
	f.disableCalls++
	return f.disableErr
}

func (f *mockScaleToZeroer) Enable(ctx context.Context) error {
	f.enableCalls++
	return f.enableErr
}

type mockWebsocketPeer struct{}

func (mockWebsocketPeer) Send(event string, payload any) {}
func (mockWebsocketPeer) Ping() error                    { return nil }
func (mockWebsocketPeer) Destroy(reason string)          {}

func newSessionManager(t *testing.T) *intsession.SessionManagerCtx {
	t.Helper()
	return intsession.New(&config.Session{})
}

func newPluginWithFakeCtrl(sm types.SessionManager) (*Manager, *mockScaleToZeroer) {
	fc := &mockScaleToZeroer{}
	m := NewManager(sm, &Config{Enabled: true})
	m.ctrl = fc
	return m, fc
}

func connect(t *testing.T, sm types.SessionManager, id string) (types.Session, types.WebSocketPeer) {
	t.Helper()
	s, ok := sm.Get(id)
	if !ok {
		var err error
		s, _, err = sm.Create(id, types.MemberProfile{CanLogin: true, CanConnect: true, CanWatch: true})
		require.NoError(t, err)
	}
	p := &mockWebsocketPeer{}
	s.ConnectWebSocketPeer(p)

	return s, p
}
