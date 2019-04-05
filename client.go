package telecom

import (
	"encoding/binary"
	"encoding/json"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/crypto/nacl/secretbox"
)

type GatewayMessage struct {
	Op   int             `json:"op"`
	Seq  int64           `json:"s"`
	Type string          `json:"t"`
	Data json.RawMessage `json:"d"`
}

type GatewayMessageReady struct {
	SSRC              uint32        `json:"ssrc"`
	Port              int           `json:"port"`
	Modes             []string      `json:"modes"`
	HeartbeatInterval time.Duration `json:"heartbeat_interval"`
}

type GatewayMessageSpeaking struct {
	Speaking bool `json:"speaking"`
	Delay    int  `json:"delay"`
}

type GatewayMessageMode struct {
	SecretKey [32]byte `json:"secret_key"`
	Mode      string   `json:"mode"`
}

type VoiceHandshakeData struct {
	ServerID  string `json:"server_id"`
	UserID    string `json:"user_id"`
	SessionID string `json:"session_id"`
	Token     string `json:"token"`
}

type SelectProtocolData struct {
	Address string `json:"address"` // Public IP of machine running this code
	Port    uint16 `json:"port"`    // UDP Port of machine running this code
	Mode    string `json:"mode"`    // always "xsalsa20_poly1305"
}

type GatewayMessageSelectProtocol struct {
	Protocol string             `json:"protocol"` // Always "udp" ?
	Data     SelectProtocolData `json:"data"`
}

type VoiceOp struct {
	Op   int         `json:"op"`
	Data interface{} `json:"d"`
}

type ServerInfo struct {
	Endpoint string
	Token    string
}

type Client struct {
	UserId        string
	GuildId       string
	AudioSendChan chan []byte
	Ready         chan bool

	endpoint  string
	sessionId string
	secretKey [32]byte

	serverInfoChan chan ServerInfo

	wsLock sync.Mutex
	ws     *websocket.Conn
	udp    *net.UDPConn
}

func NewClient(userId, guildId, sessionId string) *Client {
	return &Client{
		UserId:  userId,
		GuildId: guildId,
		Ready:   make(chan bool, 0),
		// Jitter buffer
		AudioSendChan:  make(chan []byte, 128),
		sessionId:      sessionId,
		serverInfoChan: make(chan ServerInfo, 0),
	}
}

// Starts the clients internal goroutines
func (c *Client) Run() {
	go c.runForever()
}

// Updates the clients server information
func (c *Client) SetServerInfo(endpoint, token string) {
	c.serverInfoChan <- ServerInfo{endpoint, token}
}

// Sets whether the client is speaking or not
func (c *Client) SetSpeaking(speaking bool) {
	data := VoiceOp{5, GatewayMessageSpeaking{speaking, 0}}
	c.wsLock.Lock()
	err := c.ws.WriteJSON(data)
	c.wsLock.Unlock()
	if err != nil {
		log.Printf("Error setting speaking: %v")
	}
}

// Waits until the client is ready to send voice data
func (c *Client) WaitReady() {
	if c.Ready == nil {
		return
	}
	<-c.Ready
}

func (c *Client) runForever() {
	for {
		info := <-c.serverInfoChan
		c.endpoint = info.Endpoint

		if c.ws != nil {
			panic("Would have had to close the existing connection TODO")
		}

		var err error

		c.ws, _, err = websocket.DefaultDialer.Dial("wss://"+strings.TrimSuffix(info.Endpoint, ":80"), nil)
		if err != nil {
			panic(err)
		}

		data := VoiceOp{0, VoiceHandshakeData{c.GuildId, c.UserId, c.sessionId, info.Token}}
		err = c.ws.WriteJSON(data)
		if err != nil {
			panic(err)
		}

		go c.runWebsocket()
	}
}

func (c *Client) runWebsocket() {
	for {
		_, message, err := c.ws.ReadMessage()
		if err != nil {
			log.Printf("runWebsocket terminating due to error on read: %v", err)
			return
		}

		err = c.handleWebsocketMessage(message)
		if err != nil {
			log.Printf("Error processing websocket message: %v (%v)", err, string(message))
		}
	}
}

func (c *Client) runHeartbeater(interval time.Duration) {
	var err error

	ticker := time.NewTicker(time.Duration(interval) * time.Millisecond)
	defer ticker.Stop()

	for {
		c.wsLock.Lock()
		err = c.ws.WriteJSON(VoiceOp{3, int(time.Now().Unix())})
		c.wsLock.Unlock()
		if err != nil {
			log.Printf("Error in heartbeater: %v", err)
			return
		}

		select {
		case <-ticker.C:

		}
	}
}

