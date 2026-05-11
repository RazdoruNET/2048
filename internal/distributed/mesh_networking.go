package distributed

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"sync"
	"time"
)

// MeshNetwork manages a mesh network topology
type MeshNetwork struct {
	nodeID       string
	peers        map[string]*Peer
	routingTable *RoutingTable
	discovery    *PeerDiscovery
	encryption   *MeshEncryption
	config       *MeshConfig
	mu           sync.RWMutex
	running      bool
	stopCh       chan struct{}
}

// Peer represents a mesh network peer
type Peer struct {
	ID         string
	Address    string
	Port       int
	PublicKey  []byte
	LastSeen   time.Time
	Status     PeerStatus
	Connection net.Conn
	Latency    time.Duration
	Bandwidth  int64
	mu         sync.RWMutex
}

// PeerStatus represents the status of a peer
type PeerStatus int

const (
	PeerStatusUnknown PeerStatus = iota
	PeerStatusConnected
	PeerStatusDisconnected
	PeerStatusConnecting
	PeerStatusError
)

// RoutingTable manages mesh routing
type RoutingTable struct {
	entries  map[string]*RouteEntry
	nextHops map[string]string
	mu       sync.RWMutex
}

// RouteEntry represents a routing entry
type RouteEntry struct {
	Destination string
	NextHop     string
	Metric      int
	LastUpdated time.Time
	Path        []string
}

// PeerDiscovery handles peer discovery
type PeerDiscovery struct {
	broadcastPort int
	discoveryPort int
	peers         map[string]*Peer
	mu            sync.RWMutex
}

// MeshEncryption handles mesh network encryption
type MeshEncryption struct {
	privateKey []byte
	publicKey  []byte
	sharedKeys map[string][]byte
	mu         sync.RWMutex
}

// MeshConfig defines mesh network configuration
type MeshConfig struct {
	NodeID            string        `yaml:"node_id"`
	ListenPort        int           `yaml:"listen_port"`
	BroadcastPort     int           `yaml:"broadcast_port"`
	DiscoveryPort     int           `yaml:"discovery_port"`
	MaxPeers          int           `yaml:"max_peers"`
	HeartbeatInterval time.Duration `yaml:"heartbeat_interval"`
	EnableEncryption  bool          `yaml:"enable_encryption"`
	EncryptionType    string        `yaml:"encryption_type"`
	AutoConnect       bool          `yaml:"auto_connect"`
	RouteTimeout      time.Duration `yaml:"route_timeout"`
}

// Message represents a mesh network message
type Message struct {
	Type        MessageType `json:"type"`
	Source      string      `json:"source"`
	Destination string      `json:"destination"`
	Payload     []byte      `json:"payload"`
	Timestamp   time.Time   `json:"timestamp"`
	TTL         int         `json:"ttl"`
	Path        []string    `json:"path"`
}

// MessageType represents the type of mesh message
type MessageType int

const (
	MessageTypeHeartbeat MessageType = iota
	MessageTypeDiscovery
	MessageTypeRouteUpdate
	MessageTypeData
	MessageTypeEncryption
	MessageTypePeerJoin
	MessageTypePeerLeave
)

// NewMeshNetwork creates a new mesh network
func NewMeshNetwork(config *MeshConfig) *MeshNetwork {
	if config == nil {
		config = &MeshConfig{
			NodeID:            generateNodeID(),
			ListenPort:        8082,
			BroadcastPort:     8083,
			DiscoveryPort:     8084,
			MaxPeers:          50,
			HeartbeatInterval: 30 * time.Second,
			EnableEncryption:  true,
			EncryptionType:    "aes256",
			AutoConnect:       true,
			RouteTimeout:      60 * time.Second,
		}
	}

	mn := &MeshNetwork{
		nodeID: config.NodeID,
		peers:  make(map[string]*Peer),
		routingTable: &RoutingTable{
			entries:  make(map[string]*RouteEntry),
			nextHops: make(map[string]string),
		},
		discovery: &PeerDiscovery{
			broadcastPort: config.BroadcastPort,
			discoveryPort: config.DiscoveryPort,
			peers:         make(map[string]*Peer),
		},
		encryption: NewMeshEncryption(config.EnableEncryption),
		config:     config,
		stopCh:     make(chan struct{}),
	}

	return mn
}

