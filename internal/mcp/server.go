package mcp

type Server struct{}

func NewServer() *Server {
	return &Server{}
}

func (s *Server) Start() string {
	return "pong"
}
