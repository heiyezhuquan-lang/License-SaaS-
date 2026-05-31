package api

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func (s *Server) exportCards(c *gin.Context) {
	filters := cardFilters(c)
	rows, err := s.DB.Query(`SELECT cards.card_key FROM cards JOIN apps ON apps.id=cards.app_id LEFT JOIN users ON users.id=cards.used_by_user_id LEFT JOIN agents ON agents.id=cards.agent_id WHERE (`+filters.where+`) ORDER BY cards.id DESC LIMIT 5000`, filters.args...)
	if err != nil {
		errorJSON(c, 500, err.Error())
		return
	}
	defer rows.Close()
	keys := []string{}
	for rows.Next() {
		var k string
		if rows.Scan(&k) == nil {
			keys = append(keys, k)
		}
	}
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=cards.txt")
	c.String(200, strings.Join(keys, "\n"))
}

type sqlFilter struct {
	where string
	args  []any
}

func cardFilters(c *gin.Context) sqlFilter {
	parts := []string{"1=1"}
	args := []any{}
	if v := c.Query("status"); v != "" {
		parts = append(parts, "cards.status=?")
		args = append(args, v)
	}
	if v := c.Query("app_id"); v != "" {
		parts = append(parts, "cards.app_id=?")
		args = append(args, v)
	}
	if v := c.Query("agent_id"); v != "" {
		parts = append(parts, "cards.agent_id=?")
		args = append(args, v)
	}
	if v := c.Query("card_type_id"); v != "" {
		parts = append(parts, "cards.card_type_id=?")
		args = append(args, v)
	}
	if v := strings.TrimSpace(c.Query("start_date")); v != "" {
		parts = append(parts, "date(cards.created_at) >= date(?)")
		args = append(args, v)
	}
	if v := strings.TrimSpace(c.Query("end_date")); v != "" {
		parts = append(parts, "date(cards.created_at) <= date(?)")
		args = append(args, v)
	}
	if v := strings.TrimSpace(c.Query("keyword")); v != "" {
		like := "%" + v + "%"
		parts = append(parts, "(cards.card_key LIKE ? OR users.username LIKE ? OR agents.username LIKE ?)")
		args = append(args, like, like, like)
	}
	return sqlFilter{where: strings.Join(parts, " AND "), args: args}
}

func salesFilters(c *gin.Context, forcedAgentID any) sqlFilter {
	parts := []string{"1=1"}
	args := []any{}
	timeColumn := "cards.created_at"
	if c.Query("time_field") == "used_at" || c.Query("time_field") == "activated_at" {
		timeColumn = "cards.used_at"
	}
	if forcedAgentID != nil {
		parts = append(parts, "cards.agent_id=?")
		args = append(args, forcedAgentID)
	} else if v := c.Query("agent_id"); v != "" {
		parts = append(parts, "cards.agent_id=?")
		args = append(args, v)
	}
	if v := c.Query("app_id"); v != "" {
		parts = append(parts, "cards.app_id=?")
		args = append(args, v)
	}
	if v := c.Query("card_type_id"); v != "" {
		parts = append(parts, "cards.card_type_id=?")
		args = append(args, v)
	}
	dateFiltered := false
	if v := strings.TrimSpace(c.Query("start_date")); v != "" {
		parts = append(parts, "date("+timeColumn+") >= date(?)")
		args = append(args, v)
		dateFiltered = true
	}
	if v := strings.TrimSpace(c.Query("end_date")); v != "" {
		parts = append(parts, "date("+timeColumn+") <= date(?)")
		args = append(args, v)
		dateFiltered = true
	}
	if timeColumn == "cards.used_at" && dateFiltered {
		parts = append(parts, "cards.used_at IS NOT NULL AND cards.used_at<>''")
	}
	if c.Query("include_disabled") != "1" && c.Query("include_disabled") != "true" {
		parts = append(parts, "cards.status<>'disabled'")
	}
	return sqlFilter{where: strings.Join(parts, " AND "), args: args}
}