// Start starts the mesh network
func (mn *MeshNetwork) Start(ctx context.Context) error {
	mn.mu.Lock()
	defer mn.mu.Unlock()

	if mn.running {
		return ErrMeshNetworkAlreadyRunning
	}

	mn.running = true

	// Start discovery service
	go mn.runDiscovery(ctx)

	// Start heartbeat service
	go mn.runHeartbeat(ctx)

	// Start routing service
	go mn.runRouting(ctx)

	// Start peer management
	go mn.runPeerManagement(ctx)

	return nil
}

// Stop stops the mesh network
func (mn *MeshNetwork) Stop() error {
	mn.mu.Lock()
	defer mn.mu.Unlock()

	if !mn.running {
		return ErrMeshNetworkNotRunning
	}

	mn.running = false
	close(mn.stopCh)

	// Close all peer connections
	for _, peer := range mn.peers {
		if peer.Connection != nil {
			peer.Connection.Close()
		}
	}

	return nil
}

// AddPeer adds a new peer to the mesh
func (mn *MeshNetwork) AddPeer(peer *Peer) error {
	mn.mu.Lock()
	defer mn.mu.Unlock()

	if len(mn.peers) >= mn.config.MaxPeers {
		return ErrMaxPeersReached
	}

	if _, exists := mn.peers[peer.ID]; exists {
		return ErrPeerAlreadyExists
	}

	peer.Status = PeerStatusConnecting
	peer.LastSeen = time.Now()
	mn.peers[peer.ID] = peer

	// Attempt to connect to peer
	if mn.config.AutoConnect {
		go mn.connectToPeer(peer)
	}

	return nil
}

// RemovePeer removes a peer from the mesh
func (mn *MeshNetwork) RemovePeer(peerID string) error {
	mn.mu.Lock()
	defer mn.mu.Unlock()

	peer, exists := mn.peers[peerID]
	if !exists {
		return ErrPeerNotFound
	}

	if peer.Connection != nil {
		peer.Connection.Close()
	}

	delete(mn.peers, peerID)
	mn.routingTable.RemoveRoute(peerID)

	return nil
}

// GetPeer returns a peer by ID
func (mn *MeshNetwork) GetPeer(peerID string) (*Peer, error) {
	mn.mu.RLock()
	defer mn.mu.RUnlock()

	peer, exists := mn.peers[peerID]
	if !exists {
		return nil, ErrPeerNotFound
	}

	return peer, nil
}

// GetAllPeers returns all peers in the mesh
func (mn *MeshNetwork) GetAllPeers() []*Peer {
	mn.mu.RLock()
	defer mn.mu.RUnlock()

	peers := make([]*Peer, 0, len(mn.peers))
	for _, peer := range mn.peers {
		peers = append(peers, peer)
	}

	return peers
}

// GetConnectedPeers returns all connected peers
func (mn *MeshNetwork) GetConnectedPeers() []*Peer {
	mn.mu.RLock()
	defer mn.mu.RUnlock()

	peers := make([]*Peer, 0)
	for _, peer := range mn.peers {
		if peer.Status == PeerStatusConnected {
			peers = append(peers, peer)
		}
	}

	return peers
}

// SendMessage sends a message through the mesh
func (mn *MeshNetwork) SendMessage(msg *Message) error {
	if msg.Destination == mn.nodeID {
		// Message is for this node
		return mn.handleMessage(msg)
	}

	// Find route to destination
	nextHop, err := mn.routingTable.GetNextHop(msg.Destination)
	if err != nil {
		return err
	}

	// Send message to next hop
	peer, exists := mn.peers[nextHop]
	if !exists {
		return ErrPeerNotFound
	}

	return mn.sendToPeer(peer, msg)
}

