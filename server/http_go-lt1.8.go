// +build !go1.8

// This file implements Close() function for (*Server)

package server

import (
	"fmt"
	"net"
	"time"
	"sync"
)

type serverCompat struct {
	conns []*conn
	connsMutex *sync.Mutex
	listener net.Listener
}

const (
	keepAliveTimeout = 5 * time.Minute
)

var (
	ErrNotImplemented = fmt.Errorf("Not implemented, yet")
)

// Close stops (*Server).Run()/(*Server).ListenAndServe() and closes
// all connections
func (s *Server) Close() error {
	err := s.listener.Close()
	if err != nil {
		return err
	}
	return s.closeConnections()
}

func (s *Server) closeConnections() (result error) {
	for _, conn := range s.conns {
		err := conn.Close()
		if err != nil {
			result = err
		}
	}
	s.conns = []*conn{}
	return
}

func (s *Server) addConnection(c *conn) {
	s.conns = append(s.conns, c)
}

func (s *Server) removeConnection(a *conn) { // TODO: use maps instead of slices to prevent this slow search
	for idx, b := range s.conns {
		if a != b {
			continue
		}

		s.conns = append(s.conns[:idx], s.conns[idx+1:]...)
		return
	}
}

// There're two reasons of this listener type (`tcpKeepAliveListener`).
// 1. To track connections (to be able to close them)
// 2. To be sure to enable TCP keep-alive (see below).
//
// Go1.8 documentations claims:
//
// ```
// ListenAndServe() listens on the TCP network address srv.Addr and
// then calls Serve to handle requests on incoming connections.
// Accepted connections are configured to enable TCP keep-alives.
// ```
//
// So we have to be sure that accepted connections will be configured
// to enable TCP keep-alives.
type tcpKeepAliveListener struct {
	*net.TCPListener

	server *Server
}
type conn struct {
	net.Conn

	server *Server
	mutex *sync.Mutex
	isClosed bool
}
func (conn *conn) Close() error {
	conn.mutex.Lock()
	defer conn.mutex.Unlock()
	if conn.isClosed {
		return nil
	}
	conn.isClosed = true
	conn.server.removeConnection(conn)
	return conn.Conn.Close()
}
func (conn *conn) CloseRead() error {
	return ErrNotImplemented
}
func (conn *conn) CloseWrite() error {
	return ErrNotImplemented
}
func (listener tcpKeepAliveListener) Accept() (net.Conn, error) {
	rawConn, err := listener.AcceptTCP()
	if err != nil {
		return nil, err
	}
	rawConn.SetKeepAlive(true)
	rawConn.SetKeepAlivePeriod(keepAliveTimeout)
	c := &conn{Conn: rawConn, mutex: &sync.Mutex{}, server: listener.server}
	listener.server.addConnection(c)
	return c, nil
}

// ListenAndServer does the same as (*http.Server).ListenAndServe() but
// also stores the listener into s.Listener to be able to close it in
// Go of version prior to 1.8
func (s *Server) ListenAndServe() error {
	var err error
	s.connsMutex = &sync.Mutex{}
	s.listener, err = net.Listen("tcp", s.Addr)
	if err != nil {
		return err
	}
	return s.Serve(tcpKeepAliveListener{TCPListener: s.listener.(*net.TCPListener), server: s})
}
