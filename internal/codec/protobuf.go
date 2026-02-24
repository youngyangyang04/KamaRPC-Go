package codec

import (
	"fmt"

	"google.golang.org/protobuf/proto"
)

const PROTO Type = 2

type protoCodec struct{}

func (p *protoCodec) Marshal(v interface{}) ([]byte, error) {
	msg, ok := v.(proto.Message)
	if !ok {
		return nil, fmt.Errorf("proto codec: not proto.Message")
	}
	return proto.Marshal(msg)
}

func (p *protoCodec) Unmarshal(data []byte, v interface{}) error {
	msg, ok := v.(proto.Message)
	if !ok {
		return fmt.Errorf("proto codec: not proto.Message")
	}
	return proto.Unmarshal(data, msg)
}

func init() {
	Register(PROTO, func() Codec {
		return &protoCodec{}
	})
}
