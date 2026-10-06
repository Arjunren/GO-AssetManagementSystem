package service

import (
	"context"
	"errors"
	"github.com/Arjunren/asset-management-system/internal/domain"
	"github.com/Arjunren/asset-management-system/internal/store"
	"testing"
	"time"
)

type fake struct {
	store.Store
	asset domain.Asset
	user  domain.User
}

func (f *fake) AssetByID(context.Context, int64) (domain.Asset, error) { return f.asset, nil }
func (f *fake) UserByID(context.Context, int64) (domain.User, error)   { return f.user, nil }
func TestEmployeeOwnership(t *testing.T) {
	owner := int64(2)
	s := New(&fake{asset: domain.Asset{ID: 1, AssignedTo: &owner}}, "0123456789abcdef0123456789abcdef", time.Minute)
	_, e := s.GetAsset(context.Background(), 1, 3, domain.RoleEmployee)
	if !errors.Is(e, ErrForbidden) {
		t.Fatalf("expected forbidden, got %v", e)
	}
}
func TestAssignmentRequiresActiveUser(t *testing.T) {
	id := int64(4)
	s := New(&fake{user: domain.User{ID: 4, IsActive: false}}, "0123456789abcdef0123456789abcdef", time.Minute)
	_, e := s.Transition(context.Background(), 1, domain.StatusAssigned, &id, "", 2)
	var v ValidationError
	if !errors.As(e, &v) {
		t.Fatalf("expected validation error, got %v", e)
	}
}
