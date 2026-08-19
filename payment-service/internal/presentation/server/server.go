package server

import "github.com/gin-gonic/gin"

type Server struct {
	Instance *gin.Engine
}

func NewServer() *Server {
	return &Server{
		Instance: gin.Default(),
	}
}

func (s *Server) Start(port string) error {
	return s.Instance.Run(":" + port)
}