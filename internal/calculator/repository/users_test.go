package repository

import (
	"context"
	"edu-final-calculate-api/internal/calculator/repository/models"
	"testing"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository_Register(t *testing.T) {
	db := setupTestDB(t)
	repo := New(db)
	ctx := context.Background()

	tests := []struct {
		name    string
		cmd     models.RegisterUserCmd
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "successful registration",
			cmd: models.RegisterUserCmd{
				Login:        "testuser",
				PasswordHash: "hashedpassword123",
			},
			wantErr: require.NoError,
		},
		{
			name: "duplicate user",
			cmd: models.RegisterUserCmd{
				Login:        "testuser", // same login as above
				PasswordHash: "anotherpassword",
			},
			wantErr: func(t require.TestingT, err error, _ ...interface{}) {
				require.Error(t, err)
				require.ErrorIs(t, err, models.ErrUserExists)
			},
		},
		{
			name: "register another user",
			cmd: models.RegisterUserCmd{
				Login:        "anotheruser",
				PasswordHash: "hashedpassword456",
			},
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.Register(ctx, tt.cmd)
			tt.wantErr(t, err)
		})
	}
}

func TestRepository_GetUser(t *testing.T) {
	db := setupTestDB(t)
	repo := New(db)
	ctx := context.Background()

	testUsers := []models.RegisterUserCmd{
		{
			Login:        "user1",
			PasswordHash: "password1hash",
		},
		{
			Login:        "user2",
			PasswordHash: "password2hash",
		},
	}

	for _, user := range testUsers {
		err := repo.Register(ctx, user)
		require.NoError(t, err, "Failed to set up test user")
	}

	tests := []struct {
		name    string
		cmd     models.GetUserCmd
		want    func(*models.User) bool
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "get existing user",
			cmd: models.GetUserCmd{
				Login:        "user1",
				PasswordHash: "password1hash",
			},
			want: func(user *models.User) bool {
				return user != nil && user.Login == "user1" && user.PasswordHash == "password1hash"
			},
			wantErr: require.NoError,
		},
		{
			name: "wrong password",
			cmd: models.GetUserCmd{
				Login:        "user1",
				PasswordHash: "wrongpassword",
			},
			want: func(user *models.User) bool {
				return user == nil
			},
			wantErr: func(t require.TestingT, err error, _ ...interface{}) {
				require.Error(t, err)
				require.ErrorIs(t, err, models.ErrUserNotFound)
			},
		},
		{
			name: "user not found",
			cmd: models.GetUserCmd{
				Login:        "nonexistentuser",
				PasswordHash: "anypassword",
			},
			want: func(user *models.User) bool {
				return user == nil
			},
			wantErr: func(t require.TestingT, err error, _ ...interface{}) {
				require.Error(t, err)
				require.ErrorIs(t, err, models.ErrUserNotFound)
			},
		},
		{
			name: "admin user from migrations",
			cmd: models.GetUserCmd{
				Login:        "admin",
				PasswordHash: "21232f297a57a5a743894a0e4a801fc3", // MD5 hash of "admin"
			},
			want: func(user *models.User) bool {
				return user != nil && user.Login == "admin" &&
					user.PasswordHash == "21232f297a57a5a743894a0e4a801fc3" &&
					user.ID == "00000000000000000000"
			},
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.GetUser(ctx, tt.cmd)
			tt.wantErr(t, err)
			assert.True(t, tt.want(got))
		})
	}
}

func TestRepository_RegisterAndGetUser(t *testing.T) {
	db := setupTestDB(t)
	repo := New(db)
	ctx := context.Background()

	// Register a new user
	registerCmd := models.RegisterUserCmd{
		Login:        "newuser",
		PasswordHash: "securepassword123",
	}

	err := repo.Register(ctx, registerCmd)
	require.NoError(t, err, "Failed to register new user")

	// Try to get the user we just registered
	getUserCmd := models.GetUserCmd(registerCmd)

	user, err := repo.GetUser(ctx, getUserCmd)
	require.NoError(t, err, "Failed to get registered user")
	assert.NotNil(t, user, "User should not be nil")
	assert.Equal(t, registerCmd.Login, user.Login)
	assert.Equal(t, registerCmd.PasswordHash, user.PasswordHash)

	// Verify user ID was created
	assert.NotEmpty(t, user.ID, "User ID should not be empty")

	// Verify timestamps were set
	assert.False(t, user.CreatedAt.IsZero(), "CreatedAt should be set")
	assert.False(t, user.UpdatedAt.IsZero(), "UpdatedAt should be set")
}
