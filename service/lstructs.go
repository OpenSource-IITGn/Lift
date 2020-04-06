/*
Contains all the lift service structs
and sock type functions
*/

package service

import (
	"net"
	"time"
	"encoding/gob"
	"bytes"
	//"os"
)

type Location struct {
	Priv bool
	Path []string
}

type FileReq struct {
	Token []byte
	Offset int64
	BlockSize uint64
}

type Config struct {
	Username string
	Password string
	ComPort uint16
	PubHostList map[string]int64
	PrivHostList map[string]int64
	MaskList []net.IPNet
	PublicPath string
	PrivatePath string
}

type Service struct {
	Config *Config
	Host string
	LPrivDir []string
	LPubDir []string
	RPubDir []string
	RPrivDir []string
	Files map[string]bool
}

type socket struct{
	socket net.Conn
	timeout time.Duration
	buf []byte
	err error
	reader *gob.Decoder
	writer *gob.Encoder
}

func Gensock(conn net.Conn) *socket {
	sock := socket {
		socket: conn,
		timeout: time.Minute,
		buf: make([]byte, 1024),
		err: nil,
		reader: gob.NewDecoder(conn),
		writer: gob.NewEncoder(conn),
	}

	return &sock
}

func (sock *socket) Close() {
	sock.err = sock.socket.Close()
}

func (sock *socket) Settimeout (n uint32){
	sock.timeout = time.Millisecond * time.Duration(n)
}

func (sock *socket) Read(n uint16) {
	sock.err = sock.socket.SetReadDeadline(time.Now().Add(sock.timeout))
	if sock.err == nil {
		buf := make([]byte, n)
		sock.err = sock.reader.Decode(&buf)
		sock.buf = bytes.Trim(buf, "\x00")
	}
}

func (sock *socket) ReadObj(x interface{}) {
	sock.err = sock.socket.SetReadDeadline(time.Now().Add(sock.timeout))
	if sock.err == nil {
		sock.err = sock.reader.Decode(x)
	}
}

func (sock *socket) Write(x interface{}) {
	sock.err = sock.socket.SetWriteDeadline(time.Now().Add(sock.timeout))
	if sock.err == nil {
		sock.err = sock.writer.Encode(x)
	}
}

func (sock *socket) RemoteAddr() (host, port string) {
	host, port, err := net.SplitHostPort(sock.socket.RemoteAddr().String())
	sock.err = err
	if err!=nil { return "",""}
	return host, port
}