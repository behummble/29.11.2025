package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/behummble/29-11-2025/internal/config"
	"github.com/behummble/29-11-2025/internal/models"
)

type Server struct {
	log *slog.Logger
	server *http.Server
	service Service
}

type Service interface {
	VerifyLinks(ctx context.Context, data []byte) (models.VerifyLinksResponse, error)
	PackageLinks(ctx context.Context, data []byte) ([]byte, error)
}

func NewServer(log *slog.Logger, cfg config.ServerConfig, service Service) *Server {
	server := &Server{
		log: log,
		service: service,
	}
	srv := &http.Server{
		Addr: fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
	}
	mux := newMux(server)
	srv.Handler = mux
	server.server = srv
	
	return server
}

func(s *Server) Start() {
	err := s.server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
        panic(err)
    }
}

func(s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func(s *Server) GetHandler() http.Handler {
	return s.server.Handler
}

func(s *Server) VerifyLinks(writer http.ResponseWriter, request *http.Request) {
	ctx, cancel := context.WithTimeout(request.Context(), 30 * time.Second)
	defer cancel()
	data, err := executeRequestBody(request, s.log)
	if err != nil {
		writer.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(writer, err.Error())
		return
	}

	s.log.Info("Recive request cerify links")

	if len(data) == 0 {
		writer.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(writer, "Empty body")
		return
	}

	res, err := s.service.VerifyLinks(ctx, data)
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(writer, err.Error())
		return
	}
	bytes := prepareResponse(res, s.log)
	writer.WriteHeader(http.StatusOK)
	writer.Write(bytes)
}

func(s *Server) LinksReport(writer http.ResponseWriter, request *http.Request) {
	ctx, cancel := context.WithTimeout(request.Context(), 30 * time.Second)
	defer cancel()
	data, err := executeRequestBody(request, s.log)
	if err != nil {
		writer.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(writer, err.Error())
		return
	}

	s.log.Info("Recive request to get link package report")

	if len(data) == 0 {
		writer.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(writer, "Empty body")
		return
	}
	res, err := s.service.PackageLinks(ctx, data)
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(writer, err.Error())
		return
	}
	s.log.Info("Recive request to get all question")
	writer.Header().Set("Content-Type", "application/pdf")
	writer.WriteHeader(http.StatusOK)	
	writer.Write(res)
}

func newMux(s *Server) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /links", s.VerifyLinks)
	mux.HandleFunc("POST /links/list", s.LinksReport)
	
	return mux
}

func executeRequestBody(request *http.Request, log *slog.Logger) ([]byte, error) {
	if request.Body == nil {
		log.Error(
			"ExecutionBodyError", 
			slog.String("component", "io/Read"),
			slog.Any("error", "Empty body"),
		)
		return nil, errors.New("ExecutionBodyError")
	}
	data, err := io.ReadAll(request.Body)
	if err != nil {
		log.Error(
			"ReadingRequestBodyError", 
			slog.String("component", "io/Read"),
			slog.Any("error", err),
		)
		return nil, errors.New("ReadingRequestBodyError")
	}

	return data, nil
}

func prepareResponse[T any](m T, log *slog.Logger) []byte {
	res, err := json.Marshal(m)
	if err != nil {
		log.Error(
			"MarshalingJSONError", 
			slog.String("component", "json/marshalling"),
			slog.Any("error", err),
		)
	}

	return res
}