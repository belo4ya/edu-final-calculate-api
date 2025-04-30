package service

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func InternalError(err error) error {
	return status.Errorf(codes.Internal, "oops, something went wrong: %v", err)
}
