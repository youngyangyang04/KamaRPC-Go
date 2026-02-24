package protocol

import "kamaRPC/internal/codec"

// CodecType 编解码器类型
type CodecType byte

const (
	CodecTypeJSON CodecType = iota + 1
	CodecTypeProto
)

type Header struct {
	RequestID   uint64
	ServiceName string
	MethodName  string
	Error       string
	CodecType   CodecType
	Compression codec.CompressionType
}
