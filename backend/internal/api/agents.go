package api

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"license-saas/backend/internal/auth"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func (s *Server) listAgents(c *gin.Context) {
	queryJSON(c, s.DB, `SELECT id,username,status,balance,remark,created_at,updated_at FROM agents ORDER BY id DESC`)
}

func (s *Server) createAgent(c *gin.Context) {
	var r struct {
		Username, Password, Remark string
		Balance                    float64
		AppIDs                     []int `json:"appIds"`
		CardTypeIDs                []int `json:"cardTypeIds"`
	}
	if c.ShouldBindJSON(&r) != nil || r.Username == "" || r.Password == "" {
		errorJSON(c, 400, "username/password required")
		return
	}
	h, _ := bcrypt.GenerateFromPassword([]byte(r.Password), bcrypt.DefaultCost)
	t := now()
	tx, err := s.DB.Begin()
	if err != nil {
		errorJSON(c, 500, err.Error())
		return
	}
	res, err := tx.Exec(`INSERT INTO agents(username,password_hash,status,balance,remark,created_at,updated_at) VALUES(?,?,?,?,?,?,?)`, r.Username, string(h), "active", r.Balance, r.Remark, t, t)
	if err != nil {
		tx.Rollback()
		errorJSON(c, 400, err.Error())
		return
	}
	agentID, _ := res.LastInsertId()
	if r.Balance != 0 {
		_, _ = tx.Exec(`INSERT INTO agent_balance_logs(agent_id,type,amount,before_balance,after_balance,remark,operator,created_at) VALUES(?,?,?,?,?,?,?,?)`, agentID, "init", r.Balance, 0, r.Balance, "初始余额", adminName(c), t)
	}
	saveAgentScopesTx(tx, int(agentID), r.AppIDs, r.CardTypeIDs, t)
	if err := tx.Commit(); err != nil {
		errorJSON(c, 500, err.Error())
		return
	}
	c.JSON(200, gin.H{"ok": true, "id": agentID})
}

