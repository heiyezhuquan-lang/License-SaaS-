package api

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"license-saas/backend/internal/auth"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

var beijingLocation = time.FixedZone("CST", 8*3600)

const displayTimeLayout = "2006-01-02 15:04:05"

type Server struct {
	DB               *sql.DB
	JWTSecret        string
	ClientSignSecret string
}

func New(db *sql.DB, jwtSecret string, clientSignSecret ...string) *Server {
	secret := "demo-client-secret-change-me"
	if len(clientSignSecret) > 0 && clientSignSecret[0] != "" {
		secret = clientSignSecret[0]
	}
	return &Server{DB: db, JWTSecret: jwtSecret, ClientSignSecret: secret}
}

func (s *Server) Router() *gin.Engine {
	r := gin.Default()
	r.Use(cors())
	r.GET("/api/health", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true, "time": now()}) })
	r.GET("/api/setup/status", s.setupStatus)
	r.POST("/api/setup/admin", s.setupAdmin)
	r.POST("/api/admin/login", s.adminLogin)
	admin := r.Group("/api/admin", s.adminAuth())
	admin.GET("/stats", s.stats)
	admin.GET("/apps", s.listApps)
	admin.POST("/apps", s.createApp)
	admin.PUT("/apps/:id", s.updateApp)
	admin.PUT("/apps/:id/secret/reset", s.resetAppSecret)
	admin.GET("/users", s.listUsers)
	admin.POST("/users", s.createUser)
	admin.PUT("/users/:id", s.updateUser)
	admin.PUT("/users/:id/password", s.updateUserPassword)
	admin.PUT("/users/:id/unbind", s.unbindUserMachine)
	admin.DELETE("/users/:id", s.deleteUser)
	admin.GET("/users/:id/devices", s.userDevices)
	admin.GET("/card-types", s.listCardTypes)
	admin.POST("/card-types", s.createCardType)
	admin.DELETE("/card-types/:id", s.deleteCardType)
	admin.GET("/cards", s.listCards)
	admin.GET("/cards/export", s.exportCards)
	admin.GET("/sales", s.adminSalesReport)
	admin.POST("/cards/generate", s.generateCards)
	admin.PUT("/cards/:id/disable", s.disableCard)
	admin.PUT("/cards/:id/enable", s.enableCard)
	admin.GET("/devices", s.listDevices)
	admin.PUT("/devices/:id/status", s.updateDeviceStatus)
	admin.GET("/agents", s.listAgents)
	admin.POST("/agents", s.createAgent)
	admin.PUT("/agents/:id", s.updateAgent)
	admin.PUT("/agents/:id/status", s.updateAgentStatus)
	admin.DELETE("/agents/:id", s.deleteAgent)
	admin.POST("/agents/:id/balance", s.adjustAgentBalance)
	admin.GET("/agents/:id/balance-logs", s.agentBalanceLogs)
	admin.GET("/cloud-vars", s.listCloudVars)
	admin.POST("/cloud-vars", s.createCloudVar)
	admin.PUT("/cloud-vars/:id", s.updateCloudVar)
	admin.DELETE("/cloud-vars/:id", s.deleteCloudVar)
	r.POST("/api/agent/login", s.agentLogin)
	agent := r.Group("/api/agent", s.agentAuth())
	agent.GET("/me", s.agentMe)
	agent.GET("/stats", s.agentStats)
	agent.GET("/balance-logs", s.agentOwnBalanceLogs)
	agent.GET("/scopes", s.agentScopes)
	agent.GET("/cards", s.agentCards)
	agent.GET("/sales", s.agentSalesReport)
	agent.POST("/cards/generate", s.agentGenerateCards)
	client := r.Group("/api/client", s.clientSignAuth())
	client.POST("/register", s.clientRegister)
	client.POST("/login", s.clientLogin)
	client.POST("/card-login", s.clientCardLogin)
	client.POST("/recharge", s.clientRecharge)
	client.POST("/unbind", s.clientUnbind)
	client.POST("/account-unbind", s.clientAccountUnbind)
	client.POST("/card-unbind", s.clientCardUnbind)
	client.POST("/heartbeat", s.clientHeartbeat)
	client.GET("/app-info", s.clientAppInfo)
	client.GET("/cloud-vars", s.clientCloudVars)
	return r
}

func cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-App-Key, X-Timestamp, X-Nonce, X-Signature")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
func errorJSON(c *gin.Context, code int, msg string) {
	c.JSON(code, gin.H{"ok": false, "message": msg})
}

func (s *Server) adminAuth() gin.HandlerFunc {
	return s.roleAuth("admin")
}

func (s *Server) agentAuth() gin.HandlerFunc {
	return s.roleAuth("agent")
}

func (s *Server) roleAuth(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		token := strings.TrimPrefix(h, "Bearer ")
		if token == "" {
			errorJSON(c, 401, "missing token")
			c.Abort()
			return
		}
		claims, err := auth.Parse(s.JWTSecret, token)
		if err != nil || claims.Role != role {
			errorJSON(c, 401, "登录令牌无效或已过期")
			c.Abort()
			return
		}
		if role == "admin" {
			var exists int
			if err := s.DB.QueryRow(`SELECT COUNT(*) FROM admins WHERE username=?`, claims.Username).Scan(&exists); err != nil || exists == 0 {
				errorJSON(c, 401, "登录令牌无效或已过期")
				c.Abort()
				return
			}
		} else if role == "agent" {
			var status string
			if err := s.DB.QueryRow(`SELECT status FROM agents WHERE username=?`, claims.Username).Scan(&status); err != nil || status != "active" {
				errorJSON(c, 401, "登录令牌无效或已过期")
				c.Abort()
				return
			}
		}
		c.Set(role, claims.Username)
		c.Next()
	}
}

func (s *Server) setupStatus(c *gin.Context) {
	var count int
	if err := s.DB.QueryRow(`SELECT COUNT(*) FROM admins`).Scan(&count); err != nil {
		errorJSON(c, 500, err.Error())
		return
	}
	c.JSON(200, gin.H{"ok": true, "installed": count > 0})
}

