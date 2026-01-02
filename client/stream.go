package main

import (
	"bufio"
	"io"
	"log"
	"net"
	"strconv"
	"strings"

	"z44-tunnel/common"
)

// handleStream handles an incoming stream from the server
func handleStream(stream net.Conn, portMap map[int]string) {
	defer common.CloseConn(stream)

	portStr, err := bufio.NewReader(stream).ReadString('\n')
	if err != nil && err != io.EOF {
		log.Printf("Failed to read port: %v", err)
		return
	}

	port, err := strconv.Atoi(strings.TrimSpace(portStr))
	if err != nil || !common.ValidatePort(port) {
		log.Printf("Invalid port: %s", strings.TrimSpace(portStr))
		return
	}

	localAddr, ok := portMap[port]
	if !ok {
		log.Printf("No mapping for port %d", port)
		return
	}

	local, err := net.DialTimeout("tcp", localAddr, LocalServiceTimeout)
	if err != nil {
		log.Printf("Failed to dial %s: %v", localAddr, err)
		return
	}
	defer common.CloseConn(local)

	if _, err := stream.Write([]byte("OK\n")); err != nil {
		log.Printf("Failed to send OK: %v", err)
		return
	}

	common.PipeConnections(stream, local, "stream/local")
}
