// +build !go1.8

package server

func (s *Server) Close() error {
	return s.Shutdown()
}
