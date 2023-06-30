package main

import (
	"GoBalanceProxy/pkg/config"
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"time"

	"github.com/rs/zerolog/log"
	_ "go.uber.org/automaxprocs"
)

const appName = "App"

// Build Injected with  ldflags -X
var version = "0.0.0"

func MaxParallelism() int {
	maxProcs := runtime.GOMAXPROCS(0)
	numCPU := runtime.NumCPU()
	if maxProcs < numCPU {
		return maxProcs
	}
	return numCPU
}

func main() {

	rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
	conf, err := config.GetConfig()
	if err != nil {
		fmt.Printf("Config load error: %v\n", err)
		os.Exit(1)
	}

	//configurePrometheus()  // !!!TODO
	configureLogger(&conf)

	log.Info().Str("version", version).Msg("Starting App")
	log.Info().Msgf("MaxParallelism: %d", MaxParallelism())

	app := NewApp(&conf)
	handleSignals(app)
	app.Run()
}
