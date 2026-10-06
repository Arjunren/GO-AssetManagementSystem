package postgres

import (
	"context"
	"errors"
	"fmt"
	"github.com/Arjunren/asset-management-system/internal/domain"
	basestore "github.com/Arjunren/asset-management-system/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"strings"
	"time"
)

type Store struct{ Pool *pgxpool.Pool }

func Open(c context.Context, u string) (*Store, error) {
	p, e := pgxpool.New(c, u)
	if e != nil {
		return nil, e
	}
	if e = p.Ping(c); e != nil {
		p.Close()
		return nil, e
	}
	return &Store{p}, nil
}
func (s *Store) Close() { s.Pool.Close() }
func mapErr(e error) error {
	if errors.Is(e, pgx.ErrNoRows) {
		return basestore.ErrNotFound
	}
	var p *pgconn.PgError
	if errors.As(e, &p) && (p.Code == "23505" || p.Code == "23503" || p.Code == "23514") {
		return basestore.ErrConflict
	}
	return e
}

const userCols = `id,email,password_hash,name,role,is_active,created_at,updated_at`

func (s *Store) UserByEmail(c context.Context, email string) (domain.User, error) {
	var u domain.User
	e := s.Pool.QueryRow(c, `SELECT `+userCols+` FROM users WHERE LOWER(email)=LOWER($1)`, email).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.Role, &u.IsActive, &u.CreatedAt, &u.UpdatedAt)
	return u, mapErr(e)
}
func (s *Store) UserByID(c context.Context, id int64) (domain.User, error) {
	var u domain.User
	e := s.Pool.QueryRow(c, `SELECT `+userCols+` FROM users WHERE id=$1`, id).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.Role, &u.IsActive, &u.CreatedAt, &u.UpdatedAt)
	return u, mapErr(e)
}
func (s *Store) CreateUser(c context.Context, u domain.User) (domain.User, error) {
	e := s.Pool.QueryRow(c, `INSERT INTO users(email,password_hash,name,role)VALUES(LOWER($1),$2,$3,$4)RETURNING `+userCols, u.Email, u.PasswordHash, u.Name, u.Role).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.Role, &u.IsActive, &u.CreatedAt, &u.UpdatedAt)
	return u, mapErr(e)
}
func (s *Store) CreateSessionToken(c context.Context, id string, uid int64, exp time.Time) error {
	_, e := s.Pool.Exec(c, `INSERT INTO sessions(id,user_id,expires_at)VALUES($1,$2,$3)`, id, uid, exp)
	return e
}
func (s *Store) SessionActive(c context.Context, id string, uid int64) (bool, error) {
	var ok bool
	e := s.Pool.QueryRow(c, `SELECT EXISTS(SELECT 1 FROM sessions s JOIN users u ON u.id=s.user_id WHERE s.id=$1 AND s.user_id=$2 AND s.revoked_at IS NULL AND s.expires_at>NOW() AND u.is_active)`, id, uid).Scan(&ok)
	return ok, e
}
func (s *Store) RevokeSession(c context.Context, id string, uid int64) error {
	_, e := s.Pool.Exec(c, `UPDATE sessions SET revoked_at=NOW() WHERE id=$1 AND user_id=$2`, id, uid)
	return e
}

const assetSelect = `SELECT a.id,a.asset_tag,a.name,a.category,a.serial_number,a.status,a.purchase_date,a.purchase_cost,a.assigned_to,COALESCE(u.name,''),a.created_by,a.updated_by,a.created_at,a.updated_at FROM assets a LEFT JOIN users u ON u.id=a.assigned_to`

