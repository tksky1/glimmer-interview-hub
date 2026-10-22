package handler

import (
	"github.com/gin-gonic/gin"

	"glimmer-interview-hub/backend/internal/middleware"
	"glimmer-interview-hub/backend/internal/pkg/response"
	"glimmer-interview-hub/backend/internal/service"
)

type redeemReq struct {
	Code string `json:"code"`
}

type generateInviteReq struct {
	Direction string `json:"direction"`
	Note      string `json:"note"`
	Count     int    `json:"count"`
}

// RedeemInvite 使用邀请码绑定方向（成员）
func (h *Handler) RedeemInvite(c *gin.Context) {
	var req redeemReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "请求参数格式错误")
		return
	}
	direction, err := h.invites.Redeem(req.Code, middleware.CurrentUser(c))
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, gin.H{"direction": direction})
}

// ListInvites 邀请码列表（管理员）
func (h *Handler) ListInvites(c *gin.Context) {
	list, err := h.invites.List()
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, gin.H{"invites": list})
}

// GenerateInvites 生成邀请码（管理员）
func (h *Handler) GenerateInvites(c *gin.Context) {
	var req generateInviteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "请求参数格式错误")
		return
	}
	list, err := h.invites.Generate(service.GenerateInput{
		Direction: req.Direction,
		Note:      req.Note,
		Count:     req.Count,
	}, middleware.CurrentUser(c))
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, gin.H{"invites": list})
}

// DeleteInvite 删除邀请码（管理员）
func (h *Handler) DeleteInvite(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	if err := h.invites.Delete(id); err != nil {
		fail(c, err)
		return
	}
	response.OK(c, gin.H{"deleted": true})
}
