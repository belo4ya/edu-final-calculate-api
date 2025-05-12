package service

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"edu-final-calculate-api/internal/calculator/auth"
	"edu-final-calculate-api/internal/calculator/config"
	"edu-final-calculate-api/internal/calculator/repository/models"
	"edu-final-calculate-api/internal/testutil"
	mocks "edu-final-calculate-api/internal/testutil/mocks/calculator/service"

	calculatorv1 "edu-final-calculate-api/pkg/calculator/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestUserService_Register(t *testing.T) {
	type args struct {
		ctx context.Context
		req *calculatorv1.RegisterRequest
	}
	tests := []struct {
		name       string
		setupMocks func(auth *mocks.MockAuth, repo *mocks.MockUserRepository)
		args       args
		wantErr    assert.ErrorAssertionFunc
	}{
		{
			name: "successful registration",
			setupMocks: func(_ *mocks.MockAuth, repo *mocks.MockUserRepository) {
				repo.EXPECT().Register(mock.Anything, mock.MatchedBy(func(cmd models.RegisterUserCmd) bool {
					// Password "password123" should be hashed
					return cmd.Login == "testuser" && cmd.PasswordHash != "password123"
				})).Return(nil)
			},
			args: args{
				ctx: context.Background(),
				req: &calculatorv1.RegisterRequest{
					Login:    "testuser",
					Password: "password123",
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "user already exists",
			setupMocks: func(_ *mocks.MockAuth, repo *mocks.MockUserRepository) {
				repo.EXPECT().Register(mock.Anything, mock.Anything).Return(models.ErrUserExists)
			},
			args: args{
				ctx: context.Background(),
				req: &calculatorv1.RegisterRequest{
					Login:    "existinguser",
					Password: "password123",
				},
			},
			wantErr: assert.Error,
		},
		{
			name: "repository error",
			setupMocks: func(_ *mocks.MockAuth, repo *mocks.MockUserRepository) {
				repo.EXPECT().Register(mock.Anything, mock.Anything).Return(assert.AnError)
			},
			args: args{
				ctx: context.Background(),
				req: &calculatorv1.RegisterRequest{
					Login:    "newuser",
					Password: "password123",
				},
			},
			wantErr: assert.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authMock := mocks.NewMockAuth(t)
			repo := mocks.NewMockUserRepository(t)

			tt.setupMocks(authMock, repo)
			svc := NewUserService(&config.Config{}, testutil.DiscardLogger(), authMock, repo)

			_, err := svc.Register(tt.args.ctx, tt.args.req)
			tt.wantErr(t, err, fmt.Sprintf("Register(%v, %v)", tt.args.ctx, tt.args.req))
		})
	}
}

func TestUserService_Login(t *testing.T) {
	userID := "user-id"
	userLogin := "testuser"

	type args struct {
		ctx context.Context
		req *calculatorv1.LoginRequest
	}
	tests := []struct {
		name       string
		setupMocks func(authMock *mocks.MockAuth, repo *mocks.MockUserRepository)
		args       args
		want       *calculatorv1.LoginResponse
		wantErr    assert.ErrorAssertionFunc
	}{
		{
			name: "successful login",
			setupMocks: func(authMock *mocks.MockAuth, repo *mocks.MockUserRepository) {
				repo.EXPECT().GetUser(mock.Anything, mock.MatchedBy(func(cmd models.GetUserCmd) bool {
					// Password "password123" should be hashed
					return cmd.Login == "testuser" && cmd.PasswordHash != "password123"
				})).Return(&models.User{
					ID:    userID,
					Login: userLogin,
				}, nil)

				authMock.EXPECT().GenerateJWT(auth.UserInfo{
					ID:    userID,
					Login: userLogin,
				}).Return("jwt-token", nil)
			},
			args: args{
				ctx: context.Background(),
				req: &calculatorv1.LoginRequest{
					Login:    "testuser",
					Password: "password123",
				},
			},
			want: &calculatorv1.LoginResponse{
				AccessToken: "jwt-token",
			},
			wantErr: assert.NoError,
		},
		{
			name: "user not found",
			setupMocks: func(_ *mocks.MockAuth, repo *mocks.MockUserRepository) {
				repo.EXPECT().GetUser(mock.Anything, mock.Anything).Return(nil, models.ErrUserNotFound)
			},
			args: args{
				ctx: context.Background(),
				req: &calculatorv1.LoginRequest{
					Login:    "nonexistent",
					Password: "password123",
				},
			},
			want:    nil,
			wantErr: assert.Error,
		},
		{
			name: "repository error",
			setupMocks: func(_ *mocks.MockAuth, repo *mocks.MockUserRepository) {
				repo.EXPECT().GetUser(mock.Anything, mock.Anything).Return(nil, errors.New("database connection error"))
			},
			args: args{
				ctx: context.Background(),
				req: &calculatorv1.LoginRequest{
					Login:    "testuser",
					Password: "password123",
				},
			},
			want:    nil,
			wantErr: assert.Error,
		},
		{
			name: "JWT generation error",
			setupMocks: func(authMock *mocks.MockAuth, repo *mocks.MockUserRepository) {
				repo.EXPECT().GetUser(mock.Anything, mock.Anything).Return(&models.User{
					ID:    userID,
					Login: userLogin,
				}, nil)

				authMock.EXPECT().GenerateJWT(mock.Anything).Return("", errors.New("JWT signing error"))
			},
			args: args{
				ctx: context.Background(),
				req: &calculatorv1.LoginRequest{
					Login:    "testuser",
					Password: "password123",
				},
			},
			want:    nil,
			wantErr: assert.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authMock := mocks.NewMockAuth(t)
			repo := mocks.NewMockUserRepository(t)

			tt.setupMocks(authMock, repo)
			svc := NewUserService(&config.Config{}, testutil.DiscardLogger(), authMock, repo)

			got, err := svc.Login(tt.args.ctx, tt.args.req)
			if !tt.wantErr(t, err, fmt.Sprintf("Login(%v, %v)", tt.args.ctx, tt.args.req)) {
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
