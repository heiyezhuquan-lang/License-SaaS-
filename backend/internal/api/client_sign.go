package api

import (
	"bytes"
	"crypto/subtle"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const clientSignWindowSeconds = 300

func (s *Server) clientSignAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		ts := c.GetHeader("X-Timestamp")
		nonce := strings.TrimSpace(c.GetHeader("X-Nonce"))
		gotSig := strings.ToLower(strings.TrimSpace(c.GetHeader("X-Signature")))
		if ts == "" || nonce == "" || gotSig == "" {
			errorJSON(c, 401, "缺少客户端签名请求头")
			c.Abort()
			return
		}
		unix, err := strconv.ParseInt(ts, 10, 64)
		if err != nil {
			errorJSON(c, 401, "时间戳格式错误")
			c.Abort()
			return
		}
		nowUnix := time.Now().Unix()
		if unix < nowUnix-clientSignWindowSeconds || unix > nowUnix+clientSignWindowSeconds {
			errorJSON(c, 401, "请求时间戳已过期")
			c.Abort()
			return
		}
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			errorJSON(c, 400, "读取请求内容失败")
			c.Abort()
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		appKey := c.GetHeader("X-App-Key")
		if appKey == "" {
			appKey = c.Query("app_key")
		}
		if appKey == "" {
			appKey = c.Query("appKey")
		}
		if appKey == "" {
			errorJSON(c, 401, "缺少软件标识")
			c.Abort()
			return
		}
		var appSecret string
		if err := s.DB.QueryRow(`SELECT client_secret FROM apps WHERE app_key=? AND status='active'`, appKey).Scan(&appSecret); err != nil || appSecret == "" {
			errorJSON(c, 401, "软件不存在或已禁用")
			c.Abort()
			return
		}
		pathWithQuery := c.Request.URL.RequestURI()
		canonical := c.Request.Method + "\n" + pathWithQuery + "\n" + ts + "\n" + nonce + "\n" + string(bodyBytes)
		expect := sign(appSecret, canonical)
		if subtleCompareHex(expect, gotSig) == false {
			errorJSON(c, 401, "签名校验失败")
			c.Abort()
			return
		}
		_, _ = s.DB.Exec(`DELETE FROM client_nonces WHERE created_at < ?`, nowUnix-clientSignWindowSeconds)
		if _, err := s.DB.Exec(`INSERT INTO client_nonces(nonce,app_key,created_at) VALUES(?,?,?)`, nonce, appKey, unix); err != nil {
			errorJSON(c, 409, "请求已被使用，请更换随机串")
			c.Abort()
			return
		}
		c.Next()
	}
}

func subtleCompareHex(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func clientSigningString(method, pathWithQuery, timestamp, nonce, body string) string {
	return fmt.Sprintf("%s\n%s\n%s\n%s\n%s", method, pathWithQuery, timestamp, nonce, body)
}
