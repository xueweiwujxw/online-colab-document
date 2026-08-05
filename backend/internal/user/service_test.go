package user

import (
	"context"
	"testing"
)

type searchRepository struct {
	users  []User
	limit  int
	offset int
}

func (r *searchRepository) Search(_ context.Context, _ string, limit, offset int) ([]User, error) {
	r.limit = limit
	r.offset = offset
	return r.users, nil
}

func (r *searchRepository) ListAdmin(context.Context, string, int, int) ([]User, error) {
	return nil, nil
}

func TestSearchReturnsPageAndHasMore(t *testing.T) {
	repo := &searchRepository{users: []User{
		{ID: "one", Email: "one@example.test", DisplayName: "一"},
		{ID: "two", Email: "two@example.test", DisplayName: "二"},
		{ID: "three", Email: "three@example.test", DisplayName: "三"},
	}}
	service := NewService(repo)

	result, err := service.Search(context.Background(), "", 2, 4)
	if err != nil {
		t.Fatalf("search users: %v", err)
	}
	if repo.limit != 3 || repo.offset != 4 {
		t.Fatalf("unexpected page request: limit=%d offset=%d", repo.limit, repo.offset)
	}
	if !result.HasMore || len(result.Items) != 2 {
		t.Fatalf("unexpected page result: %#v", result)
	}
	if result.Items[0].ID != "one" || result.Items[1].ID != "two" {
		t.Fatalf("unexpected users: %#v", result.Items)
	}
}
