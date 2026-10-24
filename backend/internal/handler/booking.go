package handler

import (
	"strings"

	"github.com/gin-gonic/gin"

	"glimmer-interview-hub/backend/internal/pkg/response"
)

// SessionBookings 某场次的全部预约详情（逐人心仪时间 + 未提交名单 + 时段）
func (h *Handler) SessionBookings(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	data, err := h.bookings.Detail(id)
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, data)
}

// DeleteBooking 删除某成员在本场次的预约（心仪时间 + 分配记录一并撤下）
func (h *Handler) DeleteBooking(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	username := strings.TrimSpace(c.Param("username"))
	if username == "" {
		response.Fail(c, 400, "缺少用户名")
		return
	}
	if err := h.bookings.DeleteUser(id, username); err != nil {
		fail(c, err)
		return
	}
	response.OK(c, gin.H{"deleted": true})
}
