package user

import (
	"context"
	"strings"
)

const (
	defaultSearchLimit = 50
	maxSearchLimit     = 50
)

type Repository interface {
	Search(ctx context.Context, query string, limit, offset int) ([]User, error)
	ListAdmin(ctx context.Context, query string, limit int, offset int) ([]User, error)
}

func (s *Service) ListAdmin(ctx context.Context, query string, limit, offset int) ([]AdminUser, error) {
	if limit <= 0 || limit > maxSearchLimit {
		limit = maxSearchLimit
	}
	if offset < 0 {
		offset = 0
	}
	users, err := s.repo.ListAdmin(ctx, strings.TrimSpace(query), limit, offset)
	if err != nil {
		return nil, err
	}
	items := make([]AdminUser, 0, len(users))
	for _, u := range users {
		items = append(items, ToAdmin(u))
	}
	return items, nil
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

type SearchResult struct {
	Items   []PublicUser
	HasMore bool
}

func (s *Service) Search(ctx context.Context, query string, limit, offset int) (SearchResult, error) {
	if limit <= 0 {
		limit = defaultSearchLimit
	}
	if limit > maxSearchLimit {
		limit = maxSearchLimit
	}
	if offset < 0 {
		offset = 0
	}
	users, err := s.repo.Search(ctx, strings.TrimSpace(query), limit+1, offset)
	if err != nil {
		return SearchResult{}, err
	}
	hasMore := len(users) > limit
	if hasMore {
		users = users[:limit]
	}
	items := make([]PublicUser, 0, len(users))
	for _, u := range users {
		items = append(items, ToPublic(u))
	}
	return SearchResult{Items: items, HasMore: hasMore}, nil
}
