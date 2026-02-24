package transport

import (
	"bufio"
	"io"
	"kamaRPC/internal/protocol"
	"net"
	"sync"
)

const BufferSize = 4096

// 包缓冲区（处理粘包）
type PacketBuffer struct {
	buf  []byte
	lock sync.Mutex
}

func (pb *PacketBuffer) Write(data []byte) {
	pb.lock.Lock()
	pb.buf = append(pb.buf, data...)
	pb.lock.Unlock()
}

func (pb *PacketBuffer) Read() []byte {
	pb.lock.Lock()
	defer pb.lock.Unlock()

	// 最小包头长度校验
	if len(pb.buf) < 10 {
		return nil
	}

	headerLen := int(protocol.DecodeHeaderLen(pb.buf[2:6]))
	bodyLen := int(protocol.DecodeBodyLen(pb.buf[6:10]))
	totalLen := 10 + headerLen + bodyLen

	if len(pb.buf) < totalLen {
		return nil
	}

	packet := make([]byte, totalLen)
	copy(packet, pb.buf[:totalLen])

	// 移动窗口
	pb.buf = pb.buf[totalLen:]
	return packet
}

type TCPConnection struct {
	conn   net.Conn
	reader *bufio.Reader
	buffer *PacketBuffer

	writeMu sync.Mutex
}

// 创建连接
func NewTCPConnection(conn net.Conn) *TCPConnection {
	return &TCPConnection{
		conn:   conn,
		reader: bufio.NewReaderSize(conn, BufferSize),
		buffer: &PacketBuffer{
			buf: make([]byte, 0, BufferSize*2),
		},
	}
}

func (tc *TCPConnection) Read() (*protocol.Message, error) {
	for {
		// 尝试从缓冲区取完整包
		if packet := tc.buffer.Read(); packet != nil {
			return protocol.Decode(packet)
		}

		tmp := make([]byte, BufferSize)
		n, err := tc.reader.Read(tmp)
		if err != nil {
			if err == io.EOF {
				return nil, err
			}
			return nil, err
		}

		if n > 0 {
			tc.buffer.Write(tmp[:n])
		}
	}
}

func (tc *TCPConnection) Write(msg *protocol.Message) error {
	data, err := protocol.Encode(msg)
	if err != nil {
		return err
	}

	tc.writeMu.Lock()
	defer tc.writeMu.Unlock()

	total := 0
	for total < len(data) {
		n, err := tc.conn.Write(data[total:])
		if err != nil {
			return err
		}
		total += n
	}

	return nil
}

// 关闭连接
func (tc *TCPConnection) Close() error {
	if tcp, ok := tc.conn.(*net.TCPConn); ok {
		tcp.SetLinger(0)
	}
	return tc.conn.Close()
}

func (tc *TCPConnection) RemoteAddr() string {
	return tc.conn.RemoteAddr().String()
}
