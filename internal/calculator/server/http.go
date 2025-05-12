package server

import (
	"context"

	"github.com/belo4ya/edu-final-calculate-api/api"
	"github.com/belo4ya/edu-final-calculate-api/internal/calculator/config"

	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	httpSwagger "github.com/swaggo/http-swagger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

type HTTPServer struct {
	HTTP  *http.Server
	GWMux *runtime.ServeMux
	conf  *config.Config
}

func NewHTTPServer(conf *config.Config) *HTTPServer {
	s := &HTTPServer{conf: conf}

	mux := http.NewServeMux()
	s.setupDocsRoutes(mux)

	gwmux := runtime.NewServeMux(
		runtime.WithForwardResponseOption(s.grpcGatewayResponseModifier),
		runtime.WithErrorHandler(s.grpcGatewayErrorHandler),
	)
	mux.Handle("/", gwmux)

	s.GWMux = gwmux
	s.HTTP = &http.Server{
		Addr:    conf.HTTPAddr,
		Handler: mux,
	}
	return s
}

func (s *HTTPServer) Start(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		slog.InfoContext(ctx, "http server start listening on "+s.conf.HTTPAddr)
		if err := s.HTTP.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("start http server: %w", err)
		}
		close(errCh)
	}()
	select {
	case <-ctx.Done():
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		slog.InfoContext(ctx, "shutting down http server")
		if err := s.HTTP.Shutdown(ctx); err != nil {
			return fmt.Errorf("shutdown http server: %w", err)
		}
		return nil
	case err := <-errCh:
		return err
	}
}

const mdHeaderHTTPCode = "x-http-code"

func WithHTTPResponseCode(ctx context.Context, code int) {
	_ = grpc.SetHeader(ctx, metadata.Pairs(mdHeaderHTTPCode, strconv.Itoa(code)))
}

func (s *HTTPServer) setupDocsRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/docs/openapi.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(api.OpenAPISpec)
	})
	mux.Handle("/docs/", httpSwagger.Handler(httpSwagger.URL("/docs/openapi.json")))
}

func (s *HTTPServer) grpcGatewayResponseModifier(ctx context.Context, w http.ResponseWriter, _ proto.Message) error {
	md, ok := runtime.ServerMetadataFromContext(ctx)
	if !ok {
		return nil
	}

	if code, ok := s.getHTTPStatusFromMetadata(md, w); ok {
		w.WriteHeader(code)
	}
	return nil
}

func (s *HTTPServer) grpcGatewayErrorHandler(
	ctx context.Context,
	mux *runtime.ServeMux,
	marshaler runtime.Marshaler,
	w http.ResponseWriter,
	r *http.Request,
	err error,
) {
	md, ok := runtime.ServerMetadataFromContext(ctx)
	if !ok {
		runtime.DefaultHTTPErrorHandler(ctx, mux, marshaler, w, r, err)
		return
	}

	if code, ok := s.getHTTPStatusFromMetadata(md, w); ok {
		err = &runtime.HTTPStatusError{HTTPStatus: code, Err: err}
	}
	runtime.DefaultHTTPErrorHandler(ctx, mux, marshaler, w, r, err)
}

func (s *HTTPServer) getHTTPStatusFromMetadata(md runtime.ServerMetadata, w http.ResponseWriter) (int, bool) {
	if vals := md.HeaderMD.Get(mdHeaderHTTPCode); len(vals) > 0 {
		if code, err := strconv.Atoi(vals[0]); err == nil {
			delete(md.HeaderMD, mdHeaderHTTPCode)
			delete(w.Header(), "Grpc-Metadata-X-Http-Code")
			return code, true
		}
	}
	return 0, false
}
