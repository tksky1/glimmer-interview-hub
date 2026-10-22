package handler

import (
	"github.com/gin-gonic/gin"

	"glimmer-interview-hub/backend/internal/middleware"
	"glimmer-interview-hub/backend/internal/pkg/response"
)

type savePrefReq struct {
	SessionID uint   `json:"sessionId"`
	SlotIDs   []uint `json:"slotIds"`
}

// SavePreferences 保存三个心仪时间（整体替换）
func (h *Handler) SavePreferences(c *gin.Context) {
	var req savePrefReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "请求参数格式错误")
		return
	}
	v := h.currentViewer(c)
	if err := h.prefs.Save(req.SessionID, v.Username, middleware.CurrentStudentID(c), v, req.SlotIDs); err != nil {
		fail(c, err)
		return
	}
	response.OK(c, gin.H{"saved": true})
}

// MySchedule 我的预约总览（各场次心仪时间 + 已确认分配）
func (h *Handler) MySchedule(c *gin.Context) {
	data, err := h.prefs.MySchedule(middleware.CurrentUser(c))
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, data)
}