func (s *Server) setupAdmin(c *gin.Context) {
	var req struct{ Username, Password string }
	if c.ShouldBindJSON(&req) != nil {
		errorJSON(c, 400, "请求格式错误")
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || len(req.Password) < 6 {
		errorJSON(c, 400, "请输入管理员账号，并设置至少 6 位密码")
		return
	}
	tx, err := s.DB.Begin()
	if err != nil {
		errorJSON(c, 500, err.Error())
		return
	}
	defer tx.Rollback()
	var count int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM admins`).Scan(&count); err != nil {
		errorJSON(c, 500, err.Error())
		return
	}
	if count > 0 {
		errorJSON(c, 409, "系统已完成初始化")
		return
	}
	h, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if _, err := tx.Exec(`INSERT INTO admins(username,password_hash,created_at) VALUES(?,?,?)`, req.Username, string(h), now()); err != nil {
		errorJSON(c, 400, err.Error())
		return
	}
	if err := tx.Commit(); err != nil {
		errorJSON(c, 500, err.Error())
		return
	}
	token, _ := auth.Issue(s.JWTSecret, "admin", req.Username, 24*time.Hour)
	c.JSON(200, gin.H{"ok": true, "token": token, "username": req.Username})
}

func (s *Server) adminLogin(c *gin.Context) {
	var req struct{ Username, Password string }
	if c.ShouldBindJSON(&req) != nil {
		errorJSON(c, 400, "bad request")
		return
	}
	var hash string
	if err := s.DB.QueryRow(`SELECT password_hash FROM admins WHERE username=?`, req.Username).Scan(&hash); err != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)) != nil {
		errorJSON(c, 401, "账号或密码错误")
		return
	}
	token, _ := auth.Issue(s.JWTSecret, "admin", req.Username, 24*time.Hour)
	c.JSON(200, gin.H{"ok": true, "token": token, "username": req.Username})
}

func (s *Server) stats(c *gin.Context) {
	counts := gin.H{}
	queries := map[string]string{
		"apps":              "SELECT COUNT(*) FROM apps",
		"active_apps":       "SELECT COUNT(*) FROM apps WHERE status='active'",
		"users":             "SELECT COUNT(*) FROM users",
		"active_users":      "SELECT COUNT(*) FROM users WHERE status='active'",
		"expired_users":     "SELECT COUNT(*) FROM users WHERE expire_at IS NOT NULL AND expire_at!='' AND datetime(expire_at) < datetime('now')",
		"expiring_users":    "SELECT COUNT(*) FROM users WHERE expire_at IS NOT NULL AND expire_at!='' AND datetime(expire_at) BETWEEN datetime('now') AND datetime('now','+7 day')",
		"cards":             "SELECT COUNT(*) FROM cards",
		"unused_cards":      "SELECT COUNT(*) FROM cards WHERE status='unused'",
		"used_cards":        "SELECT COUNT(*) FROM cards WHERE status='used'",
		"disabled_cards":    "SELECT COUNT(*) FROM cards WHERE status='disabled'",
		"devices":           "SELECT COUNT(*) FROM devices",
		"active_devices":    "SELECT COUNT(*) FROM devices WHERE status='active'",
		"today_cards":       "SELECT COUNT(*) FROM cards WHERE date(created_at)=date('now','localtime')",
		"today_activations": "SELECT COUNT(*) FROM cards WHERE used_at IS NOT NULL AND used_at!='' AND date(used_at)=date('now','localtime')",
		"agents":            "SELECT COUNT(*) FROM agents",
		"active_agents":     "SELECT COUNT(*) FROM agents WHERE status='active'",
		"cloud_vars":        "SELECT COUNT(*) FROM cloud_vars",
	}
	for k, q := range queries {
		var n int
		_ = s.DB.QueryRow(q).Scan(&n)
		counts[k] = n
	}
	var revenue, agentCost float64
	_ = s.DB.QueryRow(`SELECT COALESCE(SUM(cost),0) FROM cards WHERE status='used'`).Scan(&revenue)
	_ = s.DB.QueryRow(`SELECT COALESCE(SUM(cost),0) FROM cards WHERE agent_id IS NOT NULL`).Scan(&agentCost)
	counts["revenue"] = revenue
	counts["agent_cost"] = agentCost
	counts["recent_cards"] = rowsToMaps(s.DB, `SELECT cards.card_key,cards.status,cards.created_at,cards.used_at,apps.name app_name,users.username used_by FROM cards JOIN apps ON apps.id=cards.app_id LEFT JOIN users ON users.id=cards.used_by_user_id ORDER BY cards.id DESC LIMIT 8`)
	counts["app_overview"] = rowsToMaps(s.DB, `SELECT apps.name,apps.app_key,apps.status,COUNT(DISTINCT users.id) users,COUNT(DISTINCT cards.id) cards FROM apps LEFT JOIN users ON users.app_id=apps.id LEFT JOIN cards ON cards.app_id=apps.id GROUP BY apps.id ORDER BY apps.id DESC LIMIT 5`)
	c.JSON(200, gin.H{"ok": true, "data": counts})
}

func queryJSON(c *gin.Context, db *sql.DB, q string, args ...any) {
	rows, err := db.Query(q, args...)
	if err != nil {
		errorJSON(c, 500, err.Error())
		return
	}
	defer rows.Close()
	cols, _ := rows.Columns()
	out := []map[string]any{}
	for rows.Next() {
		vals := make([]any, len(cols))
		ptr := make([]any, len(cols))
		for i := range vals {
			ptr[i] = &vals[i]
		}
		rows.Scan(ptr...)
		m := map[string]any{}
		for i, col := range cols {
			if b, ok := vals[i].([]byte); ok {
				m[col] = string(b)
			} else {
				m[col] = vals[i]
			}
		}
		out = append(out, m)
	}
	c.JSON(200, gin.H{"ok": true, "data": out})
}
func now() string { return time.Now().In(beijingLocation).Format(displayTimeLayout) }

func (s *Server) listApps(c *gin.Context) {
	queryJSON(c, s.DB, `SELECT id,name,app_key,status,CAST(version AS INTEGER) version,CAST(min_version AS INTEGER) min_version,force_update,force_update_message,download_url,backup_download_url,announcement,heartbeat_interval,heartbeat_timeout,client_secret,created_at,updated_at FROM apps ORDER BY id DESC`)
}
func (s *Server) createApp(c *gin.Context) {
	var r struct {
		Name, AppKey, Announcement, DownloadURL, BackupDownloadURL string
		Version, MinVersion, HeartbeatInterval, HeartbeatTimeout   int
	}
	if c.ShouldBindJSON(&r) != nil || r.Name == "" || r.AppKey == "" {
		errorJSON(c, 400, "name/app_key required")
		return
	}
	sec := randKey(24)
	t := now()
	_, err := s.DB.Exec(`INSERT INTO apps(name,app_key,status,version,min_version,announcement,download_url,backup_download_url,heartbeat_interval,heartbeat_timeout,client_secret,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`, r.Name, r.AppKey, "active", maxInt(r.Version, 1), maxInt(r.MinVersion, 0), r.Announcement, r.DownloadURL, r.BackupDownloadURL, normalizeSeconds(r.HeartbeatInterval, 60), normalizeSeconds(r.HeartbeatTimeout, 180), sec, t, t)
	if err != nil {
		errorJSON(c, 400, err.Error())
		return
	}
	c.JSON(200, gin.H{"ok": true})
}
func (s *Server) updateApp(c *gin.Context) {
	id := c.Param("id")
	var r struct {
		Name, Status, Announcement, DownloadURL, BackupDownloadURL, ForceUpdateMessage string
		Version, MinVersion, ForceUpdate, HeartbeatInterval, HeartbeatTimeout          int
	}
	if c.ShouldBindJSON(&r) != nil {
		errorJSON(c, 400, "bad request")
		return
	}
	_, err := s.DB.Exec(`UPDATE apps SET name=?,status=?,version=?,min_version=?,force_update=?,force_update_message=?,announcement=?,download_url=?,backup_download_url=?,heartbeat_interval=?,heartbeat_timeout=?,updated_at=? WHERE id=?`, r.Name, fallback(r.Status, "active"), maxInt(r.Version, 1), maxInt(r.MinVersion, 0), r.ForceUpdate, r.ForceUpdateMessage, r.Announcement, r.DownloadURL, r.BackupDownloadURL, normalizeSeconds(r.HeartbeatInterval, 60), normalizeSeconds(r.HeartbeatTimeout, 180), now(), id)
	if err != nil {
		errorJSON(c, 400, err.Error())
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

func (s *Server) listUsers(c *gin.Context) {
	parts := []string{"1=1"}
	args := []any{}
	if v := c.Query("status"); v != "" {
		parts = append(parts, "users.status=?")
		args = append(args, v)
	}
	if v := c.Query("app_id"); v != "" {
		parts = append(parts, "users.app_id=?")
		args = append(args, v)
	}
	if v := strings.TrimSpace(c.Query("keyword")); v != "" {
		like := "%" + v + "%"
		parts = append(parts, "(users.username LIKE ? OR users.machine_code LIKE ?)")
		args = append(args, like, like)
	}
	q := `SELECT users.id,apps.name app_name,users.app_id,users.username,users.status,users.expire_at,users.machine_code,users.max_devices,users.free_unbinds,users.max_unbinds,users.unbind_used,users.unbind_deduct_hours,users.created_at,users.updated_at FROM users JOIN apps ON apps.id=users.app_id WHERE ` + strings.Join(parts, " AND ") + ` ORDER BY users.id DESC LIMIT 500`
	queryJSON(c, s.DB, q, args...)
}
func (s *Server) createUser(c *gin.Context) {
	var r struct {
		AppID              int
		Username, Password string
		MaxDevices         int
	}
	if c.ShouldBindJSON(&r) != nil || r.AppID == 0 || r.Username == "" || r.Password == "" {
		errorJSON(c, 400, "required")
		return
	}
	h, _ := bcrypt.GenerateFromPassword([]byte(r.Password), bcrypt.DefaultCost)
	t := now()
	_, err := s.DB.Exec(`INSERT INTO users(app_id,username,password_hash,status,max_devices,created_at,updated_at) VALUES(?,?,?,?,?,?,?)`, r.AppID, r.Username, string(h), "active", maxInt(r.MaxDevices, 1), t, t)
	if err != nil {
		errorJSON(c, 400, err.Error())
		return
	}
	c.JSON(200, gin.H{"ok": true})
}
func (s *Server) updateUser(c *gin.Context) {
	id := c.Param("id")
	var r struct {
		Status, ExpireAt, MachineCode              string
		MaxDevices                                 int
		FreeUnbinds, MaxUnbinds, UnbindDeductHours *int
	}
	if c.ShouldBindJSON(&r) != nil {
		errorJSON(c, 400, "bad request")
		return
	}
	freeUnbinds, maxUnbinds, deductHours := 0, 0, 0
	_ = s.DB.QueryRow(`SELECT free_unbinds,max_unbinds,unbind_deduct_hours FROM users WHERE id=?`, id).Scan(&freeUnbinds, &maxUnbinds, &deductHours)
	if r.FreeUnbinds != nil {
		freeUnbinds = maxInt(*r.FreeUnbinds, 0)
	}
	if r.MaxUnbinds != nil {
		maxUnbinds = maxInt(*r.MaxUnbinds, 0)
	}
	if r.UnbindDeductHours != nil {
		deductHours = maxInt(*r.UnbindDeductHours, 0)
	}
	_, err := s.DB.Exec(`UPDATE users SET status=?,expire_at=?,machine_code=?,max_devices=?,free_unbinds=?,max_unbinds=?,unbind_deduct_hours=?,updated_at=? WHERE id=?`, fallback(r.Status, "active"), emptyNil(r.ExpireAt), emptyNil(r.MachineCode), maxInt(r.MaxDevices, 1), freeUnbinds, maxUnbinds, deductHours, now(), id)
	if err != nil {
		errorJSON(c, 400, err.Error())
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

func (s *Server) updateUserPassword(c *gin.Context) {
	var r struct{ Password string }
	if c.ShouldBindJSON(&r) != nil || r.Password == "" {
		errorJSON(c, 400, "password required")
		return
	}
	h, _ := bcrypt.GenerateFromPassword([]byte(r.Password), bcrypt.DefaultCost)
	_, err := s.DB.Exec(`UPDATE users SET password_hash=?,updated_at=? WHERE id=?`, string(h), now(), c.Param("id"))
	if err != nil {
		errorJSON(c, 400, err.Error())
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

func (s *Server) unbindUserByID(id int) (used int, expireAt string, deducted bool, deductHours int, err error) {
	var freeUnbinds, maxUnbinds int
	var exp sql.NullString
	if err = s.DB.QueryRow(`SELECT free_unbinds,max_unbinds,unbind_used,unbind_deduct_hours,expire_at FROM users WHERE id=?`, id).Scan(&freeUnbinds, &maxUnbinds, &used, &deductHours, &exp); err != nil {
		return 0, "", false, 0, fmt.Errorf("用户不存在")
	}
	if maxUnbinds > 0 && used >= maxUnbinds {
		return used, strings.TrimSpace(exp.String), false, deductHours, fmt.Errorf("解绑次数已达到上限")
	}
	newExpire := strings.TrimSpace(exp.String)
	nextUsed := used + 1
	if nextUsed > freeUnbinds && deductHours > 0 && exp.Valid && strings.TrimSpace(exp.String) != "" {
		if t, parseErr := parseTime(exp.String); parseErr == nil {
			newExpire = t.Add(-time.Duration(deductHours) * time.Hour).In(beijingLocation).Format(displayTimeLayout)
			deducted = true
		}
	}
	tx, err := s.DB.Begin()
	if err != nil {
		return used, newExpire, deducted, deductHours, err
	}
	if _, err = tx.Exec(`UPDATE users SET machine_code='',expire_at=?,unbind_used=unbind_used+1,updated_at=? WHERE id=?`, newExpire, now(), id); err != nil {
		tx.Rollback()
		return used, newExpire, deducted, deductHours, err
	}
	if err = tx.Commit(); err != nil {
		return used, newExpire, deducted, deductHours, err
	}
	return used + 1, newExpire, deducted, deductHours, nil
}

func (s *Server) unbindCardByID(id int) (used int, expireAt string, deducted bool, deductHours int, err error) {
	var days, hours, freeUnbinds, maxUnbinds int
	var status string
	var usedAt, machine sql.NullString
	if err = s.DB.QueryRow(`SELECT status,expire_days,expire_hours,used_at,machine_code,free_unbinds,max_unbinds,unbind_used,unbind_deduct_hours FROM cards WHERE id=?`, id).Scan(&status, &days, &hours, &usedAt, &machine, &freeUnbinds, &maxUnbinds, &used, &deductHours); err != nil {
		return 0, "", false, 0, fmt.Errorf("卡密无效、已使用或已被禁用")
	}
	if status != "used" {
		return used, "", false, deductHours, fmt.Errorf("卡密无效、已使用或已被禁用")
	}
	if strings.TrimSpace(machine.String) == "" {
		return used, "", false, deductHours, fmt.Errorf("卡密尚未绑定机器")
	}
	if maxUnbinds > 0 && used >= maxUnbinds {
		return used, "", false, deductHours, fmt.Errorf("解绑次数已达到上限")
	}
	if hours <= 0 {
		hours = days * 24
	}
	start := time.Now().In(beijingLocation)
	if usedAt.Valid && strings.TrimSpace(usedAt.String) != "" {
		if parsed, parseErr := parseTime(usedAt.String); parseErr == nil {
			start = parsed
		}
	}
	expTime := start.Add(time.Duration(hours) * time.Hour)
	if time.Now().In(beijingLocation).After(expTime) {
		return used, expTime.In(beijingLocation).Format(displayTimeLayout), false, deductHours, fmt.Errorf("卡密已到期")
	}
	nextUsed := used + 1
	newHours := hours
	if nextUsed > freeUnbinds && deductHours > 0 {
		newHours = hours - deductHours
		if newHours < 0 {
			newHours = 0
		}
		deducted = true
		expTime = start.Add(time.Duration(newHours) * time.Hour)
		if time.Now().In(beijingLocation).After(expTime) {
			return used, expTime.In(beijingLocation).Format(displayTimeLayout), deducted, deductHours, fmt.Errorf("解绑扣时后卡密已到期")
		}
	}
	if _, err = s.DB.Exec(`UPDATE cards SET machine_code='',expire_hours=?,expire_days=?,unbind_used=unbind_used+1 WHERE id=?`, newHours, (newHours+23)/24, id); err != nil {
		return used, expTime.In(beijingLocation).Format(displayTimeLayout), deducted, deductHours, err
	}
	return nextUsed, expTime.In(beijingLocation).Format(displayTimeLayout), deducted, deductHours, nil
}

func (s *Server) unbindUserMachine(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	used, exp, deducted, deductHours, err := s.unbindUserByID(id)
	if err != nil {
		status := 400
		if strings.Contains(err.Error(), "不存在") {
			status = 404
		}
		errorJSON(c, status, err.Error())
		return
	}
	c.JSON(200, gin.H{"ok": true, "message": "解绑成功", "unbind_used": used, "expire_at": exp, "deducted": deducted, "deduct_hours": deductHours})
}

func (s *Server) userDevices(c *gin.Context) {
	q := `SELECT devices.id,devices.app_id,apps.name app_name,apps.heartbeat_timeout,devices.user_id,users.username,devices.machine_code,devices.status,devices.client_version,devices.ip,devices.last_seen,devices.created_at FROM devices JOIN apps ON apps.id=devices.app_id JOIN users ON users.id=devices.user_id WHERE devices.user_id=? ORDER BY devices.last_seen DESC`
	data, onlineCount := s.devicesWithOnline(q, c.Param("id"))
	c.JSON(200, gin.H{"ok": true, "data": data, "online_count": onlineCount, "total": len(data)})
}

func (s *Server) deleteUser(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id == 0 {
		errorJSON(c, 400, "用户ID无效")
		return
	}
	tx, err := s.DB.Begin()
	if err != nil {
		errorJSON(c, 500, err.Error())
		return
	}
	if _, err = tx.Exec(`UPDATE cards SET used_by_user_id=NULL WHERE used_by_user_id=?`, id); err == nil {
		_, err = tx.Exec(`DELETE FROM devices WHERE user_id=?`, id)
	}
	var res sql.Result
	if err == nil {
		res, err = tx.Exec(`DELETE FROM users WHERE id=?`, id)
	}
	if err != nil {
		tx.Rollback()
		errorJSON(c, 400, err.Error())
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		tx.Rollback()
		errorJSON(c, 404, "用户不存在")
		return
	}
	if err := tx.Commit(); err != nil {
		errorJSON(c, 500, err.Error())
		return
	}
	c.JSON(200, gin.H{"ok": true, "message": "用户已删除"})
}

func (s *Server) listCardTypes(c *gin.Context) {
	queryJSON(c, s.DB, `SELECT card_types.id,apps.name app_name,card_types.app_id,card_types.name,card_types.days,card_types.hours,card_types.max_devices,card_types.free_unbinds,card_types.max_unbinds,card_types.unbind_deduct_hours,card_types.price,card_types.status FROM card_types JOIN apps ON apps.id=card_types.app_id ORDER BY card_types.id DESC`)
}
func (s *Server) createCardType(c *gin.Context) {
	var r struct {
		AppID, Days, Hours, MaxDevices             int
		FreeUnbinds, MaxUnbinds, UnbindDeductHours int
		Name                                       string
		Price                                      float64
	}
	if c.ShouldBindJSON(&r) != nil || r.AppID == 0 || r.Name == "" || (r.Hours <= 0 && r.Days <= 0) {
		errorJSON(c, 400, "required")
		return
	}
	if r.Hours <= 0 {
		r.Hours = r.Days * 24
	}
	r.Days = (r.Hours + 23) / 24
	t := now()
	res, err := s.DB.Exec(`INSERT INTO card_types(app_id,name,days,hours,max_devices,free_unbinds,max_unbinds,unbind_deduct_hours,price,status,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`, r.AppID, r.Name, r.Days, r.Hours, maxInt(r.MaxDevices, 1), maxInt(r.FreeUnbinds, 0), maxInt(r.MaxUnbinds, 0), maxInt(r.UnbindDeductHours, 0), r.Price, "active", t, t)
	if err != nil {
		errorJSON(c, 400, err.Error())
		return
	}
	id, _ := res.LastInsertId()
	c.JSON(200, gin.H{"ok": true, "id": id})
}

func (s *Server) deleteCardType(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id == 0 {
		errorJSON(c, 400, "invalid id")
		return
	}
	var used int
	_ = s.DB.QueryRow(`SELECT COUNT(*) FROM cards WHERE card_type_id=?`, id).Scan(&used)
	if used > 0 {
		errorJSON(c, 400, "该套餐已有卡密使用，不能删除；可新建套餐替代")
		return
	}
	res, err := s.DB.Exec(`DELETE FROM card_types WHERE id=?`, id)
	if err != nil {
		errorJSON(c, 500, err.Error())
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		errorJSON(c, 404, "套餐不存在")
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

func (s *Server) listCards(c *gin.Context) {
	filters := cardFilters(c)
	q := `SELECT cards.id,apps.name app_name,cards.app_id,cards.card_key,cards.status,cards.expire_days,cards.expire_hours,cards.max_devices,cards.free_unbinds,cards.max_unbinds,cards.unbind_used,cards.unbind_deduct_hours,cards.used_at,cards.created_at,cards.created_by,cards.agent_id,agents.username agent_name,cards.cost,users.username used_by FROM cards JOIN apps ON apps.id=cards.app_id LEFT JOIN users ON users.id=cards.used_by_user_id LEFT JOIN agents ON agents.id=cards.agent_id WHERE ` + filters.where + ` ORDER BY cards.id DESC LIMIT 500`
	queryJSON(c, s.DB, q, filters.args...)
}
func (s *Server) generateCards(c *gin.Context) {
	var r struct{ AppID, CardTypeID, Count int }
	if c.ShouldBindJSON(&r) != nil || r.AppID == 0 || r.Count <= 0 {
		errorJSON(c, 400, "软件和数量不能为空")
		return
	}
	if r.CardTypeID == 0 {
		errorJSON(c, 400, "请选择卡类套餐")
		return
	}
	if r.Count > 200 {
		r.Count = 200
	}
	var days, hours, dev, freeUnbinds, maxUnbinds, deductHours int
	if err := s.DB.QueryRow(`SELECT days,hours,max_devices,free_unbinds,max_unbinds,unbind_deduct_hours FROM card_types WHERE id=? AND app_id=? AND status='active'`, r.CardTypeID, r.AppID).Scan(&days, &hours, &dev, &freeUnbinds, &maxUnbinds, &deductHours); err != nil {
		errorJSON(c, 400, "卡类套餐不存在或已禁用")
		return
	}
	if hours <= 0 {
		hours = days * 24
	}
	days = (hours + 23) / 24
	tx, err := s.DB.Begin()
	if err != nil {
		errorJSON(c, 500, err.Error())
		return
	}
	keys := []string{}
	for i := 0; i < r.Count; i++ {
		key := "LS-" + strings.ToUpper(randKey(4)+"-"+randKey(4)+"-"+randKey(4))
		_, err := tx.Exec(`INSERT INTO cards(app_id,card_type_id,card_key,status,expire_days,expire_hours,max_devices,free_unbinds,max_unbinds,unbind_deduct_hours,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, r.AppID, r.CardTypeID, key, "unused", days, hours, dev, freeUnbinds, maxUnbinds, deductHours, now())
		if err != nil {
			tx.Rollback()
			errorJSON(c, 500, err.Error())
			return
		}
		keys = append(keys, key)
	}
	if err := tx.Commit(); err != nil {
		errorJSON(c, 500, err.Error())
		return
	}
	c.JSON(200, gin.H{"ok": true, "data": keys})
}
func (s *Server) disableCard(c *gin.Context) {
	_, err := s.DB.Exec(`UPDATE cards SET status='disabled' WHERE id=?`, c.Param("id"))
	if err != nil {
		errorJSON(c, 400, err.Error())
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

func (s *Server) appByKey(key string) (id int, secret, status string, err error) {
	err = s.DB.QueryRow(`SELECT id,client_secret,status FROM apps WHERE app_key=?`, key).Scan(&id, &secret, &status)
	return
}

func (s *Server) appTiming(appID int) (heartbeatInterval int, heartbeatTimeout int) {
	heartbeatInterval, heartbeatTimeout = 60, 180
	_ = s.DB.QueryRow(`SELECT heartbeat_interval,heartbeat_timeout FROM apps WHERE id=?`, appID).Scan(&heartbeatInterval, &heartbeatTimeout)
	heartbeatInterval = normalizeSeconds(heartbeatInterval, 60)
	heartbeatTimeout = normalizeSeconds(heartbeatTimeout, 180)
	return
}

func parseTime(v string) (time.Time, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return time.Time{}, fmt.Errorf("empty time")
	}
	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return t, nil
	}
	if t, err := time.ParseInLocation(displayTimeLayout, v, beijingLocation); err == nil {
		return t, nil
	}
	if t, err := time.ParseInLocation("2006-01-02", v, beijingLocation); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("invalid time: %s", v)
}

func isExpired(exp string) bool {
	exp = strings.TrimSpace(exp)
	if exp == "" {
		return false
	}
	if t, err := parseTime(exp); err == nil {
		return time.Now().In(beijingLocation).After(t)
	}
	return false
}

func (s *Server) bindClientDevice(appID, userID int, machineCode, clientVersion, ip string) error {
	machineCode = strings.TrimSpace(machineCode)
	if machineCode == "" {
		return fmt.Errorf("machine code required")
	}
	var status, boundMachine string
	var exp sql.NullString
	var maxDevices int
	if err := s.DB.QueryRow(`SELECT status,expire_at,max_devices,COALESCE(machine_code,'') FROM users WHERE id=? AND app_id=?`, userID, appID).Scan(&status, &exp, &maxDevices, &boundMachine); err != nil {
		return fmt.Errorf("user not found")
	}
	if status != "active" {
		return fmt.Errorf("user disabled")
	}
	if isExpired(exp.String) {
		return fmt.Errorf("user expired")
	}
	boundMachine = strings.TrimSpace(boundMachine)
	if boundMachine != "" && boundMachine != machineCode {
		return fmt.Errorf("device limit exceeded")
	}
	if sourceAgentDisabled, err := s.userHasDisabledAgentSource(appID, userID); err != nil {
		return err
	} else if sourceAgentDisabled {
		return fmt.Errorf("agent disabled")
	}
	var deviceStatus string
	err := s.DB.QueryRow(`SELECT status FROM devices WHERE app_id=? AND user_id=? AND machine_code=?`, appID, userID, machineCode).Scan(&deviceStatus)
	if err == nil {
		t := now()
		if deviceStatus != "active" {
			if strings.TrimSpace(boundMachine) == "" && deviceStatus == "disabled" {
				_, err = s.DB.Exec(`UPDATE devices SET status='active',client_version=?,ip=?,last_seen=? WHERE app_id=? AND user_id=? AND machine_code=?`, clientVersion, ip, t, appID, userID, machineCode)
				if err == nil {
					_, err = s.DB.Exec(`UPDATE users SET machine_code=?,updated_at=? WHERE id=? AND app_id=?`, machineCode, t, userID, appID)
				}
				return err
			}
			return fmt.Errorf("device %s", deviceStatus)
		}
		_, err = s.DB.Exec(`UPDATE devices SET client_version=?,ip=?,last_seen=? WHERE app_id=? AND user_id=? AND machine_code=?`, clientVersion, ip, t, appID, userID, machineCode)
		return err
	}
	if err != sql.ErrNoRows {
		return err
	}
	t := now()
	if _, err := s.DB.Exec(`UPDATE users SET machine_code=?,updated_at=? WHERE id=? AND app_id=? AND COALESCE(machine_code,'')=''`, machineCode, t, userID, appID); err != nil {
		return err
	}
	_, err = s.DB.Exec(`INSERT INTO devices(app_id,user_id,machine_code,status,client_version,ip,last_seen,created_at) VALUES(?,?,?,?,?,?,?,?)`, appID, userID, machineCode, "active", clientVersion, ip, t, t)
	return err
}

func (s *Server) userHasDisabledAgentSource(appID, userID int) (bool, error) {
	var disabled int
	err := s.DB.QueryRow(`SELECT COUNT(*) FROM cards JOIN agents ON agents.id=cards.agent_id WHERE cards.app_id=? AND cards.used_by_user_id=? AND agents.status!='active'`, appID, userID).Scan(&disabled)
	if err != nil {
		return false, err
	}
	return disabled > 0, nil
}

func (s *Server) bindCardDevice(appID, cardID int, machineCode, clientVersion, ip string) error {
	machineCode = strings.TrimSpace(machineCode)
	if machineCode == "" {
		return fmt.Errorf("machine code required")
	}
	var deviceStatus string
	err := s.DB.QueryRow(`SELECT status FROM devices WHERE app_id=? AND auth_mode='card' AND card_id=? AND machine_code=?`, appID, cardID, machineCode).Scan(&deviceStatus)
	t := now()
	if err == nil {
		if deviceStatus != "active" {
			return fmt.Errorf("device %s", deviceStatus)
		}
		_, err = s.DB.Exec(`UPDATE devices SET client_version=?,ip=?,last_seen=? WHERE app_id=? AND auth_mode='card' AND card_id=? AND machine_code=?`, clientVersion, ip, t, appID, cardID, machineCode)
		return err
	}
	if err != sql.ErrNoRows {
		return err
	}
	_, err = s.DB.Exec(`INSERT INTO devices(app_id,user_id,card_id,auth_mode,machine_code,status,client_version,ip,last_seen,created_at) VALUES(?,?,?,?,?,?,?,?,?,?)`, appID, 0, cardID, "card", machineCode, "active", clientVersion, ip, t, t)
	return err
}

func (s *Server) clientRegister(c *gin.Context) {
	var r struct{ AppKey, Username, Password, CardKey, MachineCode string }
	c.ShouldBindJSON(&r)
	r.Username = strings.TrimSpace(r.Username)
	r.CardKey = strings.TrimSpace(r.CardKey)
	r.MachineCode = strings.TrimSpace(r.MachineCode)
	if r.Username == "" || r.Password == "" || r.CardKey == "" || r.MachineCode == "" {
		errorJSON(c, 400, "用户名、密码、卡密和机器码不能为空")
		return
	}
	app, _, st, err := s.appByKey(r.AppKey)
	if err != nil || st != "active" {
		errorJSON(c, 404, "软件不存在或已禁用")
		return
	}
	h, _ := bcrypt.GenerateFromPassword([]byte(r.Password), bcrypt.DefaultCost)
	t := now()
	tx, err := s.DB.Begin()
	if err != nil {
		errorJSON(c, 500, err.Error())
		return
	}
	res, err := tx.Exec(`INSERT INTO users(app_id,username,password_hash,status,machine_code,created_at,updated_at) VALUES(?,?,?,?,?,?,?)`, app, r.Username, string(h), "active", r.MachineCode, t, t)
	if err != nil {
		tx.Rollback()
		if strings.Contains(err.Error(), "UNIQUE") {
			errorJSON(c, 400, "用户名已存在")
		} else {
			errorJSON(c, 400, err.Error())
		}
		return
	}
	uid64, _ := res.LastInsertId()
	exp, _, err := s.consumeCardForUser(tx, app, r.CardKey, int(uid64))
	if err != nil {
		tx.Rollback()
		errorJSON(c, 400, zhClientError(err.Error()))
		return
	}
	_, err = tx.Exec(`INSERT INTO devices(app_id,user_id,machine_code,status,client_version,ip,last_seen,created_at) VALUES(?,?,?,?,?,?,?,?)`, app, int(uid64), r.MachineCode, "active", "", c.ClientIP(), t, t)
	if err != nil {
		tx.Rollback()
		errorJSON(c, 400, err.Error())
		return
	}
	if err := tx.Commit(); err != nil {
		errorJSON(c, 500, err.Error())
		return
	}
	token, _ := auth.Issue(s.JWTSecret, "client", fmt.Sprintf("%d:%d:%s:%s", app, int(uid64), r.Username, r.MachineCode), 24*time.Hour)
	interval, timeout := s.appTiming(app)
	c.JSON(200, gin.H{"ok": true, "client_token": token, "user_id": int(uid64), "app_id": app, "expire_at": exp, "machine_code": r.MachineCode, "heartbeat_interval": interval, "heartbeat_timeout": timeout})
}
func (s *Server) clientLogin(c *gin.Context) {
	var r struct{ AppKey, Username, Password, MachineCode string }
	c.ShouldBindJSON(&r)
	r.MachineCode = strings.TrimSpace(r.MachineCode)
	if r.MachineCode == "" {
		errorJSON(c, 400, "机器码不能为空")
		return
	}
	app, _, st, err := s.appByKey(r.AppKey)
	if err != nil || st != "active" {
		errorJSON(c, 404, "软件不存在或已禁用")
		return
	}
	var id, maxDevices int
	var hash, status string
	var exp sql.NullString
	err = s.DB.QueryRow(`SELECT id,password_hash,status,expire_at,max_devices FROM users WHERE app_id=? AND username=?`, app, r.Username).Scan(&id, &hash, &status, &exp, &maxDevices)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(r.Password)) != nil {
		errorJSON(c, 401, "账号或密码错误")
		return
	}
	if status != "active" {
		errorJSON(c, 403, "账号已被禁用")
		return
	}
	if isExpired(exp.String) {
		errorJSON(c, 403, "账号已到期")
		return
	}
	if err := s.bindClientDevice(app, id, r.MachineCode, "", c.ClientIP()); err != nil {
		errorJSON(c, 403, zhClientError(err.Error()))
		return
	}
	token, _ := auth.Issue(s.JWTSecret, "client", fmt.Sprintf("%d:%d:%s:%s", app, id, r.Username, r.MachineCode), 24*time.Hour)
	interval, timeout := s.appTiming(app)
	c.JSON(200, gin.H{"ok": true, "client_token": token, "expire_at": exp.String, "max_devices": maxDevices, "machine_code": r.MachineCode, "heartbeat_interval": interval, "heartbeat_timeout": timeout})
}
func (s *Server) consumeCardForUser(tx *sql.Tx, app int, cardKey string, uid int) (string, int, error) {
	var cid, days, hours, dev, freeUnbinds, maxUnbinds, deductHours int
	if err := tx.QueryRow(`SELECT cards.id,cards.expire_days,cards.expire_hours,cards.max_devices,cards.free_unbinds,cards.max_unbinds,cards.unbind_deduct_hours FROM cards LEFT JOIN agents ON agents.id=cards.agent_id WHERE cards.app_id=? AND cards.card_key=? AND cards.status='unused' AND (cards.agent_id IS NULL OR agents.status='active')`, app, cardKey).Scan(&cid, &days, &hours, &dev, &freeUnbinds, &maxUnbinds, &deductHours); err != nil {
		return "", 0, fmt.Errorf("卡密无效、已使用或已被禁用")
	}
	if hours <= 0 {
		hours = days * 24
	}
	base := time.Now().In(beijingLocation)
	var oldExp sql.NullString
	_ = tx.QueryRow(`SELECT expire_at FROM users WHERE id=? AND app_id=?`, uid, app).Scan(&oldExp)
	if oldExp.Valid {
		if parsed, err := parseTime(oldExp.String); err == nil && parsed.After(base) {
			base = parsed
		}
	}
	exp := base.Add(time.Duration(hours) * time.Hour).In(beijingLocation).Format(displayTimeLayout)
	if _, err := tx.Exec(`UPDATE users SET expire_at=?,max_devices=?,free_unbinds=?,max_unbinds=?,unbind_used=0,unbind_deduct_hours=?,updated_at=? WHERE id=?`, exp, dev, freeUnbinds, maxUnbinds, deductHours, now(), uid); err != nil {
		return "", 0, err
	}
	if _, err := tx.Exec(`UPDATE cards SET status='used',used_by_user_id=?,used_at=? WHERE id=?`, uid, now(), cid); err != nil {
		return "", 0, err
	}
	return exp, cid, nil
}

func (s *Server) clientRecharge(c *gin.Context) {
	var r struct{ AppKey, Username, CardKey string }
	c.ShouldBindJSON(&r)
	app, _, _, err := s.appByKey(r.AppKey)
	if err != nil {
		errorJSON(c, 404, "软件不存在或已禁用")
		return
	}
	var uid int
	if s.DB.QueryRow(`SELECT id FROM users WHERE app_id=? AND username=?`, app, r.Username).Scan(&uid) != nil {
		errorJSON(c, 404, "用户不存在")
		return
	}
	tx, err := s.DB.Begin()
	if err != nil {
		errorJSON(c, 500, err.Error())
		return
	}
	exp, _, err := s.consumeCardForUser(tx, app, strings.TrimSpace(r.CardKey), uid)
	if err != nil {
		tx.Rollback()
		errorJSON(c, 400, zhClientError(err.Error()))
		return
	}
	if err := tx.Commit(); err != nil {
		errorJSON(c, 500, err.Error())
		return
	}
	c.JSON(200, gin.H{"ok": true, "expire_at": exp})
}
func (s *Server) clientCardLogin(c *gin.Context) {
	var r struct{ AppKey, CardKey, MachineCode string }
	c.ShouldBindJSON(&r)
	r.CardKey = strings.TrimSpace(r.CardKey)
	r.MachineCode = strings.TrimSpace(r.MachineCode)
	if r.CardKey == "" || r.MachineCode == "" {
		errorJSON(c, 400, "卡密和机器码不能为空")
		return
	}
	app, _, st, err := s.appByKey(r.AppKey)
	if err != nil || st != "active" {
		errorJSON(c, 404, "软件不存在或已禁用")
		return
	}
	var cid, days, hours, maxDevices int
	var status string
	var usedAt, machine sql.NullString
	if err := s.DB.QueryRow(`SELECT cards.id,cards.status,cards.expire_days,cards.expire_hours,cards.max_devices,cards.used_at,cards.machine_code FROM cards LEFT JOIN agents ON agents.id=cards.agent_id WHERE cards.app_id=? AND cards.card_key=? AND cards.status IN ('unused','used') AND (cards.agent_id IS NULL OR agents.status='active')`, app, r.CardKey).Scan(&cid, &status, &days, &hours, &maxDevices, &usedAt, &machine); err != nil {
		errorJSON(c, 400, "卡密无效、已使用或已被禁用")
		return
	}
	if hours <= 0 {
		hours = days * 24
	}
	if maxDevices <= 0 {
		maxDevices = 1
	}
	start := time.Now().In(beijingLocation)
	if usedAt.Valid && strings.TrimSpace(usedAt.String) != "" {
		if parsed, err := parseTime(usedAt.String); err == nil {
			start = parsed
		}
	}
	expTime := start.Add(time.Duration(hours) * time.Hour)
	if time.Now().In(beijingLocation).After(expTime) {
		errorJSON(c, 403, "卡密已到期")
		return
	}
	boundMachine := strings.TrimSpace(machine.String)
	if boundMachine != "" && boundMachine != r.MachineCode {
		errorJSON(c, 403, "机器码不匹配")
		return
	}
	if status == "unused" {
		if _, err := s.DB.Exec(`UPDATE cards SET status='used',machine_code=?,used_at=? WHERE id=?`, r.MachineCode, now(), cid); err != nil {
			errorJSON(c, 500, err.Error())
			return
		}
	} else if boundMachine == "" {
		if _, err := s.DB.Exec(`UPDATE cards SET machine_code=? WHERE id=?`, r.MachineCode, cid); err != nil {
			errorJSON(c, 500, err.Error())
			return
		}
	}
	if err := s.bindCardDevice(app, cid, r.MachineCode, "", c.ClientIP()); err != nil {
		errorJSON(c, 403, zhClientError(err.Error()))
		return
	}
	token, _ := auth.Issue(s.JWTSecret, "client", fmt.Sprintf("card:%d:%d:%s:%s", app, cid, r.CardKey, r.MachineCode), 24*time.Hour)
	interval, timeout := s.appTiming(app)
	c.JSON(200, gin.H{"ok": true, "mode": "card", "client_token": token, "expire_at": expTime.In(beijingLocation).Format(displayTimeLayout), "max_devices": maxDevices, "machine_code": r.MachineCode, "heartbeat_interval": interval, "heartbeat_timeout": timeout})
}

func (s *Server) clientUnbind(c *gin.Context) {
	token := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
	claims, err := auth.Parse(s.JWTSecret, token)
	if err != nil || claims.Role != "client" {
		errorJSON(c, 401, "登录令牌无效或已过期")
		return
	}
	parts := strings.Split(claims.Username, ":")
	if len(parts) < 4 || parts[0] == "card" {
		errorJSON(c, 400, "当前授权模式不支持账号解绑")
		return
	}
	app, _ := strconv.Atoi(parts[0])
	uid, _ := strconv.Atoi(parts[1])
	tokenMachine := parts[3]
	var r struct{ AppKey, MachineCode string }
	c.ShouldBindJSON(&r)
	r.MachineCode = strings.TrimSpace(r.MachineCode)
	if r.MachineCode == "" {
		errorJSON(c, 400, "机器码不能为空")
		return
	}
	if tokenMachine != "" && tokenMachine != r.MachineCode {
		errorJSON(c, 403, "机器码不匹配")
		return
	}
	if strings.TrimSpace(r.AppKey) != "" {
		checkApp, _, st, appErr := s.appByKey(r.AppKey)
		if appErr != nil || st != "active" || checkApp != app {
			errorJSON(c, 404, "软件不存在或已禁用")
			return
		}
	}
	if !s.userCanUnbind(uid, app, false, c) {
		return
	}
	s.respondUserUnbind(c, uid)
}

func (s *Server) clientAccountUnbind(c *gin.Context) {
	var r struct{ AppKey, Username, Password string }
	c.ShouldBindJSON(&r)
	r.Username = strings.TrimSpace(r.Username)
	if strings.TrimSpace(r.AppKey) == "" || r.Username == "" || r.Password == "" {
		errorJSON(c, 400, "appKey、账号和密码不能为空")
		return
	}
	app, _, st, err := s.appByKey(r.AppKey)
	if err != nil || st != "active" {
		errorJSON(c, 404, "软件不存在或已禁用")
		return
	}
	var uid int
	var hash string
	if err := s.DB.QueryRow(`SELECT id,password_hash FROM users WHERE app_id=? AND username=?`, app, r.Username).Scan(&uid, &hash); err != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(r.Password)) != nil {
		errorJSON(c, 401, "账号或密码错误")
		return
	}
	if !s.userCanUnbind(uid, app, true, c) {
		return
	}
	s.respondUserUnbind(c, uid)
}

