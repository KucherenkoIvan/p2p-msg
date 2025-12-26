package signaling

import "errors"

// related to http so should probably live in http-specific context
type ClientResolveRequest struct {
	PeerName string `json:"peerName"`
}

type ClientResolveResponse struct {
	DisplayName string `json:"displayName"`
	Address     string `json:"address"`
}

type ClientReadyRequest struct {
	DisplayName string `json:"displayName"`
	// TODO: add keys
}

type ClientReadyResponse struct {
	Result string `json:"result"`
}

var activeClients map[string]string = make(map[string]string)

func AddToActiveClientsList(name string, addr string) {
	activeClients[name] = addr
}

func RemoveFromActiveClientsList(name string) {
	delete(activeClients, name)
}

func GetAddressByPeerName(name string) (string, error) {
	addr, exists := activeClients[name]
	if exists {
		return addr, nil
	}
	return "", errors.New("Name was not found in the list of active clients")
}
func DEBUG_ResetClients() {

	activeClients = make(map[string]string)
}
