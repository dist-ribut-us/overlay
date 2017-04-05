package main

import (
	"github.com/dist-ribut-us/log"
	"github.com/dist-ribut-us/overlay"
	"github.com/dist-ribut-us/prog"
	"path"
)

func main() {
	log.Contents = log.Truncate
	log.Panic(log.ToFile(prog.Root() + "overlay.log"))
	log.Go()
	log.SetDebug(true)

	proc, _, key, err := prog.ReadArgs()
	log.Panic(err)

	log.Info(log.Lbl("building_overlay_node"))
	overlayNode, err := overlay.NewServer(proc, 7667)
	log.Panic(err)
	err = overlayNode.Forest(key, path.Join(prog.UserHomeDir(), "overlay"))
	log.Panic(err)

	overlayNode.SetupNetwork()

	overlayNode.Run()
}
