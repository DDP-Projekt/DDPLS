package main

import (
	"github.com/DDP-Projekt/DDPLS/ddpls"
	"github.com/tliron/kutil/logging"

	// Must include a backend implementation. See kutil's logging/ for other options.
	_ "github.com/tliron/kutil/logging/simple"
)

func main() {
	// This increases logging verbosity (optional)
	logging.Configure(1, nil)

	ddpls := ddpls.NewDDPLS()
	ddpls.Server.RunStdio()
}
