package api

import (
	"database/sql"
	"strconv"
	"strings"
	"time"

	"license-saas/backend/internal/auth"

	"github.com/gin-gonic/gin"
)

type cloudVarReq struct {
	AppID     int    `json:"appId"`
	VarKey    string `json:"varKey"`
	VarValue  string `json:"varValue"`
	ValueType string `json:"valueType"`
	Status    string `json:"status"`
	Remark    string `json:"remark"`
}

func (s *Server) listCloudVars(c *gin.Context) {
	parts := []string{"1=1"}
	args := []any{}
	if v := c.Query("app_id"); v != "" {
		parts = append(parts, "cloud_vars.app_id=?")
		args = append(args, v)
	}
	if v := c.Query("status"); v != "" {
		parts = append(parts, "cloud_vars.status=?")
		args = append(args, v)
	}
	if v := strings.TrimSpace(c.Query("keyword")); v != "" {
		like := "%" + v + "%"
		parts = append(parts, "(cloud_vars.var_key LIKE ? OR cloud_vars.var_value LIKE ? OR cloud_vars.remark LIKE ?)")
		args = append(args, like, like, like)
	}
	q := `SELECT cloud_vars.id,cloud_vars.app_id,apps.name app_name,cloud_vars.var_key,cloud_vars.var_value,cloud_vars.value_type,cloud_vars.status,cloud_vars.remark,cloud_vars.created_at,cloud_vars.updated_at FROM cloud_vars JOIN apps ON apps.id=cloud_vars.app_id WHERE ` + strings.Join(parts, " AND ") + ` ORDER BY cloud_vars.id DESC LIMIT 500`
	queryJSON(c, s.DB, q, args...)
}

func (s *Server) createCloudVar(c *gin.Context) {
	var r cloudVarReq
	if c.ShouldBindJSON(&r) != nil || r.AppID == 0 || strings.TrimSpace(r.VarKey) == "" {
		errorJSON(c, 400, "app_id/var_key required")
		return
	}
	status := normalizeStatus(r.Status)
	t := now()
	_, err := s.DB.Exec(`INSERT INTO cloud_vars(app_id,var_key,var_value,value_type,status,remark,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?)`, r.AppID, strings.TrimSpace(r.VarKey), r.VarValue, "text", status, r.Remark, t, t)
	if err != nil {
		errorJSON(c, 400, err.Error())
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

func (s *Server) updateCloudVar(c *gin.Context) {
	var r cloudVarReq
	if c.ShouldBindJSON(&r) != nil || strings.TrimSpace(r.VarKey) == "" {
		errorJSON(c, 400, "bad request")
		return
	}
	status := normalizeStatus(r.Status)
	_, err := s.DB.Exec(`UPDATE cloud_vars SET var_key=?,var_value=?,value_type=?,status=?,remark=?,updated_at=? WHERE id=?`, strings.TrimSpace(r.VarKey), r.VarValue, "text", status, r.Remark, now(), c.Param("id"))
	if err != nil {
		errorJSON(c, 400, err.Error())
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

func (s *Server) deleteCloudVar(c *gin.Context) {
	_, err := s.DB.Exec(`DELETE FROM cloud_vars WHERE id=?`, c.Param("id"))
	if err != nil {
		errorJSON(c, 400, err.Error())
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

func (s *Server) clientCloudVars(c *gin.Context) {
	key := c.Query("app_key")
	if key == "" {
		key = c.Query("appKey")
	}
	appID, _, st, err := s.appByKey(key)
	if err != nil || st != "active" {
		errorJSON(c, 404, "软件不存在或已禁用")
		return
	}
	if !s.validateClientTokenForApp(c, appID) {
		return
	}
	rows, err := s.DB.Query(`SELECT var_key,var_value FROM cloud_vars WHERE app_id=? AND status='active' ORDER BY id ASC`, appID)
	if err != nil {
		errorJSON(c, 500, err.Error())
		return
	}
	defer rows.Close()
	data := gin.H{}
	raw := []gin.H{}
	for rows.Next() {
		var k, v string
		if rows.Scan(&k, &v) != nil {
			continue
		}
		data[k] = v
		raw = append(raw, gin.H{"key": k, "value": v, "type": "text"})
	}
	c.JSON(200, gin.H{"ok": true, "data": data, "items": raw})
}

func (s *Server) validateClientTokenForApp(c *gin.Context, appID int) bool {
	token := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
	claims, err := auth.Parse(s.JWTSecret, token)
	if err != nil || claims.Role != "client" {
		errorJSON(c, 401, "登录令牌无效或已过期")
		return false
	}
	parts := strings.Split(claims.Username, ":")
	if len(parts) < 3 {
		errorJSON(c, 401, "登录令牌格式错误")
		return false
	}
	if parts[0] == "card" {
		if len(parts) < 5 {
			errorJSON(c, 401, "登录令牌格式错误")
			return false
		}
		tokenApp, _ := strconv.Atoi(parts[1])
		if tokenApp != appID {
			errorJSON(c, 403, "登录令牌不属于当前软件")
			return false
		}
		cid, _ := strconv.Atoi(parts[2])
		var days, hours int
		var status string
		var usedAt, machine sql.NullString
		if err := s.DB.QueryRow(`SELECT status,expire_days,expire_hours,used_at,machine_code FROM cards WHERE id=? AND app_id=?`, cid, appID).Scan(&status, &days, &hours, &usedAt, &machine); err != nil || status != "used" {
			errorJSON(c, 403, "卡密无效、已使用或已被禁用")
			return false
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
		if time.Now().In(beijingLocation).After(start.Add(time.Duration(hours) * time.Hour)) {
			errorJSON(c, 403, "卡密已到期")
			return false
		}
		return true
	}
	uid, _ := strconv.Atoi(parts[1])
	tokenApp, _ := strconv.Atoi(parts[0])
	if tokenApp != appID {
		errorJSON(c, 403, "登录令牌不属于当前软件")
		return false
	}
	var status string
	var exp sql.NullString
	if err := s.DB.QueryRow(`SELECT status,expire_at FROM users WHERE id=? AND app_id=?`, uid, appID).Scan(&status, &exp); err != nil {
		errorJSON(c, 403, "账号不存在")
		return false
	}
	if status != "active" {
		errorJSON(c, 403, "账号已被禁用")
		return false
	}
	if exp.Valid && strings.TrimSpace(exp.String) != "" {
		if t, parseErr := parseTime(exp.String); parseErr == nil && time.Now().In(beijingLocation).After(t) {
			errorJSON(c, 403, "账号已到期")
			return false
		}
	}
	return true
}

func normalizeStatus(v string) string {
	if v == "disabled" || v == "inactive" {
		return "disabled"
	}
	return "active"
}

func cloudVarExists(db *sql.DB, id string) bool {
	var n int
	return db.QueryRow(`SELECT COUNT(*) FROM cloud_vars WHERE id=?`, id).Scan(&n) == nil && n > 0
}