func (c *Client) runUDP(port int, ssrc uint32) {
	if c.udp != nil {
		log.Printf("Error: UDP connection already open?")
		return
	}

	host := strings.TrimSuffix(c.endpoint, ":80") + ":" + strconv.Itoa(port)
	addr, err := net.ResolveUDPAddr("udp", host)
	if err != nil {
		log.Printf("Error: failed to resolve UDP addr %v: %v", host, err)
		return
	}

	c.udp, err = net.DialUDP("udp", nil, addr)
	if err != nil {
		log.Printf("Error: failed to dial udp: %v", err)
		return
	}

	// Create a 70 byte array and put the SSRC code from the Op 2 VoiceConnection event
	// into it.  Then send that over the UDP connection to Discord
	sb := make([]byte, 70)
	binary.BigEndian.PutUint32(sb, ssrc)
	_, err = c.udp.Write(sb)
	if err != nil {
		log.Printf("Error: failed to write udp: %v", err)
		return
	}

	// Create a 70 byte array and listen for the initial handshake response
	// from Discord.  Once we get it parse the IP and PORT information out
	// of the response.  This should be our public IP and PORT as Discord
	// saw us.
	rb := make([]byte, 70)
	rlen, _, err := c.udp.ReadFromUDP(rb)
	if err != nil {
		log.Printf("Error: failed to read from udp: %v", err)
		return
	}

	if rlen < 70 {
		log.Printf("Error: invalid udp packet size %v", rlen)
		return
	}

	// Loop over position 4 through 20 to grab the IP address
	// Should never be beyond position 20.
	var ip string
	for i := 4; i < 20; i++ {
		if rb[i] == 0 {
			break
		}
		ip += string(rb[i])
	}

	// Grab port from position 68 and 69
	ourPort := binary.LittleEndian.Uint16(rb[68:70])

	// Take the data from above and send it back to Discord to finalize
	// the UDP connection handshake.
	data := VoiceOp{1, GatewayMessageSelectProtocol{
		"udp",
		SelectProtocolData{
			ip, ourPort, "xsalsa20_poly1305",
		},
	}}

	c.wsLock.Lock()
	err = c.ws.WriteJSON(data)
	c.wsLock.Unlock()
	if err != nil {
		log.Printf("Failed to send SELECT PROTOCOL: %v", err)
		return
	}

	go c.runUDPReader()
	go c.runUDPWriter(ssrc, 48000, 960)
	// start udpKeepAlive
	// go v.udpKeepAlive(v.udpConn, v.close, 5*time.Second)

	return
}

func (c *Client) runUDPReader() {
	buffer := make([]byte, 1024)
	for {
		_, err := c.udp.Read(buffer)
		if err != nil {
			log.Printf("UDP READ ERROR: %v", err)
			continue
		}

		// TODO: voice decrypting and processing
	}
}

func (c *Client) runUDPWriter(ssrc uint32, rate, size int) {
	var sequence uint16
	var timestamp uint32
	var nonce [24]byte
	udpHeader := make([]byte, 12)

	// build the parts that don't change in the udpHeader
	udpHeader[0] = 0x80
	udpHeader[1] = 0x78
	binary.BigEndian.PutUint32(udpHeader[8:], ssrc)

	c.SetSpeaking(true)
	time.Sleep(time.Second * 1)
	c.SetSpeaking(true)

	close(c.Ready)
	c.Ready = nil

	// start a send loop that loops until buf chan is closed
	ticker := time.NewTicker(time.Millisecond * time.Duration(size/(rate/1000)))
	defer ticker.Stop()
	for {
		recvbuf := <-c.AudioSendChan

		// Add sequence and timestamp to udpPacket
		binary.BigEndian.PutUint16(udpHeader[2:], sequence)
		binary.BigEndian.PutUint32(udpHeader[4:], timestamp)

		// encrypt the opus data
		copy(nonce[:], udpHeader)
		sendbuf := secretbox.Seal(udpHeader, recvbuf, &nonce, &c.secretKey)

		// block here until we're exactly at the right time :)
		// Then send rtp audio packet to Discord over UDP
		<-ticker.C
		_, err := c.udp.Write(sendbuf)
		if err != nil {
			log.Printf("error writing opus to udp: %v", err)
			return
		}

		if (sequence) == 0xFFFF {
			sequence = 0
		} else {
			sequence++
		}

		if (timestamp + uint32(size)) >= 0xFFFFFFFF {
			timestamp = 0
		} else {
			timestamp += uint32(size)
		}
	}
}

func (c *Client) handleWebsocketMessage(data []byte) error {
	var message GatewayMessage

	err := json.Unmarshal(data, &message)
	if err != nil {
		return err
	}

	switch message.Op {
	case 2:
		var ready GatewayMessageReady
		err = json.Unmarshal(message.Data, &ready)
		if err != nil {
			return err
		}

		go c.runHeartbeater(ready.HeartbeatInterval)

		go c.runUDP(ready.Port, ready.SSRC)
	case 3:
	case 4:
		var mode GatewayMessageMode
		err = json.Unmarshal(message.Data, &mode)
		if err != nil {
			return err
		}

		c.secretKey = mode.SecretKey
	default:
		log.Printf("Unhandled gateway op %v: %v", message.Op, string(message.Data))
	}

	return nil
}
