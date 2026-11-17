// Package auth управляет аккаунтами и непрозрачными серверными сессиями.
package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"blood/apps/api/internal/contracts"
	"github.com/alexedwards/argon2id"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const SessionLifetime = 30 * 24 * time.Hour

var (
	ErrCredentials = errors.New("неверный логин или пароль")
	ErrConflict    = errors.New("логин уже занят")
	ErrInvalid     = errors.New("недопустимые учётные данные")
	ErrSession     = errors.New("требуется вход в аккаунт")
	loginPattern   = regexp.MustCompile(`^[A-Za-z0-9_-]{3,32}$`)
	// RFC 9106, второй рекомендуемый вариант: 64 МиБ, 3 прохода, 4 потока.
	passwordParams = &argon2id.Params{Memory: 64 * 1024, Iterations: 3, Parallelism: 4, SaltLength: 16, KeyLength: 32}
)

type User struct {
	ID           string `gorm:"primaryKey;size:64"`
	Login        string `gorm:"uniqueIndex;size:32;not null"`
	PasswordHash string `gorm:"not null" json:"-"`
	CreatedAt    time.Time
}

type Session struct {
	TokenHash string    `gorm:"primaryKey;size:64" json:"-"`
	UserID    string    `gorm:"size:64;not null;index"`
	User      User      `gorm:"constraint:OnDelete:CASCADE"`
	ExpiresAt time.Time `gorm:"not null;index"`
	CreatedAt time.Time
}

type Identity struct {
	User      contracts.User
	TokenHash string
	ExpiresAt time.Time
}

type Service struct {
	db        *gorm.DB
	dummyHash string
	// Ограничивает суммарную память Argon2 независимо от количества IP.
	work chan struct{}
}

func New(db *gorm.DB) (*Service, error) {
	hash, err := argon2id.CreateHash(RandomToken(), passwordParams)
	if err != nil {
		return nil, err
	}
	return &Service{db: db, dummyHash: hash, work: make(chan struct{}, 2)}, nil
}

func RandomToken() string {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic("недоступен системный генератор случайных чисел")
	}
	return base64.RawURLEncoding.EncodeToString(b[:])
}

func Digest(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func CSRFToken(token string) string {
	mac := hmac.New(sha256.New, []byte(token))
	_, _ = mac.Write([]byte("blood-session-csrf-v1"))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func ValidCredentials(c contracts.Credentials) bool {
	n := utf8.RuneCountInString(c.Password)
	return loginPattern.MatchString(c.Login) && utf8.ValidString(c.Password) && n >= 12 && n <= 128
}

// Authenticate заменяет только текущую сессию устройства. Другие устройства
// продолжают работать под тем же UserID и сохраняют членство в комнатах.
func (s *Service) Authenticate(ctx context.Context, c contracts.Credentials, register bool, oldToken string) (contracts.Session, string, error) {
	if !ValidCredentials(c) {
		return contracts.Session{}, "", ErrInvalid
	}
	select {
	case s.work <- struct{}{}:
		defer func() { <-s.work }()
	case <-ctx.Done():
		return contracts.Session{}, "", ctx.Err()
	}
	login := strings.ToLower(c.Login)
	var user User
	if register {
		hash, err := argon2id.CreateHash(c.Password, passwordParams)
		if err != nil {
			return contracts.Session{}, "", err
		}
		user = User{ID: RandomToken(), Login: login, PasswordHash: hash}
	} else {
		err := s.db.WithContext(ctx).Where(&User{Login: login}).Take(&user).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return contracts.Session{}, "", err
		}
		hash := user.PasswordHash
		if hash == "" {
			hash = s.dummyHash
		}
		match, err := argon2id.ComparePasswordAndHash(c.Password, hash)
		if err != nil {
			return contracts.Session{}, "", err
		}
		if !match || user.ID == "" {
			return contracts.Session{}, "", ErrCredentials
		}
	}
	token := RandomToken()
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if register {
			result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&user)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return ErrConflict
			}
		}
		if oldToken != "" {
			if err := tx.Where(&Session{TokenHash: Digest(oldToken)}).Delete(&Session{}).Error; err != nil {
				return err
			}
		}
		// Удаление истёкших записей не затрагивает действующие устройства.
		if err := tx.Where(clause.Lte{Column: "expires_at", Value: time.Now().UTC()}).Delete(&Session{}).Error; err != nil {
			return err
		}
		return tx.Create(&Session{TokenHash: Digest(token), UserID: user.ID, ExpiresAt: time.Now().UTC().Add(SessionLifetime)}).Error
	})
	if err != nil {
		return contracts.Session{}, "", err
	}
	return contracts.Session{User: contracts.User{Id: user.ID, Login: user.Login}, CsrfToken: CSRFToken(token)}, token, nil
}

func (s *Service) Resolve(ctx context.Context, token string) (Identity, error) {
	if len(token) != 43 {
		return Identity{}, ErrSession
	}
	var session Session
	err := s.db.WithContext(ctx).Preload("User").Where(&Session{TokenHash: Digest(token)}).
		Where(clause.Gt{Column: "expires_at", Value: time.Now().UTC()}).Take(&session).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Identity{}, ErrSession
	}
	if err != nil {
		return Identity{}, err
	}
	return Identity{User: contracts.User{Id: session.User.ID, Login: session.User.Login}, TokenHash: session.TokenHash, ExpiresAt: session.ExpiresAt}, nil
}

func (s *Service) Logout(ctx context.Context, token string) error {
	return s.db.WithContext(ctx).Where(&Session{TokenHash: Digest(token)}).Delete(&Session{}).Error
}
