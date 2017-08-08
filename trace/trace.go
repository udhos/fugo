package trace

import (
	"fmt"
	"net"
	//"log"
)

// Trace sends log to UDP socket.
type Trace struct {
	conn net.Conn
}

// New creates new Trace.
func New(server string) (*Trace, error) {
	conn, errDial := net.Dial("udp", server)
	if errDial != nil {
		return nil, errDial
	}
	t := &Trace{conn: conn}
	return t, nil
}

// Printf writes log to Trace.
func (t *Trace) Printf(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	//log.Printf("trace.Printf: " + msg)
	t.conn.Write([]byte(msg))
}

// Write writes log to Trace.
func (t *Trace) Write(b []byte) (int, error) {
	return t.conn.Write(b)
}
