package main

import "./service"
import "net"
import "os"
import "fmt"
import "strings"
import "bufio"
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
		PublicPath: "E:\\Projects\\ML\\putty",
		PrivatePath: "E:\\Forms\\garuda",
	}

	config.PubHostList["127.0.0.1"] = 0

	serv := service.Service{
		Config: &config,
		Host: "127.0.0.1",
		LPrivDir: make([]string, 1),
		LPubDir: make([]string, 1),
		RPrivDir: make([]string, 1),
		RPubDir: make([]string, 1),
		Files: make(map[string]bool),
	}

	//go serv.HostRenewal()

	go (&serv).Lserver()

	in := bufio.NewReader(os.Stdin)

	for {
		fmt.Print(">>> ")
		first, _ := in.ReadString('\n')
		args := strings.Split(strings.TrimSpace(first), " ")
		if args[0]=="ls"{
			serv.Host = args[1]
			location := service.Location{
				Priv: false,
				Path: make([]string, 1),
			}
			if args[2]=="1"{
				location.Priv=true
			}
			(&serv).GetDir(location)
			for name, b := range (serv.Files) {
				fmt.Println(name, b)
			}
		}
	}

}