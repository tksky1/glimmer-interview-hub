package service

import (
	"glimmer-interview-hub/backend/internal/model"
	"glimmer-interview-hub/backend/internal/repository"
)

// PreferenceService 偏好业务
type PreferenceService struct {
	repo *repository.Repo
}

func NewPreferenceService(repo *repository.Repo) *PreferenceService {
	return &PreferenceService{repo: repo}
}

// Save 保存三个心仪时间（整体替换，按序即第 1/2/3 心仪）
func (s *PreferenceService) Save(sessionID uint, username, studentid string, v Viewer, slotIDs []uint) error {
	if len(slotIDs) != 3 {
		return BadRequest("必须选择 3 个心仪时间")
	}
	seen := make(map[uint]bool, 3)
	for _, id := range slotIDs {
		if id == 0 {
			return BadRequest("存在无效的时段")
		}
		if seen[id] {
			return BadRequest("三个心仪时间不能选择相同的时段")
		}
		seen[id] = true
	}
	sess, err := s.repo.GetSession(sessionID)
	if err != nil {
		return NotFound("场次不存在")
	}
	if len(v.Directions) == 0 && v.Direction == "" {
		return BadRequest("请先通过邀请码绑定方向，再选择心仪时间")
	}
	if !v.CanAccessDirection(sess.Direction) {
		return BadRequest("只能选择你已收到邀请的方向的面试日程")
	}
	if sess.Status == model.StatusDraft {
		return BadRequest("场次尚未发布")
	}
	if sess.Status == model.StatusConfirmed {
		return BadRequest("分配已确认，心仪时间已锁定，如需修改请联系管理员")
	}
	slots, err := s.repo.GetSlots(sessionID)
	if err != nil {
		return err
	}
	valid := make(map[uint]bool, len(slots))
	for _, slot := range slots {
		valid[slot.ID] = true
	}
	for _, id := range slotIDs {
		if !valid[id] {
			return BadRequest("存在不属于本场次的时段")
		}
	}
	// 同一方向只能预约一个场次：保存本场次前清空该方向其它场次的心仪时间
	locked, err := s.repo.ConfirmedSessionsWithPreferences(username, sess.Direction, sessionID)
	if err != nil {
		return err
	}
	if len(locked) > 0 {
		return BadRequest("该方向已有已确认的面试场次，无法更换；如需调整请联系管理员")
	}
	if _, err := s.repo.ClearPreferencesInDirection(username, sess.Direction, sessionID); err != nil {
		return err
	}
	return s.repo.ReplacePreferences(sessionID, username, studentid, slotIDs)
}

// MySchedule 我的预约总览：各场次心仪时间 + 已确认分配
func (s *PreferenceService) MySchedule(username string) (map[string]interface{}, error) {
	prefs, err := s.repo.MyAllPreferences(username)
	if err != nil {
		return nil, err
	}
	assigns, err := s.repo.MyAllConfirmedAssignments(username)
	if err != nil {
		return nil, err
	}
	// 汇总涉及的场次
	sessionIDs := make(map[uint]bool)
	for _, p := range prefs {
		sessionIDs[p.SessionID] = true
	}
	for _, a := range assigns {
		sessionIDs[a.SessionID] = true
	}
	sessions := make([]model.Session, 0, len(sessionIDs))
	for id := range sessionIDs {
		sess, err := s.repo.GetSession(id)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, *sess)
	}
	slots := make(map[uint]model.Slot)
	for id := range sessionIDs {
		list, err := s.repo.GetSlots(id)
		if err != nil {
			return nil, err
		}
		for _, sl := range list {
			slots[sl.ID] = sl
		}
	}
	// 各场次的日程列表（多日程场次展示日期区间）
	partitionsOf := make(map[uint][]PartitionView)
	for id := range sessionIDs {
		list, err := s.repo.GetSlots(id)
		if err != nil {
			return nil, err
		}
		partitionsOf[id] = BuildPartitionViews(list)
	}
	type prefItem struct {
		Session    model.Session   `json:"session"`
		Partitions []PartitionView `json:"partitions"`
		Rank       int             `json:"rank"`
		Slot       model.Slot      `json:"slot"`
	}
	type assignItem struct {
		Session    model.Session   `json:"session"`
		Partitions []PartitionView `json:"partitions"`
		Slot       model.Slot      `json:"slot"`
	}
	prefItems := make([]prefItem, 0, len(prefs))
	for _, p := range prefs {
		sess, err := s.repo.GetSession(p.SessionID)
		if err != nil {
			return nil, err
		}
		prefItems = append(prefItems, prefItem{
			Session:    *sess,
			Partitions: partitionsOf[p.SessionID],
			Rank:       p.Rank,
			Slot:       slots[p.SlotID],
		})
	}
	assignItems := make([]assignItem, 0, len(assigns))
	for _, a := range assigns {
		sess, err := s.repo.GetSession(a.SessionID)
		if err != nil {
			return nil, err
		}
		assignItems = append(assignItems, assignItem{
			Session:    *sess,
			Partitions: partitionsOf[a.SessionID],
			Slot:       slots[a.SlotID],
		})
	}
	return map[string]interface{}{
		"preferences": prefItems,
		"assignments": assignItems,
	}, nil
}
