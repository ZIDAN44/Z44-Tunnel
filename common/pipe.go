package common

import (
	"io"
	"log"
	"net"
)

// PipeConnections pipes data bidirectionally between two connections
// It handles panic recovery and proper error logging
func PipeConnections(src, dst net.Conn, label string) {
	done := make(chan struct{}, 2)

	// Copy from src to dst
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Panic in %s copy (src->dst): %v", label, r)
			}
			done <- struct{}{}
		}()
		_, err := io.Copy(dst, src)
		if err != nil && err != io.EOF {
			log.Printf("Error copying %s (src->dst): %v", label, err)
		}
	}()

	// Copy from dst to src
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Panic in %s copy (dst->src): %v", label, r)
			}
			done <- struct{}{}
		}()
		_, err := io.Copy(src, dst)
		if err != nil && err != io.EOF {
			log.Printf("Error copying %s (dst->src): %v", label, err)
		}
	}()

	// Wait for both copies to complete
	<-done
	<-done
}
