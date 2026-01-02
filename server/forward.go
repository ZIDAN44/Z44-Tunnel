package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"time"

	"z44-tunnel/common"
)

// forwardLoop handles forwarding connections from local listeners to the tunnel
func forwardLoop(ln net.Listener, port int, server *TunnelServer) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic in forwardLoop port %d: %v", port, r)
		}
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			if netErr, ok := err.(net.Error); ok && !netErr.Temporary() {
				return
			}
			time.Sleep(100 * time.Millisecond)
			continue
		}

		sess := server.GetActiveSession()
		if sess == nil || sess.IsClosed() {
			common.CloseConn(conn)
			continue
		}

		// Rate limiting: check token bucket and stream count
		if !server.rateLimiter.Allow() {
			log.Printf("Rate limit exceeded for port %d", port)
			common.CloseConn(conn)
			continue
		}

		if !server.IncrementStreamCount() {
			log.Printf("Max concurrent streams reached for port %d", port)
			common.CloseConn(conn)
			continue
		}

		stream, err := sess.Open()
		if err != nil {
			server.DecrementStreamCount()
			log.Printf("Failed to open stream port %d: %v", port, err)
			common.CloseConn(conn)
			continue
		}

		if _, err := fmt.Fprintf(stream, "%d\n", port); err != nil {
			server.DecrementStreamCount()
			common.CloseConn(conn)
			common.CloseConn(stream)
			continue
		}

		stream.SetReadDeadline(time.Now().Add(HandshakeTimeout))
		buf := make([]byte, 3)
		n, err := io.ReadFull(stream, buf)
		stream.SetReadDeadline(time.Time{})

		if err != nil || n < 2 || string(buf[:2]) != "OK" {
			server.DecrementStreamCount()
			log.Printf("❌ Zombie detected port %d", port)
			sess.Close()
			common.CloseConn(conn)
			common.CloseConn(stream)
			continue
		}

		go func(c, s net.Conn) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Panic in pipe port %d: %v", port, r)
				}
			}()
			defer server.DecrementStreamCount()
			defer common.CloseConn(c)
			defer common.CloseConn(s)
			common.PipeConnections(c, s, "conn/stream")
		}(conn, stream)
	}
}
