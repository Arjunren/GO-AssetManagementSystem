//go:build integration

package postgres

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"

	"github.com/Arjunren/asset-management-system/internal/domain"
	"github.com/Arjunren/asset-management-system/internal/service"
	basestore "github.com/Arjunren/asset-management-system/internal/store"
	"github.com/Arjunren/asset-management-system/migrations"
)

func TestConcurrentAssignmentAllowsOneWinner(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	db, err := Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := migrations.Apply(ctx, db.Pool); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Pool.Exec(ctx, `TRUNCATE asset_events,assets,sessions,users RESTART IDENTITY CASCADE`); err != nil {
		t.Fatal(err)
	}
	svc := service.New(db, "0123456789abcdef0123456789abcdef", 0)
	admin, _ := svc.CreateUser(ctx, "Admin", "admin@test.local", "TestPassword123!", domain.RoleAdmin)
	first, _ := svc.CreateUser(ctx, "First", "first@test.local", "TestPassword123!", domain.RoleEmployee)
	second, _ := svc.CreateUser(ctx, "Second", "second@test.local", "TestPassword123!", domain.RoleEmployee)
	asset, err := db.CreateAsset(ctx, domain.Asset{AssetTag: "LAP-001", Name: "Laptop", Category: "Computers", CreatedBy: admin.ID})
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for _, target := range []int64{first.ID, second.ID} {
		wg.Add(1)
		go func(target int64) {
			defer wg.Done()
			_, err := db.TransitionAsset(ctx, asset.ID, domain.StatusAssigned, &target, "test", admin.ID)
			errs <- err
		}(target)
	}
	wg.Wait()
	close(errs)
	success, rejected := 0, 0
	for err := range errs {
		if err == nil {
			success++
		} else if errors.Is(err, basestore.ErrUnavailable) {
			rejected++
		} else {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if success != 1 || rejected != 1 {
		t.Fatalf("success=%d rejected=%d", success, rejected)
	}
}
