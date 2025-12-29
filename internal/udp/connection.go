package udp

import (
	"errors"
	"net"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"
)

type UDPConnection struct {
	Connection net.PacketConn
	Timeout    time.Duration
	BufferSize int

	buffer    []byte
	readMutex sync.Mutex
}

func NewConnection(listenAddr net.Addr, idleTimeoutMs int, bufferSize int) (*UDPConnection, error) {
	c := UDPConnection{}

	if idleTimeoutMs <= 0 || idleTimeoutMs > 1_000*60 { // must be non-zero value less than 1 minute
		return nil, errors.New("timeout must be non-zero value less than 1 minute")
	}
	c.Timeout = time.Duration(idleTimeoutMs) * time.Millisecond

	if bufferSize < 128 || bufferSize > 1024*1024 { // must be between 128 bytes and 1 mb
		return nil, errors.New("buffer size must be between 128 bytes and 1 MB")
	}
	c.BufferSize = bufferSize
	c.buffer = make([]byte, bufferSize)

	conn, err := net.ListenPacket("udp", listenAddr.String())
	if err != nil {
		return nil, err
	}
	c.Connection = conn

	return &c, nil
}

func (c *UDPConnection) Close() error {
	return c.Connection.Close()
}

func ReadProtobuf[T any, PT interface {
	*T
	proto.Message
}](c *UDPConnection, out PT) (net.Addr, error) {
	buf, addr, err := c.Read()
	if err != nil {
		return nil, err
	}

	if err = proto.Unmarshal(buf, out); err != nil {
		return nil, err
	}

	return addr, nil
}

func WriteProtobuf[T any, PT interface {
	*T
	proto.Message
}](c *UDPConnection, message PT, addr net.Addr) error {
	marshalled, err := proto.Marshal(message)
	if err != nil {
		return err
	}
	return c.Write(marshalled, addr)
}

func (c *UDPConnection) Read() ([]byte, net.Addr, error) {
	c.readMutex.Lock()
	defer c.readMutex.Unlock()

	if err := c.Connection.SetReadDeadline(time.Now().Add(c.Timeout)); err != nil {
		return nil, nil, err
	}

	n, addr, err := c.Connection.ReadFrom(c.buffer)
	if err != nil {
		return nil, nil, err
	}

	result := make([]byte, n)
	copy(result, c.buffer[:n])
	clear(c.buffer)

	return result, addr, nil
}

func (c *UDPConnection) Write(buf []byte, addr net.Addr) error {
	if err := c.Connection.SetWriteDeadline(time.Now().Add(c.Timeout)); err != nil {
		return err
	}

	if _, err := c.Connection.WriteTo(buf, addr); err != nil {
		return err
	}

	return nil
}
