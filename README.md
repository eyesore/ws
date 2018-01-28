# ws
High level websocket server library for Go, built on top of gorilla.websocket

## Install

`go get github.com/eyesore/ws`

## Usage

[Check out the complete documentation on godoc](https://godoc.org/github.com/eyesore/ws)

### Implement the Interface

This package exposes a pair of interfaces that might be implemented on any type to allow it to handle WebSocket connections.

`Connector` should implement [Tornado-style](http://www.tornadoweb.org/en/stable/websocket.html) event handlers for the stages of the WebSocket lifecycle:

* `OnConnect`
* `OnOpen`
* `OnClose`

All callbacks must be implemented to fulfill the interface, but no-ops are fine.  Returning an error from any callback will terminate the WebSocket connection.

The `*Conn` returned by `Connector.Conn()` is an io.ReadWriteCloser, and as such is safe for writing by multiple goroutines.

A `Server` must implement an additional callback `OnMessage`, which is how this type will interact with incoming messages rather than reading them.  **A `Server` may not access messages with `Conn().Read`** - the `Conn` is already being read to in order for the callback to receive messages.  See [the godoc](https://godoc.org/github.com/eyesore/ws) for full documentation.


### Handle Incoming Connections Automatically With Handler

`Handler` is an http.Handler ready to serve incoming connections and upgrade them to WebSockets.

`Factory` is a function that returns a `Connector` (and an error, optionally).

Create a handler and serve it with package `http`:

```go
h := ws.NewHandler(NewConnector)
http.Handle("/", h)

err := http.ListenAndServe(":80", nil)
```

Your Connectors will then manage new incoming connections with the callbacks you define.

### Connector Example

```go
type MyConnector struct {
    conn *ws.Conn
}

func (c *MyConnector) Conn() *ws.Conn {
    return c.conn
}

func (c *MyConnector) SetConn(conn *ws.Conn) {
    c.conn = conn
}

func (c *MyConnector) OnConnect(r *http.Request) error {
    return nil
}

func (c *MyConnector) OnOpen() error {
    return nil
}

func (c *MyConnector) OnClose(wasClean bool, code int, reason error) error {
    return nil
}

func New() (ws.Connector, error) {
    return &MyConnector{}, nil
}

func main() {
    h := ws.NewHandler(New)
    http.Handle("/", h)
    http.ListenAndServe(":8080", nil)
}
```

### Advanced Configuration

#### Configure the Handler

`Handler` exports a reference to the underlying `websocket.Upgrader` that allows tight control over the configuration of the WebSocket connection.  You can replace the `Upgrader` with your own, or configure as desired.  The `Handler` also contains several configs that apply default values to each instance of the `Connector`.  See [the godoc](https://godoc.org/github.com/eyesore/ws) for details.

#### Cross-Origin Access

`Handler.Upgrader` has a `CheckOrigin` method that can be overridden to control which origins are allowed to connect to your WebSocket server.  There is also a convenience method: `Handler.AllowAnyOrigin()` that sets up a permissive policy.  Do not call this method without [understanding the risks!](https://www.owasp.org/index.php/HTML5_Security_Cheat_Sheet#Communication_APIs)

In addition to configuring default settings on `Handler`, each `Connector` may modify `Conn` settings (both `ws.Conn` and `websocket.Conn`) during the `OnOpen` callback.

```go
func(c *MyConnector) OnOpen() error {
    if c.shouldLimitMessageSize() {
        c.Conn().MaxMessageSize = 1024
    }
    // you can also access the underlying websocket.Conn
    c.Conn().Conn.EnableWriteCompression(true)
}
```

The `OnConnect` callback can be used to set the ResponseHeader for the upgrade request.  You **should** check for `sec-websocket-protocol` header in the request and negotiate a subprotocol if appropriate.

```go
func (c *MyConnector) OnConnect(r *http.Request) error {
    // do a better check than this
    if protocols := r.Header.Get("sec-websocket-protocol"); protocols != "" {
        var p string
        switch {
        case strings.Contains(protocols, "json"):
            p = "json"
        case strings.Contains(protocols, "xml"):
            p = "xml"
        }
        c.Conn().ResponseHeader.Add("sec-websocket-protocol", p)
    }
    return nil
}
```

