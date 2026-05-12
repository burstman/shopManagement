package db

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"shopDashboard/app/config"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var pool *pgxpool.Pool

func Connect(cfg *config.Config) error {
	connStr := cfg.DatabaseURL
	if connStr == "" {
		connStr = fmt.Sprintf("host=%s port=%s user=%s dbname=%s sslmode=disable",
			cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBName)
		if cfg.DBPassword != "" {
			connStr += " password=" + cfg.DBPassword
		}
	}
	var err error
	pool, err = pgxpool.New(context.Background(), connStr)
	if err != nil {
		return fmt.Errorf("unable to connect to database: %w", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		return fmt.Errorf("unable to ping database: %w", err)
	}
	return AutoMigrate()
}

func Close() {
	if pool != nil {
		pool.Close()
	}
}

func GetPool() *pgxpool.Pool {
	return pool
}

func AutoMigrate() error {
	_, err := pool.Exec(context.Background(),
		`ALTER TABLE affiliates ADD COLUMN IF NOT EXISTS api_key TEXT`)
	if err != nil {
		return fmt.Errorf("failed to add api_key column: %w", err)
	}

	_, err = pool.Exec(context.Background(),
		`ALTER TABLE affiliates ADD COLUMN IF NOT EXISTS dashboard_url TEXT DEFAULT ''`)
	if err != nil {
		return fmt.Errorf("failed to add dashboard_url column: %w", err)
	}

	_, err = pool.Exec(context.Background(),
		`ALTER TABLE affiliates ADD COLUMN IF NOT EXISTS shop_url TEXT DEFAULT ''`)
	if err != nil {
		return fmt.Errorf("failed to add shop_url column: %w", err)
	}

	_, err = pool.Exec(context.Background(),
		`CREATE TABLE IF NOT EXISTS api_errors (
			id SERIAL PRIMARY KEY,
			affiliate_id INTEGER NOT NULL REFERENCES affiliates(id),
			error_type TEXT NOT NULL,
			message TEXT NOT NULL,
			details TEXT DEFAULT '',
			stack TEXT DEFAULT '',
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`)
	if err != nil {
		return fmt.Errorf("failed to create api_errors table: %w", err)
	}

	_, err = pool.Exec(context.Background(),
		`ALTER TABLE api_errors ADD COLUMN IF NOT EXISTS stack TEXT DEFAULT ''`)
	if err != nil {
		return fmt.Errorf("failed to add stack column: %w", err)
	}

	_, err = pool.Exec(context.Background(),
		`CREATE INDEX IF NOT EXISTS idx_api_errors_affiliate_id ON api_errors(affiliate_id)`)
	if err != nil {
		return fmt.Errorf("failed to create index on api_errors: %w", err)
	}

	_, err = pool.Exec(context.Background(),
		`CREATE TABLE IF NOT EXISTS api_warnings (
			id SERIAL PRIMARY KEY,
			affiliate_id INTEGER NOT NULL REFERENCES affiliates(id),
			warn_type TEXT NOT NULL,
			message TEXT NOT NULL,
			details TEXT DEFAULT '',
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`)
	if err != nil {
		return fmt.Errorf("failed to create api_warnings table: %w", err)
	}

	_, err = pool.Exec(context.Background(),
		`CREATE INDEX IF NOT EXISTS idx_api_warnings_affiliate_id ON api_warnings(affiliate_id)`)
	if err != nil {
		return fmt.Errorf("failed to create index on api_warnings: %w", err)
	}

	_, err = pool.Exec(context.Background(),
		`CREATE TABLE IF NOT EXISTS super_admins (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`)
	if err != nil {
		return fmt.Errorf("failed to create super_admins table: %w", err)
	}

	return nil
}

type Affiliate struct {
	ID           int
	AffiliateID  string
	Name         *string
	Email        *string
	ShopURL      string
	Rate         float64
	APIKey       string
	DashboardURL string
}

type AffiliateError struct {
	ID          int       `json:"id"`
	AffiliateID int       `json:"affiliate_id"`
	ErrorType   string    `json:"error_type"`
	Message     string    `json:"message"`
	Details     string    `json:"details"`
	Stack       string    `json:"stack"`
	CreatedAt   time.Time `json:"created_at"`
}

func GenerateAPIKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate api key: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func GenerateAndEnsureAPIKey(affiliateID int) (string, error) {
	var existing string
	err := pool.QueryRow(context.Background(),
		"SELECT COALESCE(api_key, '') FROM affiliates WHERE id = $1", affiliateID,
	).Scan(&existing)
	if err != nil {
		return "", fmt.Errorf("failed to query affiliate %d: %w", affiliateID, err)
	}
	if existing != "" {
		return existing, nil
	}

	key, err := GenerateAPIKey()
	if err != nil {
		return "", err
	}

	_, err = pool.Exec(context.Background(),
		"UPDATE affiliates SET api_key = $1 WHERE id = $2", key, affiliateID)
	if err != nil {
		return "", fmt.Errorf("failed to set api_key for affiliate %d: %w", affiliateID, err)
	}
	return key, nil
}

func GetAffiliate(id int) (*Affiliate, error) {
	var a Affiliate
	err := pool.QueryRow(context.Background(),
		`SELECT id, affiliate_id, name, email, shop_url, rate, COALESCE(api_key, ''), COALESCE(dashboard_url, '')
		 FROM affiliates WHERE id = $1 AND active = true`,
		id,
	).Scan(&a.ID, &a.AffiliateID, &a.Name, &a.Email, &a.ShopURL, &a.Rate, &a.APIKey, &a.DashboardURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get affiliate %d: %w", id, err)
	}
	return &a, nil
}

func GetAffiliateByAPIKey(apiKey string) (*Affiliate, error) {
	var a Affiliate
	err := pool.QueryRow(context.Background(),
		`SELECT id, affiliate_id, name, email, shop_url, rate, COALESCE(api_key, ''), COALESCE(dashboard_url, '')
		 FROM affiliates WHERE api_key = $1 AND active = true`,
		apiKey,
	).Scan(&a.ID, &a.AffiliateID, &a.Name, &a.Email, &a.ShopURL, &a.Rate, &a.APIKey, &a.DashboardURL)
	if err != nil {
		return nil, fmt.Errorf("invalid api key: %w", err)
	}
	return &a, nil
}

func UpdateAffiliateShopURL(id int, shopURL string) error {
	_, err := pool.Exec(context.Background(),
		"UPDATE affiliates SET shop_url = $1 WHERE id = $2 AND active = true",
		shopURL, id,
	)
	if err != nil {
		return fmt.Errorf("failed to update shop_url for affiliate %d: %w", id, err)
	}
	return nil
}

func UpdateAffiliateDashboardURL(id int, url string) error {
	_, err := pool.Exec(context.Background(),
		"UPDATE affiliates SET dashboard_url = $1 WHERE id = $2 AND active = true",
		url, id,
	)
	if err != nil {
		return fmt.Errorf("failed to update dashboard_url for affiliate %d: %w", id, err)
	}
	return nil
}

func DeleteAffiliateCredentials(id int) error {
	var email string
	err := pool.QueryRow(context.Background(),
		"SELECT COALESCE(email, '') FROM affiliates WHERE id = $1 AND active = true", id,
	).Scan(&email)
	if err != nil {
		return fmt.Errorf("failed to look up affiliate %d: %w", id, err)
	}

	_, err = pool.Exec(context.Background(),
		"UPDATE affiliates SET name = NULL, password_hash = NULL WHERE id = $1 AND active = true",
		id,
	)
	if err != nil {
		return fmt.Errorf("failed to reset credentials for affiliate %d: %w", id, err)
	}

	if email != "" {
		_, _ = pool.Exec(context.Background(),
			"DELETE FROM sessions WHERE user_id = (SELECT id FROM users WHERE email = $1)", email)
		_, _ = pool.Exec(context.Background(),
			"DELETE FROM users WHERE email = $1", email)
	}

	return nil
}

func GetAffiliates() ([]Affiliate, error) {
	rows, err := pool.Query(context.Background(),
		`SELECT id, affiliate_id, name, email, shop_url, rate, COALESCE(api_key, ''), COALESCE(dashboard_url, '')
		 FROM affiliates WHERE active = true
		 ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("failed to query affiliates: %w", err)
	}
	defer rows.Close()

	var affiliates []Affiliate
	for rows.Next() {
		var a Affiliate
		if err := rows.Scan(&a.ID, &a.AffiliateID, &a.Name, &a.Email, &a.ShopURL, &a.Rate, &a.APIKey, &a.DashboardURL); err != nil {
			return nil, fmt.Errorf("failed to scan affiliate: %w", err)
		}
		affiliates = append(affiliates, a)
	}
	return affiliates, nil
}

func CreateAffiliateWarn(affiliateID int, warnType, message, details string) (*AffiliateWarn, error) {
	var w AffiliateWarn
	err := pool.QueryRow(context.Background(),
		`INSERT INTO api_warnings (affiliate_id, warn_type, message, details)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, affiliate_id, warn_type, message, details, created_at`,
		affiliateID, warnType, message, details,
	).Scan(&w.ID, &w.AffiliateID, &w.WarnType, &w.Message, &w.Details, &w.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create affiliate warning: %w", err)
	}
	return &w, nil
}

func GetAffiliateWarnings(affiliateID int, limit int) ([]AffiliateWarn, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := pool.Query(context.Background(),
		`SELECT id, affiliate_id, warn_type, message, COALESCE(details, ''), created_at
		 FROM api_warnings
		 WHERE affiliate_id = $1
		 ORDER BY created_at DESC
		 LIMIT $2`,
		affiliateID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query affiliate warnings: %w", err)
	}
	defer rows.Close()

	var warnings []AffiliateWarn
	for rows.Next() {
		var w AffiliateWarn
		if err := rows.Scan(&w.ID, &w.AffiliateID, &w.WarnType, &w.Message, &w.Details, &w.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan affiliate warning: %w", err)
		}
		warnings = append(warnings, w)
	}
	return warnings, nil
}

func CreateAffiliateError(affiliateID int, errorType, message, details, stack string) (*AffiliateError, error) {
	var e AffiliateError
	err := pool.QueryRow(context.Background(),
		`INSERT INTO api_errors (affiliate_id, error_type, message, details, stack)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, affiliate_id, error_type, message, details, stack, created_at`,
		affiliateID, errorType, message, details, stack,
	).Scan(&e.ID, &e.AffiliateID, &e.ErrorType, &e.Message, &e.Details, &e.Stack, &e.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create affiliate error: %w", err)
	}
	return &e, nil
}

func GetAffiliateErrors(affiliateID int, limit int) ([]AffiliateError, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := pool.Query(context.Background(),
		`SELECT id, affiliate_id, error_type, message, details, COALESCE(stack, ''), created_at
		 FROM api_errors
		 WHERE affiliate_id = $1
		 ORDER BY created_at DESC
		 LIMIT $2`,
		affiliateID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query affiliate errors: %w", err)
	}
	defer rows.Close()

	var errors []AffiliateError
	for rows.Next() {
		var e AffiliateError
		if err := rows.Scan(&e.ID, &e.AffiliateID, &e.ErrorType, &e.Message, &e.Details, &e.Stack, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan affiliate error: %w", err)
		}
		errors = append(errors, e)
	}
	return errors, nil
}

type AffiliateWarn struct {
	ID          int       `json:"id"`
	AffiliateID int       `json:"affiliate_id"`
	WarnType    string    `json:"warn_type"`
	Message     string    `json:"message"`
	Details     string    `json:"details"`
	CreatedAt   time.Time `json:"created_at"`
}

type SuperAdmin struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

func HasAnySuperAdmin() (bool, error) {
	var count int
	err := pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM super_admins").Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to count super admins: %w", err)
	}
	return count > 0, nil
}

func CreateSuperAdmin(name, email, passwordHash string) (*SuperAdmin, error) {
	var a SuperAdmin
	err := pool.QueryRow(context.Background(),
		`INSERT INTO super_admins (name, email, password_hash)
		 VALUES ($1, $2, $3)
		 RETURNING id, name, email, password_hash, created_at`,
		name, email, passwordHash,
	).Scan(&a.ID, &a.Name, &a.Email, &a.PasswordHash, &a.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create super admin: %w", err)
	}
	return &a, nil
}

func GetSuperAdminByEmail(email string) (*SuperAdmin, error) {
	var a SuperAdmin
	err := pool.QueryRow(context.Background(),
		`SELECT id, name, email, password_hash, created_at
		 FROM super_admins WHERE email = $1`, email,
	).Scan(&a.ID, &a.Name, &a.Email, &a.PasswordHash, &a.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get super admin by email: %w", err)
	}
	return &a, nil
}

func GetSuperAdminByID(id int) (*SuperAdmin, error) {
	var a SuperAdmin
	err := pool.QueryRow(context.Background(),
		`SELECT id, name, email, password_hash, created_at
		 FROM super_admins WHERE id = $1`, id,
	).Scan(&a.ID, &a.Name, &a.Email, &a.PasswordHash, &a.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get super admin by id: %w", err)
	}
	return &a, nil
}

func GetAllSuperAdmins() ([]SuperAdmin, error) {
	rows, err := pool.Query(context.Background(),
		`SELECT id, name, email, password_hash, created_at
		 FROM super_admins ORDER BY created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("failed to query super admins: %w", err)
	}
	defer rows.Close()

	var admins []SuperAdmin
	for rows.Next() {
		var a SuperAdmin
		if err := rows.Scan(&a.ID, &a.Name, &a.Email, &a.PasswordHash, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan super admin: %w", err)
		}
		admins = append(admins, a)
	}
	return admins, nil
}
