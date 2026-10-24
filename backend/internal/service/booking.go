package service

import (
	"errors"
	"sort"

	"glimmer-interview-hub/backend/internal/model"
	"gorm.io/gorm"

	"glimmer-interview-hub/backend/internal/repository"
)

// PreferencePick 一个心仪时段（含序）
type PreferencePick struct {
	Rank int        `json:"rank"`
	Slot model.Slot `json:"slot"`
}

// BookingUserItem 某场次下一位成员的预约详情（已提交至少 1 个心仪时间）
type BookingUserItem struct {
	Username  string           `json:"username"`
	Name      string           `json:"name"`
	StudentID string           `json:"studentid"`
	QQ        string           `json:"qq"`
	Phone     string           `json:"phone"`
	Full      bool             `json:"full"` // 是否已提交满 3 个
	Prefs     []PreferencePick `json:"prefs"`
}

// SessionBookings 某场次的全部预约详情（含时段，供前端日历可视化）
type SessionBookings struct {
	Session    model.Session     `json:"session"`
	Slots      []model.Slot      `json:"slots"`
	Partitions []PartitionView   `json:"partitions"`
	Bookings   []BookingUserItem `json:"bookings"`
	Pending    []model.User      `json:"pending"` // 未提交满 3 个心仪时间的成员
}

// BookingService 预约（心仪时间）管理业务
type BookingService struct {
	repo *repository.Repo
}

func NewBookingService(repo *repository.Repo) *BookingService {
	return &BookingService{repo: repo}
}

// DeleteUser 删除某成员在本场次的预约：心仪时间 + 本场次的分配记录一并撤下。
// 已确认场次同样允许（成员临时退出等场景），调用方需明确提示成员端结果会消失。
func (s *BookingService) DeleteUser(sessionID uint, username string) error {
	if _, err := s.repo.GetSession(sessionID); err != nil {
		return NotFound("场次不存在")
	}
	if _, err := s.repo.DeleteUserBooking(sessionID, username); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return NotFound("该成员在本场次没有预约")
		}
		return err
	}
	return nil
}

// Detail 某场次的全部预约详情：逐人心仪时间 + 未提交名单 + 全部时段
func (s *BookingService) Detail(sessionID uint) (*SessionBookings, error) {
	sess, err := s.repo.GetSession(sessionID)
	if err != nil {
		return nil, NotFound("场次不存在")
	}
	prefs, err := s.repo.AllPreferences(sessionID)
	if err != nil {
		return nil, err
	}
	slots, err := s.repo.GetSlots(sessionID)
	if err != nil {
		return nil, err
	}
	slotOf := make(map[uint]model.Slot, len(slots))
	for _, sl := range slots {
		slotOf[sl.ID] = sl
	}
	byUser := make(map[string][]PreferencePick)
	order := make([]string, 0, len(prefs)/2+1)
	for _, p := range prefs {
		if _, ok := byUser[p.Username]; !ok {
			order = append(order, p.Username)
		}
		byUser[p.Username] = append(byUser[p.Username], PreferencePick{Rank: p.Rank, Slot: slotOf[p.SlotID]})
	}
	profiles, err := s.repo.UsersByNames(order)
	if err != nil {
		return nil, err
	}
	bookings := make([]BookingUserItem, 0, len(order))
	for _, u := range order {
		prof := profiles[u]
		picks := byUser[u]
		sort.Slice(picks, func(i, j int) bool { return picks[i].Rank < picks[j].Rank })
		bookings = append(bookings, BookingUserItem{
			Username:  u,
			Name:      prof.Name,
			StudentID: prof.StudentID,
			QQ:        prof.QQ,
			Phone:     prof.Phone,
			Full:      len(picks) >= 3,
			Prefs:     picks,
		})
	}
	// 已提交满 3 个的排前面，其余按姓名/用户名排序，方便对照名单
	sort.Slice(bookings, func(i, j int) bool {
		if bookings[i].Full != bookings[j].Full {
			return bookings[i].Full
		}
		a, b := bookings[i].Name, bookings[j].Name
		if a == "" {
			a = bookings[i].Username
		}
		if b == "" {
			b = bookings[j].Username
		}
		return a < b
	})
	countByUser := make(map[string]int, len(prefs))
	for _, p := range prefs {
		countByUser[p.Username]++
	}
	users, err := s.repo.ListUsersByDirection(sess.Direction)
	if err != nil {
		return nil, err
	}
	pending := make([]model.User, 0)
	for _, u := range users {
		if countByUser[u.Username] < 3 {
			pending = append(pending, u)
		}
	}
	return &SessionBookings{
		Session:    *sess,
		Slots:      slots,
		Partitions: BuildPartitionViews(slots),
		Bookings:   bookings,
		Pending:    pending,
	}, nil
}
