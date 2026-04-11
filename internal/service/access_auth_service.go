package service

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

	"neko-tool/internal/config"
)

const userAuthHeader = "X-Neko-Auth-Key"

type AccessAuthService struct {
	configManager *config.ApplicationConfigManager
}

type AuthLoginResult struct {
	AuthKey string `json:"authKey"`
}

func NewAccessAuthService(configManager *config.ApplicationConfigManager) *AccessAuthService {
	return &AccessAuthService{configManager: configManager}
}

func UserAuthHeader() string {
	return userAuthHeader
}

func (s *AccessAuthService) AccessToken() string {
	return strings.TrimSpace(s.configManager.GetConfig().AuthConfig.AccessToken)
}

func (s *AccessAuthService) ValidateAccessToken(input string) bool {
	configured := s.AccessToken()
	provided := strings.TrimSpace(input)
	if configured == "" || provided == "" {
		return false
	}
	return hmac.Equal([]byte(configured), []byte(provided))
}

func (s *AccessAuthService) IssueAuthKey(input string) (AuthLoginResult, error) {
	if !s.ValidateAccessToken(input) {
		return AuthLoginResult{}, fmt.Errorf("访问 token 不正确")
	}
	nonce, err := generateAuthNonce()
	if err != nil {
		return AuthLoginResult{}, err
	}
	signature := s.signPayload(nonce)
	token := base64.RawURLEncoding.EncodeToString([]byte(nonce)) + "." + signature
	return AuthLoginResult{
		AuthKey: token,
	}, nil
}

func (s *AccessAuthService) VerifyAuthKey(authKey string) bool {
	trimmedKey := strings.TrimSpace(authKey)
	parts := strings.Split(trimmedKey, ".")
	if len(parts) != 2 {
		return false
	}
	nonceBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}
	nonce := string(nonceBytes)
	return hmac.Equal([]byte(s.signPayload(nonce)), []byte(parts[1]))
}

func (s *AccessAuthService) signPayload(payload string) string {
	mac := hmac.New(sha256.New, []byte(s.AccessToken()))
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

func generateAuthNonce() (string, error) {
	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}
