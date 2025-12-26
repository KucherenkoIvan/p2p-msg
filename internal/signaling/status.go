package signaling

// TODO: expand
type SignalingServerStatus struct {
	Available bool `json:"available"`
}

func GetCurrentServerStatus() SignalingServerStatus {
	// TODO: respond with actual status
	return SignalingServerStatus{Available: true}
}
