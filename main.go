package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"

	"grpc_setup/app"
	"grpc_setup/proto"
	"grpc_setup/protoservices"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

func init() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)
}

var cfg *app.Config

func main() {
	cfg = app.LoadConfig()

	if cfg.IsDebuggingEnabled {
		log.SetLevel(log.DebugLevel)
		log.Info("Logging Level Debug set.")
	} else {
		log.SetLevel(log.InfoLevel)
		log.Info("Logging Level Info set.")
	}

	if err := Run(cfg.APIPort); err != nil {
		log.Fatal(err)
	}
}

// Run ...
func Run(address string, opts ...runtime.ServeMuxOption) error {

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errorChannel := make(chan error, 2)

	go func() { errorChannel <- newGRPCService() }()
	go func() { errorChannel <- newRESTService(ctx, address, opts...) }()

	if err := <-errorChannel; err != nil {
		return err
	}
	return nil
}

func newGRPCService() error {
	lis, err := net.Listen("tcp", cfg.GrpcPort)
	if err != nil {
		panic("Failed to start GRPC Services")
	}

	grpcServer := grpc.NewServer()
	proto.RegisterHealthServer(grpcServer, protoservices.NewHealthService())

	return grpcServer.Serve(lis)
}

func newRESTService(ctx context.Context, address string, opts ...runtime.ServeMuxOption) error {
	mux := http.NewServeMux()
	gw, err := newGateway(ctx, opts...)
	if err != nil {
		return err
	}
	mux.Handle("/", gw)

	return http.ListenAndServe(address, allowCORS(mux))
}

// newGateway returns a new gateway server which translates HTTP into gRPC.
func newGateway(ctx context.Context, opts ...runtime.ServeMuxOption) (http.Handler, error) {
	mux := runtime.NewServeMux(opts...)
	dialOpts := []grpc.DialOption{grpc.WithInsecure()}

	err := proto.RegisterHealthHandlerFromEndpoint(ctx, mux, fmt.Sprintf("%s%s", cfg.GrpcHost, cfg.GrpcPort), dialOpts)
	if err != nil {
		return nil, err
	}

	return mux, nil
}

// allowCORS allows Cross Origin Resoruce Sharing from any origin.
// Don't do this without consideration in production systems.
func allowCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			if r.Method == "OPTIONS" && r.Header.Get("Access-Control-Request-Method") != "" {
				preflightHandler(w, r)
				return
			}
		}
		h.ServeHTTP(w, r)
	})
}

func preflightHandler(w http.ResponseWriter, r *http.Request) {
	headers := []string{"Content-Type", "Accept"}
	w.Header().Set("Access-Control-Allow-Headers", strings.Join(headers, ","))
	methods := []string{"GET", "HEAD", "POST", "PUT", "DELETE"}
	w.Header().Set("Access-Control-Allow-Methods", strings.Join(methods, ","))
	log.Infof("preflight request for %s", r.URL.Path)
	return
}
