package ws

import (
	"bytes"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// ModeBinary causes Conn.Write to write binary messages.
	ModeBinary = websocket.BinaryMessage

	// ModeText causes Conn.Write to write text messages.
	ModeText = websocket.TextMessage
)

// Conn exposes per-socket connection configs, and implements io.ReadWriteCloser
// TODO - implement net.Conn?
type Conn struct {
	Conn *websocket.Conn
	// ResponseHeader can be modified eg. in OnConnect to be included in the initial http response
	ResponseHeader http.Header

	// PingInterval is how often we send a ping frame to make sure someone is still listening
	PingInterval time.Duration

	// PongTimeout is how long after sending a ping frame we will wait for a pong frame before closing the connection
	PongTimeout time.Duration

	MaxMessageSize int64
	WriteTimeout   time.Duration

	messageMode int
	outbox      chan *message
	inbox       chan *message
	closeSignal chan bool
}

// Read implements io.Reader.  If type information is needed, you can use ReadMessage.
// Both methods consume the same stream and should probably not be used together.
// WARNING: Read should only be used by ws.Handler that does not expose OnMessage
// Read will block indefinitely by default if ws.Server is implemented
func (c *Conn) Read(p []byte) (int, error) {
	msg := <-c.inbox
	r := bytes.NewReader(msg.content)
	return r.Read(p)
}

// ReadMessage is an alternative to Read and Server.OnMessage that provides websocket messagetype information.
// Blocks until message is available for reading.
func (c *Conn) ReadMessage(p []byte) (bytesRead int, isBinary bool, err error) {
	msg := <-c.inbox
	r := bytes.NewReader(msg.content)
	bytesRead, err = r.Read(p)
	if err != nil {
		return
	}
	isBinary = msg.isBinary
	return
}

// Write sends a message over the underlying websocket connection.  Safe for concurrent use.
// The message type will be equal to c.messageType which can be configured with c.SetMessageMode.
// If writing both types is required, Conn.Conn().WriteMessage is available
// Implements io.ReadWriteCloser
func (c *Conn) Write(m []byte) (int, error) {
	out := &message{m, c.messageMode == websocket.BinaryMessage}
	c.outbox <- out

	// this is not an accurate portrayal of what happens...
	return len(m), nil
}

// Close causes the connection to close.  How about that?
func (c *Conn) Close() error {
	timeout := time.NewTimer(30 * time.Second)
	select {
	case c.closeSignal <- true:
	case <-timeout.C:
		// avoid memory leak if this channel is no longer being consumed
		return errors.New("close timed out")
	}

	return nil
}

// SetMessageMode configures the types of messages that Conn will send with Write.  This has no bearning on
// messages gotten from Read.  Valid values for m are ws.ModeBinary or ws.ModeText
func (c *Conn) SetMessageMode(m int) {
	if m != ModeBinary && m != ModeText {
		log.Println("[ws] \t WARNING: Illegal message mode.  Please use ws.ModeBinary or " +
			"ws.ModeText")
		return
	}
	c.messageMode = m
}

type message struct {
	content  []byte
	isBinary bool
}
