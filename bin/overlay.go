package main

import (
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
	log.Contents = log.Truncate
	log.Panic(log.ToFile())
	log.Go()
	log.SetDebug(true)

	proc, _, _, err := prog.ReadArgs()
	log.Panic(err)

	if err := igdp.Setup(); err == nil {
		_, err = igdp.AddPortMapping(proc.Port(), proc.Port())
		log.Panic(err)
	}
	ip, err := igdp.GetExternalIP()
	log.Panic(err)

	log.Info(log.Lbl("building_overlay_node"))
	overlayNode, err := overlay.NewServer(proc, ip)
	log.Panic(err)

	log.Info(log.Lbl("IPC>"), proc.Port().On("127.0.0.1"), log.Lbl("Net>"), overlayNode.NetPort().On(ip), overlayNode.PubStr())

	netCh := overlayNode.NetChan()
	ipcCh := overlayNode.IPCChan()
	log.Info("overlay_listening")
	for {
		select {
		case msg := <-netCh:
			log.Info("NET: ", string(msg.Body))
		case msg := <-ipcCh:
			go overlayNode.HandleMessage(msg)
		}
	}
}
