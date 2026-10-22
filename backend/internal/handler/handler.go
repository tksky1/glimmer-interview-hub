package handler

import (
	"log"
	"strconv"

	"github.com/gin-gonic/gin"

	"glimmer-interview-hub/backend/internal/auth"
	"glimmer-interview-hub/backend/internal/middleware"
	"glimmer-interview-hub/backend/internal/pkg/response"
	"glimmer-interview-hub/backend/internal/repository"
	"glimmer-interview-hub/backend/internal/service"
)

// Handler 聚合各业务服务与路由注册
type Handler struct {
	repo      *repository.Repo
	sessions  *service.SessionService
	prefs     *service.PreferenceService
	assigns   *service.AssignmentService
	admins    *service.AdminService
	invites   *service.InviteService
	jwtSecret string
	validator *auth.LegacyValidator
}

// New 构造 Handler
func New(repo *repository.Repo, jwtSecret string, validator *auth.LegacyValidator) *Handler {
	return &Handler{
		repo:      repo,
		sessions:  service.NewSessionService(repo),
		prefs:     service.NewPreferenceService(repo),
		assigns:   service.NewAssignmentService(repo),
		admins:    service.NewAdminService(repo),
		invites:   service.NewInviteService(repo),
		jwtSecret: jwtSecret,
		validator: validator,
	}
}

// Register 注册全部路由
func (h *Handler) Register(r *gin.Engine) {
	api := r.Group("/api")
	api.GET("/health", h.Health)

	authed := api.Group("", middleware.AuthRequired(h.jwtSecret, h.validator))
	{
		authed.GET("/me", h.Me)
		authed.PUT("/profile", h.SaveProfile)
		authed.GET("/sessions", h.ListSessions)
		authed.GET("/sessions/calendar", h.Calendar)
		authed.GET("/sessions/:id", h.SessionDetail)
		authed.PUT("/preferences", h.SavePreferences)
		authed.GET("/my/schedule", h.MySchedule)
		authed.POST("/invite/redeem", h.RedeemInvite)
	}

	admin := authed.Group("/admin", middleware.RequireAdmin(h.repo.DB))
	{
		admin.POST("/sessions", h.CreateSession)
		admin.PUT("/sessions/:id", h.UpdateSession)
		admin.DELETE("/sessions/:id", h.DeleteSession)
		admin.POST("/sessions/:id/publish", h.PublishSession)
		admin.POST("/sessions/:id/assign", h.GenerateAssignment)
		admin.GET("/sessions/:id/assignment", h.GetAssignment)
		admin.PUT("/assignments/:id", h.Reassign)
		admin.POST("/sessions/:id/manual-assign", h.ManualAssign)
		admin.POST("/assignments/:id/swap", h.Swap)
		admin.POST("/sessions/:id/confirm", h.ConfirmAssignment)
		admin.GET("/admins", h.ListAdmins)
		admin.POST("/admins", h.AddAdmin)
		admin.DELETE("/admins/:username", h.RemoveAdmin)
		admin.GET("/invites", h.ListInvites)
		admin.POST("/invites", h.GenerateInvites)
		admin.DELETE("/invites/:id", h.DeleteInvite)
	}
}

// parseID 解析路径中的数字 ID
func parseID(c *gin.Context, name string) (uint, bool) {
	v, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil || v == 0 {
		response.Fail(c, 400, "参数无效")
		return 0, false
	}
	return uint(v), true
}

// fail 统一错误响应
func fail(c *gin.Context, err error) {
	code, msg := service.ToHTTP(err)
	if code >= 500 {
		log.Printf("内部错误 %s %s: %v", c.Request.Method, c.FullPath(), err)
	}
	response.Fail(c, code, msg)
}
