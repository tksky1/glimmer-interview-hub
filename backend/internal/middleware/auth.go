package middleware

import (
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"glimmer-interview-hub/backend/internal/auth"
	"glimmer-interview-hub/backend/internal/model"
	"glimmer-interview-hub/backend/internal/pkg/response"
)

// 上下文键
const (
	CtxUsername  = "username"
	CtxStudentID = "studentid"
	CtxRole      = "role"
)

// AuthRequired 校验旧系统下发的 JWT。
// 策略：先用本地密钥验签（快路径，命中则零网络开销）；
// 验签失败则免验签提取身份，并委派旧系统 API 确认 token 有效（兼容线上任意密钥）。
func AuthRequired(secret string, validator *auth.LegacyValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("token")
		if token == "" {
			response.Fail(c, 401, "未登录")
			c.Abort()
			return
		}

		// 快路径：本地密钥验签通过
		if claims, err := auth.Parse(token, secret); err == nil {
			injectUser(c, claims)
			c.Next()
			return
		}

		// 委派路径：旧系统 API 是 token 的真伪权威
		claims, err := auth.DecodeUnverified(token)
		if err != nil {
			response.Fail(c, 401, "登录状态已失效，请重新登录")
			c.Abort()
			return
		}
		valid, reachable := validator.Validate(token)
		if !valid {
			// 旧 API 网络不可达时的降级：token 结构合法且未过期则放行，避免误锁
			if !reachable && !claims.IsExpired() {
				log.Printf("警告：旧系统 API 暂不可达，按未过期 token 放行用户 %s", claims.Username)
			} else {
				response.Fail(c, 401, "登录状态已失效，请重新登录")
				c.Abort()
				return
			}
		}
		injectUser(c, claims)
		c.Next()
	}
}

func injectUser(c *gin.Context, claims *auth.Claims) {
	c.Set(CtxUsername, claims.Username)
	c.Set(CtxStudentID, claims.StudentID)
	c.Set(CtxRole, claims.Role)
}

// IsAdminUser 是否管理员：内置名单 或 旧系统令牌中的 role=admin
func IsAdminUser(db *gorm.DB, username, role string) bool {
	if role == "admin" {
		return true
	}
	var count int64
	db.Model(&model.Admin{}).Where("username = ?", username).Count(&count)
	return count > 0
}

// RequireAdmin 要求当前用户为管理员
func RequireAdmin(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !IsAdminUser(db, c.GetString(CtxUsername), c.GetString(CtxRole)) {
			response.Fail(c, 403, "无管理员权限")
			c.Abort()
			return
		}
		c.Next()
	}
}

// CurrentUser 便捷取当前登录用户名
func CurrentUser(c *gin.Context) string {
	return c.GetString(CtxUsername)
}

// CurrentStudentID 便捷取当前登录学号
func CurrentStudentID(c *gin.Context) string {
	return c.GetString(CtxStudentID)
}
