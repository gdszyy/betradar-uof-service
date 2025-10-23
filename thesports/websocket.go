package thesports

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// DefaultWSURL is the default WebSocket URL
	DefaultWSURL = "wss://ws.thesports.com/football"
	
	// PingInterval is the interval for sending ping messages
	PingInterval = 30 * time.Second
	
	// ReconnectDelay is the delay before reconnecting
	ReconnectDelay = 5 * time.Second
)

// WSClient represents a WebSocket client
type WSClient struct {
	url          string
	apiToken     string
	conn         *websocket.Conn
	mu           sync.RWMutex
	handlers     map[string][]MessageHandler
	isConnected  bool
	autoReconnect bool
	stopChan     chan struct{}
	doneChan     chan struct{}
}

// MessageHandler is a function that handles WebSocket messages
type MessageHandler func(message *WSMessage)

// WSMessage represents a WebSocket message
type WSMessage struct {
	Type      string          `json:"type"`       // match_update, incident, statistics, etc.
	MatchID   int             `json:"match_id"`
	Timestamp time.Time       `json:"timestamp"`
	Data      json.RawMessage `json:"data"`
}

// WSMatchUpdate represents a match update message
type WSMatchUpdate struct {
	Match      *Match      `json:"match"`
	Statistics *MatchStats `json:"statistics,omitempty"`
}

// WSIncident represents an incident message
type WSIncident struct {
	Incident *Incident `json:"incident"`
}

// WSSubscribeRequest represents a subscription request
type WSSubscribeRequest struct {
	Action   string `json:"action"`   // subscribe, unsubscribe
	Type     string `json:"type"`     // match, competition, team
	ID       int    `json:"id"`
	Token    string `json:"token"`
}

// WSSubscribeResponse represents a subscription response
type WSSubscribeResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Type    string `json:"type"`
	ID      int    `json:"id"`
}

// NewWSClient creates a new WebSocket client
func NewWSClient(apiToken string) *WSClient {
	return NewWSClientWithURL(DefaultWSURL, apiToken)
}

// NewWSClientWithURL creates a new WebSocket client with custom URL
func NewWSClientWithURL(url, apiToken string) *WSClient {
	return &WSClient{
		url:           url,
		apiToken:      apiToken,
		handlers:      make(map[string][]MessageHandler),
		autoReconnect: true,
		stopChan:      make(chan struct{}),
		doneChan:      make(chan struct{}),
	}
}

// Connect establishes a WebSocket connection
func (c *WSClient) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isConnected {
		return fmt.Errorf("already connected")
	}

	// Add authentication header
	header := make(map[string][]string)
	header["Authorization"] = []string{"Bearer " + c.apiToken}

	// Connect to WebSocket
	conn, _, err := websocket.DefaultDialer.Dial(c.url, header)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	c.conn = conn
	c.isConnected = true

	// Start message handler
	go c.readMessages()
	
	// Start ping handler
	go c.pingHandler()

	return nil
}

// Disconnect closes the WebSocket connection
func (c *WSClient) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.isConnected {
		return fmt.Errorf("not connected")
	}

	// Stop auto-reconnect
	c.autoReconnect = false
	
	// Signal stop
	close(c.stopChan)

	// Close connection
	err := c.conn.WriteMessage(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		log.Printf("Error sending close message: %v", err)
	}

	err = c.conn.Close()
	c.isConnected = false

	// Wait for handlers to finish
	<-c.doneChan

	return err
}

// IsConnected returns whether the client is connected
func (c *WSClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isConnected
}

// SubscribeMatch subscribes to a specific match
func (c *WSClient) SubscribeMatch(matchID int) error {
	return c.subscribe("match", matchID)
}

// UnsubscribeMatch unsubscribes from a specific match
func (c *WSClient) UnsubscribeMatch(matchID int) error {
	return c.unsubscribe("match", matchID)
}

// SubscribeCompetition subscribes to all matches in a competition
func (c *WSClient) SubscribeCompetition(competitionID int) error {
	return c.subscribe("competition", competitionID)
}

// UnsubscribeCompetition unsubscribes from a competition
func (c *WSClient) UnsubscribeCompetition(competitionID int) error {
	return c.unsubscribe("competition", competitionID)
}

// SubscribeTeam subscribes to all matches of a team
func (c *WSClient) SubscribeTeam(teamID int) error {
	return c.subscribe("team", teamID)
}

