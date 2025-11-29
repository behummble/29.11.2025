package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/behummble/29-11-2025/internal/config"
	"github.com/behummble/29-11-2025/internal/models"
)

type Server struct {
	log *slog.Logger
	server *http.Server
	service Service
	isClosed atomic.Bool
}

type Service interface {
	VerifyLinks(ctx context.Context, data []byte) (models.VerifyLinksResponse, error)
	PackageLinks(ctx context.Context, data []byte) ([]byte, error)
}

func NewServer(ctx context.Context, log *slog.Logger, cfg config.ServerConfig, service Service) *Server {
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

func(s *Server) Shutdown(ctx context.Context) {
	s.isClosed.CompareAndSwap(true, false)
}

func(s *Server) Restart() {
	s.isClosed.CompareAndSwap(false, true)
}

func(s *Server) GetHandler() http.Handler {
	return s.server.Handler
}

func(s *Server) VerifyLinks(writer http.ResponseWriter, request *http.Request) {
	if s.isClosed.Load() {
		writer.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	ctx, cancel := context.WithTimeout(request.Context(), 30 * time.Second)
	defer cancel()
	data, err := executeRequestBody(request, s.log)
	if err != nil {
		writer.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(writer, err.Error())
		return
	}

	s.log.Info("Recive request to create question")

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
	writer.WriteHeader(http.StatusCreated)
	writer.Write(bytes)
}

func(s *Server) LinksReport(writer http.ResponseWriter, request *http.Request) {
	if s.isClosed.Load() {
		writer.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	ctx, cancel := context.WithTimeout(request.Context(), 30 * time.Second)
	defer cancel()
	data, err := executeRequestBody(request, s.log)
	if err != nil {
		writer.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(writer, err.Error())
		return
	}

	s.log.Info("Recive request to create question")

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
	bytes := prepareResponse(res, s.log)
	writer.WriteHeader(http.StatusOK)
	writer.Header().Add("Content-type", "application/pdf")
	writer.Write(bytes)
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

func getID(r *http.Request) (int, error) {
	idStr := r.PathValue("id")
	if idStr == "" {
		return 0, errors.New("ParameterNotFound")
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return 0, err
	}

	return id, nil
}