func (s *Server) salesReport(c *gin.Context, forcedAgentID any) {
	filters := salesFilters(c, forcedAgentID)
	var total, unused, used, disabled int
	var amount float64
	_ = s.DB.QueryRow(`SELECT COUNT(*),COALESCE(SUM(cards.cost),0),SUM(CASE WHEN cards.status='unused' THEN 1 ELSE 0 END),SUM(CASE WHEN cards.status='used' THEN 1 ELSE 0 END),SUM(CASE WHEN cards.status='disabled' THEN 1 ELSE 0 END) FROM cards WHERE `+filters.where, filters.args...).Scan(&total, &amount, &unused, &used, &disabled)
	recent := rowsToMaps(s.DB, `SELECT cards.id,apps.name app_name,card_types.name card_type_name,cards.card_key,cards.status,cards.cost,cards.created_at,cards.used_at,agents.username agent_name,users.username used_by FROM cards JOIN apps ON apps.id=cards.app_id LEFT JOIN card_types ON card_types.id=cards.card_type_id LEFT JOIN agents ON agents.id=cards.agent_id LEFT JOIN users ON users.id=cards.used_by_user_id WHERE `+filters.where+` ORDER BY cards.id DESC LIMIT 500`, filters.args...)
	byAgent := rowsToMaps(s.DB, `SELECT COALESCE(agents.username,'管理员') agent_name,COUNT(*) count,COALESCE(SUM(cards.cost),0) amount FROM cards LEFT JOIN agents ON agents.id=cards.agent_id WHERE `+filters.where+` GROUP BY cards.agent_id,agents.username ORDER BY count DESC`, filters.args...)
	byType := rowsToMaps(s.DB, `SELECT COALESCE(card_types.name,'未指定') card_type_name,COUNT(*) count,COALESCE(SUM(cards.cost),0) amount FROM cards LEFT JOIN card_types ON card_types.id=cards.card_type_id WHERE `+filters.where+` GROUP BY cards.card_type_id,card_types.name ORDER BY count DESC`, filters.args...)
	c.JSON(200, gin.H{"ok": true, "data": gin.H{"summary": gin.H{"cards": total, "amount": amount, "unused_cards": unused, "used_cards": used, "disabled_cards": disabled}, "recent": recent, "by_agent": byAgent, "by_type": byType}})
}

func (s *Server) adminSalesReport(c *gin.Context) { s.salesReport(c, nil) }

func (s *Server) listDevices(c *gin.Context) {
	parts := []string{"1=1"}
	args := []any{}
	if v := c.Query("status"); v != "" {
		parts = append(parts, "devices.status=?")
		args = append(args, v)
	}
	if v := c.Query("app_id"); v != "" {
		parts = append(parts, "devices.app_id=?")
		args = append(args, v)
	}
	if v := strings.TrimSpace(c.Query("keyword")); v != "" {
		like := "%" + v + "%"
		parts = append(parts, "(devices.machine_code LIKE ? OR users.username LIKE ? OR cards.card_key LIKE ? OR devices.ip LIKE ? OR devices.client_version LIKE ?)")
		args = append(args, like, like, like, like, like)
	}
	q := `SELECT devices.id,devices.app_id,apps.name app_name,apps.heartbeat_timeout,COALESCE(devices.auth_mode,'account') auth_mode,COALESCE(devices.user_id,0) user_id,COALESCE(devices.card_id,0) card_id,CASE WHEN COALESCE(devices.auth_mode,'account')='card' THEN cards.card_key ELSE users.username END username,devices.machine_code,devices.status,devices.client_version,devices.ip,devices.last_seen,devices.created_at FROM devices JOIN apps ON apps.id=devices.app_id LEFT JOIN users ON users.id=devices.user_id LEFT JOIN cards ON cards.id=devices.card_id WHERE ` + strings.Join(parts, " AND ") + ` ORDER BY devices.last_seen DESC LIMIT 500`
	allData, onlineCount := s.devicesWithOnline(q, args...)
	data := allData
	if c.Query("online") == "1" || c.Query("online") == "true" {
		data = []map[string]any{}
		for _, d := range allData {
			if d["online_status"] == "online" {
				data = append(data, d)
			}
		}
	}
	c.JSON(200, gin.H{"ok": true, "data": data, "online_count": onlineCount, "total": len(data), "all_total": len(allData)})
}

