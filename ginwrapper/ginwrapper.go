package ginwrapper

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-contrib/gzip"
	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"github.com/sgostarter/i/l"
)

type GinHTTPServerConfig struct {
	Debug   bool   `json:"debug" yaml:"debug"`
	Listens string `json:"listens" yaml:"listens"`
}

type FnRegisterRoutes func(r *gin.RouterGroup)

func RunGinHTTPServer(cfg GinHTTPServerConfig, register FnRegisterRoutes,
	logger l.Wrapper, middlewares ...gin.HandlerFunc) {
	if logger == nil {
		logger = l.NewConsoleLoggerWrapper()
	}

	if cfg.Listens == "" || register == nil {
		logger.Fatal("no listenList or handler")
	}

	logger = logger.WithFields(l.StringField(l.RoutineKey, "RunHTTPServer"))

	logger.Debug("enter")

	defer logger.Debug("leave")

	if cfg.Debug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gzip.Gzip(gzip.DefaultCompression))
	r.Use(gin.Recovery())
	r.Use(requestid.New())

	for _, middleware := range middlewares {
		r.Use(middleware)
	}

	r.Any("/healthy", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	register(&r.RouterGroup)

	fnListen := func(listen string) {
		srv := &http.Server{
			Addr:        listen,
			ReadTimeout: time.Second,
			Handler:     r,
		}

		logger.WithFields(l.StringField("listen", listen)).Debug("start listen")

		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.WithFields(l.ErrorField(err), l.StringField("listen", listen)).Error("listen failed")
		}
	}

	listens := strings.Split(cfg.Listens, " ")

	for idx := 0; idx < len(listens)-1; idx++ {
		go fnListen(listens[idx])
	}

	fnListen(listens[len(listens)-1])
}

func JSONMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Content-Type", "application/json")
		c.Next()
	}
}
