package main

import (
	"context"
	"fmt"
	"kamaRPC/internal/client"
	"kamaRPC/internal/codec"
	"kamaRPC/internal/registry"
	"kamaRPC/internal/transport"
	"kamaRPC/pkg/api"
	"log"
	"time"
)

func main() {
	reg, _ := registry.NewRegistry([]string{"localhost:2379"})

	c, err := client.NewClient(
		reg,
		client.WithClientCodec(codec.JSON),
	)
	if err != nil {
		log.Println("NewClient error:", err)
		return
	}

	type call struct {
		future *transport.Future
		args   *api.Args
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	log.Println("开始周期性发送请求...")

	counter := 0

	for {
		<-ticker.C

		fmt.Println("====== 新一轮请求 ======")

		var calls []call

		// 每轮发送 3 个请求
		for i := 0; i < 3; i++ {
			args := &api.Args{
				A: counter,
				B: counter,
			}
			counter++

			f, err := c.InvokeAsync(
				context.Background(),
				"Arith",
				"Add",
				args,
			)
			if err != nil {
				log.Println("send error:", err)
				continue
			}

			calls = append(calls, call{f, args})
		}

		// 等待这一轮的 3 个完成
		doneCh := make(chan call)
		log.Println("需要等待", len(calls), "个任务完成")
		for _, item := range calls {
			go func(it call) {
				<-it.future.DoneChan()
				doneCh <- it
			}(item)
		}

		for i := 0; i < len(calls); i++ {
			item := <-doneCh

			reply := &api.Reply{}
			err := item.future.GetResult(reply)
			if err != nil {
				log.Println("get error:", err)
				continue
			}

			fmt.Printf("Add %v+%v result: %v\n",
				item.args.A,
				item.args.B,
				reply.Result)
		}
	}
}
