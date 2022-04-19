package utils

import (
	"context"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Config struct {
	Port              int
	GracefullShutdown chan os.Signal
	UserHeaderName    string
	SaltApiUrl        *url.URL
	ReverseProxy      bool
}

func (c *Config) GetListeningAddr() string {
	return fmt.Sprintf(":%d", c.Port)
}

func (c *Config) GetInternalURL() string {
	return fmt.Sprintf("http://127.0.0.1:%d/api/v1", c.Port)
}

func GetConfig() *Config {
	// Enable VIPER to read Environment Variables
	viper.AutomaticEnv() // To get the value from the config file using key// viper package read .env

	config := Config{
		Port:              8080,
		GracefullShutdown: make(chan os.Signal, 1),
		UserHeaderName:    "X-Forwarded-User",
		SaltApiUrl: &url.URL{
			Host:   "127.0.0.1:8000",
			Scheme: "http",
		},
		ReverseProxy: false,
	}

	if p := viper.GetInt("PORT"); p != 0 {
		config.Port = p
	}

	if n := viper.GetString("USER_HEADER_NAME"); n != "" {
		config.UserHeaderName = n
	}

	if urlstr := viper.GetString("SALT_API_URL"); urlstr != "" {
		saltApiUrl, err := url.Parse(urlstr)
		if err != nil {
			log.Fatal().Err(err).Msg("SALT_API_URL is invalid")
		}
		config.SaltApiUrl = saltApiUrl
	}

	config.ReverseProxy = viper.GetBool("REVERSE_PROXY")

	log.Info().Msgf("listen port %d", config.Port)
	log.Info().Msgf("User header is %s", config.UserHeaderName)
	log.Info().Msgf("SaltAPI is %s", config.SaltApiUrl)
	log.Info().Msgf("ReverseProxy is %t", config.ReverseProxy)
	return &config
}

func (c *Config) WaitForInterruptSignal(srv *http.Server) {
	// kill (no param) default send syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can't be catch, so don't need add it
	signal.Notify(c.GracefullShutdown, syscall.SIGINT, syscall.SIGTERM)
	// wait for signal
	<-c.GracefullShutdown
	log.Info().Msg("Interrupt signal received, shutting down...")

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()
	if srv != nil {
		if err := srv.Shutdown(ctx); err != nil {
			log.Fatal().Err(err).Msg("server forced to shutdown")
		}
	}
	log.Info().Msg("server exiting")
}
