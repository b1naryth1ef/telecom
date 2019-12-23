package telecom

import (
	"encoding/binary"
	"encoding/json"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
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
	IP                string        `json:"ip"`
	Port              int           `json:"port"`
	Modes             []string      `json:"modes"`

	// This heartbeat interval should not be used.
	HeartbeatInterval time.Duration `json:"heartbeat_interval"`
}

type GatewayMessageHello struct {
    HeartbeatInterval float32 `json:"heartbeat_interval"`
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

type GatewayMessageSelectProtocol struct {
	Protocol string                           `json:"protocol"`
	Data     GatewayMessageSelectProtocolData `json:"data"`
}

type GatewayMessageSelectProtocolData struct {
	Address string `json:"address"`
	Port    uint16 `json:"port"`
	Mode    string `json:"mode"`
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
	UserId  string
	GuildId string
	Ready   chan bool

	playableQueue chan Playable
	playable      Playable
	closer        chan struct{}
	endpoint      string
	sessionId     string
	secretKey     [32]byte

	serverInfoChan chan ServerInfo

	wsLock sync.Mutex
	ws     *websocket.Conn
	udp    *net.UDPConn
}

func NewClient(userId, guildId, sessionId string) *Client {
	return &Client{
		UserId:         userId,
		GuildId:        guildId,
		Ready:          make(chan bool, 0),
		playableQueue:  make(chan Playable, 0),
		sessionId:      sessionId,
		serverInfoChan: make(chan ServerInfo, 0),
	}
}

// Starts the clients internal goroutines
func (c *Client) Run() {
	if c.closer != nil {
		return
	}

	go c.runForever()
}

// Updates the clients server information
func (c *Client) UpdateServerInfo(endpoint, token string) {
	c.serverInfoChan <- ServerInfo{endpoint, token}
}

// Sets whether the client is speaking or not
func (c *Client) SetSpeaking(speaking bool) {
	data := VoiceOp{5, GatewayMessageSpeaking{speaking, 0}}
	c.wsLock.Lock()
	err := c.ws.WriteJSON(data)
	c.wsLock.Unlock()
	if err != nil {
		log.WithError(err).Errorf("Failed to set speaking to %v", speaking)
	}
}

// Waits until the client is ready to send voice data
func (c *Client) WaitReady() {
	if c.Ready == nil {
		return
	}
	<-c.Ready
}

func (c *Client) Disconnect() {
	if c.closer != nil {
		close(c.closer)
		c.closer = nil
	}
}

func (c *Client) Play(p Playable) {
	c.playableQueue <- p
}

func (c *Client) runForever() {
	var info ServerInfo

	c.closer = make(chan struct{}, 0)

	for {
		select {
		case info = <-c.serverInfoChan:
			if c.ws != nil {
				log.Debug("Tearing down existing connection, endpoint has moved")
				c.ws.Close()
			}
		case <-c.closer:
			log.Info("Tearing down client due to upstream close request")
			if c.ws != nil {
				c.ws.Close()
			}
			return
		}

		var err error
		host := "wss://" + strings.TrimSuffix(info.Endpoint, ":80") + ":443/?v=4"
		c.ws, _, err = websocket.DefaultDialer.Dial(host, nil)
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
	var message GatewayMessage
	wsCloser := make(chan struct{}, 0)

	log.Debug("Running websocket connection")

	for {
		_, data, err := c.ws.ReadMessage()
		if err != nil {
			log.WithError(err).Error("Failed to ReadMessage from websocket, closing child routines")
			close(wsCloser)
			if c.udp != nil {
				c.udp.Close()
			}
			return
		}

		err = json.Unmarshal(data, &message)
		if err != nil {
			log.WithError(err).Errorf("Failed to process gateway message: %v", string(data))
			continue
		}

		switch message.Op {
		case 2:
			var ready GatewayMessageReady
			err = json.Unmarshal(message.Data, &ready)
			if err != nil {
				log.WithError(err).Errorf("Failed to process Ready message: %v", string(message.Data))
				continue
			}

			go c.runUDP(wsCloser, ready.IP, ready.Port, ready.SSRC)
		case 3:
		case 4:
			var mode GatewayMessageMode
			err = json.Unmarshal(message.Data, &mode)
			if err != nil {
				log.WithError(err).Errorf("Failed to process Mode message: %v", string(message.Data))
				continue
			}

			c.secretKey = mode.SecretKey
		case 6:
			log.Debug("Heartbeat acknowledged")
		case 8:
			var hello GatewayMessageHello
			err = json.Unmarshal(message.Data, &hello)
			if err != nil {
				log.WithError(err).Errorf("Failed to process Hello message: %v", string(message.Data))
				continue
			}

			go c.runHeartbeater(wsCloser, hello.HeartbeatInterval)
		default:
			log.WithFields(log.Fields{
				"op":   message.Op,
				"data": string(message.Data),
			}).Warning("Unhandled gateway message")
		}
	}
}

func (c *Client) runHeartbeater(closer chan struct{}, interval float32) {
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
		case <-closer:
			return
		}
	}
}

func (c *Client) runUDP(closer chan struct{}, connection_ip string, port int, ssrc uint32) {
	if c.udp != nil {
		log.Printf("Error: UDP connection already open?")
		return
	}

	host := connection_ip + ":" + strconv.Itoa(port)
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
		GatewayMessageSelectProtocolData{
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

	// Closes due to udp being closed
	go c.runUDPReader()
	go c.runUDPWriter(closer, ssrc, 48000, 960)

	// TODO start udpKeepAlive
	// go v.udpKeepAlive(v.udpConn, v.close, 5*time.Second)

	return
}

func (c *Client) runUDPReader() {
	buffer := make([]byte, 1024)
	for {
		_, err := c.udp.Read(buffer)
		if err != nil {
			log.WithError(err).Error("Error on UDP read")
			return
		}

		// TODO: voice decrypting and processing
	}
}

func (c *Client) runUDPWriter(closer chan struct{}, ssrc uint32, rate, size int) {
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
	c.SetSpeaking(false)
	time.Sleep(time.Second * 1)
	c.SetSpeaking(true)

	log.Debug("Ready to transmit voice data...")
	close(c.Ready)
	c.Ready = nil

	// start a send loop that loops until buf chan is closed
	ticker := time.NewTicker(time.Millisecond * time.Duration(size/(rate/1000)))
	defer ticker.Stop()

	var err error
	var recv chan []byte
	for {
		if c.playable == nil {
			log.Debug("Waiting for playable")
			c.playable = <-c.playableQueue
			recv, err = c.playable.Output()
			log.Debug("Got new playable")
			if err != nil {
				log.WithError(err).Warning("Error opening playable output")
				continue
			}
		}

		recvbuf, ok := <-recv
		if !ok {
			log.Warning("Error reading from playable")
			c.playable = nil
			continue
		}

		// Add sequence and timestamp to udpPacket
		binary.BigEndian.PutUint16(udpHeader[2:], sequence)
		binary.BigEndian.PutUint32(udpHeader[4:], timestamp)

		// encrypt the opus data
		copy(nonce[:], udpHeader)
		sendbuf := secretbox.Seal(udpHeader, recvbuf, &nonce, &c.secretKey)

		// block here until we're exactly at the right time :)
		// Then send rtp audio packet to Discord over UDP
		select {
		case <-ticker.C:
		case <-closer:
			return
		}

		_, err := c.udp.Write(sendbuf)
		if err != nil {
			log.WithError(err).Error("UDP write error")
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
