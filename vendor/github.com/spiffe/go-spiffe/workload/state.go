package workload

import (
	"errors"
	"sync"
)

type clientState int

const (
	clientStateInit clientState = iota
	clientStateStarted
	clientStateStopped
)

type clientStateManager struct {
	sync.Mutex
	state clientState
}

func newClientStateManager() *clientStateManager {
	return &clientStateManager{}
}

func (s *clientStateManager) StartIfStartable() error {
	s.Lock()
	defer s.Unlock()
	if s.state == clientStateStarted {
		return errors.New("client already started")
	}
	if s.state == clientStateStopped {
		return errors.New("client cannot start once stopped")
	}
	s.state = clientStateStarted
	return nil
}

func (s *clientStateManager) StopIfStoppable() error {
	s.Lock()
	defer s.Unlock()
	if s.state == clientStateInit {
		return errors.New("client hasn't started")
	}
	if s.state == clientStateStopped {
		return errors.New("client is already stopped")
	}
	s.state = clientStateStopped
	return nil
}
