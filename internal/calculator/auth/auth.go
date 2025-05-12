package auth

import (
	"context"
	calculatorv1 "edu-final-calculate-api/pkg/calculator/v1"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/auth"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/selector"
	"github.com/samber/lo"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UserInfo struct {
	ID    string `json:"id"`
	Login string `json:"login"`
}

type Manager struct {
	jwtSecret         string
	jwtExpirationTime time.Duration
}

func NewManager(secret string) *Manager {
	return &Manager{jwtSecret: secret}
}

// Claims represents the JWT claims for authentication.
type Claims struct {
	UserInfo UserInfo `json:"user_info"`
	jwt.RegisteredClaims
}

// GenerateJWT creates a JWT token for the given user.
func (m *Manager) GenerateJWT(user UserInfo) (string, error) {
	now := time.Now().UTC()
	claims := &Claims{
		UserInfo: user,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(m.jwtExpirationTime)),
			IssuedAt:  jwt.NewNumericDate(now),
			Subject:   user.ID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(m.jwtSecret))
	if err != nil {
		return "", fmt.Errorf("sign jwt: %w", err)
	}

	return tokenString, nil
}

// UnaryServerInterceptor returns a gRPC unary server interceptor for authentication.
func (m *Manager) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	authFn := func(ctx context.Context) (context.Context, error) {
		token, err := auth.AuthFromMD(ctx, "bearer")
		if err != nil {
			return nil, err
		}
		claims, err := m.validateJWT(token)
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "invalid auth token: %v", err)
		}
		return WithContext(ctx, claims.UserInfo), nil
	}

	matchFn := func(ctx context.Context, callMeta interceptors.CallMeta) bool {
		return calculatorv1.CalculatorService_ServiceDesc.ServiceName == callMeta.Service
	}

	return selector.UnaryServerInterceptor(auth.UnaryServerInterceptor(authFn), selector.MatchFunc(matchFn))
}

func (m *Manager) validateJWT(s string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(s, claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(m.jwtSecret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse jwt: %w", err)
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

type ctxKey struct{}

func WithContext(ctx context.Context, user UserInfo) context.Context {
	return context.WithValue(ctx, ctxKey{}, user)
}

// UserFromContext retrieves UserInfo from context.
func UserFromContext(ctx context.Context) (UserInfo, bool) {
	userInfo, ok := ctx.Value(ctxKey{}).(UserInfo)
	if !ok {
		return UserInfo{}, false
	}
	return userInfo, true
}

func MustUserIDFromContext(ctx context.Context) string {
	return lo.Must(UserFromContext(ctx)).ID
}
