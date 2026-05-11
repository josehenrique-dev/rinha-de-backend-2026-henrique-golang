package server

import (
	"fmt"
	"net"
	"os"
	"sync"
)

type Handler func(path []byte, body []byte) []byte

type Server struct {
	ln      net.Listener
	handler Handler
}

const maxRequestSize = 8 * 1024

var readBufPool = sync.Pool{
	New: func() any {
		b := make([]byte, 4096)
		return &b
	},
}

var Responses [7][]byte

func init() {
	bodies := [6]string{
		`{"approved":true,"fraud_score":0.0}`,
		`{"approved":true,"fraud_score":0.2}`,
		`{"approved":true,"fraud_score":0.4}`,
		`{"approved":false,"fraud_score":0.6}`,
		`{"approved":false,"fraud_score":0.8}`,
		`{"approved":false,"fraud_score":1.0}`,
	}
	for i, body := range bodies {
		Responses[i] = []byte(fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: %d\r\n\r\n%s", len(body), body))
	}
	Responses[6] = []byte("HTTP/1.1 200 OK\r\nContent-Length: 0\r\n\r\n")
}

func Listen(socketPath string, handler Handler) (*Server, error) {
	_ = os.Remove(socketPath)
	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, err
	}
	_ = os.Chmod(socketPath, 0o666)
	srv := &Server{ln: ln, handler: handler}
	go srv.acceptLoop()
	return srv, nil
}

func ListenTCP(addr string, handler Handler) (*Server, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	srv := &Server{ln: ln, handler: handler}
	go srv.acceptLoop()
	return srv, nil
}

func (s *Server) acceptLoop() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return
		}
		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()

	bufRef := readBufPool.Get().(*[]byte)
	buf := *bufRef
	used := 0

	defer func() {
		if cap(buf) <= maxRequestSize {
			*bufRef = buf[:cap(buf)]
			readBufPool.Put(bufRef)
		}
	}()

	for {
		var headEnd int
		for {
			if used >= len(buf) {
				if used >= maxRequestSize {
					return
				}
				newBuf := make([]byte, len(buf)*2)
				copy(newBuf, buf[:used])
				buf = newBuf
			}
			n, err := conn.Read(buf[used:])
			if n > 0 {
				used += n
				if idx := indexHeaderEnd(buf[:used]); idx >= 0 {
					headEnd = idx + 4
					break
				}
			}
			if err != nil {
				return
			}
		}

		path, contentLen := parseRequestLine(buf[:headEnd])
		if contentLen > maxRequestSize-headEnd {
			return
		}
		bodyEnd := headEnd + contentLen
		for used < bodyEnd {
			if used >= len(buf) {
				newBuf := make([]byte, len(buf)*2)
				copy(newBuf, buf[:used])
				buf = newBuf
			}
			n, err := conn.Read(buf[used:])
			if n > 0 {
				used += n
			}
			if err != nil {
				return
			}
		}

		resp := s.handler(path, buf[headEnd:bodyEnd])
		if _, err := conn.Write(resp); err != nil {
			return
		}

		copy(buf, buf[bodyEnd:used])
		used -= bodyEnd
	}
}

func indexHeaderEnd(b []byte) int {
	for i := 0; i+3 < len(b); i++ {
		if b[i] == '\r' && b[i+1] == '\n' && b[i+2] == '\r' && b[i+3] == '\n' {
			return i
		}
	}
	return -1
}

func parseRequestLine(buf []byte) (path []byte, contentLen int) {
	i := 0
	for i < len(buf) && buf[i] != ' ' {
		i++
	}
	i++
	pathStart := i
	for i < len(buf) && buf[i] != ' ' {
		i++
	}
	path = buf[pathStart:i]
	contentLen = findContentLength(buf)
	return
}

func findContentLength(buf []byte) int {
	for i := 0; i+16 < len(buf); i++ {
		if (buf[i] == 'C' || buf[i] == 'c') && isContentLengthPrefix(buf[i:]) {
			j := i + 16
			for j < len(buf) && buf[j] == ' ' {
				j++
			}
			n := 0
			for j < len(buf) && buf[j] >= '0' && buf[j] <= '9' {
				n = n*10 + int(buf[j]-'0')
				j++
			}
			return n
		}
	}
	return 0
}

func isContentLengthPrefix(b []byte) bool {
	const name = "content-length: "
	if len(b) < len(name) {
		return false
	}
	for i := range 15 {
		c := b[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		if c != name[i] {
			return false
		}
	}
	return true
}
