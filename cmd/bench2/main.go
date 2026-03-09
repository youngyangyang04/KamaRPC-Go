package main

import (
	"context"
	"flag"
	"fmt"
	"kamaRPC/internal/client"
	"kamaRPC/internal/codec"
	"kamaRPC/internal/registry"
	"kamaRPC/pkg/api"
	"log"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

var (
	concurrency = flag.Int("c", 50, "并发客户端数量")
	durationSec = flag.Int("d", 10, "基准测试持续时间（单位为秒）")
	etcdAddr    = flag.String("etcd", "localhost:2379", "etcd的地址")
	serviceName = flag.String("s", "Arith", "服务名")
	methodName  = flag.String("m", "Add", "方法名")
)

type metrics struct {
	success int64
	fail    int64
	latency []int64
	mu      sync.Mutex
}

func (m *metrics) recordLatency(us int64) {
	m.mu.Lock()
	m.latency = append(m.latency, us)
	m.mu.Unlock()
}

func main() {

	flag.Parse()

	log.Printf("Starting benchmark: concurrency=%d duration=%ds\n",
		*concurrency, *durationSec)

	reg, err := registry.NewRegistry([]string{*etcdAddr})
	if err != nil {
		log.Fatal(err)
	}
	defer reg.Close()

	c, err := client.NewClient(
		reg,
		client.WithClientCodec(codec.JSON),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	var wg sync.WaitGroup
	var m metrics

	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Duration(*durationSec)*time.Second,
	)
	defer cancel()

	start := time.Now()

	for i := 0; i < *concurrency; i++ {

		wg.Add(1)

		go func() {

			defer wg.Done()

			for {

				select {

				case <-ctx.Done():
					return

				default:

					args := &api.Args{A: 1, B: 2}

					reqStart := time.Now()

					reply := &api.Reply{}

					err := c.Invoke(
						context.Background(),
						*serviceName,
						*methodName,
						args,
						reply,
					)

					lat := time.Since(reqStart).Microseconds()

					if err != nil {
						atomic.AddInt64(&m.fail, 1)
						continue
					}

					atomic.AddInt64(&m.success, 1)

					m.recordLatency(lat)
				}
			}

		}()
	}

	wg.Wait()

	totalTime := time.Since(start)

	printStats(&m, totalTime)
}

func printStats(m *metrics, duration time.Duration) {

	total := m.success + m.fail

	qps := float64(m.success) / duration.Seconds()

	sort.Slice(m.latency, func(i, j int) bool {
		return m.latency[i] < m.latency[j]
	})

	p50 := percentile(m.latency, 50)
	p90 := percentile(m.latency, 90)
	p99 := percentile(m.latency, 99)

	var avg float64

	if len(m.latency) > 0 {

		var sum int64

		for _, v := range m.latency {
			sum += v
		}

		avg = float64(sum) / float64(len(m.latency)) / 1000
	}

	fmt.Println()
	fmt.Println("========= Benchmark Result =========")

	fmt.Printf("Duration:        %v\n", duration)
	fmt.Printf("Total Requests:  %d\n", total)
	fmt.Printf("Success:         %d\n", m.success)
	fmt.Printf("Failed:          %d\n", m.fail)

	fmt.Printf("QPS:             %.2f\n", qps)

	fmt.Printf("Avg Latency:     %.2f ms\n", avg)
	fmt.Printf("P50 Latency:     %.2f ms\n", float64(p50)/1000)
	fmt.Printf("P90 Latency:     %.2f ms\n", float64(p90)/1000)
	fmt.Printf("P99 Latency:     %.2f ms\n", float64(p99)/1000)
}

func percentile(data []int64, p int) int64 {

	if len(data) == 0 {
		return 0
	}

	k := int(float64(len(data)) * float64(p) / 100.0)

	if k >= len(data) {
		k = len(data) - 1
	}

	return data[k]
}
