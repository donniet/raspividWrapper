package main

import (
	"log"
	"net"
	"sync"
	"time"
)

/*
SocketServer implements io.Writer and to write to all connection on a TCP Listener
*/
type SocketServer struct {
	// map of all connections
	connections map[*net.TCPConn]bool
	// locks the connection map
	lock            sync.Locker
	listener        *net.TCPListener
	KeepAlivePeriod time.Duration
}

/*
NewSocketServer creates a new socket server
*/
func NewSocketServer() *SocketServer {
	return &SocketServer{
		connections: make(map[*net.TCPConn]bool),
		lock:        new(sync.Mutex),
	}
}

/*
ServeTCP opens a TCP Listener and accepts incoming connections
*/
func (s *SocketServer) ServeTCP(port string) error {
	var addr *net.TCPAddr
	var err error

	addr, err = net.ResolveTCPAddr("tcp", port)
	if err != nil {
		return err
	}
	s.listener, err = net.ListenTCP("tcp", addr)
	if err != nil {
		return err
	}

	defer s.listener.Close()

	for {
		var conn *net.TCPConn

		conn, err = s.listener.AcceptTCP()
		if err != nil {
			break
		}
		if s.KeepAlivePeriod != 0 {
			if err = conn.SetKeepAlivePeriod(s.KeepAlivePeriod); err != nil {
				log.Printf("error setting keepalive period: %v", err)
				break
			}
		}
		if err = conn.SetKeepAlive(true); err != nil {
			log.Printf("error setting keepalive: %v", err)
			break
		}
		log.Printf("accepted connection %s", conn.RemoteAddr())

		s.lock.Lock()
		s.connections[conn] = true
		s.lock.Unlock()
	}

	return err
}

/*
Close closes the underlying tcp listener and any open connections.
*/
func (s *SocketServer) Close() error {
	err := s.listener.Close()

	// extract active connections
	var conns []*net.TCPConn
	s.lock.Lock()
	for c := range s.connections {
		conns = append(conns, c)
	}
	s.lock.Unlock()

	for _, c := range conns {
		c.Close()
	}

	s.lock.Lock()
	for c := range s.connections {
		delete(s.connections, c)
	}
	s.lock.Unlock()

	return err
}

/*
Write implements the io.Writer and writes to all connections listening.  Writer will also
remove connectsion which error on the write
*/
func (s *SocketServer) Write(b []byte) (int, error) {

	// extract active connections
	var conns []*net.TCPConn
	s.lock.Lock()
	for c := range s.connections {
		conns = append(conns, c)
	}
	s.lock.Unlock()

	wg := new(sync.WaitGroup)

	// write to all open connections in parallel
	toRemove := make([]bool, len(conns))

	for i, c := range conns {
		wg.Add(1)
		go func(i int, c *net.TCPConn) {
			defer wg.Done()

			if _, err := c.Write(b); err != nil {
				log.Printf("removing connection %s err: %v", c.RemoteAddr(), err)
				toRemove[i] = true
				c.Close()
			} else {
				toRemove[i] = false
			}
		}(i, c)
	}

	// wait for all the writers to finish
	wg.Wait()

	// remove connections which are closed
	s.lock.Lock()
	for i, closed := range toRemove {
		if closed {
			delete(s.connections, conns[i])
		}
	}
	s.lock.Unlock()

	return len(b), nil
}
