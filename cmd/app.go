package main

import (
	"GoBalanceProxy/pkg/balancer"
	"GoBalanceProxy/pkg/checker"
	"GoBalanceProxy/pkg/config"
	"GoBalanceProxy/pkg/endpoints"
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func configureLogger(conf *config.Config) {
	zerolog.TimeFieldFormat = "2006-01-02 15:04:05.000"
	lvl := zerolog.InfoLevel
	if conf.Debug {
		lvl = zerolog.DebugLevel
	}
	log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
	zerolog.SetGlobalLevel(lvl)
}

type App struct {
	ctx             context.Context //nolint
	Stopper         context.CancelFunc
	conf            *config.Config
	proxy           *balancer.Balancer
	checker         *checker.Checker
	proxyDoneChan   chan struct{}
	checkerDoneChan chan struct{}
}

func (app *App) WaitDone() {
	<-app.ctx.Done()
	log.Info().Msg("WaitDone: catch done signal")
	//<-app.proxyDoneChan
	<-app.checkerDoneChan
	log.Info().Msg("WaitDone: checker stopped")
}

func (app *App) Run() {
	//app.logger.Info().Msg("Run: Start http proxy application")
	go app.proxy.StartHTTPServer()
	go app.checker.StartHealthChecker()
	app.WaitDone()
	//app.logger.Info().Msg("Run: Stop http proxy application")
}

func NewApp(conf *config.Config) *App {
	log.Info().Msg("App init: started")

	ctx, cancel := context.WithCancel(context.Background())
	activeEndpoints := endpoints.ActiveEndpoints{}

	// http proxy init
	balanceProxy := balancer.NewBalancer(ctx, conf.Balancer, &activeEndpoints)

	// probe checker init
	checkerDoneChan := make(chan struct{})
	probeChecker := checker.NewChecker(ctx, conf.Endpoints, conf.Checker, &activeEndpoints, checkerDoneChan)

	log.Info().Msg("App init: finished")
	return &App{
		ctx:             ctx,
		Stopper:         cancel,
		conf:            conf,
		proxy:           balanceProxy,
		checker:         probeChecker,
		checkerDoneChan: checkerDoneChan,
	}
}

func Finalize(app *App) {
	app.Stopper()
	go time.AfterFunc(time.Second*30, func() {
		log.Fatal().Msg("force exit after deadline")
	})
	err := app.proxy.Server.Shutdown(context.Background())
	if err != nil {
		log.Fatal().Err(err).Msg("Balancer shutdown")
	}
}

func handleSignals(app *App) {
	signalChannel := make(chan os.Signal, 1)

	go func() {
		sig := <-signalChannel
		log.Warn().Msgf("catch interrupt signal: %v", sig)
		log.Warn().Msg("Finalizing application")
		Finalize(app)
	}()

	signal.Notify(signalChannel, syscall.SIGINT, syscall.SIGTERM)
}