func scanAsset(r interface{ Scan(...any) error }) (domain.Asset, error) {
	var a domain.Asset
	e := r.Scan(&a.ID, &a.AssetTag, &a.Name, &a.Category, &a.SerialNumber, &a.Status, &a.PurchaseDate, &a.PurchaseCost, &a.AssignedTo, &a.AssigneeName, &a.CreatedBy, &a.UpdatedBy, &a.CreatedAt, &a.UpdatedAt)
	return a, e
}
func (s *Store) AssetByID(c context.Context, id int64) (domain.Asset, error) {
	a, e := scanAsset(s.Pool.QueryRow(c, assetSelect+` WHERE a.id=$1`, id))
	return a, mapErr(e)
}
func (s *Store) CreateAsset(c context.Context, a domain.Asset) (domain.Asset, error) {
	var id int64
	e := s.Pool.QueryRow(c, `INSERT INTO assets(asset_tag,name,category,serial_number,purchase_date,purchase_cost,created_by,updated_by)VALUES(UPPER($1),$2,$3,$4,$5,$6,$7,$7)RETURNING id`, a.AssetTag, a.Name, a.Category, a.SerialNumber, a.PurchaseDate, a.PurchaseCost, a.CreatedBy).Scan(&id)
	if e != nil {
		return a, mapErr(e)
	}
	return s.AssetByID(c, id)
}
func (s *Store) UpdateAsset(c context.Context, a domain.Asset) (domain.Asset, error) {
	res, e := s.Pool.Exec(c, `UPDATE assets SET asset_tag=UPPER($2),name=$3,category=$4,serial_number=$5,purchase_date=$6,purchase_cost=$7,updated_by=$8,updated_at=NOW() WHERE id=$1`, a.ID, a.AssetTag, a.Name, a.Category, a.SerialNumber, a.PurchaseDate, a.PurchaseCost, a.UpdatedBy)
	if e != nil {
		return a, mapErr(e)
	}
	if res.RowsAffected() == 0 {
		return a, basestore.ErrNotFound
	}
	return s.AssetByID(c, a.ID)
}
func (s *Store) ListAssets(c context.Context, p domain.ListParams) ([]domain.Asset, int64, error) {
	clauses := []string{`(LOWER(a.asset_tag) LIKE $1 OR LOWER(a.name) LIKE $1 OR LOWER(a.serial_number) LIKE $1)`}
	args := []any{"%" + strings.ToLower(p.Search) + "%"}
	add := func(q string, v any) { args = append(args, v); clauses = append(clauses, fmt.Sprintf(q, len(args))) }
	if p.Status != "" {
		add(`a.status=$%d`, p.Status)
	}
	if p.Category != "" {
		add(`LOWER(a.category)=LOWER($%d)`, p.Category)
	}
	if p.ActorRole == domain.RoleEmployee {
		add(`a.assigned_to=$%d`, p.ActorID)
	}
	where := " WHERE " + strings.Join(clauses, " AND ")
	var total int64
	if e := s.Pool.QueryRow(c, `SELECT COUNT(*) FROM assets a`+where, args...).Scan(&total); e != nil {
		return nil, 0, e
	}
	args = append(args, p.Limit, (p.Page-1)*p.Limit)
	q := fmt.Sprintf(`%s%s ORDER BY a.updated_at DESC,a.id DESC LIMIT $%d OFFSET $%d`, assetSelect, where, len(args)-1, len(args))
	rows, e := s.Pool.Query(c, q, args...)
	if e != nil {
		return nil, 0, e
	}
	defer rows.Close()
	out := []domain.Asset{}
	for rows.Next() {
		a, e := scanAsset(rows)
		if e != nil {
			return nil, 0, e
		}
		out = append(out, a)
	}
	return out, total, rows.Err()
}
func (s *Store) TransitionAsset(c context.Context, id int64, to domain.AssetStatus, target *int64, notes string, actor int64) (domain.Asset, error) {
	tx, e := s.Pool.Begin(c)
	if e != nil {
		return domain.Asset{}, e
	}
	defer func() { _ = tx.Rollback(c) }()
	var from domain.AssetStatus
	var current *int64
	if e = tx.QueryRow(c, `SELECT status,assigned_to FROM assets WHERE id=$1 FOR UPDATE`, id).Scan(&from, &current); e != nil {
		return domain.Asset{}, mapErr(e)
	}
	allowed := false
	event := domain.EventAvailable
	switch {
	case from == domain.StatusAvailable && to == domain.StatusAssigned && target != nil:
		allowed = true
		event = domain.EventAssigned
	case from == domain.StatusAssigned && to == domain.StatusAvailable && target == nil:
		allowed = true
		event = domain.EventReturned
	case from == domain.StatusAvailable && to == domain.StatusMaintenance && target == nil:
		allowed = true
		event = domain.EventMaintenance
	case from == domain.StatusMaintenance && to == domain.StatusAvailable && target == nil:
		allowed = true
		event = domain.EventAvailable
	case (from == domain.StatusAvailable || from == domain.StatusMaintenance) && to == domain.StatusRetired && target == nil:
		allowed = true
		event = domain.EventRetired
	}
	if !allowed {
		return domain.Asset{}, basestore.ErrUnavailable
	}
	if _, e = tx.Exec(c, `UPDATE assets SET status=$2,assigned_to=$3,updated_by=$4,updated_at=NOW() WHERE id=$1`, id, to, target, actor); e != nil {
		return domain.Asset{}, mapErr(e)
	}
	if _, e = tx.Exec(c, `INSERT INTO asset_events(asset_id,type,from_user_id,to_user_id,notes,performed_by)VALUES($1,$2,$3,$4,$5,$6)`, id, event, current, target, notes, actor); e != nil {
		return domain.Asset{}, mapErr(e)
	}
	if e = tx.Commit(c); e != nil {
		return domain.Asset{}, e
	}
	return s.AssetByID(c, id)
}
func (s *Store) ListEvents(c context.Context, id int64) ([]domain.AssetEvent, error) {
	rows, e := s.Pool.Query(c, `SELECT e.id,e.asset_id,e.type,e.from_user_id,e.to_user_id,e.notes,e.performed_by,u.name,e.created_at FROM asset_events e JOIN users u ON u.id=e.performed_by WHERE e.asset_id=$1 ORDER BY e.created_at DESC,e.id DESC`, id)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []domain.AssetEvent{}
	for rows.Next() {
		var x domain.AssetEvent
		if e := rows.Scan(&x.ID, &x.AssetID, &x.Type, &x.FromUserID, &x.ToUserID, &x.Notes, &x.PerformedBy, &x.PerformerName, &x.CreatedAt); e != nil {
			return nil, e
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
func (s *Store) Dashboard(c context.Context, uid int64, role domain.Role) (domain.Dashboard, error) {
	where := ""
	args := []any{}
	if role == domain.RoleEmployee {
		where = " WHERE assigned_to=$1"
		args = append(args, uid)
	}
	var d domain.Dashboard
	q := `SELECT COUNT(*),COUNT(*)FILTER(WHERE status='AVAILABLE'),COUNT(*)FILTER(WHERE status='ASSIGNED'),COUNT(*)FILTER(WHERE status='MAINTENANCE'),COUNT(*)FILTER(WHERE status='RETIRED'),COALESCE(SUM(purchase_cost),0) FROM assets` + where
	e := s.Pool.QueryRow(c, q, args...).Scan(&d.Total, &d.Available, &d.Assigned, &d.Maintenance, &d.Retired, &d.TotalValue)
	return d, e
}
