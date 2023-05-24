package checker

import (
	"GoBalanceProxy/pkg/config"
	"bytes"
	"context"
	"fmt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"net/http"
	"strings"
	"time"
)

type Checker struct {
	ctx             context.Context //nolint
	logger          *zerolog.Logger
	endpointsConf   []*config.EndpointsConf
	checkerConf     *config.CheckerConf
	httpClient      *http.Client
	activeEndpoints *[]string
	checkerDoneChan chan struct{}
}

func NewChecker(
	ctx context.Context,
	endpointsConf []*config.EndpointsConf,
	checkerConf *config.CheckerConf,
	activeEndpoints *[]string,
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
	activeEndpoints := []string{}
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
		resp, err := c.httpClient.Do(checkRequest)
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
		activeEndpoints = append(activeEndpoints, val.Server)
	}
	*c.activeEndpoints = activeEndpoints
	c.logger.Debug().Msgf("checkEndpointHealth: %s", strings.Join(*c.activeEndpoints, ","))
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
