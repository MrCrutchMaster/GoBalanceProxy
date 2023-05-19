package main

import (
	"GoBalanceProxy/pkg/config"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/rs/zerolog/log"
	_ "go.uber.org/automaxprocs"
)

const appName = "GoBalanceProxy"

// Build Injected with  ldflags -X
var version = "0.0.0"

func main() {
	rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
	conf, err := config.GetConfig()
	if err != nil {
		fmt.Printf("Config load error: %v\n", err)
		os.Exit(1)
	}

	//configurePrometheus()  // !!!TODO
	configureLogger(&conf)
	log.Info().Str("version", version).Msg("Starting GoBalanceProxy")

	goBalanceProxy := NewGoBalanceProxy(&conf)
	handleSignals(goBalanceProxy.server)
	goBalanceProxy.server.Run()
}
