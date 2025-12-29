package stun

import (
	"context"
	"errors"
	"log"
	"net"
	"sync"
	"time"

	"p2p-msg/internal/udp"
	pb "p2p-msg/pkg/protobuf/gen/p2p-msg/pkg/protobuf/gen/stun"

	"google.golang.org/protobuf/proto"
)

type _PeerRegistryEntry struct {
	name    string
	addr    net.Addr
	key     []byte
	expires time.Time
}

type StunServer struct {
	ListenAddress net.Addr

	connection    *udp.UDPConnection
	disposeChan   chan any
	connectionMux sync.Mutex

	peerRegistry        map[string]_PeerRegistryEntry
	registryTTLDuration time.Duration
	registryMux         sync.Mutex
	registryTicker      *time.Ticker
}

func NewServer(addr net.Addr) (*StunServer, error) {
	server := StunServer{
		ListenAddress: addr,
	}

	// TODO: get timeout and buffer size from config
	conn, err := udp.NewConnection(addr, 1000, 512)
	if err != nil {
		log.Println("Error creating socket:", err)
		return nil, err
	}

	server.connection = conn
	server.disposeChan = make(chan any)

	server.peerRegistry = make(map[string]_PeerRegistryEntry)
	server.registryTTLDuration = time.Duration(30) * time.Second
	server.registryTicker = time.NewTicker(server.registryTTLDuration)

	return &server, nil
}

func (s *StunServer) Serve(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			log.Println("Received context cancellation, exiting read cycle")
			return
		case <-s.disposeChan:
			log.Println("Received shutdown command, exiting read cycle")
			return
		case <-s.registryTicker.C:
			s.renewRegistry()
		default:
			if err := s.processRead(); err != nil {
				log.Println("Got error processing udp read: ", err)
			}
		}
	}
}

func (s *StunServer) renewRegistry() {
	log.Println("Removing expired peer registry entries")

	log.Println("Locking registry mutex...")
	s.registryMux.Lock()
	defer s.registryMux.Unlock()

	before := len(s.peerRegistry)

	log.Println("Cleaning registry...")
	for key := range s.peerRegistry {
		item, _ := s.peerRegistry[key]
		if item.expires.Unix() <= time.Now().Unix() {
			delete(s.peerRegistry, key)
		}
	}

	delta := before - len(s.peerRegistry)
	log.Printf("Removed %d expired entries", delta)
}

func (s *StunServer) addToRegistry(r *_PeerRegistryEntry) {
	log.Println("Adding new record to registry...", r)

	log.Println("Locking registry mutex...")
	s.registryMux.Lock()
	defer s.registryMux.Unlock()

	log.Println("Mutex locked, inserting...")

	s.peerRegistry[r.name] = *r

	log.Println("Inserted")
}

func (s *StunServer) lookupRegistry(name string) (_PeerRegistryEntry, bool) {
	log.Println("Registry lookup by name", name)

	log.Println("Locking registry mutex...")
	s.registryMux.Lock()
	defer s.registryMux.Unlock()

	log.Println("Mutex locked, reading...")

	res, exist := s.peerRegistry[name]

	log.Println("Lookup result", res, exist)

	return res, exist
}

func (s *StunServer) processRead() error {
	if locked := s.connectionMux.TryLock(); !locked {
		return errors.New("Can't lock connection mutex")
	}
	defer s.connectionMux.Unlock()

	request := pb.StunMessage{}
	from, err := udp.ReadProtobuf(s.connection, &request)
	if err != nil {
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			return nil
		}

		return err
	}

	var response *pb.StunMessage

	if request.MessageClass != pb.MsgClass_REQUEST && request.MessageClass != pb.MsgClass_INDICATION {
		return errors.New("Read non-request packet, ignoring")
	}

	switch request.MessageMethod {
	case pb.MsgMethod_DISCOVER:
		response, err = s.handleDiscover(&request, from)
	case pb.MsgMethod_BINDING:
		response, err = s.handleBinding(&request)
	default:
		response = s.handleError(&request)
	}

	if err != nil || response == nil {
		return err
	}

	if err := udp.WriteProtobuf(s.connection, response, from); err != nil {
		return err
	}

	return nil
}

func (s *StunServer) handleDiscover(req *pb.StunMessage, from net.Addr) (*pb.StunMessage, error) {
	discoverRequest := pb.DiscoverRequest{}
	if err := proto.Unmarshal(req.Content, &discoverRequest); err != nil {
		return nil, err
	}

	log.Println("Received DISCOVER request: ", discoverRequest.String())

	s.addToRegistry(&_PeerRegistryEntry{
		name: discoverRequest.Name,
		addr: from,
		key:  discoverRequest.Key,

		expires: time.Now().Add(s.registryTTLDuration),
	})

	log.Println("Added to registry: ", discoverRequest.Name, from.String())

	discoverData := pb.DiscoverResponse{
		Address: from.String(),
	}

	content, err := proto.Marshal(&discoverData)
	if err != nil {
		return nil, err
	}

	response := pb.StunMessage{
		MessageClass:  pb.MsgClass_SUCCESS_RESPONSE,
		MessageMethod: pb.MsgMethod_DISCOVER,
		Content:       content,
	}

	log.Println("Formed discover response: ", response.String())

	return &response, nil
}

func (s *StunServer) handleBinding(req *pb.StunMessage) (*pb.StunMessage, error) {
	bindingRequest := pb.BindingRequest{}
	if err := proto.Unmarshal(req.Content, &bindingRequest); err != nil {
		return nil, err
	}

	log.Println("Received BINDING request: ", bindingRequest.String())

	peerInfo, exist := s.lookupRegistry(bindingRequest.PeerName)
	if !exist {
		return s.handleError(req), nil
	}

	bindingData := pb.BindingResponse{
		Name:    peerInfo.name,
		Address: peerInfo.addr.String(),
		Key:     peerInfo.key,
	}

	content, err := proto.Marshal(&bindingData)
	if err != nil {
		return nil, err
	}
	response := pb.StunMessage{
		MessageClass:  pb.MsgClass_SUCCESS_RESPONSE,
		MessageMethod: pb.MsgMethod_BINDING,
		Content:       content,
	}

	log.Println("Formed binding response: ", response.String())

	return &response, nil
}

func (s *StunServer) handleError(req *pb.StunMessage) *pb.StunMessage {
	log.Println("Received unprocessable request: ", req.String())

	response := pb.StunMessage{
		MessageClass:  pb.MsgClass_FAILURE_RESPONSE,
		MessageMethod: req.MessageMethod,
		Content:       nil,
	}

	log.Println("Formed failure response: ", response.String())

	return &response
}

func (s *StunServer) Dispose() error {
	log.Println("Received dispose command, processing...")

	log.Println("Stopping registry ticker")
	s.registryTicker.Stop()
	log.Println("Registry ticker stopped")

	log.Println("Waiting to lock connection mutex...")
	s.connectionMux.Lock()
	defer s.connectionMux.Unlock()

	log.Println("Connection mutex locked, sending shutdown command...")
	s.disposeChan <- 1

	log.Println("Closing connection...")
	return s.connection.Close()
}
