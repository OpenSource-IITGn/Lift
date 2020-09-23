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
	//"io/ioutil"
	"fmt"
	"time"
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

// Client side sending the filepath
func sendFilePath(sock *socket, config *Config, location Location) bool {
	sock.Read(64)
	if sock.err != nil { return false }

	if (!bytes.Equal(sock.buf, []byte("CONT"))) {
		return  false
	}

	sock.Write(location)
	if sock.err != nil {return false}

	if location.Priv {
		if startAuth(sock, config) { return true }
	} else {
		sock.Read(64)
		if sock.err != nil { return false }
	}

	if (bytes.Equal(sock.buf, []byte("PASS"))) {
			return true
	}

	return false

}

func lsService(sock *socket, config *Config) int {
	fpath := getFilePath(sock, config)
	if fpath == "" {return 0}

	sock.Read(64)
	if sock.err != nil { return 0 }

	if (!bytes.Equal(sock.buf, []byte("CONT"))) {
		return  0
	}

	file, _ := os.Open(fpath)

	filelist, err := file.Readdir(0)
	if err != nil { return 0}

	obj := make(map[string]bool)

	for _,fi := range filelist {
		obj[fi.Name()] = fi.IsDir()
	}

	sock.Write(obj)

	return 0

}


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

		if (!bytes.Equal(token, fileReq.Token)) {
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


// Client make connection
func makeConnection(addr string) *socket {
	dial := net.Dialer{Timeout: time.Minute,}
	conn, err := dial.Dial("tcp", addr)

	if err!= nil { return nil }

	sock := Gensock(conn)

	if !bannerXcng(sock) {sock.Close(); return nil }

	return sock
}


// Client side get directory listing
func (serv *Service) GetDir(location Location) int {
	sock := makeConnection(
		fmt.Sprintf(
			"%s:%v",
			serv.Host,
			(*serv.Config).ComPort),
		)

	if sock == nil { return 0 }

	defer sock.Close()

	sock.Write([]byte("LIST"))

	if !sendFilePath(sock, serv.Config, location) { return 0 }

	sock.Write([]byte("CONT"))
	if sock.err!= nil { return 0 }

	serv.Files = make(map[string]bool)

	sock.ReadObj(&serv.Files)

	return 0
}

/*

Client side file request. This will be split in two parts.
First will be obtaining size and connection token from server.

This part has to be optimized using the File related services.
Current implemenation is a simple single connection request.

*/

func (serv *Service) FileServiceReq(location Location) int {
	sock := makeConnection(
		fmt.Sprintf(
			"%s:%v",
			serv.Host,
			(*serv.Config).ComPort),
		)

	if sock == nil { return 0 }

	defer sock.Close()

	sock.Write([]byte("FILESZ"))

	if !sendFilePath(sock, serv.Config, location) { return 0 }

	sock.Write([]byte("CONT"))
	if sock.err != nil { return 0 }

	var size int64

	sock.ReadObj(&size)

	if size==0 { return 0 }

	sock.Close()

	sock = makeConnection(
		fmt.Sprintf(
			"%s:%v",
			serv.Host,
			(*serv.Config).ComPort),
		)

	sock.Write([]byte("FILE"))

	if !sendFilePath(sock, serv.Config, location) { return 0 }

	token := RandStringBytes(6)
	sock.Write(token)
	if sock.err != nil { return 0 }

	var port int
	sock.ReadObj(&port)

	if port == 0 { return 0}

	sock.Close()

	var fpath string

	if location.Priv {
		fpath = path.Join(serv.LPrivDir...)
		fpath = path.Join(
			serv.Config.PrivatePath,
			fpath,
			location.Path[len(location.Path)-1],
		)
	} else {
		fpath = path.Join(serv.LPubDir...)
		fpath = path.Join(
			serv.Config.PublicPath,
			fpath,
			location.Path[len(location.Path)-1],
		)
	}

	file, err := os.OpenFile(fpath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0774)
	if err!= nil { return 0 }
	defer file.Close()

	conn, err := net.DialTimeout(
		"tcp",
		fmt.Sprintf("%s:%v", serv.Host, port,),
		time.Millisecond * 300,
	)

	if err != nil { return 0 }

	sock = Gensock(conn)

	fileReq := FileReq{
		Token: token,
		Offset: 0,
		BlockSize: uint64(size),
	}

	sock.Write(fileReq)
	if sock.err != nil { return 0 }

	recvBuffer := make([]byte, BUFFERSIZE)

	recv := 0

	for {
		if size < BUFFERSIZE {
			recvBuffer = make([]byte, size)
		}
		if size==0 {
			return 0
		}

		sock.ReadObj(&recvBuffer)
		if sock.err != nil { return 0 }
		recv, err = file.Write(recvBuffer)
		if err!= nil { return 0 }
		size = size - int64(recv)
	}
}