package rawhttp

import (
	"io"
	"net"
	"sync"
	"time"

	"github.com/projectdiscovery/rawhttp/client"
)

// Dialer can dial a remote HTTP server.
type Dialer interface {
	// Dial dials a remote http server returning a Conn.
	Dial(network, addr string) (Conn, error)
}

type dialer struct {
	sync.Mutex                   // protects following fields
	conns      map[string][]Conn // maps addr to a, possibly empty, slice of existing Conns
}

func (d *dialer) Dial(network, addr string) (Conn, error) {
	d.Lock()
	if d.conns == nil {
		d.conns = make(map[string][]Conn)
	}
	if c, ok := d.conns[addr]; ok {
		if len(c) > 0 {
			conn := c[0]
			c[0], c = c[len(c)-1], c[:len(c)-1]
			d.Unlock()
			return conn, nil
		}
	}
	d.Unlock()
	c, err := net.Dial(network, addr)
	return &conn{
		Client: client.NewClient(c),
		Conn:   c,
		dialer: d,
	}, err
}

type Conn interface {
	client.Client
	io.Closer

	SetDeadline(time.Time) error
	SetReadDeadline(time.Time) error
	SetWriteDeadline(time.Time) error
	Release()
}

type conn struct {
	client.Client
	net.Conn
	*dialer
}

func (c *conn) Release() {
	c.dialer.Lock()
	defer c.dialer.Unlock()
	addr := c.Conn.RemoteAddr().String()
	c.dialer.conns[addr] = append(c.dialer.conns[addr], c)
}
