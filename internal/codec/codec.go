package codec

import (
	"fmt"
	"sync"
)

type Codec interface {
	Marshal(v interface{}) ([]byte, error)
	Unmarshal(data []byte, v interface{}) error
}

// Type 用于标识不同的编解码类型
type Type byte

// Factory 定义 codec 构造函数
type Factory func() Codec

var (
	mu        sync.RWMutex
	factories = make(map[Type]Factory)
)

func Register(t Type, f Factory) {
	mu.Lock()
	defer mu.Unlock()

	if f == nil {
		panic("codec: factory is nil")
	}

	if _, exists := factories[t]; exists {
		panic(fmt.Sprintf("codec: type %d already registered", t))
	}

	factories[t] = f
}

// New 根据 Type 创建 codec 实例
func New(t Type) (Codec, error) {
	mu.RLock()
	defer mu.RUnlock()

	f, ok := factories[t]
	if !ok {
		return nil, fmt.Errorf("codec: type %d not registered", t)
	}

	return f(), nil
}
