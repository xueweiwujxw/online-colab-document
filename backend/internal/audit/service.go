package audit

import (
	"context"
	"crypto/rand"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

type Service struct {
	repo  Repository
	now   func() time.Time
	newID func() (string, error)
}

func NewService(repo Repository) *Service {
	return &Service{
		repo:  repo,
		now:   time.Now,
		newID: newUUID,
	}
}

func (s *Service) Record(ctx context.Context, input RecordInput) error {
	if input.Action == "" || input.TargetType == "" || input.TargetID == "" {
		return nil
	}
	id, err := s.newID()
	if err != nil {
		return err
	}
	ipAddr := nullable(input.IPAddr)
	userAgent := nullable(input.UserAgent)
	return s.repo.Create(ctx, Log{
		ID:          id,
		ActorUserID: input.ActorUserID,
		Action:      input.Action,
		TargetType:  input.TargetType,
		TargetID:    input.TargetID,
		IPAddr:      ipAddr,
		UserAgent:   userAgent,
		Metadata:    sanitizeMetadata(input.Metadata),
		CreatedAt:   s.now().UTC(),
	})
}

func (s *Service) List(ctx context.Context, filter ListFilter) ([]Log, error) {
	return s.repo.List(ctx, filter)
}

func RequestInfo(r *http.Request) (string, string) {
	ip := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if ip != "" {
		parts := strings.Split(ip, ",")
		ip = strings.TrimSpace(parts[0])
	} else {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err == nil {
			ip = host
		} else {
			ip = r.RemoteAddr
		}
	}
	return ip, r.UserAgent()
}

func nullable(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func sanitizeMetadata(metadata map[string]any) map[string]any {
	if metadata == nil {
		return map[string]any{}
	}
	clean := make(map[string]any, len(metadata))
	for key, value := range metadata {
		lower := strings.ToLower(key)
		if strings.Contains(lower, "password") ||
			strings.Contains(lower, "token") ||
			strings.Contains(lower, "secret") ||
			strings.Contains(lower, "cookie") {
			continue
		}
		clean[key] = value
	}
	return clean
}

func newUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate uuid: %w", err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}
