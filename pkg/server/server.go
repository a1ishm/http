package server

import (
	"bytes"
	"io"
	"log"
	"net"
	"strings"
	"sync"
)

type HandlerFunc func(conn net.Conn)

type Server struct {
	addr     string
	mu       sync.RWMutex
	handlers map[string]HandlerFunc
}

func NewServer(addr string) *Server {
	return &Server{addr: addr, handlers: make(map[string]HandlerFunc)}
}

func (s *Server) Register(path string, handler HandlerFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[path] = handler
}

func (s *Server) Start() error {
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		log.Print(err)
		return err
	}
	defer func() {
		if cerr := listener.Close(); cerr != nil {
			if err == nil {
				err = cerr
				return
			}
			log.Print(cerr)
		}
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Print(err)
			continue
		}

		go s.Handle(conn)
		if err != nil {
			log.Print(err)
			continue
		}
	}
}

func (s *Server) Handle(conn net.Conn) (err error) {
	defer func() {
		if cerr := conn.Close(); cerr != nil {
			if err == nil {
				err = cerr
				return
			}
			log.Print(err)
		}
	}()
	
	s.mu.RLock()

	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err == io.EOF {
		log.Printf("%s", buf[:n])
		return nil
	}
	if err != nil {
		return err
	}

	data := buf[:n]
	requestLineDelim := []byte{'\r', '\n'}
	requestLineEnd := bytes.Index(data, requestLineDelim)
	if requestLineEnd == -1 {
		log.Print("no line end")
		return err
	}

	requestLine := string(data[:requestLineEnd])
	parts := strings.Split(requestLine, " ")
	if len(parts) != 3 {
		log.Print("not enough parts")
		return err
	}

	path, version := parts[1], parts[2]
	
	if version != "HTTP/1.1" {
		log.Print("bad version")
		return err
	}

	handler, ok := s.handlers[path]
	if !ok {
		log.Print("bad addres")
		return err
	}

	s.mu.RUnlock()
	handler(conn)

	return nil
}
