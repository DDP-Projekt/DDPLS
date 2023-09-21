package main

import (
	"flag"
	"os"
	"runtime/pprof"

	"github.com/DDP-Projekt/DDPLS/ddpls"
	"github.com/DDP-Projekt/DDPLS/log"
	"github.com/tliron/kutil/logging"

	// Must include a backend implementation. See kutil's logging/ for other options.
	_ "github.com/tliron/kutil/logging/simple"
)

func main() {
	var cpuprofile string
	flag.StringVar(&cpuprofile, "cpuprofile", "", "write cpu profile to file")
	flag.Parse()

	// This increases logging verbosity (optional)
	logging.Configure(1, nil)

	ls := ddpls.NewDDPLS()

	if cpuprofile != "" {
		f, err := os.Create(cpuprofile)
		if err != nil {
			log.Warningf("error creating cpuprofile file: %w", err)
		} else {
			defer f.Close()
			if err := pprof.StartCPUProfile(f); err != nil {
				log.Warningf("error starting cpuprofile: %w", err)
			} else {
				defer pprof.StopCPUProfile()
			}
		}
	}

	ls.Server.RunStdio()
}
