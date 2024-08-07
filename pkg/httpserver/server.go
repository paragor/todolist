package httpserver

import (
	"bytes"
	"context"
	"fmt"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/paragor/todo/pkg/models"
	"github.com/paragor/todo/public"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"os"
	"path"
	"strconv"
	"sync"
	"time"
)

type httpServer struct {
	listen     string
	mux        *mux.Router
	repository models.Repository
	authConfig *AuthChainConfig
	oidc       *authOidcContext

	cancel       func()
	shutdownChan chan struct{}
}

func NewHttpServer(
	listen string,
	repository models.Repository,
	authConfig *AuthChainConfig,
	serverPublicUrl string,
	diagnosticEndpointsEnabled bool,
) (*httpServer, error) {
	server := &httpServer{listen: listen, mux: mux.NewRouter(), repository: repository, authConfig: authConfig}
	server.mux.Use(
		handlers.RecoveryHandler(),
		func(handler http.Handler) http.Handler {
			return handlers.LoggingHandler(os.Stdout, handler)
		},
		handlers.CompressHandler,
	)
	server.mux.Name("static").PathPrefix("/static/").Handler(
		restartEtag(
			cacheMiddleware(
				http.FileServer(
					http.FS(
						public.Static,
					),
				),
				5*time.Minute,
			),
		),
	)
	if authConfig != nil {
		server.mux.Path("/login").HandlerFunc(server.htmxPageLogin)
	}
	if authConfig != nil && authConfig.AuthOidcConfig != nil {
		oidc, err := newOidcContext(*authConfig.AuthOidcConfig, serverPublicUrl+"/oidc/callback", "/")
		if err != nil {
			return nil, fmt.Errorf("cant init oidc: %w", err)
		}
		server.oidc = oidc
		server.mux.Path("/oidc/callback").Handler(server.oidc.AuthCallbackHandler())
		server.mux.Path("/oidc/login").Handler(server.oidc.AuthLoginHandler())
	}
	if diagnosticEndpointsEnabled {
		server.mux.Path("/metrics").Handler(promhttp.Handler())
		server.mux.Path("/healthz").HandlerFunc(server.apiPing)
		server.mux.Path("/readyz").HandlerFunc(server.apiPing)
	}

	htmx := server.mux.Name("htmx").Subrouter()
	if authConfig != nil {
		htmx.Use(server.AuthChainMiddleware())
	}
	htmx.Path("/").HandlerFunc(server.htmxPageMain)
	htmx.Path("/projects").HandlerFunc(server.htmxPageProjects)
	htmx.Path("/agenda").HandlerFunc(server.htmxPageAgenda)
	htmx.Path("/task").HandlerFunc(server.htmxPageTask)
	htmx.Path("/htmx/get_task").HandlerFunc(server.htmxGetTask)
	htmx.Path("/htmx/edit_task").HandlerFunc(server.htmxEditTask)
	htmx.Path("/htmx/copy_task").HandlerFunc(server.htmxCopyTask)
	htmx.Path("/htmx/new_task").HandlerFunc(server.htmxNewTask)
	htmx.Path("/htmx/api/save_status").Methods("PUT").HandlerFunc(server.htmxSaveStatus)
	htmx.Path("/htmx/api/save_task").Methods("PUT").HandlerFunc(server.htmxSaveTask)

	api := server.mux.Name("api").PathPrefix("/api/").Subrouter()
	if authConfig != nil {
		api.Use(server.AuthChainMiddleware())
	}
	api.Path("/ping").HandlerFunc(server.apiPing)
	api.Path("/all").HandlerFunc(server.apiAllTask)
	api.Path("/get_task").HandlerFunc(server.apiGetTask)
	api.Path("/insert_task").Methods("PUT").HandlerFunc(server.apiInsertTask)

	return server, nil
}

func (h *httpServer) Stop() {
	if h.cancel != nil {
		h.cancel()
	}
	if h.shutdownChan != nil {
		<-h.shutdownChan
	}
}
func (h *httpServer) Start(ctx context.Context, stopper chan<- error) error {
	h.shutdownChan = make(chan struct{}, 1)
	server := &http.Server{Addr: h.listen, Handler: h.mux}
	go func() {
		err := server.ListenAndServe()
		stopper <- fmt.Errorf("stop httpserver: %w", err)
	}()
	go func() {
		<-ctx.Done()
		close(h.shutdownChan)
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()
	return nil
}

func must[T any](result T, err error) T {
	if err != nil {
		panic(err)
	}
	return result
}

var bytesBufferPool = &sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(nil)
	},
}

func restartEtag(handler http.Handler) http.Handler {
	start := time.Now()
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		etag := start.String() + "@" + path.Clean(request.URL.Path)
		if requestEtag := request.Header.Get("If-None-Match"); requestEtag == etag {
			writer.WriteHeader(304)
			return
		}
		writer.Header().Set("ETag", etag)
		handler.ServeHTTP(writer, request)
	})
}

func cacheMiddleware(handler http.Handler, duration time.Duration) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Cache-Control", "max-age="+strconv.Itoa(int(duration.Seconds())))
		handler.ServeHTTP(writer, request)
	})
}
