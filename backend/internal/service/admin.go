package service

import (
	"glimmer-interview-hub/backend/internal/model"
	"glimmer-interview-hub/backend/internal/repository"
)

// AdminService 管理员名单业务
type AdminService struct {
	repo *repository.Repo
}

func NewAdminService(repo *repository.Repo) *AdminService {
	return &AdminService{repo: repo}
}

// List 管理员列表
func (s *AdminService) List() ([]model.Admin, error) {
	return s.repo.ListAdmins()
}

// Add 添加管理员
func (s *AdminService) Add(username, operator string) error {
	if username == "" {
		return BadRequest("用户名不能为空")
	}
	if username == operator {
		return BadRequest("不能添加自己")
	}
	if s.repo.IsAdmin(username) {
		return Conflict("该用户已是管理员")
	}
	return s.repo.AddAdmin(username)
}

// Remove 移除管理员（至少保留一名）
func (s *AdminService) Remove(username, operator string) error {
	if username == operator {
		return BadRequest("不能移除自己")
	}
	n, err := s.repo.CountAdmins()
	if err != nil {
		return err
	}
	if n <= 1 {
		return BadRequest("至少需要保留一名管理员")
	}
	return s.repo.DeleteAdmin(username)
}
