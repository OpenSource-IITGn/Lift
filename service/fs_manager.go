/*
Contains functions for handling
file system related services
*/

package service

import (
	"os"
	"path"
	"net"
	"bytes"
	"io"
)

const BUFFERSIZE = 4096

// Receive the file path object
// Return the path
// Supporting function for file services
func getFilePath(sock *socket, config *Config) string {
	sock.Write([]byte("CONT"))
	filepath := new(Location)
	sock.ReadObj(filepath)
	fpath := path.Join(filepath.Path...)
	if filepath.Priv {
		if !authenticate(sock, config) {
			return ""
		}

		fpath = path.Join(config.PrivatePath, fpath)
	} else {
		sock.Write([]byte("PASS"))
		fpath = path.Join(config.PublicPath, fpath)
	}

	return fpath
}

func lsService(sock *socket, config *Config) {}
func hashService(sock *socket, config *Config) {}


// Returns the size of the file
// Returns 0 if there is error
func sizeService(sock *socket, config *Config) int {
	fpath := getFilePath(sock, config)
	if fpath == "" { return 0 }

	// Opening file and reading size
	file, err := os.Open(fpath)
	defer file.Close()
	if err != nil { return 0 }

	sock.Read(64)
	if sock.err != nil { return 0 }

	if (!bytes.Equal(sock.buf, []byte("CONT"))) {
		return  0
	}

	if fi, ferr := file.Stat(); ferr == nil {
		sock.Write(fi.Size())
	} else {
		sock.Write(int64(0))
	}
	return 0
}

// Locates file and make it ready to be transfered
// Receives token from client for access for file transfer connection
// Transfers the port for making the connection
func fileService(sock *socket, config *Config) int {
	fpath := getFilePath(sock, config)
	if fpath == "" { return 0 }

	// Receiving the token
	sock.Read(64)

	if sock.err != nil { return 0 }

	token := new([]byte)
	copy(*token, sock.buf)

	if _, err := os.Stat(fpath); err == nil {
		listen, lerr := net.Listen("tcp", ":0")
		
		if lerr != nil {
			sock.Write(int(0))
			return 0
		}
		port:= listen.Addr().(*net.TCPAddr).Port
		sock.Write(port)

		go fileReqHandle(fpath, *token, listen)
	} else {
		sock.Write(int(0))
	}
	return 0
}


// Handles the file Transfer
func fileReqHandle(
	fpath string,
	token []byte,
	listen net.Listener) int {
		file, err:= os.Open(fpath)
		defer file.Close()
		if err != nil { return 0 }

		// Need to add timeout for Listener

		conn, cerr := listen.Accept()

		listen.Close()

		if cerr != nil { return 0 }

		sock := Gensock(conn)
		defer sock.Close()

		fileReq := new(FileReq)

		sock.ReadObj(fileReq)

		if sock.err != nil { return 0 }

		if (!bytes.Equal(sock.buf, fileReq.token)) {
			return 0
		}

		file.Seek(fileReq.Offset, 0)

		sendBuffer := make([]byte, BUFFERSIZE)

		sent := 0

		for {
			if fileReq.BlockSize < BUFFERSIZE {
				sendBuffer = make([]byte, fileReq.BlockSize)
			}


			sent, err = file.Read(sendBuffer)
			if (err == io.EOF || fileReq.BlockSize==0) {
				return 0
			}
			sock.Write(sendBuffer)
			if sock.err != nil { return 0 }
			fileReq.BlockSize = fileReq.BlockSize - uint64(sent)
		}

	}