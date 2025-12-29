package stun

import (
	"context"
	"errors"
	"log"
	"net"
	"p2p-msg/internal/udp"
	"sync"

	pb "p2p-msg/pkg/protobuf/gen/p2p-msg/pkg/protobuf/gen/stun"

	"google.golang.org/protobuf/proto"
)

type StunClient struct {
	Name   string
	Server net.Addr

	connection *udp.UDPConnection
	publicKey  []byte
	privateKey []byte

	connectionMux sync.Mutex
}

func NewClient(listenAddr net.Addr, server net.Addr, name string, publicKey []byte, privateKey []byte) (*StunClient, error) {
	client := StunClient{}

	conn, err := udp.NewConnection(listenAddr, 1000, 512)
	if err != nil {
		log.Println("Error creating socket:", err)

		return nil, err
	}

	client.connection = conn

	client.Name = name
	client.Server = server
	client.publicKey = publicKey
	client.privateKey = privateKey

	return &client, nil
}

func (c *StunClient) Serve(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			log.Println("Received context cancellation, exiting read cycle")
			return
		default:
			if err := c.processRead(); err != nil {
				log.Println("Got error processing udp read: ", err)
			}
		}
	}
}

func (c *StunClient) processRead() error {
	c.connectionMux.Lock()
	defer c.connectionMux.Unlock()

	request := pb.StunMessage{}
	from, err := udp.ReadProtobuf(c.connection, &request)
	if err != nil {
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			return nil
		}

		return err
	}

	var response *pb.StunMessage

	switch request.MessageMethod {
	case pb.MsgMethod_ECHO:
		response = c.handleEcho(&request, from)
	default:
		response = c.handleError(&request)
	}
	if err != nil {
		return err
	}

	if err := udp.WriteProtobuf(c.connection, response, from); err != nil {
		return err
	}

	return nil
}

func (c *StunClient) handleEcho(req *pb.StunMessage, from net.Addr) *pb.StunMessage {
	response := pb.StunMessage{
		MessageClass:  pb.MsgClass_SUCCESS_RESPONSE,
		MessageMethod: req.MessageMethod,
		Content:       req.Content,
	}

	log.Println("Got ECHO from ", from.String())

	return &response
}

func (c *StunClient) handleError(req *pb.StunMessage) *pb.StunMessage {
	log.Println("Received unprocessable request: ", req.String())

	response := pb.StunMessage{
		MessageClass:  pb.MsgClass_FAILURE_RESPONSE,
		MessageMethod: req.MessageMethod,
		Content:       nil,
	}

	log.Println("Formed failure response: ", response.String())

	return &response
}

func (c *StunClient) Discover() (*pb.DiscoverResponse, error) {
	c.connectionMux.Lock()
	defer c.connectionMux.Unlock()

	// form and send request
	discoverRequest := pb.DiscoverRequest{
		Name: c.Name,
		Key:  c.publicKey,
	}

	content, err := proto.Marshal(&discoverRequest)
	if err != nil {
		return nil, err
	}

	message := pb.StunMessage{
		MessageClass:  pb.MsgClass_REQUEST,
		MessageMethod: pb.MsgMethod_DISCOVER,
		Content:       content,
	}

	if err = udp.WriteProtobuf(c.connection, &message, c.Server); err != nil {
		return nil, err
	}

	// read and process response
	responseMsg := pb.StunMessage{}

	if _, err = udp.ReadProtobuf(c.connection, &responseMsg); err != nil {
		return nil, err
	}

	if responseMsg.MessageClass != pb.MsgClass_SUCCESS_RESPONSE {
		return nil, errors.New("Recieved FAILURE response")
	}

	response := pb.DiscoverResponse{}
	if err = proto.Unmarshal(responseMsg.Content, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *StunClient) Bind(peerName string) (*pb.BindingResponse, error) {
	c.connectionMux.Lock()
	defer c.connectionMux.Unlock()

	// form and send request
	bindRequest := pb.BindingRequest{
		PeerName: peerName,
	}

	content, err := proto.Marshal(&bindRequest)
	if err != nil {
		return nil, err
	}

	message := pb.StunMessage{
		MessageClass:  pb.MsgClass_REQUEST,
		MessageMethod: pb.MsgMethod_BINDING,
		Content:       content,
	}

	if err = udp.WriteProtobuf(c.connection, &message, c.Server); err != nil {
		return nil, err
	}

	// read and process response
	responseMsg := pb.StunMessage{}

	if _, err = udp.ReadProtobuf(c.connection, &responseMsg); err != nil {
		return nil, err
	}

	if responseMsg.MessageClass != pb.MsgClass_SUCCESS_RESPONSE {
		return nil, errors.New("Recieved FAILURE response")
	}

	response := pb.BindingResponse{}
	if err = proto.Unmarshal(responseMsg.Content, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *StunClient) SendEcho(to net.Addr) error {
	c.connectionMux.Lock()
	defer c.connectionMux.Unlock()

	// form and send request
	discoverRequest := pb.EchoRequest{
		Message: "Echo",
	}

	content, err := proto.Marshal(&discoverRequest)
	if err != nil {
		return err
	}

	message := pb.StunMessage{
		MessageClass:  pb.MsgClass_REQUEST,
		MessageMethod: pb.MsgMethod_ECHO,
		Content:       content,
	}

	if err = udp.WriteProtobuf(c.connection, &message, to); err != nil {
		return err
	}

	return nil
}

func (c *StunClient) Dispose() {
	c.connectionMux.Lock()
	defer c.connectionMux.Unlock()

	c.connection.Close()
}
