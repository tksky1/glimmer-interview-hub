package service

import (
	"crypto/rand"
	"time"

	"glimmer-interview-hub/backend/internal/model"
	"glimmer-interview-hub/backend/internal/repository"
)

// InviteService 方向邀请码业务
type InviteService struct {
	repo *repository.Repo
}

func NewInviteService(repo *repository.Repo) *InviteService {
	return &InviteService{repo: repo}
}

// GenerateInput 生成邀请码入参
type GenerateInput struct {
	Direction string
	Note      string
	Count     int
}

// Generate 批量生成邀请码
func (s *InviteService) Generate(in GenerateInput, admin string) ([]model.Invite, error) {
	if !model.ValidDirection(in.Direction) {
		return nil, BadRequest("请选择方向")
	}
	if in.Count <= 0 || in.Count > 50 {
		in.Count = 1
	}
	out := make([]model.Invite, 0, in.Count)
	for i := 0; i < in.Count; i++ {
		code, err := randomCode(8)
		if err != nil {
			return nil, err
		}
		inv := model.Invite{
			Code:      code,
			Direction: in.Direction,
			Note:      in.Note,
			CreatedBy: admin,
			CreatedAt: time.Now(),
		}
		if err := s.repo.CreateInvite(&inv); err != nil {
			return nil, err
		}
		out = append(out, inv)
	}
	return out, nil
}

// List 邀请码列表
func (s *InviteService) List() ([]model.Invite, error) {
	return s.repo.ListInvites()
}

// Delete 删除邀请码
func (s *InviteService) Delete(id uint) error {
	return s.repo.DeleteInvite(id)
}

// Redeem 使用邀请码绑定当前用户的方向。
// 一个同学可能参加多个方向的面试，因此核销邀请码是「追加方向」而不是覆盖：
// 已绑定 A 方向的同学再核销 B 方向邀请码后，A、B 两个方向都可以预约。
func (s *InviteService) Redeem(code, username string) (string, error) {
	if code == "" {
		return "", BadRequest("请输入邀请码")
	}
	inv, err := s.repo.GetInviteByCode(code)
	if err != nil {
		return "", BadRequest("邀请码不存在，请检查后重试")
	}
	user, err := s.repo.GetUser(username)
	if err != nil {
		return "", err
	}
	if user == nil {
		return "", BadRequest("用户不存在，请刷新后重试")
	}
	dirs, err := s.repo.ListUserDirections(username)
	if err != nil {
		return "", err
	}
	bound := false
	for _, d := range dirs {
		if d == inv.Direction {
			bound = true
			break
		}
	}
	if !bound {
		if err := s.repo.AddUserDirection(username, inv.Direction); err != nil {
			return "", err
		}
	}
	// 兼容旧字段：保留首个绑定方向
	if user.Direction == "" {
		if err := s.repo.SetUserDirection(username, inv.Direction); err != nil {
			return "", err
		}
	}
	// 邀请码不限制使用次数：同一人重复核销幂等，不同人核销累加人数
	first, err := s.repo.AddInviteRedemption(inv.Code, username)
	if err != nil {
		return "", err
	}
	if first {
		now := time.Now()
		inv.UsedBy = username
		inv.UsedAt = &now
		inv.UsedCount++
		if err := s.repo.UpdateInvite(inv); err != nil {
			return "", err
		}
	}
	return inv.Direction, nil
}

// randomCode 生成 Crockford Base32 随机码（去掉易混淆字符）
func randomCode(n int) (string, error) {
	const alphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	for i := range b {
		b[i] = alphabet[int(b[i])%len(alphabet)]
	}
	return string(b), nil
}
