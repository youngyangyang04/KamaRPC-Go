package client

import (
	"context"
	"errors"
	"kamaRPC/internal/breaker"
	"kamaRPC/internal/codec"
	"kamaRPC/internal/limiter"
	"kamaRPC/internal/loadbalance"
	"kamaRPC/internal/protocol"
	"kamaRPC/internal/registry"
	"kamaRPC/internal/transport"
	"log"
	"sync"
	"time"
)

type Client struct {
	reg     *registry.Registry
	lb      loadbalance.LoadBalancer
	limiter *limiter.TokenBucket
	timeout time.Duration
	codec   codec.Codec
	breaker sync.Map // map[string]*CircuitBreaker

	pools sync.Map // map[string]*transport.ConnectionPool
}

func NewClient(reg *registry.Registry, opts ...ClientOption) (*Client, error) {
	c := &Client{
		reg:     reg,
		lb:      &loadbalance.RoundRobin{},
		limiter: limiter.NewTokenBucket(10000),
		timeout: 5 * time.Second,
	}
	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}
	return c, nil
}

func (c *Client) InvokeAsync(ctx context.Context, service string, method string, args interface{}) (*transport.Future, error) {

	if !c.limiter.Allow() {
		return nil, errors.New("rate limit exceeded")
	}

	addr, err := c.getAddr(service)
	if err != nil {
		return nil, err
	}
	br := c.getBreaker(service, addr)

	if !br.Allow() {
		return nil, errors.New("circuit breaker open")
	}

	pool := c.getPool(addr)

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}

	body, err := c.codec.Marshal(args)
	if err != nil {
		return nil, err
	}

	req := &protocol.Message{
		Header: &protocol.Header{
			ServiceName: service,
			MethodName:  method,
			Compression: codec.CompressionGzip,
		},
		Body: body,
	}
	future, err := conn.SendAsync(req)
	if err != nil {
		br.RecordFailure()
		return nil, err
	}

	future.OnComplete(func(err error) {
		if err != nil {
			br.RecordFailure()
		} else {
			br.RecordSuccess()
		}
	})

	return future, nil
}

// 同步接口 = 异步 + 等待
func (c *Client) Invoke(ctx context.Context, service string, method string, args interface{}, reply interface{}) error {

	future, err := c.InvokeAsync(ctx, service, method, args)
	if err != nil {
		return err
	}

	return future.GetResultWithContext(ctx, reply)
}

func (c *Client) getPool(addr string) *transport.ConnectionPool {
	if pool, ok := c.pools.Load(addr); ok {
		return pool.(*transport.ConnectionPool)
	}

	newPool := transport.NewConnectionPool(addr, 0, 1)
	actual, _ := c.pools.LoadOrStore(addr, newPool)
	return actual.(*transport.ConnectionPool)
}

func (c *Client) getAddr(service string) (string, error) {
	if c.reg == nil {
		return "", errors.New("registry not configured")
	}

	instances, err := c.reg.Discover(service)
	if err != nil {
		return "", err
	}

	if len(instances) == 0 {
		return "", errors.New("no instance available")
	}

	instance := c.lb.Select(instances)
	log.Println("选择的地址为:", instance.Addr)
	return instance.Addr, nil
}

func (c *Client) Close() {
	c.pools.Range(func(key, value interface{}) bool {
		pool := value.(*transport.ConnectionPool)
		pool.Close()
		return true
	})
}

func (c *Client) getBreaker(service, addr string) *breaker.CircuitBreaker {

	key := service + "|" + addr

	if val, ok := c.breaker.Load(key); ok {
		return val.(*breaker.CircuitBreaker)
	}

	newBreaker := breaker.NewCircuitBreaker(
		10,
		0.6,           // 60% 错误率熔断
		5*time.Second, // 熔断 5 秒
	)

	actual, _ := c.breaker.LoadOrStore(key, newBreaker)

	return actual.(*breaker.CircuitBreaker)
}
