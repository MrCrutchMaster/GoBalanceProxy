package balancer

import (
	"GoBalanceProxy/pkg/config"
	"GoBalanceProxy/pkg/metrics"
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
	"strings"
	"time"
)

type Balancer struct {
	Server    *http.Server
	ctx       context.Context
	Stopper   context.CancelFunc
	logger    *zerolog.Logger
	limitChan chan struct{}

	httpClient      *http.Client
	proxyConf       *config.BalancerConf
	activeEndpoints *[]string
}

const (
	urlBalancerStatus = "/balancer/status"
)

var (
	ErrSelectEndpoint = errors.New("can't select destination server, no active endpoints available")
	ErrRequestLimit   = errors.New("balancer request limit reached")
)

func giveErrorResp(err error, statusCode int, w http.ResponseWriter) {
	w.WriteHeader(statusCode)
	_, err = w.Write([]byte(fmt.Sprintf("error: %s", err)))
}
func (b *Balancer) selectDestServer() (string, error) {
	l := len(*b.activeEndpoints)
	if l == 0 {
		return "", fmt.Errorf("selectDestServer: %w", ErrSelectEndpoint)
	}
	ind := rand.Intn(l)
	srv := (*b.activeEndpoints)[ind]
	return srv, nil

}
func (b *Balancer) sendRequest(origReq *http.Request) (*http.Response, error) {
	endpoint, err := b.selectDestServer()
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

	requestPath := fmt.Sprintf("%s%s", endpoint, uri)
	b.logger.Info().
		Str("host", host).
		Str("method", method).
		Str("endpoint", endpoint).
		Str("uri", uri).
		Str("queryString", fmt.Sprintf("%b", queryString)).
		Msgf("sendRequest: request info")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	proxyRequest, err := http.NewRequestWithContext(ctx, method, requestPath, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	proxyRequest.Header = header
	proxyRequest.Host = host
	resp, err := b.httpClient.Do(proxyRequest)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (b *Balancer) freeChannel() {
	_ = <-b.limitChan
}
func (b *Balancer) baseHandler(w http.ResponseWriter, r *http.Request) {
	reqNum := len(b.limitChan)
	b.limitChan <- struct{}{}
	defer b.freeChannel()
	if reqNum >= b.proxyConf.MaxConn {
		giveErrorResp(ErrRequestLimit, http.StatusTooManyRequests, w)
		metrics.RequestLimit.Inc()
		return
	}
	resp, err := b.sendRequest(r)
	if err != nil {
		b.logger.Error().Err(err)
		giveErrorResp(err, http.StatusBadGateway, w)
		metrics.RequestFail.Inc()
		return
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		b.logger.Error().Err(err)
		giveErrorResp(err, http.StatusBadGateway, w)
		metrics.RequestFail.Inc()
		return
	}
	for hdr, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(hdr, value)
		}
	}
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(body)
	if err != nil {
		b.logger.Error().Err(err)
		giveErrorResp(err, http.StatusBadGateway, w)
		metrics.RequestFail.Inc()
		return
	}
	metrics.RequestOk.Inc()
}

func (b *Balancer) StartHTTPServer() {
	probe := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(strings.Join(*b.activeEndpoints, ",")))
	}

	mux := http.NewServeMux()
	b.Server = &http.Server{
		Addr:         b.proxyConf.ListenAddr,
		ReadTimeout:  b.proxyConf.ReadTimeout,
		WriteTimeout: b.proxyConf.WriteTimeout,
		Handler:      mux,
	}

	mux.HandleFunc(urlBalancerStatus, probe)
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/", b.baseHandler)

	err := b.Server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		b.logger.Fatal().Err(err).Msg("ListenAndServe error")
	}
}

func NewBalancer(
	ctx context.Context,
	proxyConf *config.BalancerConf,
	activeEndpoints *[]string,
) *Balancer {
	logger := log.With().Str("me", "Balancer").Logger()
	logger.Info().Msgf("Balancer addr: %s", proxyConf.ListenAddr)
	httpClient := &http.Client{
		Timeout: 6 * time.Second, //!TODO
		Transport: &http.Transport{
			MaxConnsPerHost: 256,
			MaxIdleConns:    10,
			IdleConnTimeout: time.Second * 15,
		}}
	limitChan := make(chan struct{}, proxyConf.MaxConn)
	Srv := &Balancer{
		ctx:             ctx,
		logger:          &logger,
		limitChan:       limitChan,
		proxyConf:       proxyConf,
		activeEndpoints: activeEndpoints,
		httpClient:      httpClient,
	}

	return Srv
}