func (s *Server) devicesWithOnline(q string, args ...any) ([]map[string]any, int) {
	rows, err := s.DB.Query(q, args...)
	if err != nil {
		return []map[string]any{}, 0
	}
	defer rows.Close()
	cols, _ := rows.Columns()
	out := []map[string]any{}
	onlineCount := 0
	for rows.Next() {
		vals := make([]any, len(cols))
		ptr := make([]any, len(cols))
		for i := range vals {
			ptr[i] = &vals[i]
		}
		if rows.Scan(ptr...) != nil {
			continue
		}
		m := map[string]any{}
		for i, col := range cols {
			if b, ok := vals[i].([]byte); ok {
				m[col] = string(b)
			} else {
				m[col] = vals[i]
			}
		}
		timeout := int64(180)
		if v, ok := m["heartbeat_timeout"].(int64); ok && v > 0 {
			timeout = v
		}
		lastSeen, _ := m["last_seen"].(string)
		isOnline := m["status"] == "active" && seenWithin(lastSeen, time.Duration(timeout)*time.Second)
		if isOnline {
			onlineCount++
			m["online_status"] = "online"
			m["online_text"] = "在线"
		} else {
			m["online_status"] = "offline"
			m["online_text"] = "离线"
		}
		delete(m, "heartbeat_timeout")
		out = append(out, m)
	}
	return out, onlineCount
}

func seenWithin(lastSeen string, maxAge time.Duration) bool {
	if maxAge <= 0 {
		maxAge = 180 * time.Second
	}
	t, err := parseTime(lastSeen)
	if err != nil {
		return false
	}
	return time.Since(t) <= maxAge
}

func (s *Server) updateDeviceStatus(c *gin.Context) {
	var r struct{ Status string }
	if c.ShouldBindJSON(&r) != nil {
		errorJSON(c, 400, "bad request")
		return
	}
	if r.Status != "active" && r.Status != "banned" && r.Status != "disabled" {
		errorJSON(c, 400, "bad status")
		return
	}
	_, err := s.DB.Exec(`UPDATE devices SET status=? WHERE id=?`, r.Status, c.Param("id"))
	if err != nil {
		errorJSON(c, 400, err.Error())
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

func (s *Server) enableCard(c *gin.Context) {
	_, err := s.DB.Exec(`UPDATE cards SET status='unused' WHERE id=? AND status='disabled'`, c.Param("id"))
	if err != nil {
		errorJSON(c, 400, err.Error())
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

func (s *Server) agentStats(c *gin.Context) {
	id, _ := agentID(c)
	data := gin.H{}
	queries := map[string]string{
		"cards":        "SELECT COUNT(*) FROM cards WHERE agent_id=?",
		"unused_cards": "SELECT COUNT(*) FROM cards WHERE agent_id=? AND status='unused'",
		"used_cards":   "SELECT COUNT(*) FROM cards WHERE agent_id=? AND status='used'",
		"today_cards":  "SELECT COUNT(*) FROM cards WHERE agent_id=? AND date(created_at)=date('now','localtime')",
		"month_cards":  "SELECT COUNT(*) FROM cards WHERE agent_id=? AND strftime('%Y-%m',created_at)=strftime('%Y-%m','now','localtime')",
	}
	for k, q := range queries {
		var n int
		_ = s.DB.QueryRow(q, id).Scan(&n)
		data[k] = n
	}
	var cost float64
	_ = s.DB.QueryRow(`SELECT COALESCE(SUM(cost),0) FROM cards WHERE agent_id=?`, id).Scan(&cost)
	data["total_cost"] = cost
	c.JSON(200, gin.H{"ok": true, "data": data})
}

func (s *Server) agentOwnBalanceLogs(c *gin.Context) {
	id, _ := agentID(c)
	queryJSON(c, s.DB, `SELECT * FROM agent_balance_logs WHERE agent_id=? ORDER BY id DESC LIMIT 100`, id)
}
