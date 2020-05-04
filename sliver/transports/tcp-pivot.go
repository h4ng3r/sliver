package transports

import (
	"net"

	pb "github.com/bishopfox/sliver/protobuf/sliver"
	"github.com/bishopfox/sliver/sliver/pivots"
)

func tcpPivoteWriteEnvelope(conn *net.Conn, envelope *pb.Envelope) error {
	return pivots.PivotWriteEnvelope(conn, envelope)
}

func tcpPivotReadEnvelope(conn *net.Conn) (*pb.Envelope, error) {
	return pivots.PivotReadEnvelope(conn)
}
