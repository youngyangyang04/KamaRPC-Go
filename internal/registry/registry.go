package registry

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

type Instance struct {
	Addr string
}

type Registry struct {
	client *clientv3.Client
	prefix string

	mu       sync.RWMutex
	services map[string]map[string]Instance // service -> addr -> Instance

	ctx    context.Context
	cancel context.CancelFunc
}

func NewRegistry(endpoints []string) (*Registry, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Registry{
		client:   cli,
		prefix:   "/kamaRPC/services/",
		services: make(map[string]map[string]Instance),
		ctx:      ctx,
		cancel:   cancel,
	}, nil
}

func (r *Registry) Register(service string, ins Instance, ttl int64) error {
	leaseResp, err := r.client.Grant(r.ctx, ttl)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("%s%s/%s", r.prefix, service, ins.Addr)

	_, err = r.client.Put(r.ctx, key, ins.Addr, clientv3.WithLease(leaseResp.ID))
	if err != nil {
		return err
	}

	ch, err := r.client.KeepAlive(r.ctx, leaseResp.ID)
	if err != nil {
		return err
	}

	go func() {
		for {
			_, ok := <-ch
			if !ok {
				return
			}
		}
	}()

	return nil
}

// Discover：只在第一次调用时建立缓存 + watch
func (r *Registry) Discover(service string) ([]Instance, error) {
	r.mu.RLock()
	if _, ok := r.services[service]; ok {
		instances := r.copyInstances(service)
		r.mu.RUnlock()
		return instances, nil
	}
	r.mu.RUnlock()

	// 第一次发现，初始化
	if err := r.initService(service); err != nil {
		return nil, err
	}

	return r.copyInstances(service), nil
}

// 初始化服务缓存
func (r *Registry) initService(service string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 防止重复初始化
	if _, ok := r.services[service]; ok {
		return nil
	}

	key := fmt.Sprintf("%s%s/", r.prefix, service)

	// 先 Get 一次
	resp, err := r.client.Get(r.ctx, key, clientv3.WithPrefix())
	if err != nil {
		return err
	}

	r.services[service] = make(map[string]Instance)

	for _, kv := range resp.Kvs {
		addr := string(kv.Value)
		r.services[service][addr] = Instance{Addr: addr}
	}

	// 启动 watch
	go r.watch(service)

	return nil
}

func (r *Registry) watch(service string) {
	key := fmt.Sprintf("%s%s/", r.prefix, service)

	for {
		watchChan := r.client.Watch(r.ctx, key, clientv3.WithPrefix())

		for watchResp := range watchChan {
			for _, event := range watchResp.Events {
				switch event.Type {

				case clientv3.EventTypePut:
					addr := string(event.Kv.Value)
					r.mu.Lock()
					r.services[service][addr] = Instance{Addr: addr}
					r.mu.Unlock()

				case clientv3.EventTypeDelete:
					deletedKey := string(event.Kv.Key)
					addr := strings.TrimPrefix(deletedKey, r.prefix+service+"/")
					r.mu.Lock()
					delete(r.services[service], addr)
					r.mu.Unlock()
				}
			}
		}

		// watch 断了，稍后重连
		time.Sleep(time.Second)
	}
}

func (r *Registry) copyInstances(service string) []Instance {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var instances []Instance
	for _, ins := range r.services[service] {
		instances = append(instances, ins)
	}
	return instances
}

func (r *Registry) Close() error {
	r.cancel()
	return r.client.Close()
}
