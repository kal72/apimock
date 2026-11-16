package usecase

import (
	"apimock/internal/domain"
	"context"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

var ErrUnauthorized = errors.New("authentication required")
var ErrCredentials = errors.New("invalid username or password")
var ErrForbidden = errors.New("administrator role required")
var ErrSetupClosed = errors.New("initial setup has already been completed")
var ErrLastAdmin = errors.New("cannot remove or disable the last active administrator")
var userPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{2,63}$`)

const passwordIterations = 600000
const SessionDuration = 12 * time.Hour

type IdentityRepository interface {
	HasUsers(context.Context) (bool, error)
	CreateFirstUser(context.Context, domain.User) error
	Users(context.Context) ([]domain.User, error)
	UserByUsername(context.Context, string) (domain.User, error)
	SaveUser(context.Context, domain.User, string, bool) error
	DeleteUser(context.Context, string, string) error
	CreateSession(context.Context, string, string, string, time.Time) error
	SessionUser(context.Context, string) (domain.User, error)
	DeleteSession(context.Context, string) error
	ChangePassword(context.Context, string, string, string) error
	APIKeys(ctx context.Context, userID string) ([]domain.APIKey, error)
	CreateAPIKey(ctx context.Context, key domain.APIKey) error
	DeleteAPIKey(ctx context.Context, userID, keyID string) error
	AuthenticateAPIKey(ctx context.Context, keyHash string) (domain.User, error)
}
type AuthService struct{ Repo IdentityRepository }
type UserInput struct {
	Username string `json:"username"`
	Name     string `json:"name"`
	Role     string `json:"role"`
	Active   bool   `json:"active"`
	Password string `json:"password"`
}

func randomToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
func TokenHash(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
func CSRFToken(token string) string { return TokenHash("mocklab-csrf:" + token) }
func HashPassword(password string) (string, error) {
	if utf8.RuneCountInString(password) < 8 || len(password) > 1024 {
		return "", fmt.Errorf("%w: password must contain at least 8 characters and at most 1024 bytes", domain.ErrValidation)
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key, err := pbkdf2.Key(sha256.New, password, salt, passwordIterations, 32)
	if err != nil {
		return "", err
	}
	return "pbkdf2-sha256$600000$" + base64.RawStdEncoding.EncodeToString(salt) + "$" + base64.RawStdEncoding.EncodeToString(key), nil
}
func VerifyPassword(encoded, password string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 4 || parts[0] != "pbkdf2-sha256" || parts[1] != "600000" || len(password) > 1024 {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[2])
	if err != nil || len(salt) != 16 {
		return false
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil || len(want) != 32 {
		return false
	}
	key, err := pbkdf2.Key(sha256.New, password, salt, passwordIterations, 32)
	return err == nil && subtle.ConstantTimeCompare(key, want) == 1
}
func (s AuthService) Setup(ctx context.Context, input UserInput) (domain.User, error) {
	u, err := newUser(input)
	if err != nil {
		return u, err
	}
	u.Role = "admin"
	u.Active = true
	err = s.Repo.CreateFirstUser(ctx, u)
	return u, err
}
func (s AuthService) Bootstrap(ctx context.Context, username, password string) error {
	exists, err := s.Repo.HasUsers(ctx)
	if err != nil || exists || username == "" {
		return err
	}
	_, err = s.Setup(ctx, UserInput{Username: username, Name: username, Password: password})
	if errors.Is(err, ErrSetupClosed) {
		return nil
	}
	return err
}
func newUser(input UserInput) (domain.User, error) {
	u := domain.User{ID: randomToken(), Username: strings.ToLower(strings.TrimSpace(input.Username)), Name: strings.TrimSpace(input.Name), Role: input.Role, Active: input.Active}
	if !userPattern.MatchString(u.Username) || u.Name == "" || len(u.Name) > 120 {
		return u, fmt.Errorf("%w: name is required (up to 120 bytes); username must use 3–64 lowercase letters, numbers, dots, underscores or hyphens", domain.ErrValidation)
	}
	if u.Role != "admin" && u.Role != "user" && u.Role != "" {
		return u, fmt.Errorf("%w: role must be admin or user", domain.ErrValidation)
	}
	if u.Role == "" {
		u.Role = "user"
	}
	hash, err := HashPassword(input.Password)
	u.PasswordHash = hash
	return u, err
}
func (s AuthService) Login(ctx context.Context, username, password string) (domain.User, string, error) {
	u, err := s.Repo.UserByUsername(ctx, strings.ToLower(strings.TrimSpace(username)))
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return domain.User{}, "", err
	}
	hash := u.PasswordHash
	if err != nil {
		hash = "pbkdf2-sha256$600000$AAAAAAAAAAAAAAAAAAAAAA$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	}
	valid := VerifyPassword(hash, password)
	if !valid || !u.Active || err != nil {
		return domain.User{}, "", ErrCredentials
	}
	token := randomToken()
	err = s.Repo.CreateSession(ctx, TokenHash(token), u.ID, u.PasswordHash, time.Now().Add(SessionDuration))
	return u, token, err
}
func (s AuthService) Authenticate(ctx context.Context, token string) (domain.User, error) {
	if len(token) != 43 {
		return domain.User{}, ErrUnauthorized
	}
	u, err := s.Repo.SessionUser(ctx, TokenHash(token))
	if errors.Is(err, domain.ErrNotFound) {
		return u, ErrUnauthorized
	}
	return u, err
}
func (s AuthService) SaveUser(ctx context.Context, actor domain.User, id string, input UserInput) (domain.User, error) {
	if actor.Role != "admin" {
		return domain.User{}, ErrForbidden
	}
	// A blank password on update keeps the stored hash; it never becomes an empty password.
	password := input.Password
	if id != "" && password == "" {
		input.Password = "validation-placeholder-password"
	}
	u, err := newUser(input)
	if err != nil {
		return u, err
	}
	if id != "" {
		u.ID = id
		if password == "" {
			u.PasswordHash = ""
		}
	}
	if actor.ID == u.ID && (u.Role != "admin" || !u.Active) {
		return u, fmt.Errorf("%w: you cannot demote or deactivate your own account", domain.ErrValidation)
	}
	return u, s.Repo.SaveUser(ctx, u, actor.ID, id == "")
}
func (s AuthService) DeleteUser(ctx context.Context, actor domain.User, id string) error {
	if actor.Role != "admin" {
		return ErrForbidden
	}
	if actor.ID == id {
		return fmt.Errorf("%w: you cannot delete your own account", domain.ErrValidation)
	}
	return s.Repo.DeleteUser(ctx, id, actor.ID)
}
func (s AuthService) ChangePassword(ctx context.Context, u domain.User, current, next string) error {
	stored, err := s.Repo.UserByUsername(ctx, u.Username)
	if err != nil {
		return err
	}
	if !VerifyPassword(stored.PasswordHash, current) {
		return ErrCredentials
	}
	hash, err := HashPassword(next)
	if err != nil {
		return err
	}
	return s.Repo.ChangePassword(ctx, u.ID, stored.PasswordHash, hash)
}

func (s AuthService) APIKeys(ctx context.Context, userID string) ([]domain.APIKey, error) {
	return s.Repo.APIKeys(ctx, userID)
}

func (s AuthService) CreateAPIKey(ctx context.Context, userID, name string) (domain.GeneratedAPIKey, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return domain.GeneratedAPIKey{}, fmt.Errorf("%w: key name is required", domain.ErrValidation)
	}
	if len(name) > 100 {
		return domain.GeneratedAPIKey{}, fmt.Errorf("%w: key name cannot exceed 100 characters", domain.ErrValidation)
	}

	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return domain.GeneratedAPIKey{}, err
	}
	rawHex := hex.EncodeToString(b)
	rawKey := "mlk_" + rawHex
	prefix := "mlk_" + rawHex[:8] + "..."

	key := domain.APIKey{
		ID:        ID(),
		UserID:    userID,
		Name:      name,
		KeyHash:   TokenHash(rawKey),
		Prefix:    prefix,
		CreatedAt: time.Now(),
	}

	if err := s.Repo.CreateAPIKey(ctx, key); err != nil {
		return domain.GeneratedAPIKey{}, err
	}

	return domain.GeneratedAPIKey{
		APIKey: key,
		RawKey: rawKey,
	}, nil
}

func (s AuthService) DeleteAPIKey(ctx context.Context, userID, keyID string) error {
	return s.Repo.DeleteAPIKey(ctx, userID, keyID)
}

func (s AuthService) AuthenticateAPIKey(ctx context.Context, rawKey string) (domain.User, error) {
	rawKey = strings.TrimSpace(rawKey)
	if rawKey == "" {
		return domain.User{}, ErrUnauthorized
	}
	hash := TokenHash(rawKey)
	return s.Repo.AuthenticateAPIKey(ctx, hash)
}
