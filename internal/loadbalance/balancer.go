package loadbalance

import "kamaRPC/internal/registry"

type LoadBalancer interface {
	Select([]registry.Instance) registry.Instance
}
