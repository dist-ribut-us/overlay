package main

import (
	"fmt"
	"github.com/dist-ribut-us/natt/igdp"
	"github.com/dist-ribut-us/overlay"
	"github.com/dist-ribut-us/prog"
)

const (
	Send = byte(iota)
	Register
)

func main() {
	proc, _, _, err := prog.ReadArgs()
	check(err)

	if err := igdp.Setup(); err == nil {
		_, err = igdp.AddPortMapping(proc.Port(), proc.Port())
		check(err)
	}
	ip, err := igdp.GetExternalIP()
	check(err)

	overlayNode, err := overlay.NewServer(proc, ip)
	check(err)

	fmt.Printf("IPC> 127.0.0.1:%d\n", proc.Port())
	fmt.Printf("NET> %s:%d\n", ip, overlayNode.Port())
	fmt.Println(overlayNode.PubStr())

	onCh := overlayNode.Chan()
	ipcCh := overlayNode.IPCChan()
	for {
		select {
		case msg := <-onCh:
			fmt.Println("NET: ", string(msg.Body))
		case msg := <-ipcCh:
			fmt.Println("IPC: ", string(msg.Body))
		}
	}
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
