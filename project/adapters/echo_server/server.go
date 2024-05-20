package echo_server

import (
	"log/slog"
	"net/http"
	"tickets/adapters/echo_server/ticket"

	commonHTTP "github.com/ThreeDotsLabs/go-event-driven/common/http"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/labstack/echo/v4"
)

type Server struct {
	port   string
	router *echo.Echo
	logger *slog.Logger
}

func NewServer(port string, pub message.Publisher, logger *slog.Logger) *Server {
	router := commonHTTP.NewEcho()
	ticketController := ticket.NewController(pub)
	router.POST("/tickets-status", ticketController.ProcessTicket)
	router.GET("/health", ticketController.Health)

	return &Server{
		port:   port,
		logger: logger,
		router: router,
	}
}

func (s *Server) Start() error {
	err := s.router.Start(":8080")
	if err != nil && err != http.ErrServerClosed {
		s.logger.Error("Server Stopped")
	}
	return err
}

func (s *Server) Stop() error {
	return nil
}
