package circuit

import (
	"fmt"
	"sync"
	"time"

	"github.com/kelrob/shared/logger"
)

type State string

const (
	StateClosed   State = "closed"
	StateOpen     State = "open"
	StateHalfOpen State = "half-open"
)

type Breaker struct {
	mu               sync.Mutex
	state            State
	failureCount     int
	failureThreshold int
	cooldown         time.Duration
	lastFailureTime  time.Time
	appLog           *logger.Logger
}

func NewBreaker(failureThreshold int, cooldown time.Duration, appLog *logger.Logger) *Breaker {
	return &Breaker{
		state:            StateClosed,
		failureThreshold: failureThreshold,
		cooldown:         cooldown,
		appLog:           appLog,
	}
}

func (b *Breaker) Execute(fn func() error) error {
	b.mu.Lock()

	switch b.state {
	case StateOpen:
		if time.Since(b.lastFailureTime) >= b.cooldown {
			b.appLog.Log("Circuit half-open — testing recovery", nil)
			b.state = StateHalfOpen
		} else {
			b.mu.Unlock()
			return fmt.Errorf("circuit breaker open — service unavailable")
		}
	}

	b.mu.Unlock()

	err := fn()

	b.mu.Lock()
	defer b.mu.Unlock()

	if err != nil {
		b.failureCount++
		b.lastFailureTime = time.Now()
		b.appLog.Log("Circuit breaker failure count", map[string]any{
			"failure_count":     b.failureCount,
			"failure_threshold": b.failureThreshold,
		})

		if b.failureCount >= b.failureThreshold {
			b.state = StateOpen
			b.appLog.Log("Circuit breaker opened", nil)
		}
		return err
	}

	if b.state == StateHalfOpen {
		b.appLog.Log("Circuit breaker closed — service recovered", nil)
		b.state = StateClosed
		b.failureCount = 0
	}

	return nil
}

func (b *Breaker) State() State {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.state
}
