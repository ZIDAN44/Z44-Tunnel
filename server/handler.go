package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"

	"z44-tunnel/common"

	"github.com/hashicorp/yamux"
)

// handleClient handles a new client connection
func handleClient(conn net.Conn, server *TunnelServer) {
	defer common.CloseConn(conn)

	if conn.RemoteAddr() == nil {
		return
	}
	log.Println("New connection:", conn.RemoteAddr())

	session, err := yamux.Server(conn, common.YamuxConfig(PingInterval, WriteTimeout))
	if err != nil {
		log.Printf("Failed to create yamux session: %v", err)
		return
	}
	defer common.CloseSession(session)

	stream, err := session.Accept()
	if err != nil {
		log.Printf("Failed to accept handshake: %v", err)
		return
	}

	var h Handshake
	if err := json.NewDecoder(stream).Decode(&h); err != nil {
		log.Printf("Failed to decode handshake: %v", err)
		common.CloseConn(stream)
		return
	}
	common.CloseConn(stream)

	if len(h.Mappings) == 0 {
		return
	}

	server.SetActiveSession(session)

	for _, m := range h.Mappings {
		if !common.ValidatePort(m.RemotePort) || server.HasListener(m.RemotePort) {
			continue
		}

		addr := fmt.Sprintf("127.0.0.1:%d", m.RemotePort)
		l, err := net.Listen("tcp", addr)
		if err != nil {
			log.Printf("Failed to listen on %s: %v", addr, err)
			continue
		}
		server.AddListener(m.RemotePort, l)
		log.Printf("✅ Forwarding port %d", m.RemotePort)
		go func(listener net.Listener, p int) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Panic in forwardLoop port %d: %v", p, r)
				}
			}()
			forwardLoop(listener, p, server)
		}(l, m.RemotePort)
	}

	<-session.CloseChan()
	log.Println("⚠️ Client disconnected")
	server.ClearActiveSession(session)
}