func (s *Server) updateAgent(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var r struct {
		Username, Password, Status, Remark string
		AppIDs                             []int `json:"appIds"`
		CardTypeIDs                        []int `json:"cardTypeIds"`
	}
	if c.ShouldBindJSON(&r) != nil || id == 0 || r.Username == "" {
		errorJSON(c, 400, "bad request")
		return
	}
	tx, err := s.DB.Begin()
	if err != nil {
		errorJSON(c, 500, err.Error())
		return
	}
	if r.Password != "" {
		h, _ := bcrypt.GenerateFromPassword([]byte(r.Password), bcrypt.DefaultCost)
		_, err = tx.Exec(`UPDATE agents SET username=?,password_hash=?,status=?,remark=?,updated_at=? WHERE id=?`, r.Username, string(h), fallback(r.Status, "active"), r.Remark, now(), id)
	} else {
		_, err = tx.Exec(`UPDATE agents SET username=?,status=?,remark=?,updated_at=? WHERE id=?`, r.Username, fallback(r.Status, "active"), r.Remark, now(), id)
	}
	if err != nil {
		tx.Rollback()
		errorJSON(c, 400, err.Error())
		return
	}
	_, _ = tx.Exec(`DELETE FROM agent_app_scopes WHERE agent_id=?`, id)
	_, _ = tx.Exec(`DELETE FROM agent_card_type_scopes WHERE agent_id=?`, id)
	saveAgentScopesTx(tx, id, r.AppIDs, r.CardTypeIDs, now())
	if err := tx.Commit(); err != nil {
		errorJSON(c, 500, err.Error())
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

func saveAgentScopesTx(tx *sql.Tx, agentID int, appIDs, typeIDs []int, t string) {
	for _, appID := range appIDs {
		if appID > 0 {
			_, _ = tx.Exec(`INSERT OR IGNORE INTO agent_app_scopes(agent_id,app_id,created_at) VALUES(?,?,?)`, agentID, appID, t)
		}
	}
	for _, typeID := range typeIDs {
		if typeID > 0 {
			_, _ = tx.Exec(`INSERT OR IGNORE INTO agent_card_type_scopes(agent_id,card_type_id,created_at) VALUES(?,?,?)`, agentID, typeID, t)
		}
	}
}

func (s *Server) updateAgentStatus(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var r struct{ Status string }
	if c.ShouldBindJSON(&r) != nil || id == 0 {
		errorJSON(c, 400, "bad request")
		return
	}
	if r.Status != "active" && r.Status != "disabled" {
		errorJSON(c, 400, "bad status")
		return
	}
	res, err := s.DB.Exec(`UPDATE agents SET status=?,updated_at=? WHERE id=?`, r.Status, now(), id)
	if err != nil {
		errorJSON(c, 400, err.Error())
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		errorJSON(c, 404, "agent not found")
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

func (s *Server) deleteAgent(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id == 0 {
		errorJSON(c, 400, "bad request")
		return
	}
	tx, err := s.DB.Begin()
	if err != nil {
		errorJSON(c, 500, err.Error())
		return
	}
	if _, err = tx.Exec(`DELETE FROM cards WHERE agent_id=?`, id); err == nil {
		_, _ = tx.Exec(`DELETE FROM agent_app_scopes WHERE agent_id=?`, id)
		_, _ = tx.Exec(`DELETE FROM agent_card_type_scopes WHERE agent_id=?`, id)
		_, _ = tx.Exec(`DELETE FROM agent_balance_logs WHERE agent_id=?`, id)
		var res sql.Result
		res, err = tx.Exec(`DELETE FROM agents WHERE id=?`, id)
		if err == nil {
			if n, _ := res.RowsAffected(); n == 0 {
				err = sql.ErrNoRows
			}
		}
	}
	if err != nil {
		tx.Rollback()
		if err == sql.ErrNoRows {
			errorJSON(c, 404, "agent not found")
		} else {
			errorJSON(c, 400, err.Error())
		}
		return
	}
	if err := tx.Commit(); err != nil {
		errorJSON(c, 500, err.Error())
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

func (s *Server) adjustAgentBalance(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var r struct {
		Type, Remark string
		Amount       float64
	}
	if c.ShouldBindJSON(&r) != nil || id == 0 || r.Amount <= 0 {
		errorJSON(c, 400, "bad request")
		return
	}
	if r.Type != "deduct" {
		r.Type = "add"
	}
	tx, err := s.DB.Begin()
	if err != nil {
		errorJSON(c, 500, err.Error())
		return
	}
	var before float64
	if err := tx.QueryRow(`SELECT balance FROM agents WHERE id=?`, id).Scan(&before); err != nil {
		tx.Rollback()
		errorJSON(c, 404, "agent not found")
		return
	}
	after := before + r.Amount
	amount := r.Amount
	if r.Type == "deduct" {
		after = before - r.Amount
		amount = -r.Amount
	}
	if after < 0 {
		tx.Rollback()
		errorJSON(c, 400, "余额不足")
		return
	}
	_, err = tx.Exec(`UPDATE agents SET balance=?,updated_at=? WHERE id=?`, after, now(), id)
	if err == nil {
		_, err = tx.Exec(`INSERT INTO agent_balance_logs(agent_id,type,amount,before_balance,after_balance,remark,operator,created_at) VALUES(?,?,?,?,?,?,?,?)`, id, r.Type, amount, before, after, r.Remark, adminName(c), now())
	}
	if err != nil {
		tx.Rollback()
		errorJSON(c, 500, err.Error())
		return
	}
	if err := tx.Commit(); err != nil {
		errorJSON(c, 500, err.Error())
		return
	}
	c.JSON(200, gin.H{"ok": true, "balance": after})
}

func (s *Server) agentBalanceLogs(c *gin.Context) {
	queryJSON(c, s.DB, `SELECT * FROM agent_balance_logs WHERE agent_id=? ORDER BY id DESC LIMIT 200`, c.Param("id"))
}

func (s *Server) agentLogin(c *gin.Context) {
	var r struct{ Username, Password string }
	if c.ShouldBindJSON(&r) != nil {
		errorJSON(c, 400, "bad request")
		return
	}
	var id int
	var hash, status string
	if err := s.DB.QueryRow(`SELECT id,password_hash,status FROM agents WHERE username=?`, r.Username).Scan(&id, &hash, &status); err != nil || status != "active" || bcrypt.CompareHashAndPassword([]byte(hash), []byte(r.Password)) != nil {
		errorJSON(c, 401, "账号或密码错误")
		return
	}
	token, _ := auth.Issue(s.JWTSecret, "agent", fmt.Sprintf("%d:%s", id, r.Username), 24*time.Hour)
	c.JSON(200, gin.H{"ok": true, "token": token, "username": r.Username, "id": id})
}

func (s *Server) agentMe(c *gin.Context) {
	id, _ := agentID(c)
	queryJSON(c, s.DB, `SELECT id,username,status,balance,remark,created_at FROM agents WHERE id=?`, id)
}

func (s *Server) agentScopes(c *gin.Context) {
	id, _ := agentID(c)
	apps := rowsToMaps(s.DB, `SELECT apps.id,apps.name,apps.app_key FROM apps JOIN agent_app_scopes s ON s.app_id=apps.id WHERE s.agent_id=? AND apps.status='active'`, id)
	types := rowsToMaps(s.DB, `SELECT card_types.id,card_types.app_id,apps.name app_name,card_types.name,card_types.days,card_types.hours,card_types.max_devices,card_types.price FROM card_types JOIN apps ON apps.id=card_types.app_id JOIN agent_card_type_scopes s ON s.card_type_id=card_types.id WHERE s.agent_id=? AND card_types.status='active'`, id)
	c.JSON(200, gin.H{"ok": true, "data": gin.H{"apps": apps, "cardTypes": types}})
}

func (s *Server) agentCards(c *gin.Context) {
	id, _ := agentID(c)
	queryJSON(c, s.DB, `SELECT cards.id,apps.name app_name,cards.app_id,cards.card_key,cards.status,cards.expire_days,cards.expire_hours,cards.max_devices,cards.free_unbinds,cards.max_unbinds,cards.unbind_used,cards.unbind_deduct_hours,cards.cost,cards.used_at,cards.created_at,users.username used_by FROM cards JOIN apps ON apps.id=cards.app_id LEFT JOIN users ON users.id=cards.used_by_user_id WHERE cards.agent_id=? ORDER BY cards.id DESC LIMIT 500`, id)
}

func (s *Server) agentSalesReport(c *gin.Context) {
	id, _ := agentID(c)
	s.salesReport(c, id)
}

func (s *Server) agentGenerateCards(c *gin.Context) {
	agentID, agentUser := agentID(c)
	var r struct{ AppID, CardTypeID, Count int }
	if c.ShouldBindJSON(&r) != nil || r.AppID == 0 || r.CardTypeID == 0 || r.Count <= 0 {
		errorJSON(c, 400, "required")
		return
	}
	if r.Count > 200 {
		r.Count = 200
	}
	if !s.agentHasScope(agentID, r.AppID, r.CardTypeID) {
		errorJSON(c, 403, "无此软件或卡类权限")
		return
	}
	var days, hours, dev, freeUnbinds, maxUnbinds, deductHours int
	var price float64
	if err := s.DB.QueryRow(`SELECT days,hours,max_devices,free_unbinds,max_unbinds,unbind_deduct_hours,price FROM card_types WHERE id=? AND app_id=? AND status='active'`, r.CardTypeID, r.AppID).Scan(&days, &hours, &dev, &freeUnbinds, &maxUnbinds, &deductHours, &price); err != nil {
		errorJSON(c, 400, "card type invalid")
		return
	}
	if hours <= 0 {
		hours = days * 24
	}
	days = (hours + 23) / 24
	cost := price * float64(r.Count)
	tx, err := s.DB.Begin()
	if err != nil {
		errorJSON(c, 500, err.Error())
		return
	}
	var before float64
	if err := tx.QueryRow(`SELECT balance FROM agents WHERE id=? AND status='active'`, agentID).Scan(&before); err != nil {
		tx.Rollback()
		errorJSON(c, 401, "agent invalid")
		return
	}
	if before < cost {
		tx.Rollback()
		errorJSON(c, 400, "余额不足")
		return
	}
	after := before - cost
	_, err = tx.Exec(`UPDATE agents SET balance=?,updated_at=? WHERE id=?`, after, now(), agentID)
	keys := []string{}
	for i := 0; i < r.Count && err == nil; i++ {
		key := "LS-" + strings.ToUpper(randKey(4)+"-"+randKey(4)+"-"+randKey(4))
		_, err = tx.Exec(`INSERT INTO cards(app_id,card_type_id,card_key,status,expire_days,expire_hours,max_devices,free_unbinds,max_unbinds,unbind_deduct_hours,created_by,agent_id,cost,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, r.AppID, r.CardTypeID, key, "unused", days, hours, dev, freeUnbinds, maxUnbinds, deductHours, "agent:"+agentUser, agentID, price, now())
		if err == nil {
			keys = append(keys, key)
		}
	}
	if err == nil {
		_, err = tx.Exec(`INSERT INTO agent_balance_logs(agent_id,type,amount,before_balance,after_balance,remark,operator,created_at) VALUES(?,?,?,?,?,?,?,?)`, agentID, "issue_card", -cost, before, after, fmt.Sprintf("生成卡密 %d 张", r.Count), agentUser, now())
	}
	if err != nil {
		tx.Rollback()
		errorJSON(c, 500, err.Error())
		return
	}
	if err := tx.Commit(); err != nil {
		errorJSON(c, 500, err.Error())
		return
	}
	c.JSON(200, gin.H{"ok": true, "data": keys, "balance": after, "cost": cost})
}

func (s *Server) agentHasScope(agentID, appID, typeID int) bool {
	var n int
	_ = s.DB.QueryRow(`SELECT COUNT(*) FROM agent_app_scopes WHERE agent_id=? AND app_id=?`, agentID, appID).Scan(&n)
	if n == 0 {
		return false
	}
	_ = s.DB.QueryRow(`SELECT COUNT(*) FROM agent_card_type_scopes WHERE agent_id=? AND card_type_id=?`, agentID, typeID).Scan(&n)
	return n > 0
}

func agentID(c *gin.Context) (int, string) {
	raw, _ := c.Get("agent")
	parts := strings.SplitN(fmt.Sprint(raw), ":", 2)
	id, _ := strconv.Atoi(parts[0])
	name := "agent"
	if len(parts) > 1 {
		name = parts[1]
	}
	return id, name
}

func adminName(c *gin.Context) string {
	v, _ := c.Get("admin")
	if s := fmt.Sprint(v); s != "" && s != "<nil>" {
		return s
	}
	return "admin"
}

func rowsToMaps(db *sql.DB, q string, args ...any) []map[string]any {
	rows, err := db.Query(q, args...)
	if err != nil {
		return []map[string]any{}
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
	return out
}
