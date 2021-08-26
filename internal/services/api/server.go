package api

import (
	"context"

	apipb "github.com/denkoren/mi-labs-test/proto/v1"
)

type Server struct {
	apipb.UnimplementedZapuskatorAPIServer
}

func (s *Server) Calculate(ctx context.Context, request *apipb.Calculate_Request) (*apipb.Calculate_Response, error) {
	return &apipb.Calculate_Response{Data: []byte("Here is my response")}, nil
}
