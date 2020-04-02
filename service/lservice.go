/*
The lift service
*/

package service

import (
    "bytes"
    "net"
    //"time"
    "fmt"
)

/*
This will handle the client requests
It has a deadline of 1 sec for reading the request
*/

func handleClient(conn net.Conn, config *Config) int {

	sock := Gensock(conn)

	defer sock.Close()

	sock.Write([]byte("LIFT"))

	sock.Settimeout(60000)

	if sock.err != nil { return 0 }

	sock.Read(64)

	if sock.err != nil {
		return 0
	}

	if bytes.Equal(sock.buf, []byte("HXCNG")){
		infXcng(sock, config)
	} else if bytes.Equal(sock.buf, []byte("LIST")){
		lsService(sock, config)
	} else if bytes.Equal(sock.buf, []byte("FILESZ")) {
		sizeService(sock, config)
	} else if bytes.Equal(sock.buf, []byte("FILEH")) {
		hashService(sock, config)
	} else if bytes.Equal(sock.buf, []byte("FILE")) {
		fileService(sock, config)
	}
	return 0
}

/*

This is the remote server started on ComPort

*/
func (serv Service) Lserver() int {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%v", (*serv.Config).ComPort))
	if err != nil {
		return 0
	}
	defer ln.Close()
	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		go handleClient(conn, serv.Config)
	}

	return 0

}

