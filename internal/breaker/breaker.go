package breaker

import (
	"log"
	"sync"
	"time"
)

type State int

const (
	Closed State = iota
	Open
	HalfOpen
)

type CircuitBreaker struct {
	mu sync.Mutex

	state State

	// 统计数据
	failureCount int
	successCount int

	// 配置参数
	windowSize       int           // 统计窗口大小（次数）
	failureThreshold float64       // 失败率阈值
	openTimeout      time.Duration // 熔断持续时间

	// 状态控制
	lastStateChange time.Time
	halfOpenProbe   bool // 半开状态下是否已有探测请求
}

func NewCircuitBreaker(windowSize int, failureThreshold float64, openTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:            Closed,
		windowSize:       windowSize,
		failureThreshold: failureThreshold,
		openTimeout:      openTimeout,
		lastStateChange:  time.Now(),
	}
}
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {

	case Closed:
		return true

	case Open:
		// 熔断时间到了，进入半开
		if time.Since(cb.lastStateChange) > cb.openTimeout {
			cb.state = HalfOpen
			cb.halfOpenProbe = false
			return true
		}
		return false

	case HalfOpen:
		// 只允许一个探测请求
		if cb.halfOpenProbe {
			return false
		}
		cb.halfOpenProbe = true
		return true
	}

	return true
}
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {

	case Closed:
		cb.successCount++

	case HalfOpen:
		// 探测成功 → 恢复
		cb.toClosed()

	case Open:
		//按道理是不会进入这块的
		log.Println("理论不发生触发")
	}
}
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {

	case Closed:
		cb.failureCount++

		total := cb.failureCount + cb.successCount
		if total < cb.windowSize {
			return
		}

		rate := float64(cb.failureCount) / float64(total)
		if rate >= cb.failureThreshold {
			cb.toOpen()
			return
		}

		cb.resetCounts()

	case HalfOpen:
		// 探测失败 → 重新熔断
		cb.toOpen()

	case Open:
		// 已经熔断，不处理
	}
}
func (cb *CircuitBreaker) toOpen() {
	cb.state = Open
	cb.lastStateChange = time.Now()
	cb.resetCounts()
	cb.halfOpenProbe = false
}

func (cb *CircuitBreaker) toClosed() {
	cb.state = Closed
	cb.lastStateChange = time.Now()
	cb.resetCounts()
	cb.halfOpenProbe = false
}
func (cb *CircuitBreaker) resetCounts() {
	cb.failureCount = 0
	cb.successCount = 0
}

func (cb *CircuitBreaker) State() State {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}
