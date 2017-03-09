package main

import (
	"fmt"
	"github.com/dist-ribut-us/log"
	"github.com/dist-ribut-us/natt/igdp"
	"github.com/dist-ribut-us/overlay"
	"github.com/dist-ribut-us/prog"
)

const (
	Send = byte(iota)
	Register
)

func main() {
	log.Panic(log.ToFile())
	log.Go()
	log.SetDebug(true)
	log.Contents = log.Truncate

	proc, _, _, err := prog.ReadArgs()
	log.Panic(err)

	if err := igdp.Setup(); err == nil {
		_, err = igdp.AddPortMapping(proc.Port(), proc.Port())
		log.Panic(err)
	}
	ip, err := igdp.GetExternalIP()
	log.Panic(err)

	log.Info("starting_server")
	overlayNode, err := overlay.NewServer(proc, ip)
	log.Panic(err)
	log.Info("server_started")

	log.Info(fmt.Sprintf("IPC> 127.0.0.1:%d NET> %s:%d %s", proc.Port(), ip, overlayNode.Port(), overlayNode.PubStr()))

	onCh := overlayNode.Chan()
	ipcCh := overlayNode.IPCChan()
	for {
		select {
		case msg := <-onCh:
			log.Info("NET: ", string(msg.Body))
		case msg := <-ipcCh:
			log.Info("IPC: ", string(msg.Body))
		}
	}
}
