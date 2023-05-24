package main

import (
	"GoBalanceProxy/pkg/config"
	"GoBalanceProxy/pkg/server"
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

	hook := zerolog.NewLevelHook()
	//hook.ErrorHook = zerolog.HookFunc(func(e *zerolog.Event, level zerolog.Level, message string) {
	//	stats.Increment("log_errors")
	//})
	log.Logger = zerolog.New(os.Stdout).Hook(hook).With().Timestamp().Logger()
	zerolog.SetGlobalLevel(lvl)
}

type GoBalanceProxy struct {
	conf   *config.Config
	server *server.Srv
}

func NewGoBalanceProxy(conf *config.Config) *GoBalanceProxy {
	log.Info().Msg("GoBalanceProxy init: started")

	// http server init
	srv := server.NewHTTPServer(conf.BalanceProxy, conf.DestServer)
	log.Info().Msg("GoBalanceProxy init: finished")
	return &GoBalanceProxy{
		conf:   conf,
		server: srv,
	}
}

func Finalize(s *server.Srv) {
	s.Stopper()
	go time.AfterFunc(time.Second*30, func() {
		log.Fatal().Msg("force exit after deadline")
	})
	err := s.Srv.Shutdown(context.Background())
	if err != nil {
		log.Fatal().Err(err).Msg("BalanceProxy server Shutdown")
	}
}

func handleSignals(srv *server.Srv) {
	signalChannel := make(chan os.Signal, 1)

	go func() {
		sig := <-signalChannel
		log.Warn().Msgf("catch interrupt signal: %v", sig)
		log.Warn().Msg("Finalizing application")
		Finalize(srv)
	}()

	signal.Notify(signalChannel, syscall.SIGINT, syscall.SIGTERM)
}