// BroadcastMessage broadcasts a message to all peers
func (mn *MeshNetwork) BroadcastMessage(msg *Message) error {
	for _, peer := range mn.GetConnectedPeers() {
		msgCopy := *msg
		msgCopy.Destination = peer.ID
		if err := mn.sendToPeer(peer, &msgCopy); err != nil {
			// Log error but continue broadcasting
			continue
		}
	}

	return nil
}

// sendToPeer sends a message to a specific peer
func (mn *MeshNetwork) sendToPeer(peer *Peer, msg *Message) error {
	if peer.Connection == nil {
		return ErrPeerNotConnected
	}

	// Encrypt message if encryption is enabled
	if mn.config.EnableEncryption {
		encryptedPayload, err := mn.encryption.Encrypt(msg.Payload, peer.ID)
		if err != nil {
			return err
		}
		msg.Payload = encryptedPayload
	}

	// Serialize and send message
	data, err := serializeMessage(msg)
	if err != nil {
		return err
	}

	_, err = peer.Connection.Write(data)
	return err
}

// handleMessage handles an incoming message
func (mn *MeshNetwork) handleMessage(msg *Message) error {
	switch msg.Type {
	case MessageTypeHeartbeat:
		return mn.handleHeartbeat(msg)
	case MessageTypeDiscovery:
		return mn.handleDiscovery(msg)
	case MessageTypeRouteUpdate:
		return mn.handleRouteUpdate(msg)
	case MessageTypeData:
		return mn.handleData(msg)
	case MessageTypePeerJoin:
		return mn.handlePeerJoin(msg)
	case MessageTypePeerLeave:
		return mn.handlePeerLeave(msg)
	default:
		return ErrUnknownMessageType
	}
}

// handleHeartbeat handles heartbeat messages
func (mn *MeshNetwork) handleHeartbeat(msg *Message) error {
	// Update peer last seen time
	if peer, exists := mn.peers[msg.Source]; exists {
		peer.mu.Lock()
		peer.LastSeen = time.Now()
		peer.Status = PeerStatusConnected
		peer.mu.Unlock()
	}

	return nil
}

// handleDiscovery handles discovery messages
func (mn *MeshNetwork) handleDiscovery(msg *Message) error {
	// Respond with peer information
	response := &Message{
		Type:        MessageTypeDiscovery,
		Source:      mn.nodeID,
		Destination: msg.Source,
		Payload:     mn.getPeerInfo(),
		Timestamp:   time.Now(),
	}

	return mn.SendMessage(response)
}

// handleRouteUpdate handles routing updates
func (mn *MeshNetwork) handleRouteUpdate(msg *Message) error {
	// Update routing table
	return mn.routingTable.UpdateRoute(msg.Destination, msg.Source, 1)
}

// handleData handles data messages
func (mn *MeshNetwork) handleData(msg *Message) error {
	// Decrypt message if needed
	if mn.config.EnableEncryption {
		decryptedPayload, err := mn.encryption.Decrypt(msg.Payload, msg.Source)
		if err != nil {
			return err
		}
		msg.Payload = decryptedPayload
	}

	// Process data based on application logic
	return mn.processData(msg)
}

// handlePeerJoin handles peer join messages
func (mn *MeshNetwork) handlePeerJoin(msg *Message) error {
	// Add new peer to mesh
	peerInfo := parsePeerInfo(msg.Payload)
	if peerInfo != nil {
		return mn.AddPeer(peerInfo)
	}

	return nil
}

// handlePeerLeave handles peer leave messages
func (mn *MeshNetwork) handlePeerLeave(msg *Message) error {
	// Remove peer from mesh
	return mn.RemovePeer(msg.Source)
}

