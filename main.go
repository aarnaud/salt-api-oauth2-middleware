package main

import (
	"encoding/json"
	"fmt"
	"github.com/aarnaud/salt-api-oauth2-middleware/utils"
	"github.com/aarnaud/salt-api-oauth2-middleware/utils/helpers"
	"github.com/gin-contrib/logger"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"syscall"
)

type LoginData struct {
	Username string `json:"username" form:"username"`
	Password string `json:"password" form:"password"`
	Eauth    string `json:"eauth" form:"eauth"`
}
type Challenge struct {
	Data  map[string]string
	Mutex sync.Mutex
}

func (c *Challenge) NewChallenge(email string) string {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	c.Data[email] = helpers.RandomString(64)
	return c.Data[email]
}

func (c *Challenge) CheckChallenge(email string, value string) bool {
	c.Mutex.Lock()
	defer func() {
		delete(c.Data, email)
		c.Mutex.Unlock()
	}()
	return c.Data[email] == value
}

func NewChallengeDB() *Challenge {
	return &Challenge{
		Data:  make(map[string]string),
		Mutex: sync.Mutex{},
	}
}

func main() {
	log.Info().Msg("initializing...")
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	appconf := utils.GetConfig()
	router := gin.New()
	router.Use(logger.SetLogger(logger.WithSkipPath([]string{"/healthz"})))
	router.Use(gin.Recovery())
	cdb := NewChallengeDB()

	if gin.Mode() == gin.DebugMode {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	router.GET("/healthz", func(c *gin.Context) {
		cdb.Mutex.Lock()
		defer cdb.Mutex.Unlock()
		c.JSON(http.StatusOK, gin.H{})
	})

	router.POST("/login", func(c *gin.Context) {
		email := c.Request.Header.Get(appconf.UserHeaderName)
		if email != "" {
			data := LoginData{
				Username: email,
				Password: cdb.NewChallenge(email),
				Eauth:    "rest",
			}

			jsonstring, err := json.Marshal(data)
			if err != nil {
				log.Err(err).Msg("failed to marshall data")
				c.AbortWithStatus(http.StatusInternalServerError)
			}
			c.Request.ContentLength = int64(len(jsonstring))
			c.Request.Header.Set("Content-Length", fmt.Sprintf("%d", c.Request.ContentLength))
			c.Request.Body = io.NopCloser(strings.NewReader(string(jsonstring)))
			ReverseProxy(appconf.SaltApiUrl)(c)
			return
		}
		// allow other auth method in salt-api if no header
		if appconf.ReverseProxy {
			ReverseProxy(appconf.SaltApiUrl)(c)
		}
	})

	router.POST("/_callback", func(c *gin.Context) {
		var challengeData LoginData
		err := c.Bind(&challengeData)
		if err != nil {
			log.Err(err).Msg("failed to parse data challenge")
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		if cdb.CheckChallenge(challengeData.Username, challengeData.Password) {
			// return at least one acl to allow connexion
			c.JSON(http.StatusOK, []string{
				"test.ping",
			})
			return
		}
		c.JSON(http.StatusForbidden, gin.H{})
	})

	if appconf.ReverseProxy {
		// send all other request to salt-api as proxy
		router.NoRoute(ReverseProxy(appconf.SaltApiUrl))
	}

	srv := &http.Server{
		Addr:    appconf.GetListeningAddr(),
		Handler: router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			// if shutdown, ignore error
			if err == http.ErrServerClosed {
				return
			}
			log.Error().Err(err).Msgf("failed to listen on %s", appconf.GetListeningAddr())
			appconf.GracefullShutdown <- syscall.SIGTERM
		}
	}()

	appconf.WaitForInterruptSignal(srv)
}

func ReverseProxy(target *url.URL) gin.HandlerFunc {
	return func(c *gin.Context) {
		director := func(req *http.Request) {
			req.URL = c.Request.URL
			req.URL.Host = target.Host
			req.URL.Scheme = target.Scheme
			req.Body = c.Request.Body
		}
		proxy := &httputil.ReverseProxy{Director: director}
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}
