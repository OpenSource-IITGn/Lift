package main

import "./service"
import "net"
//import "time"

// Currently for testing
func main(){
	mask_list := make([]net.IPNet, 1)
	mask_list[0].IP = net.ParseIP("127.0.0.1")
	mask_list[0].Mask = net.IPMask(net.ParseIP("255.255.255.255"))
	config := service.Config {
		Username: "test",
		Password: "test",
		PubHostList: make(map[string]int64),
		PrivHostList: make(map[string]int64),
		ComPort: 33446,
		MaskList: mask_list,
	}

	config.PubHostList["127.0.0.1"] = 0

	serv := service.Service{
		Config: &config,
	}

	go serv.HostRenewal()

	serv.Lserver()

}