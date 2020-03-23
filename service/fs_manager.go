/*
Contains functions for handling
file system related services
*/

package service

//import "net"
import (
	"os"
	"path"
	"net"
	"time"
	"bytes"
)

const BUFFERSIZE = 4096

// Locates file and make it ready to be transfered
// Receives token from client for access for file transfer connection
// Transfers the port for making the connection
func fileService(sock *socket, config *Config) int {
	sock.Write([]byte("CONT")])
	filepath := new(Location)
	sock.ReadObj(filepath)
	fpath := path.Join(filepath.Path...)
	if filepath.Priv {
		if !authenticate(sock, config) {
			return 0
		}

		fpath = path.Join(config.PrivatePath, fpath)
	} else {
		sock.Write([]byte("PASS"))
		fpath = path.Join(config.PublicPath, fpath)
	}

	// Receiving the token
	sock.Read(64)

	if sock.err != nil { return 0 }

	token := []byte
	copy(token, sock.buf)

	if _, err := os.Stat(fpath); err == nil {
		listen, lerr := net.Listen("tcp", ":0")
		
		if lerr != nil {
			sock.Write(int(0))
			return 0
		}
		port:= listen.Addr().(*net.TCPAddr).Port
		sock.Write(port)

		go fileReqHandle(fpath, token, listen)
	} else {
		sock.Write(int(0))
	}
	return 0
}


// Handles the file Transfer
func fileReqHandle(
	fpath string,
	token []byte,
	listen net.Listen) int {
		file, err:= os.Open(fpath)
		defer file.Close()
		if err != nil { return 0 }

		listen.SetDeadline(time.Second * 10)

		conn, cerr := listen.Accept()

		listen.Close()

		if cerr != nil { return 0 }

		sock := Gensock(conn)
		defer sock.Close()

		fileReq := new(FileReq)

		sock.Read(fileReq)

		if sock.err != nil { return 0 }

		if (!bytes.Equal(sock.buf, fileReq.token)) {
			return 0
		}

		file.Seek(fileReq.Offset, 0)

		sendBuffer := make([]byte, BUFFERSIZE)

		sent := uint64(0)

		for {
			if fileReq.BlockSize < BUFFERSIZE {
				sendBuffer = make([]byte, int(fileReq.BlockSize))
			}


			sent, err = file.Read(sendBuffer)
			if (err == io.EOF || fileReq.BlockSize==0) {
				return 0
			}
			sock.Write(sendBuffer)
			if sock.err != nil { return 0 }
			fileReq.BlockSize = fileReq.BlockSize - sent
		}

	}