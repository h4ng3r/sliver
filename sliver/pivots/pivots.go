package pivots

import (
	"bytes"
	"encoding/binary"
	"math/rand"
	"net"
	"sync"
	"time"

	// {{if .Debug}}
	"log"
	// {{end}}

	pb "github.com/bishopfox/sliver/protobuf/sliver"
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

const (
	readBufSize = 1024
)

func PivotConnectionHandler(conn *net.Conn, connection *transports.Connection, pivotID uint32) {

	defer func() {
		// {{if .Debug}}
		log.Printf("Cleaning up for pivot %d", pivotID)
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
		envelope, err := PivotReadEnvelope(conn)
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

func PivotPipeAcceptNewConnection(ln net.Listener, connection *transports.Connection, pivotType string) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		rand.Seed(time.Now().UnixNano())
		pivotID := rand.Uint32()
		pivotsMap.AddPivot(pivotID, &conn)
		SendPivotOpen(pivotID, pivotType, conn.RemoteAddr().String(), connection)

		// {{if .Debug}}
		log.Println("Accepted a new connection")
		// {{end}}

		// handle connection like any other net.Conn
		go PivotConnectionHandler(&conn, connection, pivotID)
	}
}

// PivotWriteEnvelope - Writes a protobuf envolope to a generic connection
func PivotWriteEnvelope(conn *net.Conn, envelope *pb.Envelope) error {
	data, err := proto.Marshal(envelope)
	if err != nil {
		// {{if .Debug}}
		log.Print("Envelope marshaling error: ", err)
		// {{end}}
		return err
	}
	dataLengthBuf := new(bytes.Buffer)
	binary.Write(dataLengthBuf, binary.LittleEndian, uint32(len(data)))
	(*conn).Write(dataLengthBuf.Bytes())
	(*conn).Write(data)
	return nil
}

// PivotReadEnvelope - Reads a protobuf envolope from a generic connection
func PivotReadEnvelope(conn *net.Conn) (*pb.Envelope, error) {
	dataLengthBuf := make([]byte, 4)
	_, err := (*conn).Read(dataLengthBuf)
	if err != nil {
		// {{if .Debug}}
		log.Printf("Named Pipe error (read msg-length): %v\n", err)
		// {{end}}
		return nil, err
	}
	dataLength := int(binary.LittleEndian.Uint32(dataLengthBuf))
	// {{if .Debug}}
	log.Printf("Found an evelope of %d bytes\n", dataLength)
	// {{end}}
	readBuf := make([]byte, readBufSize)
	dataBuf := make([]byte, 0)
	totalRead := 0
	for {
		n, err := (*conn).Read(readBuf)
		log.Printf("Read %d bytes pending %d\n", totalRead+n, dataLength-totalRead-n)
		dataBuf = append(dataBuf, readBuf[:n]...)
		totalRead += n
		if totalRead == dataLength {
			break
		}
		if err != nil {
			// {{if .Debug}}
			log.Printf("Read error: %s\n", err)
			// {{end}}
			break
		}
	}
	envelope := &pb.Envelope{}
	err = proto.Unmarshal(dataBuf, envelope)
	if err != nil {
		// {{if .Debug}}
		log.Printf("Unmarshaling envelope error: %v", err)
		// {{end}}
		return &pb.Envelope{}, err
	}
	// {{if .Debug}}
	log.Printf("namedPipeReadEnvelope %d\n", envelope.GetType())
	// {{end}}
	return envelope, nil
}

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

// GetPivot - returns a pivot identified by the pivotID
func GetPivot(pivotID uint32) *net.Conn {
	return pivotsMap.Pivot(pivotID)
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
