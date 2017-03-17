package main

import (
	"github.com/dist-ribut-us/log"
	"github.com/dist-ribut-us/overlay"
	"github.com/dist-ribut-us/prog"
)

const (
	Send = byte(iota)
	Register
)

func main() {
	log.Contents = log.Truncate
	log.Panic(log.ToFile(prog.Root() + "overlay.log"))
	log.Go()
	log.SetDebug(true)

	proc, _, _, err := prog.ReadArgs()
	log.Panic(err)

	log.Info(log.Lbl("building_overlay_node"))
	overlayNode, err := overlay.NewServer(proc)
	log.Panic(err)

	go overlayNode.SetupNetwork()

	overlayNode.Run()
}