func (s *Server) userCanUnbind(uid int, app int, allowExpired bool, c *gin.Context) bool {
	var status string
	var exp sql.NullString
	if err := s.DB.QueryRow(`SELECT status,expire_at FROM users WHERE id=? AND app_id=?`, uid, app).Scan(&status, &exp); err != nil {
		errorJSON(c, 404, "用户不存在")
		return false
	}
	if status != "active" {
		errorJSON(c, 403, "账号已被禁用")
		return false
	}
	if !allowExpired && isExpired(exp.String) {
		errorJSON(c, 403, "账号已到期")
		return false
	}
	return true
}

func (s *Server) respondUserUnbind(c *gin.Context, uid int) {
	used, newExpire, deducted, deductHours, err := s.unbindUserByID(uid)
	if err != nil {
		errorJSON(c, 400, err.Error())
		return
	}
	c.JSON(200, gin.H{"ok": true, "message": "解绑成功，请重新登录绑定新机器", "unbind_used": used, "expire_at": newExpire, "deducted": deducted, "deduct_hours": deductHours})
}

func (s *Server) clientCardUnbind(c *gin.Context) {
	var r struct{ AppKey, CardKey string }
	c.ShouldBindJSON(&r)
	r.CardKey = strings.TrimSpace(r.CardKey)
	if strings.TrimSpace(r.AppKey) == "" || r.CardKey == "" {
		errorJSON(c, 400, "appKey 和卡密不能为空")
		return
	}
	app, _, st, err := s.appByKey(r.AppKey)
	if err != nil || st != "active" {
		errorJSON(c, 404, "软件不存在或已禁用")
		return
	}
	var cid int
	if err := s.DB.QueryRow(`SELECT cards.id FROM cards LEFT JOIN agents ON agents.id=cards.agent_id WHERE cards.app_id=? AND cards.card_key=? AND cards.status='used' AND (cards.agent_id IS NULL OR agents.status='active')`, app, r.CardKey).Scan(&cid); err != nil {
		errorJSON(c, 400, "卡密无效、已使用或已被禁用")
		return
	}
	used, expireAt, deducted, deductHours, err := s.unbindCardByID(cid)
	if err != nil {
		errorJSON(c, 400, err.Error())
		return
	}
	c.JSON(200, gin.H{"ok": true, "mode": "card", "message": "卡密解绑成功，请重新登录绑定新机器", "unbind_used": used, "expire_at": expireAt, "deducted": deducted, "deduct_hours": deductHours})
}

