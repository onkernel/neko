package scaletozero

import (
	"context"
	"sync"

	"github.com/m1k1o/neko/server/pkg/types"
	"github.com/onkernel/kernel-images/server/lib/scaletozero"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func NewManager(
	sessions types.SessionManager,
	config *Config,
) *Manager {
	logger := log.With().Str("module", "scaletozero").Logger()

	return &Manager{
		logger:   logger,
		config:   config,
		sessions: sessions,
		ctrl:     scaletozero.NewUnikraftCloudController(),
	}
}

type Manager struct {
	logger              zerolog.Logger
	config              *Config
	sessions            types.SessionManager
	ctrl                scaletozero.Controller
	mu                  sync.Mutex
	shutdown            bool
	disabledScaleToZero bool
}

func (m *Manager) Start() error {
	if !m.config.Enabled {
		return nil
	}
	m.logger.Info().Msg("plugin enabled")

	// compute initial state and toggle if needed
	m.manage()

	m.sessions.OnConnected(func(session types.Session) {
		m.manage()
	})

	m.sessions.OnDisconnected(func(session types.Session) {
		m.manage()
	})

	return nil
}

func (m *Manager) Shutdown() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shutdown = true

	if m.disabledScaleToZero {
		return m.ctrl.Enable(context.Background())
	}

	return nil
}

func (m *Manager) manage() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shutdown {
		return
	}

	connectedSessions := 0
	for _, s := range m.sessions.List() {
		if s.State().IsConnected {
			connectedSessions++
		}
	}
	hasConnectedSessions := connectedSessions > 0

	if hasConnectedSessions == m.disabledScaleToZero {
		m.logger.Info().Bool("previously_disabled", m.disabledScaleToZero).
			Bool("currently_disabled", hasConnectedSessions).
			Int("currently_connected_sessions", connectedSessions).
			Msg("no operation needed; skipping toggle")
		return
	}

	// toggle if needed but only update internal state if successful
	if hasConnectedSessions {
		m.logger.Info().Int("connected_sessions", connectedSessions).Msg("disabling scale-to-zero")
		if err := m.ctrl.Disable(context.Background()); err != nil {
			m.logger.Error().Err(err).Msg("failed to disable scale-to-zero")
			return
		}
	} else {
		m.logger.Info().Msg("enabling scale-to-zero")
		if err := m.ctrl.Enable(context.Background()); err != nil {
			m.logger.Error().Err(err).Msg("failed to enable scale-to-zero")
			return
		}
	}

	m.disabledScaleToZero = hasConnectedSessions
}
