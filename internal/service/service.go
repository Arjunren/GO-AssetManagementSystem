package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/Arjunren/asset-management-system/internal/domain"
	"github.com/Arjunren/asset-management-system/internal/store"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"net/mail"
	"strings"
	"time"
)

var (
	ErrCredentials  = errors.New("invalid credentials")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
)

type ValidationError struct{ Fields map[string]string }

func (e ValidationError) Error() string { return "validation failed" }
func invalid(k, v string) error         { return ValidationError{map[string]string{k: v}} }

type Claims struct {
	Role domain.Role `json:"role"`
	jwt.RegisteredClaims
}
type Service struct {
	db     store.Store
	secret []byte
	expiry time.Duration
	dummy  []byte
	now    func() time.Time
}

func New(db store.Store, secret string, expiry time.Duration) *Service {
	d, _ := bcrypt.GenerateFromPassword([]byte("dummy"), bcrypt.DefaultCost)
	return &Service{db, []byte(secret), expiry, d, time.Now}
}
func (s *Service) CreateUser(c context.Context, name, email, password string, role domain.Role) (domain.User, error) {
	name = strings.TrimSpace(name)
	email = strings.ToLower(strings.TrimSpace(email))
	a, e := mail.ParseAddress(email)
	if len(name) < 2 || len(name) > 120 {
		return domain.User{}, invalid("name", "must contain 2 to 120 characters")
	}
	if e != nil || a.Address != email {
		return domain.User{}, invalid("email", "must be valid")
	}
	if len(password) < 12 || len(password) > 72 {
		return domain.User{}, invalid("password", "must contain 12 to 72 characters")
	}
	if role != domain.RoleAdmin && role != domain.RoleCustodian && role != domain.RoleEmployee {
		return domain.User{}, invalid("role", "is invalid")
	}
	h, e := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if e != nil {
		return domain.User{}, e
	}
	return s.db.CreateUser(c, domain.User{Name: name, Email: email, PasswordHash: string(h), Role: role})
}
func (s *Service) Login(c context.Context, email, password string) (string, domain.User, error) {
	u, e := s.db.UserByEmail(c, email)
	if e != nil {
		_ = bcrypt.CompareHashAndPassword(s.dummy, []byte(password))
		return "", domain.User{}, ErrCredentials
	}
	if !u.IsActive || bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
		return "", domain.User{}, ErrCredentials
	}
	now := s.now().UTC()
	id := uuid.NewString()
	cl := Claims{u.Role, jwt.RegisteredClaims{Subject: fmt.Sprint(u.ID), ID: id, IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(s.expiry))}}
	token, e := jwt.NewWithClaims(jwt.SigningMethodHS256, cl).SignedString(s.secret)
	if e != nil {
		return "", u, e
	}
	if e = s.db.CreateSessionToken(c, id, u.ID, now.Add(s.expiry)); e != nil {
		return "", u, e
	}
	return token, u, nil
}
func (s *Service) Verify(c context.Context, raw string) (Claims, error) {
	var cl Claims
	t, e := jwt.ParseWithClaims(raw, &cl, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, ErrUnauthorized
		}
		return s.secret, nil
	}, jwt.WithExpirationRequired(), jwt.WithValidMethods([]string{"HS256"}))
	if e != nil || !t.Valid {
		return cl, ErrUnauthorized
	}
	var uid int64
	if _, e = fmt.Sscan(cl.Subject, &uid); e != nil {
		return cl, ErrUnauthorized
	}
	ok, e := s.db.SessionActive(c, cl.ID, uid)
	if e != nil || !ok {
		return cl, ErrUnauthorized
	}
	return cl, nil
}
func (s *Service) Logout(c context.Context, cl Claims) error {
	var id int64
	_, _ = fmt.Sscan(cl.Subject, &id)
	return s.db.RevokeSession(c, cl.ID, id)
}
func (s *Service) SaveAsset(c context.Context, a domain.Asset) (domain.Asset, error) {
	a.AssetTag = strings.ToUpper(strings.TrimSpace(a.AssetTag))
	a.Name = strings.TrimSpace(a.Name)
	a.Category = strings.TrimSpace(a.Category)
	a.SerialNumber = strings.ToUpper(strings.TrimSpace(a.SerialNumber))
	if len(a.AssetTag) < 2 || len(a.AssetTag) > 40 {
		return a, invalid("asset_tag", "must contain 2 to 40 characters")
	}
	if len(a.Name) < 2 || len(a.Name) > 160 {
		return a, invalid("name", "must contain 2 to 160 characters")
	}
	if len(a.Category) < 2 || len(a.Category) > 100 {
		return a, invalid("category", "must contain 2 to 100 characters")
	}
	if len(a.SerialNumber) > 120 {
		return a, invalid("serial_number", "must not exceed 120 characters")
	}
	if a.PurchaseCost < 0 || a.PurchaseCost > 9999999999 {
		return a, invalid("purchase_cost", "is out of range")
	}
	if a.ID == 0 {
		return s.db.CreateAsset(c, a)
	}
	return s.db.UpdateAsset(c, a)
}
func (s *Service) GetAsset(c context.Context, id, actor int64, role domain.Role) (domain.Asset, error) {
	a, e := s.db.AssetByID(c, id)
	if e != nil {
		return a, e
	}
	if role == domain.RoleEmployee && (a.AssignedTo == nil || *a.AssignedTo != actor) {
		return domain.Asset{}, ErrForbidden
	}
	return a, nil
}
func (s *Service) ListAssets(c context.Context, p domain.ListParams) ([]domain.Asset, domain.Page, error) {
	x, n, e := s.db.ListAssets(c, p)
	return x, domain.NewPage(p, n), e
}
func (s *Service) Transition(c context.Context, id int64, to domain.AssetStatus, target *int64, notes string, actor int64) (domain.Asset, error) {
	notes = strings.TrimSpace(notes)
	if len(notes) > 1000 {
		return domain.Asset{}, invalid("notes", "must not exceed 1000 characters")
	}
	if to == domain.StatusAssigned {
		if target == nil || *target < 1 {
			return domain.Asset{}, invalid("assigned_to", "is required")
		}
		u, e := s.db.UserByID(c, *target)
		if e != nil {
			return domain.Asset{}, e
		}
		if !u.IsActive {
			return domain.Asset{}, invalid("assigned_to", "must be an active user")
		}
	} else if target != nil {
		return domain.Asset{}, invalid("assigned_to", "must be omitted for this transition")
	}
	if to != domain.StatusAssigned && to != domain.StatusAvailable && to != domain.StatusMaintenance && to != domain.StatusRetired {
		return domain.Asset{}, invalid("status", "is invalid")
	}
	return s.db.TransitionAsset(c, id, to, target, notes, actor)
}
func (s *Service) Events(c context.Context, id, actor int64, role domain.Role) ([]domain.AssetEvent, error) {
	if _, e := s.GetAsset(c, id, actor, role); e != nil {
		return nil, e
	}
	return s.db.ListEvents(c, id)
}
func (s *Service) Dashboard(c context.Context, id int64, role domain.Role) (domain.Dashboard, error) {
	return s.db.Dashboard(c, id, role)
}
