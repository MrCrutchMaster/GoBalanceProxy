package checker

import (
	"GoBalanceProxy/pkg/config"
	"GoBalanceProxy/pkg/endpoints"
	"bytes"
	"context"
	"fmt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"net/http"
	"time"
)

type Checker struct {
	ctx             context.Context //nolint
	logger          *zerolog.Logger
	endpointsConf   []*config.EndpointsConf
	checkerConf     *config.CheckerConf
	httpClient      *http.Client
	activeEndpoints *endpoints.ActiveEndpoints
	checkerDoneChan chan struct{}
}

func NewChecker(
	ctx context.Context,
	endpointsConf []*config.EndpointsConf,
	checkerConf *config.CheckerConf,
	activeEndpoints *endpoints.ActiveEndpoints,
	checkerDoneChan chan struct{},
) *Checker {
	logger := log.With().Str("me", "ProbeChecker").Logger()
	c := &Checker{
		ctx:             ctx,
		logger:          &logger,
		endpointsConf:   endpointsConf,
		checkerConf:     checkerConf,
		activeEndpoints: activeEndpoints,
		checkerDoneChan: checkerDoneChan,
	}

	return c
}

func (c *Checker) checkEndpointHealth() {
	ctx, cancel := context.WithTimeout(c.ctx, c.checkerConf.TotalCheckTimeout)
	defer cancel()
	var activeEndpoints []endpoints.Endpoint
	for _, val := range c.endpointsConf {
		endpoint := fmt.Sprintf("%s%s", val.Server, val.Probe)
		checkRequest, err := http.NewRequestWithContext(ctx, "GET", endpoint, bytes.NewReader([]byte("")))
		if err != nil {
			c.logger.Error().
				Str("endpoint", endpoint).
				Str("description", "Request create failed").
				Err(err)
			continue
		}
		start := time.Now()
		resp, err := c.httpClient.Do(checkRequest)
		respTime := time.Now().Sub(start)
		if err != nil {
			c.logger.Error().
				Str("endpoint", endpoint).
				Str("description", "Request exec failed").
				Err(err)
			continue
		}
		//c.logger.Debug().Msgf("checkEndpointHealth: result %s %d\n", endpoint, resp.StatusCode)
		if resp.StatusCode != 200 {
			c.logger.Error().
				Str("endpoint", endpoint).
				Int("status_code", resp.StatusCode).
				Str("description", "Resp bad status").
				Err(err)
			continue
		}
		endpointStruct := endpoints.Endpoint{
			Server:   val.Server,
			RespTime: respTime,
		}
		activeEndpoints = append(activeEndpoints, endpointStruct)
	}
	c.activeEndpoints.WriteEndpoints(activeEndpoints)
	c.logger.Debug().Msgf("checkEndpointHealth: %s", activeEndpoints)
}

func (c *Checker) StartHealthChecker() {
	c.logger.Info().Str("Conf", fmt.Sprintf("%s", c.checkerConf)).Msg("HealthChecker: start")
	c.httpClient = &http.Client{
		Timeout: c.checkerConf.ProbeTimeout, //!TODO
		Transport: &http.Transport{
			MaxConnsPerHost: 256,
			MaxIdleConns:    10,
			IdleConnTimeout: time.Second * 15,
		}}
	for {
		select {
		case <-c.ctx.Done():
			c.logger.Info().Msg("HealthChecker: catch stop signal")
			c.checkerDoneChan <- struct{}{}
			c.logger.Info().Msg("HealthChecker: stopped")
			return
		default:
			c.checkEndpointHealth()
			time.Sleep(c.checkerConf.RecheckPeriod)
		}

	}
}