// UnsubscribeTeam unsubscribes from a team
func (c *WSClient) UnsubscribeTeam(teamID int) error {
	return c.unsubscribe("team", teamID)
}

// subscribe sends a subscription request
func (c *WSClient) subscribe(typ string, id int) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.isConnected {
		return fmt.Errorf("not connected")
	}

	req := WSSubscribeRequest{
		Action: "subscribe",
		Type:   typ,
		ID:     id,
		Token:  c.apiToken,
	}

	return c.conn.WriteJSON(req)
}

// unsubscribe sends an unsubscription request
func (c *WSClient) unsubscribe(typ string, id int) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.isConnected {
		return fmt.Errorf("not connected")
	}

	req := WSSubscribeRequest{
		Action: "unsubscribe",
		Type:   typ,
		ID:     id,
		Token:  c.apiToken,
	}

	return c.conn.WriteJSON(req)
}

// OnMessage registers a handler for a specific message type
func (c *WSClient) OnMessage(messageType string, handler MessageHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.handlers[messageType] = append(c.handlers[messageType], handler)
}

// OnMatchUpdate registers a handler for match updates
func (c *WSClient) OnMatchUpdate(handler func(*WSMatchUpdate)) {
	c.OnMessage("match_update", func(msg *WSMessage) {
		var update WSMatchUpdate
		if err := json.Unmarshal(msg.Data, &update); err != nil {
			log.Printf("Error unmarshaling match update: %v", err)
			return
		}
		handler(&update)
	})
}

// OnIncident registers a handler for match incidents
func (c *WSClient) OnIncident(handler func(*WSIncident)) {
	c.OnMessage("incident", func(msg *WSMessage) {
		var incident WSIncident
		if err := json.Unmarshal(msg.Data, &incident); err != nil {
			log.Printf("Error unmarshaling incident: %v", err)
			return
		}
		handler(&incident)
	})
}

// OnStatistics registers a handler for statistics updates
func (c *WSClient) OnStatistics(handler func(*MatchStats)) {
	c.OnMessage("statistics", func(msg *WSMessage) {
		var stats MatchStats
		if err := json.Unmarshal(msg.Data, &stats); err != nil {
			log.Printf("Error unmarshaling statistics: %v", err)
			return
		}
		handler(&stats)
	})
}

// readMessages reads messages from the WebSocket connection
func (c *WSClient) readMessages() {
	defer func() {
		c.mu.Lock()
		c.isConnected = false
		c.mu.Unlock()
		
		// Attempt to reconnect if enabled
		if c.autoReconnect {
			go c.reconnect()
		} else {
			close(c.doneChan)
		}
	}()

	for {
		select {
		case <-c.stopChan:
			return
		default:
			var msg WSMessage
			err := c.conn.ReadJSON(&msg)
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket error: %v", err)
				}
				return
			}

			// Dispatch message to handlers
			c.dispatchMessage(&msg)
		}
	}
}

// dispatchMessage dispatches a message to registered handlers
func (c *WSClient) dispatchMessage(msg *WSMessage) {
	c.mu.RLock()
	handlers, exists := c.handlers[msg.Type]
	c.mu.RUnlock()

	if !exists {
		return
	}

	for _, handler := range handlers {
		go handler(msg)
	}
}

// pingHandler sends periodic ping messages
func (c *WSClient) pingHandler() {
	ticker := time.NewTicker(PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopChan:
			return
		case <-ticker.C:
			c.mu.RLock()
			if c.isConnected {
				err := c.conn.WriteMessage(websocket.PingMessage, nil)
				if err != nil {
					log.Printf("Error sending ping: %v", err)
				}
			}
			c.mu.RUnlock()
		}
	}
}

// reconnect attempts to reconnect to the WebSocket server
func (c *WSClient) reconnect() {
	for {
		select {
		case <-c.stopChan:
			return
		case <-time.After(ReconnectDelay):
			log.Printf("Attempting to reconnect...")
			
			err := c.Connect()
			if err != nil {
				log.Printf("Reconnection failed: %v", err)
				continue
			}
			
			log.Printf("Reconnected successfully")
			return
		}
	}
}

// SetAutoReconnect sets whether to automatically reconnect on disconnect
func (c *WSClient) SetAutoReconnect(enabled bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.autoReconnect = enabled
}

