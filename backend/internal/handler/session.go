package handler

import (
	"github.com/gin-gonic/gin"

	"glimmer-interview-hub/backend/internal/middleware"
	"glimmer-interview-hub/backend/internal/pkg/response"
	"glimmer-interview-hub/backend/internal/service"
)

type partitionReq struct {
	Date      string `json:"date"`
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
	Location  string `json:"location"`
	Capacity  int    `json:"capacity"` // 该日程每个时段可同时面试的人数（0 = 用场次级设置）
}

type slotCapacityReq struct {
	Date      string `json:"date"`
	StartTime string `json:"startTime"`
	Capacity  int    `json:"capacity"`
}

type createSessionReq struct {
	Title          string            `json:"title"`
	Direction      string            `json:"direction"`
	Date           string            `json:"date"`
	StartTime      string            `json:"startTime"`
	EndTime        string            `json:"endTime"`
	SlotMinutes    int               `json:"slotMinutes"`
	SlotCapacity   int               `json:"slotCapacity"` // 场次级：每个时段可同时面试的人数
	Location       string            `json:"location"`
	Contact        string            `json:"contact"`
	ContactQQ      string            `json:"contactQQ"`
	Partitions     []partitionReq    `json:"partitions"`
	SlotCapacities []slotCapacityReq `json:"slotCapacities"` // 逐时段容量覆盖
}

func (r createSessionReq) toInput() service.CreateSessionInput {
	in := service.CreateSessionInput{
		Title:        r.Title,
		Direction:    r.Direction,
		Date:         r.Date,
		StartTime:    r.StartTime,
		EndTime:      r.EndTime,
		SlotMinutes:  r.SlotMinutes,
		SlotCapacity: r.SlotCapacity,
		Location:     r.Location,
		Contact:      r.Contact,
		ContactQQ:    r.ContactQQ,
	}
	for _, p := range r.Partitions {
		in.Partitions = append(in.Partitions, service.PartitionInput{
			Date:      p.Date,
			StartTime: p.StartTime,
			EndTime:   p.EndTime,
			Location:  p.Location,
			Capacity:  p.Capacity,
		})
	}
	for _, sc := range r.SlotCapacities {
		in.SlotCapacities = append(in.SlotCapacities, service.SlotCapacityInput{
			Date:      sc.Date,
			StartTime: sc.StartTime,
			Capacity:  sc.Capacity,
		})
	}
	return in
}

// currentViewer 组装当前请求者视角（含全部已获邀请的方向）
func (h *Handler) currentViewer(c *gin.Context) service.Viewer {
	username := middleware.CurrentUser(c)
	dirs, _ := h.repo.ListUserDirections(username)
	v := service.Viewer{
		Username:   username,
		Directions: dirs,
		IsAdmin:    middleware.IsAdminUser(h.repo.DB, username, c.GetString(middleware.CtxRole)),
	}
	if len(dirs) > 0 {
		v.Direction = dirs[0]
	}
	return v
}

// ListSessions 场次列表（管理员看全部，成员只看本方向已发布）
func (h *Handler) ListSessions(c *gin.Context) {
	list, err := h.sessions.List(h.currentViewer(c))
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, gin.H{"sessions": list})
}

// Calendar 日历视图：区间内场次与时段（成员仅本方向）
func (h *Handler) Calendar(c *gin.Context) {
	start := c.Query("start")
	end := c.Query("end")
	if start == "" || end == "" {
		response.Fail(c, 400, "缺少 start/end 日期参数")
		return
	}
	list, err := h.sessions.Calendar(start, end, c.Query("direction"), h.currentViewer(c))
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, gin.H{"sessions": list})
}

// SessionDetail 场次详情：时段 + 首选人数 + 我的心仪时间/分配
func (h *Handler) SessionDetail(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	detail, err := h.sessions.Detail(id, h.currentViewer(c))
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, detail)
}

// CreateSession 管理员创建场次（草稿 + 自动切分时段）
func (h *Handler) CreateSession(c *gin.Context) {
	var req createSessionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "请求参数格式错误")
		return
	}
	sess, slots, err := h.sessions.Create(req.toInput(), middleware.CurrentUser(c))
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, gin.H{"session": sess, "slots": slots})
}

// UpdateSession 管理员编辑草稿场次
func (h *Handler) UpdateSession(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	var req createSessionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "请求参数格式错误")
		return
	}
	sess, slots, err := h.sessions.Update(id, req.toInput())
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, gin.H{"session": sess, "slots": slots})
}

// DeleteSession 删除草稿场次
func (h *Handler) DeleteSession(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	if err := h.sessions.Delete(id); err != nil {
		fail(c, err)
		return
	}
	response.OK(c, gin.H{"deleted": true})
}

// PublishSession 发布场次
func (h *Handler) PublishSession(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	if err := h.sessions.Publish(id); err != nil {
		fail(c, err)
		return
	}
	response.OK(c, gin.H{"published": true})
}
