package service

import (
	"context"
	"crypto/md5"
	"edu-final-calculate-api/internal/calculator/auth"
	"edu-final-calculate-api/internal/calculator/config"
	"edu-final-calculate-api/internal/calculator/repository/sqlite/models"
	"edu-final-calculate-api/internal/logging"
	calculatorv1 "edu-final-calculate-api/pkg/calculator/v1"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	Auth interface {
		GenerateJWT(auth.UserInfo) (string, error)
	}

	UserRepository interface {
		Register(ctx context.Context, cmd models.RegisterUserCmd) error
		GetUser(ctx context.Context, cmd models.GetUserCmd) (*models.User, error)
	}
)

type UserService struct {
	calculatorv1.UnimplementedUserServiceServer
	conf *config.Config
	log  *slog.Logger
	auth Auth
	repo UserRepository
}

func NewUserService(conf *config.Config, log *slog.Logger, auth Auth, repo UserRepository) *UserService {
	return &UserService{
		conf: conf,
		log:  logging.WithName(log, "user-service"),
		auth: auth,
		repo: repo,
	}
}

func (s *UserService) RegisterWith(srv *grpc.Server) {
	calculatorv1.RegisterUserServiceServer(srv, s)
}

func (s *UserService) RegisterGRPCGateway(ctx context.Context, mux *runtime.ServeMux, clientOpts []grpc.DialOption) error {
	return calculatorv1.RegisterUserServiceHandlerFromEndpoint(ctx, mux, "localhost"+s.conf.GRPCAddr, clientOpts)
}

func (s *UserService) Register(ctx context.Context, req *calculatorv1.RegisterRequest) (*emptypb.Empty, error) {
	if err := s.repo.Register(ctx, models.RegisterUserCmd{
		Login:        req.Login,
		PasswordHash: s.hashPassword(req.Password),
	}); err != nil {
		if errors.Is(err, models.ErrUserExists) {
			return nil, status.Error(codes.AlreadyExists, "user exists")
		}
		return nil, InternalError(fmt.Errorf("register user: %w", err))
	}

	return &emptypb.Empty{}, nil
}

func (s *UserService) Login(ctx context.Context, req *calculatorv1.LoginRequest) (*calculatorv1.LoginResponse, error) {
	user, err := s.repo.GetUser(ctx, models.GetUserCmd{
		Login:        req.Login,
		PasswordHash: s.hashPassword(req.Password),
	})
	if err != nil {
		if errors.Is(err, models.ErrUserNotFound) {
			return nil, status.Error(codes.FailedPrecondition, "bad login or password")
		}
		return nil, InternalError(fmt.Errorf("get user: %w", err))
	}

	token, err := s.auth.GenerateJWT(auth.UserInfo{ID: user.ID, Login: user.Login})
	if err != nil {
		return nil, InternalError(fmt.Errorf("generate jwt: %w", err))
	}
	return &calculatorv1.LoginResponse{AccessToken: token}, nil
}

func (s *UserService) hashPassword(password string) string {
	h := md5.New()
	h.Write([]byte(password))
	return hex.EncodeToString(h.Sum(nil))
}
