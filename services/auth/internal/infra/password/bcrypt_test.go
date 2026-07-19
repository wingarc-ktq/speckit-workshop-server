package password_test

import (
	"strings"
	"testing"

	"github.com/wingarc-ktq/speckit-workshop-server/services/auth/internal/infra/password"
)

const plain = "P@ssw0rd!"

func TestBcrypt_Hash(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{name: "成功", password: plain, wantErr: false},
		{name: "72バイト超はエラー", password: strings.Repeat("a", 73), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			hash, err := password.NewBcrypt().Hash(tt.password)
			if tt.wantErr {
				if err == nil {
					t.Error("err: got nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if hash == "" {
				t.Error("hash: got empty, want non-empty")
			}
			if hash == tt.password {
				t.Error("hash: got plaintext, want hashed")
			}
		})
	}
}

func TestBcrypt_Compare(t *testing.T) {
	t.Parallel()

	h := password.NewBcrypt()
	hash, err := h.Hash(plain)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{name: "パスワード一致", password: plain, wantErr: false},
		{name: "パスワード不一致", password: "wrong", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := h.Compare(hash, tt.password)
			if tt.wantErr {
				if err == nil {
					t.Error("err: got nil, want error")
				}
				return
			}
			if err != nil {
				t.Errorf("err: got %v, want nil", err)
			}
		})
	}
}
