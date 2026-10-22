package handler

import (
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"

	"glimmer-interview-hub/backend/internal/middleware"
	"glimmer-interview-hub/backend/internal/pkg/response"
)

// Health 健康检查（无需登录，供容器 healthcheck 使用）
func (h *Handler) Health(c *gin.Context) {
	response.OK(c, gin.H{"status": "ok"})
}

// Me 当前登录用户信息（同时记录用户出现过，用于未提交名单统计）
func (h *Handler) Me(c *gin.Context) {
	username := middleware.CurrentUser(c)
	studentid := middleware.CurrentStudentID(c)
	_ = h.repo.UpsertUser(username, studentid)

	dirs, _ := h.repo.ListUserDirections(username)
	name, qq, phone := "", "", ""
	if u, err := h.repo.GetUser(username); err == nil && u != nil {
		name, qq, phone = u.Name, u.QQ, u.Phone
		// 兼容历史单方向数据（未回填时也能看到方向）
		if len(dirs) == 0 && u.Direction != "" {
			dirs = []string{u.Direction}
		}
	}
	if dirs == nil {
		dirs = []string{}
	}
	direction := ""
	if len(dirs) > 0 {
		direction = dirs[0]
	}
	response.OK(c, gin.H{
		"username":   username,
		"studentid":  studentid,
		"isAdmin":    middleware.IsAdminUser(h.repo.DB, username, c.GetString(middleware.CtxRole)),
		"direction":  direction,
		"directions": dirs,
		"name":       name,
		"qq":         qq,
		"phone":      phone,
	})
}

type profileReq struct {
	Name  string `json:"name"`
	QQ    string `json:"qq"`
	Phone string `json:"phone"`
}

// SaveProfile 保存成员真实姓名与 QQ（管理端分配结果表需要）
func (h *Handler) SaveProfile(c *gin.Context) {
	var req profileReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "请求参数格式错误")
		return
	}
	name := strings.TrimSpace(req.Name)
	qq := strings.TrimSpace(req.QQ)
	phone := strings.TrimSpace(req.Phone)
	if name == "" {
		response.Fail(c, 400, "请填写真实姓名")
		return
	}
	if qq == "" {
		response.Fail(c, 400, "请填写 QQ 号")
		return
	}
	if phone == "" {
		response.Fail(c, 400, "请填写手机号")
		return
	}
	if utf8.RuneCountInString(name) > 20 || len(qq) > 15 || len(phone) > 11 {
		response.Fail(c, 400, "姓名、QQ 号或手机号过长")
		return
	}
	for _, r := range qq {
		if r < '0' || r > '9' {
			response.Fail(c, 400, "QQ 号应为纯数字")
			return
		}
	}
	if len(phone) != 11 || phone[0] != '1' {
		response.Fail(c, 400, "手机号格式不正确")
		return
	}
	for _, r := range phone {
		if r < '0' || r > '9' {
			response.Fail(c, 400, "手机号格式不正确")
			return
		}
	}
	username := middleware.CurrentUser(c)
	if err := h.repo.UpsertUser(username, middleware.CurrentStudentID(c)); err != nil {
		fail(c, err)
		return
	}
	if err := h.repo.UpdateUserProfile(username, name, qq, phone); err != nil {
		fail(c, err)
		return
	}
	response.OK(c, gin.H{"name": name, "qq": qq, "phone": phone})
}
