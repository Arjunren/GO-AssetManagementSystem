package main

import (
	"context"
	"errors"
	"github.com/Arjunren/asset-management-system/internal/domain"
	"github.com/Arjunren/asset-management-system/internal/postgres"
	"github.com/Arjunren/asset-management-system/internal/service"
	"github.com/Arjunren/asset-management-system/internal/store"
	"log"
	"os"
)

func main() {
	c := context.Background()
	db, e := postgres.Open(c, need("DATABASE_URL"))
	if e != nil {
		log.Fatal(e)
	}
	defer db.Close()
	s := service.New(db, "seed-only-secret-not-used-for-tokens", 0)
	seed(c, s, db, need("SEED_ADMIN_EMAIL"), need("SEED_ADMIN_PASSWORD"), "Development Admin", domain.RoleAdmin)
	seed(c, s, db, need("SEED_CUSTODIAN_EMAIL"), need("SEED_CUSTODIAN_PASSWORD"), "Development Custodian", domain.RoleCustodian)
	seed(c, s, db, need("SEED_EMPLOYEE_EMAIL"), need("SEED_EMPLOYEE_PASSWORD"), "Development Employee", domain.RoleEmployee)
}
func seed(c context.Context, s *service.Service, db store.Store, email, pw, name string, role domain.Role) {
	if _, e := db.UserByEmail(c, email); e == nil {
		return
	} else if !errors.Is(e, store.ErrNotFound) {
		log.Fatal(e)
	}
	if _, e := s.CreateUser(c, name, email, pw, role); e != nil {
		log.Fatal(e)
	}
}
func need(k string) string {
	v := os.Getenv(k)
	if v == "" {
		log.Fatalf("%s is required", k)
	}
	return v
}
