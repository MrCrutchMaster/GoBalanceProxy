package server

import (
	"GoBalanceProxy/pkg/config"
	"bytes"
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"io"
	"math/rand"
	"net/http"
	"time"
)

type Srv struct {
	Srv     *http.Server
	ctx     context.Context
	Stopper context.CancelFunc
	logger  *zerolog.Logger

	httpConf  *config.HTTPServerConf
	proxyConf []*config.ProxyConf
}

var (
	ErrSomeProblem = errors.New("SomeProblem")
)

func (s *Srv) baseHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	_, err := w.Write([]byte("Ok"))
	if err != nil {
		fmt.Println(err)
	}
	dstSrv := s.proxyConf[rand.Intn(len(s.proxyConf))]

	host := r.Host
	header := r.Header
	method := r.Method
	uri := r.RequestURI
	queryString := r.URL.Query()
	body, err := io.ReadAll(r.Body)

	endpoint := fmt.Sprintf("%s%s", dstSrv.Server, uri)
	fmt.Println(dstSrv)
	fmt.Println(endpoint)
	fmt.Println(host, method, uri, queryString)

	httpCli := &http.Client{
		Timeout: 1 * time.Second,
		Transport: &http.Transport{
			MaxConnsPerHost: 256,
			MaxIdleConns:    10,
			IdleConnTimeout: time.Second * 15,
		}}
	fmt.Println(httpCli)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	proxyRequest, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(body))
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(proxyRequest)
	proxyRequest.Header = header
	proxyRequest.Host = host
	resp, err := httpCli.Do(proxyRequest)
	if err != nil {
		fmt.Println(err)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(string(b))
}

func (s *Srv) startHTTPServer() {
	probe := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("OK"))
	}

	http.Handle("/metrics/", promhttp.Handler())
	http.HandleFunc("/probe/readness", probe)
	http.HandleFunc("/probe/liveness", probe)

	http.HandleFunc("/", s.baseHandler)

	s.logger.Info().Msgf("Srv addr: %s", s.httpConf.ListenAddr)
	s.Srv = &http.Server{
		Addr:         s.httpConf.ListenAddr,
		ReadTimeout:  s.httpConf.ReadTimeout,
		WriteTimeout: s.httpConf.WriteTimeout,
	}

	err := s.Srv.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		s.logger.Fatal().Err(err).Msg("ListenAndServe error")
	}
}

func (s *Srv) WaitDone() {
	<-s.ctx.Done()
	s.logger.Info().Msg("WaitDone: Server stopped")
}

func (s *Srv) Run() {
	s.logger.Info().Msg("Run: Start http server application")
	go s.startHTTPServer()
	s.WaitDone()
	s.logger.Info().Msg("Run: Stop http server application")
}

func NewHTTPServer(
	httpConf *config.HTTPServerConf,
	proxyConf []*config.ProxyConf,
) *Srv {
	ctx, cancel := context.WithCancel(context.Background())
	logger := log.With().Str("me", "HttpServer").Logger()
	Srv := &Srv{
		ctx:       ctx,
		Stopper:   cancel,
		logger:    &logger,
		httpConf:  httpConf,
		proxyConf: proxyConf,
	}

	return Srv
}
