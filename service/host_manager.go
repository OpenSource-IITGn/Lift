/*

This contains all the functions for managing
network exchange

*/

package service

import (
	"net"
	"bytes"
	"time"
	"fmt"
	"math/rand"
	"crypto/md5"
)

// For making the salt
const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"

func RandStringBytes(n int) []byte {
    b := make([]byte, n)
    for i := range b {
        b[i] = letterBytes[rand.Intn(len(letterBytes))]
    }
    return b
}

// Server side auth request start
func authenticate(sock *socket, config *Config) bool {
	salt := RandStringBytes(6)
	sock.Write([]byte("CHKAUTH"))

	if sock.err != nil { return false }

	sock.Read(1024)
	if sock.err!=nil { return false}
	
	if (!bytes.Equal(sock.buf, []byte(config.Username))) { return false}

	sock.Write(salt)

	// Generating hash of the password with salt
	passhash := md5.Sum(append(salt, []byte(config.Password)...))

	sock.Read(32)
	
	if sock.err!=nil {return false}

	if (bytes.Equal(sock.buf, passhash[:])) {
		sock.Write([]byte("PASS"))
		return true
	}

	sock.Write([]byte("FAIL"))
	return false
}

// Client side auth response
func startAuth(sock *socket, config *Config) bool {
	sock.Read(64)
	if sock.err != nil { return false }
	if (!bytes.Equal(sock.buf, []byte("CHKAUTH"))) { return false}

	sock.Write([]byte(config.Username))

	sock.Read(64)
	if sock.err != nil { return false }
	if len(sock.buf) < 6 { return false }

	passhash := md5.Sum(append(sock.buf, []byte(config.Password)...))

	sock.Write(passhash[:])

	sock.Read(64)
	if sock.err != nil { return false }
	if (bytes.Equal(sock.buf, []byte("PASS"))) { return true }

	return false
}


// Function to update the host list
func updateHosts(currMap map[string]int64, recvMap map[string]int64) {
	for host, tstamp := range recvMap {
		val, _ := currMap[host]
		if val < tstamp {
			currMap[host] = tstamp
		}
	}
}


// Function for information exchange
func infXcng(sock *socket, config *Config) int {
	//Exchanging local host
	sock.Write(config.PubHostList)
	
	if sock.err != nil { return 0 }

	recvPubList := make(map[string]int64)

	sock.ReadObj(&recvPubList)

	if sock.err != nil { return 0 }

	if authenticate(sock, config) == true {
		recvPrivList := make(map[string]int64)

		sock.ReadObj(&recvPrivList)
		if sock.err != nil { return 0 }

		sock.Write(config.PrivHostList)

		updateHosts(config.PrivHostList, recvPrivList)
		updateHosts(config.PubHostList, recvPrivList)
	}

	updateHosts(config.PubHostList, recvPubList)

	return 0
}


// Client information receive and updates
func infRecv(sock *socket, config *Config) int {

	recvPubList := make(map[string]int64)

	sock.ReadObj(&recvPubList)
	if sock.err != nil { return 0 }

	sock.Write(config.PubHostList)
	if sock.err != nil { return 0 }

	if startAuth(sock, config) {

		host, _ := sock.RemoteAddr()

		config.PrivHostList[host] = time.Now().Unix()

		sock.Write(config.PrivHostList)
		if sock.err != nil { return 0 }

		recvPrivList := make(map[string]int64)

		sock.ReadObj(&recvPrivList)

		updateHosts(config.PrivHostList, recvPrivList)
		updateHosts(config.PubHostList, recvPrivList)
	}

	updateHosts(config.PubHostList, recvPubList)

	return 0
}


// Banner Recieve client
func bannerXcng(sock *socket) bool {
	sock.Read(64)
	if sock.err!= nil {return false}
	if bytes.Equal(sock.buf, []byte("LIFT")){
		return true
	} else {
		return false
	}
}

// Check validity of IP
func checkHostValid(host string, maskList []net.IPNet) bool {
	ip := net.ParseIP(host)
	if ip==nil {return false}
	for _, mask := range maskList {
		if mask.Contains(ip) { return true }
	}
	return false
}

// Renewing Host List
func (serv Service) HostRenewal() {
	// FOR TESTING
	time.Sleep(time.Second * 10)
	// Public list iteration
	for host, tstamp := range (*serv.Config).PubHostList {

		if (!checkHostValid(host, (*serv.Config).MaskList)){
			delete((*serv.Config).PubHostList, host)
			continue
		}

		conn, err := net.DialTimeout(
			"tcp",
			fmt.Sprintf("%s:%v", host, (*serv.Config).ComPort),
			time.Millisecond * 300,)

		if (err!= nil && time.Now().Unix() - tstamp < 86400) {
			delete((*serv.Config).PubHostList, host)
			continue
		}

		sock := Gensock(conn)
		if bannerXcng(sock) {
			sock.Write([]byte("HXCNG"))
			(*serv.Config).PubHostList[host] = time.Now().Unix()
		} else {
			if time.Now().Unix() - tstamp < 86400 {
				delete((*serv.Config).PubHostList, host)
			}
			sock.Close()
			continue
		}

		infRecv(sock, serv.Config)
		sock.Close()
	}
	fmt.Println(*serv.Config)
}