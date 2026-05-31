package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

func Open(path string) (*sql.DB, error) {
	dsn := fmt.Sprintf("%s?_foreign_keys=on&_busy_timeout=5000", path)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}

func Migrate(db *sql.DB, adminUser, adminPass string) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS admins (id INTEGER PRIMARY KEY AUTOINCREMENT, username TEXT NOT NULL UNIQUE, password_hash TEXT NOT NULL, created_at TEXT NOT NULL);`,
		`CREATE TABLE IF NOT EXISTS apps (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, app_key TEXT NOT NULL UNIQUE, status TEXT NOT NULL DEFAULT 'active', version INTEGER NOT NULL DEFAULT 1, min_version INTEGER NOT NULL DEFAULT 0, force_update INTEGER NOT NULL DEFAULT 0, force_update_message TEXT NOT NULL DEFAULT '', download_url TEXT NOT NULL DEFAULT '', backup_download_url TEXT NOT NULL DEFAULT '', announcement TEXT NOT NULL DEFAULT '', heartbeat_interval INTEGER NOT NULL DEFAULT 60, heartbeat_timeout INTEGER NOT NULL DEFAULT 180, client_secret TEXT NOT NULL, created_at TEXT NOT NULL, updated_at TEXT NOT NULL);`,
		`CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY AUTOINCREMENT, app_id INTEGER NOT NULL, username TEXT NOT NULL, password_hash TEXT NOT NULL, status TEXT NOT NULL DEFAULT 'active', expire_at TEXT, machine_code TEXT NOT NULL DEFAULT '', max_devices INTEGER NOT NULL DEFAULT 1, free_unbinds INTEGER NOT NULL DEFAULT 0, max_unbinds INTEGER NOT NULL DEFAULT 0, unbind_used INTEGER NOT NULL DEFAULT 0, unbind_deduct_hours INTEGER NOT NULL DEFAULT 0, created_at TEXT NOT NULL, updated_at TEXT NOT NULL, UNIQUE(app_id, username), FOREIGN KEY(app_id) REFERENCES apps(id));`,
		`CREATE TABLE IF NOT EXISTS card_types (id INTEGER PRIMARY KEY AUTOINCREMENT, app_id INTEGER NOT NULL, name TEXT NOT NULL, days INTEGER NOT NULL DEFAULT 0, hours INTEGER NOT NULL DEFAULT 0, max_devices INTEGER NOT NULL DEFAULT 1, free_unbinds INTEGER NOT NULL DEFAULT 0, max_unbinds INTEGER NOT NULL DEFAULT 0, unbind_deduct_hours INTEGER NOT NULL DEFAULT 0, price REAL NOT NULL DEFAULT 0, status TEXT NOT NULL DEFAULT 'active', created_at TEXT NOT NULL, updated_at TEXT NOT NULL, FOREIGN KEY(app_id) REFERENCES apps(id));`,
		`CREATE TABLE IF NOT EXISTS cards (id INTEGER PRIMARY KEY AUTOINCREMENT, app_id INTEGER NOT NULL, card_type_id INTEGER, card_key TEXT NOT NULL UNIQUE, status TEXT NOT NULL DEFAULT 'unused', used_by_user_id INTEGER, used_at TEXT, machine_code TEXT NOT NULL DEFAULT '', expire_days INTEGER NOT NULL DEFAULT 0, expire_hours INTEGER NOT NULL DEFAULT 0, max_devices INTEGER NOT NULL DEFAULT 1, free_unbinds INTEGER NOT NULL DEFAULT 0, max_unbinds INTEGER NOT NULL DEFAULT 0, unbind_used INTEGER NOT NULL DEFAULT 0, unbind_deduct_hours INTEGER NOT NULL DEFAULT 0, created_by TEXT NOT NULL DEFAULT 'admin', agent_id INTEGER, cost REAL NOT NULL DEFAULT 0, created_at TEXT NOT NULL, FOREIGN KEY(app_id) REFERENCES apps(id), FOREIGN KEY(card_type_id) REFERENCES card_types(id));`,
		`CREATE TABLE IF NOT EXISTS devices (id INTEGER PRIMARY KEY AUTOINCREMENT, app_id INTEGER NOT NULL, user_id INTEGER NOT NULL DEFAULT 0, card_id INTEGER NOT NULL DEFAULT 0, auth_mode TEXT NOT NULL DEFAULT 'account', machine_code TEXT NOT NULL, status TEXT NOT NULL DEFAULT 'active', client_version TEXT NOT NULL DEFAULT '', ip TEXT NOT NULL DEFAULT '', last_seen TEXT NOT NULL, created_at TEXT NOT NULL, UNIQUE(app_id, auth_mode, user_id, card_id, machine_code), FOREIGN KEY(app_id) REFERENCES apps(id));`,
		`CREATE TABLE IF NOT EXISTS request_logs (id INTEGER PRIMARY KEY AUTOINCREMENT, app_id INTEGER, user_id INTEGER, path TEXT NOT NULL, ip TEXT NOT NULL, created_at TEXT NOT NULL);`,
		`CREATE TABLE IF NOT EXISTS client_nonces (nonce TEXT PRIMARY KEY, app_key TEXT NOT NULL DEFAULT '', created_at INTEGER NOT NULL);`,
		`CREATE TABLE IF NOT EXISTS cloud_vars (id INTEGER PRIMARY KEY AUTOINCREMENT, app_id INTEGER NOT NULL, var_key TEXT NOT NULL, var_value TEXT NOT NULL DEFAULT '', value_type TEXT NOT NULL DEFAULT 'text', status TEXT NOT NULL DEFAULT 'active', remark TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL, updated_at TEXT NOT NULL, UNIQUE(app_id, var_key), FOREIGN KEY(app_id) REFERENCES apps(id));`,
		`CREATE TABLE IF NOT EXISTS agents (id INTEGER PRIMARY KEY AUTOINCREMENT, username TEXT NOT NULL UNIQUE, password_hash TEXT NOT NULL, status TEXT NOT NULL DEFAULT 'active', balance REAL NOT NULL DEFAULT 0, remark TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL, updated_at TEXT NOT NULL);`,
		`CREATE TABLE IF NOT EXISTS agent_balance_logs (id INTEGER PRIMARY KEY AUTOINCREMENT, agent_id INTEGER NOT NULL, type TEXT NOT NULL, amount REAL NOT NULL, before_balance REAL NOT NULL, after_balance REAL NOT NULL, remark TEXT NOT NULL DEFAULT '', operator TEXT NOT NULL DEFAULT 'admin', created_at TEXT NOT NULL, FOREIGN KEY(agent_id) REFERENCES agents(id));`,
		`CREATE TABLE IF NOT EXISTS agent_app_scopes (agent_id INTEGER NOT NULL, app_id INTEGER NOT NULL, created_at TEXT NOT NULL, PRIMARY KEY(agent_id, app_id), FOREIGN KEY(agent_id) REFERENCES agents(id), FOREIGN KEY(app_id) REFERENCES apps(id));`,
		`CREATE TABLE IF NOT EXISTS agent_card_type_scopes (agent_id INTEGER NOT NULL, card_type_id INTEGER NOT NULL, created_at TEXT NOT NULL, PRIMARY KEY(agent_id, card_type_id), FOREIGN KEY(agent_id) REFERENCES agents(id), FOREIGN KEY(card_type_id) REFERENCES card_types(id));`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return err
		}
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM admins`).Scan(&count); err != nil {
		return err
	}
	beijingLocation := time.FixedZone("CST", 8*3600)
	now := time.Now().In(beijingLocation).Format("2006-01-02 15:04:05")
	if count == 0 && adminUser != "" && adminPass != "" {
		h, _ := bcrypt.GenerateFromPassword([]byte(adminPass), bcrypt.DefaultCost)
		if _, err := db.Exec(`INSERT INTO admins(username,password_hash,created_at) VALUES(?,?,?)`, adminUser, string(h), now); err != nil {
			return err
		}
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM apps`).Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		_, err := db.Exec(`INSERT INTO apps(name,app_key,status,version,announcement,client_secret,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?)`, "默认软件", "demo-app", "active", 1, "欢迎使用 License SaaS 授权平台", "demo-client-secret-change-me", now, now)
		if err != nil {
			return err
		}
		_, _ = db.Exec(`INSERT INTO card_types(app_id,name,days,hours,max_devices,free_unbinds,max_unbinds,unbind_deduct_hours,price,status,created_at,updated_at) VALUES(1,'月卡',30,720,1,1,3,24,29,'active',?,?)`, now, now)
	}
	migrations := []string{
		`ALTER TABLE cards ADD COLUMN agent_id INTEGER`,
		`ALTER TABLE cards ADD COLUMN cost REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE apps ADD COLUMN min_version INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE apps ADD COLUMN force_update_message TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE apps ADD COLUMN backup_download_url TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE apps ADD COLUMN heartbeat_interval INTEGER NOT NULL DEFAULT 60`,
		`ALTER TABLE apps ADD COLUMN heartbeat_timeout INTEGER NOT NULL DEFAULT 180`,
		`ALTER TABLE card_types ADD COLUMN hours INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE cards ADD COLUMN expire_hours INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE card_types ADD COLUMN free_unbinds INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE card_types ADD COLUMN max_unbinds INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE card_types ADD COLUMN unbind_deduct_hours INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE cards ADD COLUMN free_unbinds INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE cards ADD COLUMN max_unbinds INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE cards ADD COLUMN unbind_used INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE cards ADD COLUMN unbind_deduct_hours INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE cards ADD COLUMN machine_code TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE users ADD COLUMN free_unbinds INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE users ADD COLUMN max_unbinds INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE users ADD COLUMN unbind_used INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE users ADD COLUMN unbind_deduct_hours INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE devices ADD COLUMN card_id INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE devices ADD COLUMN auth_mode TEXT NOT NULL DEFAULT 'account'`,
	}
	for _, m := range migrations {
		_, _ = db.Exec(m)
	}
	if err := migrateDevicesForCardMode(db); err != nil {
		return err
	}
	_, _ = db.Exec(`UPDATE card_types SET hours=days*24 WHERE hours=0 AND days>0`)
	_, _ = db.Exec(`UPDATE cards SET expire_hours=expire_days*24 WHERE expire_hours=0 AND expire_days>0`)
	return nil
}

func migrateDevicesForCardMode(db *sql.DB) error {
	rows, err := db.Query(`PRAGMA foreign_key_list(devices)`)
	if err != nil {
		return err
	}
	needsRebuild := false
	for rows.Next() {
		var id, seq int
		var table, from, to, onUpdate, onDelete, match string
		if err := rows.Scan(&id, &seq, &table, &from, &to, &onUpdate, &onDelete, &match); err != nil {
			rows.Close()
			return err
		}
		if table == "users" {
			needsRebuild = true
		}
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if !needsRebuild {
		return nil
	}
	if _, err := db.Exec(`PRAGMA foreign_keys=OFF`); err != nil {
		return err
	}
	tx, err := db.Begin()
	if err != nil {
		_, _ = db.Exec(`PRAGMA foreign_keys=ON`)
		return err
	}
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS devices_new (id INTEGER PRIMARY KEY AUTOINCREMENT, app_id INTEGER NOT NULL, user_id INTEGER NOT NULL DEFAULT 0, card_id INTEGER NOT NULL DEFAULT 0, auth_mode TEXT NOT NULL DEFAULT 'account', machine_code TEXT NOT NULL, status TEXT NOT NULL DEFAULT 'active', client_version TEXT NOT NULL DEFAULT '', ip TEXT NOT NULL DEFAULT '', last_seen TEXT NOT NULL, created_at TEXT NOT NULL, UNIQUE(app_id, auth_mode, user_id, card_id, machine_code), FOREIGN KEY(app_id) REFERENCES apps(id));`,
		`INSERT OR IGNORE INTO devices_new(id,app_id,user_id,card_id,auth_mode,machine_code,status,client_version,ip,last_seen,created_at) SELECT id,app_id,COALESCE(user_id,0),COALESCE(card_id,0),COALESCE(auth_mode,'account'),machine_code,status,client_version,ip,last_seen,created_at FROM devices;`,
		`DROP TABLE devices;`,
		`ALTER TABLE devices_new RENAME TO devices;`,
	}
	for _, stmt := range stmts {
		if _, err := tx.Exec(stmt); err != nil {
			tx.Rollback()
			_, _ = db.Exec(`PRAGMA foreign_keys=ON`)
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		_, _ = db.Exec(`PRAGMA foreign_keys=ON`)
		return err
	}
	_, err = db.Exec(`PRAGMA foreign_keys=ON`)
	return err
}