func (s *Server) clientHeartbeat(c *gin.Context) {
	token := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
	claims, err := auth.Parse(s.JWTSecret, token)
	if err != nil || claims.Role != "client" {
		errorJSON(c, 401, "登录令牌无效或已过期")
		return
	}
	parts := strings.Split(claims.Username, ":")
	if len(parts) < 3 {
		errorJSON(c, 401, "登录令牌格式错误")
		return
	}
	var r struct{ MachineCode, ClientVersion string }
	c.ShouldBindJSON(&r)
	r.MachineCode = strings.TrimSpace(r.MachineCode)
	if r.MachineCode == "" {
		errorJSON(c, 400, "机器码不能为空")
		return
	}
	if parts[0] == "card" {
		if len(parts) < 5 {
			errorJSON(c, 401, "登录令牌格式错误")
			return
		}
		app, _ := strconv.Atoi(parts[1])
		cid, _ := strconv.Atoi(parts[2])
		tokenMachine := parts[4]
		if tokenMachine != "" && tokenMachine != r.MachineCode {
			errorJSON(c, 403, "机器码不匹配")
			return
		}
		var days, hours int
		var status string
		var usedAt, machine sql.NullString
		if err := s.DB.QueryRow(`SELECT cards.status,cards.expire_days,cards.expire_hours,cards.used_at,cards.machine_code FROM cards LEFT JOIN agents ON agents.id=cards.agent_id WHERE cards.id=? AND cards.app_id=? AND (cards.agent_id IS NULL OR agents.status='active')`, cid, app).Scan(&status, &days, &hours, &usedAt, &machine); err != nil || status != "used" {
			errorJSON(c, 403, "卡密无效、已使用或已被禁用")
			return
		}
		if strings.TrimSpace(machine.String) != "" && strings.TrimSpace(machine.String) != r.MachineCode {
			errorJSON(c, 403, "机器码不匹配")
			return
		}
		if hours <= 0 {
			hours = days * 24
		}
		start := time.Now().In(beijingLocation)
		if usedAt.Valid {
			if parsed, err := parseTime(usedAt.String); err == nil {
				start = parsed
			}
		}
		if time.Now().In(beijingLocation).After(start.Add(time.Duration(hours) * time.Hour)) {
			errorJSON(c, 403, "卡密已到期")
			return
		}
		if err := s.bindCardDevice(app, cid, r.MachineCode, r.ClientVersion, c.ClientIP()); err != nil {
			errorJSON(c, 403, zhClientError(err.Error()))
			return
		}
		interval, timeout := s.appTiming(app)
		c.JSON(200, gin.H{"ok": true, "mode": "card", "server_time": now(), "heartbeat_interval": interval, "heartbeat_timeout": timeout})
		return
	}
	app, _ := strconv.Atoi(parts[0])
	uid, _ := strconv.Atoi(parts[1])
	tokenMachine := ""
	if len(parts) >= 4 {
		tokenMachine = parts[3]
	}
	if tokenMachine != "" && tokenMachine != r.MachineCode {
		errorJSON(c, 403, "机器码不匹配")
		return
	}
	if err := s.bindClientDevice(app, uid, r.MachineCode, r.ClientVersion, c.ClientIP()); err != nil {
		errorJSON(c, 403, zhClientError(err.Error()))
		return
	}
	t := now()
	interval, timeout := s.appTiming(app)
	c.JSON(200, gin.H{"ok": true, "server_time": t, "heartbeat_interval": interval, "heartbeat_timeout": timeout})
}
func (s *Server) clientAppInfo(c *gin.Context) {
	key := c.Query("app_key")
	queryJSON(c, s.DB, `SELECT name,app_key,status,CAST(version AS INTEGER) version,CAST(min_version AS INTEGER) min_version,force_update,force_update_message,download_url,backup_download_url,announcement,heartbeat_interval,heartbeat_timeout FROM apps WHERE app_key=?`, key)
}

