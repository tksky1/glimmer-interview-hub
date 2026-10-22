package handler

import (
	"github.com/gin-gonic/gin"

	"glimmer-interview-hub/backend/internal/middleware"
	"glimmer-interview-hub/backend/internal/pkg/response"
)

type addAdminReq struct {
	Username string `json:"username"`
}

// ListAdmins 管理员列表
func (h *Handler) ListAdmins(c *gin.Context) {
	list, err := h.admins.List()
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, gin.H{"admins": list})
}

// AddAdmin 添加管理员
func (h *Handler) AddAdmin(c *gin.Context) {
	var req addAdminReq
	if err := c.ShouldBindJSON(&req); err != nil || req.Username == "" {
		response.Fail(c, 400, "请填写用户名")
		return
	}
	if err := h.admins.Add(req.Username, middleware.CurrentUser(c)); err != nil {
		fail(c, err)
		return
	}
	response.OK(c, gin.H{"added": true})
}

// RemoveAdmin 移除管理员
func (h *Handler) RemoveAdmin(c *gin.Context) {
	username := c.Param("username")
	if username == "" {
		response.Fail(c, 400, "参数无效")
		return
	}
	if err := h.admins.Remove(username, middleware.CurrentUser(c)); err != nil {
		fail(c, err)
		return
	}
	response.OK(c, gin.H{"removed": true})
}
