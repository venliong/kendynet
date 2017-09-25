package tcp

import (
    "net"
    "github.com/sniperHW/kendynet"
    "github.com/sniperHW/kendynet/stream_socket"
    "time"
)

type Connector struct{
	nettype string
	addr    string
}

func NewConnector(nettype string,addr string) (*Connector,error) {
	return &Connector{nettype:nettype,addr:addr},nil
}

func (this *Connector) Dial(timeout time.Duration) (kendynet.StreamSession,interface{},error) {
	dialer := &net.Dialer{Timeout:timeout}
	conn, err := dialer.Dial(this.nettype , this.addr)
	if err != nil {
		return nil,nil,err
	}
	return stream_socket.NewStreamSocket(conn),nil,nil
}