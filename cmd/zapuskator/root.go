package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"

	"github.com/denkoren/mi-labs-test/internal/services/api"
	apipb "github.com/denkoren/mi-labs-test/proto/v1"
)

var (
	// Used for flags.
	grpcPort int
	httpPort int
)

var rootCmd = &cobra.Command{
	Use:   "zapuskator",
	Short: "Zapuskator",
	Long: `...`,
	RunE: runRoot,
}

func runRoot(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	addr := fmt.Sprintf("localhost:%d", grpcPort)

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer(
		//grpc.StreamInterceptor(...),
	)
	apipb.RegisterZapuskatorAPIServer(grpcServer, &api.Server{})

	var group errgroup.Group

	group.Go(func() error {
		log.Printf("grpc server listening at %v", addr)
		return grpcServer.Serve(lis)
	})

	mux := runtime.NewServeMux(runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{OrigName: true, EmitDefaults: true}))
	opts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(50000000)),
	}

	group.Go(func() error {
		return apipb.RegisterZapuskatorAPIHandlerFromEndpoint(ctx, mux, addr, opts)
	})

	group.Go(func() error {
		addr := fmt.Sprintf("localhost:%d", httpPort)
		log.Printf("http server listening at %v", addr)
		return http.ListenAndServe(addr, mux)
	})

	// FIXME: graceful shutdown by os.signal()

	return group.Wait()
}

func init() {
	rootCmd.PersistentFlags().IntVar(&grpcPort, "grpc-port", 4334, "Port to be listened by Zapuskator gRPC service")
	rootCmd.PersistentFlags().IntVar(&httpPort, "http-port", 4224, "Port to be listened by Zapuskator HTTP service")
}