func (s *Server) resetAppSecret(c *gin.Context) {
	sec := randKey(32)
	res, err := s.DB.Exec(`UPDATE apps SET client_secret=?,updated_at=? WHERE id=?`, sec, now(), c.Param("id"))
	if err != nil {
		errorJSON(c, 400, err.Error())
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		errorJSON(c, 404, "app not found")
		return
	}
	c.JSON(200, gin.H{"ok": true, "client_secret": sec})
}

func zhClientError(msg string) string {
	switch {
	case strings.Contains(msg, "卡密无效") || strings.Contains(msg, "card invalid"):
		return "卡密无效、已使用或已被禁用"
	case strings.Contains(msg, "machine code required"):
		return "机器码不能为空"
	case strings.Contains(msg, "user not found"):
		return "用户不存在"
	case strings.Contains(msg, "user disabled"):
		return "账号已被禁用"
	case strings.Contains(msg, "user expired"):
		return "账号已到期"
	case strings.Contains(msg, "agent disabled"):
		return "授权来源代理已被禁用"
	case strings.Contains(msg, "device limit exceeded"):
		return "已达到最大绑定设备数"
	case strings.Contains(msg, "device disabled"):
		return "当前设备已被禁用"
	case strings.Contains(msg, "device banned"):
		return "当前设备已被封禁"
	case strings.Contains(msg, "machine code mismatch"):
		return "机器码不匹配"
	default:
		return msg
	}
}

func fallback(v, d string) string {
	if v == "" {
		return d
	}
	return v
}
func maxInt(v, d int) int {
	if v <= 0 {
		return d
	}
	return v
}
func emptyNil(v string) any {
	if v == "" {
		return nil
	}
	return v
}
func nullableID(v int) any {
	if v == 0 {
		return nil
	}
	return v
}
func normalizeSeconds(v, d int) int {
	if v <= 0 {
		return d
	}
	if v < 10 {
		return 10
	}
	return v
}
func randKey(n int) string { b := make([]byte, n); rand.Read(b); return hex.EncodeToString(b) }
func sign(secret, msg string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(msg))
	return hex.EncodeToString(h.Sum(nil))
}
func _unused() { _ = http.MethodGet; _ = sign }
