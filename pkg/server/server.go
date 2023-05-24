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

	httpClient       *http.Client
	balanceProxyConf *config.BalanceProxyConf
	destSrvConf      []*config.DestServerConf
	destSrvCount     int
}

var (
	ErrSomeProblem   = errors.New("SomeProblem")
	ErrSelectDestSrv = errors.New("can't select destination server")
)

func (s *Srv) checkDestServerHealth(destSrv *config.DestServerConf) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	endpoint := fmt.Sprintf("%s%s", destSrv.Server, destSrv.Probe)
	checkRequest, err := http.NewRequestWithContext(ctx, "GET", endpoint, bytes.NewReader([]byte("")))
	if err != nil {
		return false, err
	}
	resp, err := s.httpClient.Do(checkRequest)
	if err != nil {
		return false, err
	}
	fmt.Println("checkDestServerHealth: err", err)
	fmt.Println("checkDestServerHealth: resp", resp)
	fmt.Println("checkDestServerHealth: StatusCode", resp.StatusCode)
	if resp.StatusCode == 200 {
		return true, nil
	}
	return false, nil
}
func (s *Srv) selectDestServer() (*config.DestServerConf, error) {
	for _, _ = range s.destSrvConf {
		srv := s.destSrvConf[rand.Intn(s.destSrvCount)]
		isOk, err := s.checkDestServerHealth(srv)
		if isOk && err == nil {
			return srv, nil
		}
	}

	return nil, fmt.Errorf("selectDestServer: %w", ErrSelectDestSrv)
}

func (s *Srv) sendRequest(origReq *http.Request) (*http.Response, error) {
	dstSrv, err := s.selectDestServer()
	if err != nil {
		return nil, err
	}
	host := origReq.Host
	header := origReq.Header
	method := origReq.Method
	uri := origReq.RequestURI
	queryString := origReq.URL.Query()
	body, err := io.ReadAll(origReq.Body)
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("%s%s", dstSrv.Server, uri)
	fmt.Println(dstSrv)
	fmt.Println(endpoint)
	fmt.Println(host, method, uri, queryString)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	proxyRequest, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	proxyRequest.Header = header
	proxyRequest.Host = host
	resp, err := s.httpClient.Do(proxyRequest)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
func (s *Srv) baseHandler(w http.ResponseWriter, r *http.Request) {
	resp, err := s.sendRequest(r)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(200)
		_, err = w.Write([]byte(fmt.Sprintf("error: %s", err)))
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("resp headers")
	for hdr, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(hdr, value)
		}
	}

	w.WriteHeader(200)
	_, err = w.Write(body)
	if err != nil {
		fmt.Println(err)
	}

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

	s.logger.Info().Msgf("Srv addr: %s", s.balanceProxyConf.ListenAddr)

	s.httpClient = &http.Client{
		Timeout: 1 * time.Second, //!TODO
		Transport: &http.Transport{
			MaxConnsPerHost: 256,
			MaxIdleConns:    10,
			IdleConnTimeout: time.Second * 15,
		}}

	s.Srv = &http.Server{
		Addr:         s.balanceProxyConf.ListenAddr,
		ReadTimeout:  s.balanceProxyConf.ReadTimeout,
		WriteTimeout: s.balanceProxyConf.WriteTimeout,
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
	balanceProxyConf *config.BalanceProxyConf,
	destSrvConf []*config.DestServerConf,
) *Srv {
	ctx, cancel := context.WithCancel(context.Background())
	logger := log.With().Str("me", "HttpServer").Logger()
	Srv := &Srv{
		ctx:              ctx,
		Stopper:          cancel,
		logger:           &logger,
		balanceProxyConf: balanceProxyConf,
		destSrvConf:      destSrvConf,
		destSrvCount:     len(destSrvConf),
	}

	return Srv
}