// processData processes data messages
func (mn *MeshNetwork) processData(msg *Message) error {
	// Application-specific data processing
	// This would be implemented based on the specific use case
	return nil
}

// runDiscovery runs the peer discovery service
func (mn *MeshNetwork) runDiscovery(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mn.discoverPeers()
		case <-mn.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// discoverPeers discovers new peers
func (mn *MeshNetwork) discoverPeers() {
	// Broadcast discovery message
	msg := &Message{
		Type:        MessageTypeDiscovery,
		Source:      mn.nodeID,
		Destination: "broadcast",
		Payload:     mn.getPeerInfo(),
		Timestamp:   time.Now(),
	}

	mn.BroadcastMessage(msg)
}

// runHeartbeat runs the heartbeat service
func (mn *MeshNetwork) runHeartbeat(ctx context.Context) {
	ticker := time.NewTicker(mn.config.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mn.sendHeartbeat()
		case <-mn.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// sendHeartbeat sends heartbeat to all peers
func (mn *MeshNetwork) sendHeartbeat() {
	msg := &Message{
		Type:        MessageTypeHeartbeat,
		Source:      mn.nodeID,
		Destination: "broadcast",
		Payload:     []byte("heartbeat"),
		Timestamp:   time.Now(),
	}

	mn.BroadcastMessage(msg)
}

// runRouting runs the routing service
func (mn *MeshNetwork) runRouting(ctx context.Context) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mn.updateRouting()
		case <-mn.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// updateRouting updates the routing table
func (mn *MeshNetwork) updateRouting() {
	// Send route update messages
	for _, peer := range mn.GetConnectedPeers() {
		msg := &Message{
			Type:        MessageTypeRouteUpdate,
			Source:      mn.nodeID,
			Destination: peer.ID,
			Payload:     mn.getRoutingInfo(),
			Timestamp:   time.Now(),
		}

		mn.sendToPeer(peer, msg)
	}
}

// runPeerManagement runs the peer management service
func (mn *MeshNetwork) runPeerManagement(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mn.managePeers()
		case <-mn.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// managePeers manages peer connections and health
func (mn *MeshNetwork) managePeers() {
	now := time.Now()

	for _, peer := range mn.peers {
		peer.mu.RLock()
		lastSeen := peer.LastSeen
		status := peer.Status
		peer.mu.RUnlock()

		// Check if peer is stale
		if now.Sub(lastSeen) > mn.config.RouteTimeout {
			if status == PeerStatusConnected {
				mn.UpdatePeerStatus(peer.ID, PeerStatusDisconnected)
			}
		}

		// Attempt to reconnect to disconnected peers
		if status == PeerStatusDisconnected && mn.config.AutoConnect {
			go mn.connectToPeer(peer)
		}
	}
}

// connectToPeer attempts to connect to a peer
func (mn *MeshNetwork) connectToPeer(peer *Peer) {
	peer.mu.Lock()
	defer peer.mu.Unlock()

	if peer.Status == PeerStatusConnected {
		return
	}

	peer.Status = PeerStatusConnecting

	// Attempt connection
	address := fmt.Sprintf("%s:%d", peer.Address, peer.Port)
	conn, err := net.DialTimeout("tcp", address, 10*time.Second)
	if err != nil {
		peer.Status = PeerStatusError
		return
	}

	peer.Connection = conn
	peer.Status = PeerStatusConnected
	peer.LastSeen = time.Now()

	// Send peer join message
	msg := &Message{
		Type:        MessageTypePeerJoin,
		Source:      mn.nodeID,
		Destination: peer.ID,
		Payload:     mn.getPeerInfo(),
		Timestamp:   time.Now(),
	}

	mn.sendToPeer(peer, msg)
}

// UpdatePeerStatus updates peer status
func (mn *MeshNetwork) UpdatePeerStatus(peerID string, status PeerStatus) error {
	mn.mu.Lock()
	defer mn.mu.Unlock()

	peer, exists := mn.peers[peerID]
	if !exists {
		return ErrPeerNotFound
	}

	peer.mu.Lock()
	peer.Status = status
	peer.mu.Unlock()

	return nil
}

// getPeerInfo returns peer information
func (mn *MeshNetwork) getPeerInfo() []byte {
	// Serialize peer information
	info := map[string]interface{}{
		"id":      mn.nodeID,
		"address": "localhost", // Would use actual address
		"port":    mn.config.ListenPort,
		"status":  "connected",
	}

	// Simplified serialization
	return []byte(fmt.Sprintf("%v", info))
}

// getRoutingInfo returns routing information
func (mn *MeshNetwork) getRoutingInfo() []byte {
	// Serialize routing table
	return []byte("routing_info")
}

// parsePeerInfo parses peer information from payload
func parsePeerInfo(payload []byte) *Peer {
	// Simplified parsing
	return &Peer{
		ID:      "parsed_peer",
		Address: "localhost",
		Port:    8082,
		Status:  PeerStatusUnknown,
	}
}

// serializeMessage serializes a message
func serializeMessage(msg *Message) ([]byte, error) {
	// Simplified serialization
	return []byte(fmt.Sprintf("%v", msg)), nil
}

// generateNodeID generates a unique node ID
func generateNodeID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// NewMeshEncryption creates new mesh encryption
func NewMeshEncryption(enabled bool) *MeshEncryption {
	return &MeshEncryption{
		sharedKeys: make(map[string][]byte),
	}
}

// Encrypt encrypts data for a peer
func (me *MeshEncryption) Encrypt(data []byte, peerID string) ([]byte, error) {
	// Simplified encryption
	return data, nil
}

// Decrypt decrypts data from a peer
func (me *MeshEncryption) Decrypt(data []byte, peerID string) ([]byte, error) {
	// Simplified decryption
	return data, nil
}

// RoutingTable methods
func (rt *RoutingTable) UpdateRoute(destination, nextHop string, metric int) error {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	rt.entries[destination] = &RouteEntry{
		Destination: destination,
		NextHop:     nextHop,
		Metric:      metric,
		LastUpdated: time.Now(),
		Path:        []string{nextHop},
	}

	rt.nextHops[destination] = nextHop
	return nil
}

func (rt *RoutingTable) GetNextHop(destination string) (string, error) {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	nextHop, exists := rt.nextHops[destination]
	if !exists {
		return "", ErrRouteNotFound
	}

	return nextHop, nil
}

func (rt *RoutingTable) RemoveRoute(destination string) {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	delete(rt.entries, destination)
	delete(rt.nextHops, destination)
}

// Errors
var (
	ErrMeshNetworkAlreadyRunning = &MeshNetworkError{Code: "ALREADY_RUNNING", Message: "mesh network is already running"}
	ErrMeshNetworkNotRunning     = &MeshNetworkError{Code: "NOT_RUNNING", Message: "mesh network is not running"}
	ErrMaxPeersReached           = &MeshNetworkError{Code: "MAX_PEERS", Message: "maximum number of peers reached"}
	ErrPeerAlreadyExists         = &MeshNetworkError{Code: "PEER_EXISTS", Message: "peer already exists"}
	ErrPeerNotFound              = &MeshNetworkError{Code: "PEER_NOT_FOUND", Message: "peer not found"}
	ErrPeerNotConnected          = &MeshNetworkError{Code: "PEER_NOT_CONNECTED", Message: "peer not connected"}
	ErrUnknownMessageType        = &MeshNetworkError{Code: "UNKNOWN_MESSAGE_TYPE", Message: "unknown message type"}
	ErrRouteNotFound             = &MeshNetworkError{Code: "ROUTE_NOT_FOUND", Message: "route not found"}
)

// MeshNetworkError represents a mesh network error
type MeshNetworkError struct {
	Code    string
	Message string
}

func (e *MeshNetworkError) Error() string {
	return e.Message + " (" + e.Code + ")"
}
