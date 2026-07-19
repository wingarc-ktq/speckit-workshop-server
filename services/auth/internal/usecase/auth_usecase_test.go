package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/wingarc-ktq/speckit-workshop-server/services/auth/internal/domain"
	"github.com/wingarc-ktq/speckit-workshop-server/services/auth/internal/usecase"
	"github.com/wingarc-ktq/speckit-workshop-server/services/auth/internal/usecase/mock"
)

func TestAuthUsecase_Register(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     usecase.RegisterInput
		setup     func(*testing.T, *mock.MockUserRepository, *mock.MockPasswordHasher)
		wantErr   error
		wantEmail string // 期待される保存後（正規化後）のメールアドレス
	}{
		{
			name: "success",
			input: usecase.RegisterInput{
				Email:    "taro@example.com",
				Password: "P@ssw0rd!",
				Name:     "田中太郎",
			},
			setup: func(_ *testing.T, r *mock.MockUserRepository, h *mock.MockPasswordHasher) {
				r.EXPECT().FindByEmail(gomock.Any(), "taro@example.com").Return(nil, domain.ErrUserNotFound)
				h.EXPECT().Hash("P@ssw0rd!").Return("hashed-pw", nil)
				r.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
			},
			wantEmail: "taro@example.com",
		},
		{
			name: "email normalized to lowercase",
			input: usecase.RegisterInput{
				Email:    "Taro@Example.COM",
				Password: "P@ssw0rd!",
				Name:     "田中太郎",
			},
			setup: func(t *testing.T, r *mock.MockUserRepository, h *mock.MockPasswordHasher) {
				// 検索・保存ともに小文字化されたメールで行われること.
				r.EXPECT().FindByEmail(gomock.Any(), "taro@example.com").Return(nil, domain.ErrUserNotFound)
				h.EXPECT().Hash(gomock.Any()).Return("hashed-pw", nil)
				r.EXPECT().Create(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, u *domain.User) error {
						if u.Email != "taro@example.com" {
							t.Errorf("email: got %s, want taro@example.com", u.Email)
						}
						if u.PasswordHash != "hashed-pw" {
							t.Errorf("passwordHash: got %s, want hashed-pw", u.PasswordHash)
						}
						return nil
					})
			},
			wantEmail: "taro@example.com",
		},
		{
			name: "email already taken",
			input: usecase.RegisterInput{
				Email:    "taro@example.com",
				Password: "P@ssw0rd!",
				Name:     "田中太郎",
			},
			setup: func(_ *testing.T, r *mock.MockUserRepository, _ *mock.MockPasswordHasher) {
				r.EXPECT().FindByEmail(gomock.Any(), "taro@example.com").Return(&domain.User{}, nil)
			},
			wantErr: domain.ErrEmailAlreadyTaken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			repo := mock.NewMockUserRepository(ctrl)
			hasher := mock.NewMockPasswordHasher(ctrl)
			tokens := mock.NewMockTokenIssuer(ctrl)
			tt.setup(t, repo, hasher)
			uc := usecase.NewAuthUsecase(repo, hasher, tokens)

			user, err := uc.Register(context.Background(), tt.input)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("err: got %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if user.Email != tt.wantEmail {
				t.Errorf("email: got %s, want %s", user.Email, tt.wantEmail)
			}
			if user.PasswordHash != "hashed-pw" {
				t.Errorf("passwordHash: got %s, want hashed-pw", user.PasswordHash)
			}
		})
	}
}

func TestAuthUsecase_Login(t *testing.T) {
	t.Parallel()

	stored := &domain.User{
		ID:           uuid.New(),
		Email:        "taro@example.com",
		PasswordHash: "stored-hash",
		Name:         "田中太郎",
	}

	tests := []struct {
		name     string
		email    string
		password string
		setup    func(*mock.MockUserRepository, *mock.MockPasswordHasher, *mock.MockTokenIssuer)
		wantErr  error
	}{
		{
			name:     "success",
			email:    "taro@example.com",
			password: "P@ssw0rd!",
			setup: func(r *mock.MockUserRepository, h *mock.MockPasswordHasher, tk *mock.MockTokenIssuer) {
				r.EXPECT().FindByEmail(gomock.Any(), "taro@example.com").Return(stored, nil)
				h.EXPECT().Compare("stored-hash", "P@ssw0rd!").Return(nil)
				tk.EXPECT().Issue(stored.ID).Return("signed-token", 3600, nil)
			},
		},
		{
			name:     "user not found",
			email:    "ghost@example.com",
			password: "whatever",
			setup: func(r *mock.MockUserRepository, _ *mock.MockPasswordHasher, _ *mock.MockTokenIssuer) {
				r.EXPECT().FindByEmail(gomock.Any(), "ghost@example.com").Return(nil, domain.ErrUserNotFound)
			},
			wantErr: domain.ErrInvalidCredential,
		},
		{
			name:     "wrong password",
			email:    "taro@example.com",
			password: "wrong",
			setup: func(r *mock.MockUserRepository, h *mock.MockPasswordHasher, _ *mock.MockTokenIssuer) {
				r.EXPECT().FindByEmail(gomock.Any(), "taro@example.com").Return(stored, nil)
				h.EXPECT().Compare("stored-hash", "wrong").Return(errors.New("mismatch"))
			},
			wantErr: domain.ErrInvalidCredential,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			repo := mock.NewMockUserRepository(ctrl)
			hasher := mock.NewMockPasswordHasher(ctrl)
			tokens := mock.NewMockTokenIssuer(ctrl)
			tt.setup(repo, hasher, tokens)
			uc := usecase.NewAuthUsecase(repo, hasher, tokens)

			out, err := uc.Login(context.Background(), tt.email, tt.password)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("err: got %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if out.AccessToken != "signed-token" {
				t.Errorf("accessToken: got %s, want signed-token", out.AccessToken)
			}
			if out.ExpiresIn != 3600 {
				t.Errorf("expiresIn: got %d, want 3600", out.ExpiresIn)
			}
			if out.User.ID != stored.ID {
				t.Errorf("userID: got %v, want %v", out.User.ID, stored.ID)
			}
		})
	}
}

func TestAuthUsecase_Me(t *testing.T) {
	t.Parallel()

	existing := &domain.User{
		ID:    uuid.New(),
		Email: "taro@example.com",
		Name:  "田中太郎",
	}

	tests := []struct {
		name    string
		userID  uuid.UUID
		setup   func(*mock.MockUserRepository)
		wantErr error
	}{
		{
			name:   "found",
			userID: existing.ID,
			setup: func(m *mock.MockUserRepository) {
				m.EXPECT().FindByID(gomock.Any(), existing.ID).Return(existing, nil)
			},
		},
		{
			name:   "not found",
			userID: existing.ID,
			setup: func(m *mock.MockUserRepository) {
				m.EXPECT().FindByID(gomock.Any(), existing.ID).Return(nil, domain.ErrUserNotFound)
			},
			wantErr: domain.ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			repo := mock.NewMockUserRepository(ctrl)
			hasher := mock.NewMockPasswordHasher(ctrl)
			tokens := mock.NewMockTokenIssuer(ctrl)
			tt.setup(repo)
			uc := usecase.NewAuthUsecase(repo, hasher, tokens)

			user, err := uc.Me(context.Background(), tt.userID)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("err: got %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if user.ID != existing.ID {
				t.Errorf("userID: got %v, want %v", user.ID, existing.ID)
			}
			if user.Email != existing.Email {
				t.Errorf("email: got %s, want %s", user.Email, existing.Email)
			}
		})
	}
}
