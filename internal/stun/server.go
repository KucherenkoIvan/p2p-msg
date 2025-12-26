package stun

import (
	"net"
	"sync"
)

type StunServer struct {
	listenMutex sync.Mutex
}

func NewServer(addr net.Addr) {}

func (s *StunServer) ListenAndServe() {}
