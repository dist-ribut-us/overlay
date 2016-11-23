package main

import (
	"fmt"
	"github.com/dist-ribut-us/crypto"
	"github.com/dist-ribut-us/ipc"
	"github.com/dist-ribut-us/natt/igdp"
	"github.com/dist-ribut-us/overlay"
	"github.com/dist-ribut-us/rnet"
)

const (
	Send = byte(iota)
	Register
)

func main() {
	port := fmt.Sprintf(":%d", rnet.RandomPort())
	overlayNode, err := overlay.NewServer(port)
	if err != nil {
		panic(err)
	}

	err = igdp.Setup()
	if err != nil {
		panic(err)
	}

	err = igdp.AddPortMapping(overlayNode.Port(), overlayNode.Port())
	if err != nil {
		fmt.Println(err)
	}
	ip, err := igdp.GetExternalIP()
	if err != nil {
		panic(err)
	}

	ipcSrv, err := ipc.RunNew(0)

	fmt.Printf("IPC> 127.0.0.1:%d\n", ipcSrv.Port())
	fmt.Printf("NET> %s:%d\n", ip, overlayNode.Port())
	fmt.Println(overlayNode.PubStr())

	onCh := overlayNode.Chan()
	ipcCh := ipcSrv.Chan()
	for {
		select {
		case msg := <-onCh:
			fmt.Println("NET: ", string(msg.Body))
		case msg := <-ipcCh:
			fmt.Println("IPC: ", string(msg.Body))
		}
	}
}
