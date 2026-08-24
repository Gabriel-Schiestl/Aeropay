package server

import (
	"net/http/pprof"

	"github.com/Gabriel-Schiestl/Aeropay/payment-service/internal/observability"
	"github.com/gin-gonic/gin"
)

type Server struct {
	Instance *gin.Engine
}

func NewServer() *Server {
	engine := gin.Default()
	engine.Use(observability.GinMiddleware())

	engine.GET("/metrics", observability.Handler())
	registerPprof(engine)

	return &Server{
		Instance: engine,
	}
}

// go tool pprof http://localhost:8080/debug/pprof/profile.
func registerPprof(engine *gin.Engine) {
	debug := engine.Group("/debug/pprof")
	debug.GET("/", gin.WrapF(pprof.Index))
	debug.GET("/cmdline", gin.WrapF(pprof.Cmdline))
	debug.GET("/profile", gin.WrapF(pprof.Profile))
	debug.GET("/symbol", gin.WrapF(pprof.Symbol))
	debug.POST("/symbol", gin.WrapF(pprof.Symbol))
	debug.GET("/trace", gin.WrapF(pprof.Trace))
	debug.GET("/allocs", gin.WrapH(pprof.Handler("allocs")))
	debug.GET("/block", gin.WrapH(pprof.Handler("block")))
	debug.GET("/goroutine", gin.WrapH(pprof.Handler("goroutine")))
	debug.GET("/heap", gin.WrapH(pprof.Handler("heap")))
	debug.GET("/mutex", gin.WrapH(pprof.Handler("mutex")))
	debug.GET("/threadcreate", gin.WrapH(pprof.Handler("threadcreate")))
}

func (s *Server) Start(port string) error {
	return s.Instance.Run(":" + port)
}