// Package utils provides utility functions and circuit breaker implementation
package utils

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// CircuitBreakerState represents the current state of the circuit breaker
type CircuitBreakerState int32

const (
	StateClosed CircuitBreakerState = iota
	StateOpen
	StateHalfOpen
)

// CircuitBreaker implements the circuit breaker pattern for external service calls
type CircuitBreaker struct {
	name string

	// Configuration
	failureThreshold int32         // Number of failures before opening
	resetTimeout     time.Duration // Time to wait before attempting half-open
	callTimeout      time.Duration // Timeout for individual calls

	// State
	state        int32 // Current state (closed, open, half-open)
	failures     int32 // Current failure count
	lastFailTime int64 // Timestamp of last failure

	// Metrics
	totalRequests        int64
	totalFailures        int64
	totalSuccesses       int64
	consecutiveSuccesses int32

	mu sync.RWMutex
}

// CircuitBreakerConfig holds configuration for circuit breaker
type CircuitBreakerConfig struct {
	Name             string
	FailureThreshold int32
	ResetTimeout     time.Duration
	CallTimeout      time.Duration
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		name:             config.Name,
		failureThreshold: config.FailureThreshold,
		resetTimeout:     config.ResetTimeout,
		callTimeout:      config.CallTimeout,
		state:            int32(StateClosed),
	}
}

// Call executes a function with circuit breaker protection
func (cb *CircuitBreaker) Call(ctx context.Context, fn func(context.Context) error) error {
	if !cb.canExecute() {
		return NewCircuitBreakerError("circuit breaker is open", cb.getState())
	}

	// Create timeout context
	callCtx, cancel := context.WithTimeout(ctx, cb.callTimeout)
	defer cancel()

	// Execute the function
	err := fn(callCtx)

	// Update metrics and state
	atomic.AddInt64(&cb.totalRequests, 1)

	if err != nil {
		cb.recordFailure()
		atomic.AddInt64(&cb.totalFailures, 1)
		atomic.StoreInt32(&cb.consecutiveSuccesses, 0)
		return err
	}

	cb.recordSuccess()
	atomic.AddInt64(&cb.totalSuccesses, 1)
	atomic.AddInt32(&cb.consecutiveSuccesses, 1)
	return nil
}

// canExecute determines if a call can be made based on current state
func (cb *CircuitBreaker) canExecute() bool {
	state := cb.getState()

	switch state {
	case StateClosed:
		return true
	case StateOpen:
		if cb.shouldAttemptReset() {
			cb.setState(StateHalfOpen)
			return true
		}
		return false
	case StateHalfOpen:
		return true
	default:
		return false
	}
}

// recordFailure records a failure and potentially opens the circuit
func (cb *CircuitBreaker) recordFailure() {
	atomic.AddInt32(&cb.failures, 1)
	atomic.StoreInt64(&cb.lastFailTime, time.Now().UnixNano())

	if atomic.LoadInt32(&cb.failures) >= cb.failureThreshold {
		cb.setState(StateOpen)
	}
}

// recordSuccess records a success and potentially closes the circuit
func (cb *CircuitBreaker) recordSuccess() {
	if cb.getState() == StateHalfOpen {
		atomic.StoreInt32(&cb.failures, 0)
		cb.setState(StateClosed)
	}
}

// shouldAttemptReset determines if enough time has passed to attempt a reset
func (cb *CircuitBreaker) shouldAttemptReset() bool {
	lastFailTime := atomic.LoadInt64(&cb.lastFailTime)
	if lastFailTime == 0 {
		return false
	}

	return time.Since(time.Unix(0, lastFailTime)) >= cb.resetTimeout
}

// GetState returns current circuit breaker state
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	return cb.getState()
}

func (cb *CircuitBreaker) getState() CircuitBreakerState {
	return CircuitBreakerState(atomic.LoadInt32(&cb.state))
}

func (cb *CircuitBreaker) setState(state CircuitBreakerState) {
	atomic.StoreInt32(&cb.state, int32(state))
}

// GetMetrics returns current circuit breaker metrics
func (cb *CircuitBreaker) GetMetrics() CircuitBreakerMetrics {
	return CircuitBreakerMetrics{
		State:                cb.getState(),
		TotalRequests:        atomic.LoadInt64(&cb.totalRequests),
		TotalFailures:        atomic.LoadInt64(&cb.totalFailures),
		TotalSuccesses:       atomic.LoadInt64(&cb.totalSuccesses),
		ConsecutiveSuccesses: atomic.LoadInt32(&cb.consecutiveSuccesses),
		CurrentFailures:      atomic.LoadInt32(&cb.failures),
	}
}

// CircuitBreakerMetrics holds circuit breaker performance metrics
type CircuitBreakerMetrics struct {
	State                CircuitBreakerState `json:"state"`
	TotalRequests        int64               `json:"total_requests"`
	TotalFailures        int64               `json:"total_failures"`
	TotalSuccesses       int64               `json:"total_successes"`
	ConsecutiveSuccesses int32               `json:"consecutive_successes"`
	CurrentFailures      int32               `json:"current_failures"`
}

// CircuitBreakerError represents a circuit breaker error
type CircuitBreakerError struct {
	Message string
	State   CircuitBreakerState
}

func (e *CircuitBreakerError) Error() string {
	return e.Message
}

// NewCircuitBreakerError creates a new circuit breaker error
func NewCircuitBreakerError(message string, state CircuitBreakerState) *CircuitBreakerError {
	return &CircuitBreakerError{
		Message: message,
		State:   state,
	}
}

// CircuitBreakerRegistry manages multiple circuit breakers
type CircuitBreakerRegistry struct {
	breakers map[string]*CircuitBreaker
	mu       sync.RWMutex
}

// NewCircuitBreakerRegistry creates a new registry
func NewCircuitBreakerRegistry() *CircuitBreakerRegistry {
	return &CircuitBreakerRegistry{
		breakers: make(map[string]*CircuitBreaker),
	}
}

// GetOrCreate gets an existing circuit breaker or creates a new one
func (r *CircuitBreakerRegistry) GetOrCreate(name string, config CircuitBreakerConfig) *CircuitBreaker {
	r.mu.Lock()
	defer r.mu.Unlock()

	if breaker, exists := r.breakers[name]; exists {
		return breaker
	}

	breaker := NewCircuitBreaker(config)
	r.breakers[name] = breaker
	return breaker
}

// GetBreaker gets an existing circuit breaker
func (r *CircuitBreakerRegistry) GetBreaker(name string) (*CircuitBreaker, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	breaker, exists := r.breakers[name]
	return breaker, exists
}

// GetAllMetrics returns metrics for all circuit breakers
func (r *CircuitBreakerRegistry) GetAllMetrics() map[string]CircuitBreakerMetrics {
	r.mu.RLock()
	defer r.mu.RUnlock()

	metrics := make(map[string]CircuitBreakerMetrics)
	for name, breaker := range r.breakers {
		metrics[name] = breaker.GetMetrics()
	}
	return metrics
}

// Global registry instance
var globalRegistry = NewCircuitBreakerRegistry()

// GetCircuitBreaker gets or creates a circuit breaker from the global registry
func GetCircuitBreaker(name string, config CircuitBreakerConfig) *CircuitBreaker {
	return globalRegistry.GetOrCreate(name, config)
}

// GetCircuitBreakerMetrics returns metrics from the global registry
func GetCircuitBreakerMetrics() map[string]CircuitBreakerMetrics {
	return globalRegistry.GetAllMetrics()
}
