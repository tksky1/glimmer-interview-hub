package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"glimmer-interview-hub/backend/internal/middleware"
	"glimmer-interview-hub/backend/internal/pkg/response"
)

type reassignReq struct {
	SlotID uint `json:"slotId"`
}

type swapReq struct {
	OtherID uint `json:"otherId"`
}

type manualAssignReq struct {
	Username string `json:"username"`
	SlotID   uint   `json:"slotId"`
}

// GenerateAssignment 生成自动分配版本
func (h *Handler) GenerateAssignment(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	ver, err := h.assigns.Generate(id, middleware.CurrentUser(c))
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, gin.H{"version": ver})
}

// GetAssignment 查看分配（默认最新版本）
func (h *Handler) GetAssignment(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	var versionID uint
	if v := c.Query("versionId"); v != "" {
		n, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			response.Fail(c, 400, "versionId 参数无效")
			return
		}
		versionID = uint(n)
	}
	view, err := h.assigns.Get(id, versionID)
	if err != nil {
		fail(c, err)
		return
	}
	if view == nil {
		// 尚未生成分配：正常空状态
		response.OK(c, nil)
		return
	}
	response.OK(c, view)
}

// Reassign 改派到空闲时段
func (h *Handler) Reassign(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	var req reassignReq
	if err := c.ShouldBindJSON(&req); err != nil || req.SlotID == 0 {
		response.Fail(c, 400, "请选择目标时段")
		return
	}
	if err := h.assigns.Reassign(id, req.SlotID); err != nil {
		fail(c, err)
		return
	}
	response.OK(c, gin.H{"updated": true})
}

// Swap 对调两名成员的时段
func (h *Handler) Swap(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	var req swapReq
	if err := c.ShouldBindJSON(&req); err != nil || req.OtherID == 0 {
		response.Fail(c, 400, "请选择对调对象")
		return
	}
	if err := h.assigns.Swap(id, req.OtherID); err != nil {
		fail(c, err)
		return
	}
	response.OK(c, gin.H{"swapped": true})
}

// ManualAssign 把未分配成员安排到空闲时段
func (h *Handler) ManualAssign(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	var req manualAssignReq
	if err := c.ShouldBindJSON(&req); err != nil || req.Username == "" || req.SlotID == 0 {
		response.Fail(c, 400, "请提供用户名与目标时段")
		return
	}
	if err := h.assigns.AssignUser(id, req.Username, req.SlotID, middleware.CurrentUser(c)); err != nil {
		fail(c, err)
		return
	}
	response.OK(c, gin.H{"assigned": true})
}

// ConfirmAssignment 确认分配
func (h *Handler) ConfirmAssignment(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	ver, err := h.assigns.Confirm(id)
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, gin.H{"version": ver})
}
