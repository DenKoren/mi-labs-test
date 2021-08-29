package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/denkoren/mi-labs-test/internal/services/background"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"

	"github.com/denkoren/mi-labs-test/internal/interconnect/docker"
	"github.com/denkoren/mi-labs-test/internal/interconnect/registry"
	"github.com/denkoren/mi-labs-test/internal/services/api"
	apipb "github.com/denkoren/mi-labs-test/proto/api/v1"
)

var (
	// Used for flags.
	grpcPort int
	httpPort int
)

var rootCmd = &cobra.Command{
	Use:   "zapuskator",
	Short: "Zapuskator",
	Long:  `...`,
	RunE:  runRoot,
}

func runRoot(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var (
		err      error
		grpcAddr string
		group    *errgroup.Group
		groupCtx context.Context

		cRegistry *registry.ContainerRegistry
		cManager  *docker.Manager
	)

	grpcAddr = fmt.Sprintf("localhost:%d", grpcPort)

	cRegistry, err = initContainerRegistry()
	cobra.CheckErr(err)

	cManager, err = initDockerManager()
	cobra.CheckErr(err)

	group, groupCtx = errgroup.WithContext(ctx)

	initGrpcAPIServer(groupCtx, group, grpcAddr, cRegistry, cManager)
	initRestAPIServer(groupCtx, group, grpcAddr)
	initBackgroundService(groupCtx, group, cRegistry, cManager)

	// FIXME: graceful shutdown by os.signal()
	return group.Wait()
}

func init() {
	rootCmd.PersistentFlags().IntVar(&grpcPort, "grpc-port", 4334, "Port to be listened by Zapuskator gRPC service")
	rootCmd.PersistentFlags().IntVar(&httpPort, "http-port", 4224, "Port to be listened by Zapuskator HTTP service")
}

func initContainerRegistry() (*registry.ContainerRegistry, error) {
	return registry.NewContainerRegistry()
}

func initDockerManager() (*docker.Manager, error) {
	return docker.NewManager(
		docker.ManagerConfig{
			Host:           "",
			RequestTimeout: time.Second,
			ImageTag:       "mi-labs-test:latest",
		},
	)
}

func initGrpcAPIServer(_ context.Context, group *errgroup.Group, addr string, cRegistry *registry.ContainerRegistry, cManager *docker.Manager) {
	lis, err := net.Listen("tcp", addr)
	cobra.CheckErr(err)

	grpcServer := grpc.NewServer(
		//grpc.StreamInterceptor(...),
	)
	srv, err := api.NewServer(
		api.Config{
			ContainerWaitTimeout: 200 * time.Second,
		},
		cRegistry,
		cManager,
	)
	cobra.CheckErr(err)

	apipb.RegisterZapuskatorAPIServer(grpcServer, srv)

	group.Go(func() error {
		log.Printf("grpc server listening at %v", addr)
		return grpcServer.Serve(lis)
	})
}

func initRestAPIServer(ctx context.Context, group *errgroup.Group, grpcAddr string) {
	mux := runtime.NewServeMux(
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{OrigName: true, EmitDefaults: true}),
	)
	opts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(50000000)),
	}

	group.Go(func() error {
		return apipb.RegisterZapuskatorAPIHandlerFromEndpoint(ctx, mux, grpcAddr, opts)
	})

	group.Go(func() error {
		addr := fmt.Sprintf(":%d", httpPort)
		log.Printf("http server listening at %v", addr)
		return http.ListenAndServe(addr, mux)
	})
}

func initBackgroundService(ctx context.Context, group *errgroup.Group, cRegistry *registry.ContainerRegistry, cManager *docker.Manager) {
	bg, err := background.NewBackground(
		background.Config{
			InactiveContainerTimeout: 120 * time.Second,
			ContainersCheckInterval:  time.Second,
		},
		cRegistry,
		cManager,
	)

	if err != nil {
		group.Go(func() error { return err })
		return
	}

	group.Go(func() error {
		return bg.Run(ctx)
	})
}
