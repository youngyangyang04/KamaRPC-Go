package protocol

import (
	"encoding/binary"
	"fmt"
	"kamaRPC/internal/codec"
)

const Magic uint16 = 0x1234

type Message struct {
	Header *Header
	Body   []byte
}

func Encode(msg *Message) ([]byte, error) {

	if msg.Header == nil {
		return nil, fmt.Errorf("header is nil")
	}

	bodyBytes := msg.Body

	if msg.Header.Compression != codec.CompressionNone {
		var err error
		bodyBytes, err = codec.Compress(bodyBytes, msg.Header.Compression)
		if err != nil {
			return nil, err
		}
	}

	headerCodec, err := codec.New(codec.JSON)
	if err != nil {
		return nil, err
	}

	headerBytes, err := headerCodec.Marshal(msg.Header)
	if err != nil {
		return nil, err
	}

	headerLen := uint32(len(headerBytes))
	bodyLen := uint32(len(bodyBytes))

	total := 2 + 4 + 4 + headerLen + bodyLen
	buf := make([]byte, total)

	binary.BigEndian.PutUint16(buf[0:2], Magic)

	binary.BigEndian.PutUint32(buf[2:6], headerLen)

	binary.BigEndian.PutUint32(buf[6:10], bodyLen)

	copy(buf[10:], headerBytes)

	copy(buf[10+headerLen:], bodyBytes)

	return buf, nil
}

// DecodeHeaderLen 从字节切片解析 headerLen
func DecodeHeaderLen(data []byte) uint32 {
	return binary.BigEndian.Uint32(data)
}

// DecodeBodyLen 从字节切片解析 bodyLen
func DecodeBodyLen(data []byte) uint32 {
	return binary.BigEndian.Uint32(data)
}

// DecodeBytes 从字节数组解码完整的 Message（用于粘包处理）
func Decode(data []byte) (*Message, error) {

	if len(data) < 10 {
		return nil, fmt.Errorf("data too short")
	}

	// 检查 Magic
	if binary.BigEndian.Uint16(data[0:2]) != Magic {
		return nil, fmt.Errorf("invalid magic number")
	}

	headerLen := binary.BigEndian.Uint32(data[2:6])
	bodyLen := binary.BigEndian.Uint32(data[6:10])

	totalLen := 10 + int(headerLen) + int(bodyLen)
	if len(data) < totalLen {
		return nil, fmt.Errorf("incomplete packet")
	}

	headerBytes := data[10 : 10+headerLen]

	headerCodec, err := codec.New(codec.JSON)
	if err != nil {
		return nil, err
	}

	var header Header
	if err := headerCodec.Unmarshal(headerBytes, &header); err != nil {
		return nil, err
	}

	// 读取 body
	bodyBytes := data[10+headerLen : 10+headerLen+bodyLen]

	if header.Compression != codec.CompressionNone {
		bodyBytes, err = codec.Decompress(bodyBytes, header.Compression)
		if err != nil {
			return nil, err
		}
	}

	return &Message{
		Header: &header,
		Body:   bodyBytes,
	}, nil
}
