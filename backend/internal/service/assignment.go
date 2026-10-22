package service

import (
	"encoding/json"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"glimmer-interview-hub/backend/internal/algorithm"
	"glimmer-interview-hub/backend/internal/model"
	"glimmer-interview-hub/backend/internal/repository"
)

// AssignmentStats 分配统计
type AssignmentStats struct {
	TotalParticipants int `json:"totalParticipants"`
	First             int `json:"first"`
	Second            int `json:"second"`
	Third             int `json:"third"`
	Unassigned        int `json:"unassigned"`
	Unsubmitted       int `json:"unsubmitted"`
	SlotCount         int `json:"slotCount"`
}

// AssignmentItem 一条分配（含时段信息与成员资料）
type AssignmentItem struct {
	model.Assignment
	Name        string     `json:"name"`  // 真实姓名
	QQ          string     `json:"qq"`    // 联系 QQ
	Phone       string     `json:"phone"` // 手机号
	Slot        model.Slot `json:"slot"`
	SlotOccupied int       `json:"slotOccupied"` // 该时段已占用人数（容量 >1 时 >1）
}

// AssignmentView 分配预览
type AssignmentView struct {
	Version         model.AssignmentVersion `json:"version"`
	Session         model.Session           `json:"session"`
	Assignments     []AssignmentItem        `json:"assignments"`
	Stats           AssignmentStats         `json:"stats"`
	Unsubmitted     []model.User            `json:"unsubmitted"`
	UnassignedUsers []model.User            `json:"unassignedUsers"` // 提交了心仪但本版本未分到的人
	FreeSlots       []model.Slot            `json:"freeSlots"`
	Partitions      []PartitionView         `json:"partitions"`      // 本场次的日程（可跨多天）
	ConfirmedActive bool                    `json:"confirmedActive"` // 当前展示版本是否已确认
}

// AssignmentService 分配业务
type AssignmentService struct {
	repo *repository.Repo
}

func NewAssignmentService(repo *repository.Repo) *AssignmentService {
	return &AssignmentService{repo: repo}
}

// Generate 运行分配算法，生成新版本
func (s *AssignmentService) Generate(sessionID uint, admin string) (*model.AssignmentVersion, error) {
	sess, err := s.repo.GetSession(sessionID)
	if err != nil {
		return nil, NotFound("场次不存在")
	}
	if sess.Status == model.StatusDraft {
		return nil, BadRequest("请先发布场次再生成分配")
	}
	prefs, err := s.repo.AllPreferences(sessionID)
	if err != nil {
		return nil, err
	}
	slots, err := s.repo.GetSlots(sessionID)
	if err != nil {
		return nil, err
	}
	slotIDs := make([]uint, 0, len(slots))
	caps := make(map[uint]int, len(slots))
	for _, sl := range slots {
		slotIDs = append(slotIDs, sl.ID)
		caps[sl.ID] = slotCapacityOf(sl)
	}

	// 按成员聚合心仪时间（rank 顺序）
	order := make([]string, 0)
	byUser := make(map[string][]uint)
	studentOf := make(map[string]string)
	for _, p := range prefs {
		if _, ok := byUser[p.Username]; !ok {
			order = append(order, p.Username)
			studentOf[p.Username] = p.StudentID
		}
		byUser[p.Username] = append(byUser[p.Username], p.SlotID)
	}
	cands := make([]algorithm.Candidate, 0, len(order))
	for _, u := range order {
		ranks := byUser[u]
		slotRanked := make([]uint, 3)
		// prefs 已按 username, rank 排序，直接按序取
		for i := 0; i < 3 && i < len(ranks); i++ {
			slotRanked[i] = ranks[i]
		}
		cands = append(cands, algorithm.Candidate{Username: u, StudentID: studentOf[u], SlotIDs: slotRanked})
	}

	res := algorithm.AssignWithCapacity(cands, slotIDs, caps)

	assigns := make([]model.Assignment, 0, len(res.SlotOf))
	for _, u := range order {
		if slotID, ok := res.SlotOf[u]; ok {
			assigns = append(assigns, model.Assignment{
				SessionID:   sessionID,
				SlotID:      slotID,
				Username:    u,
				StudentID:   studentOf[u],
				MatchedRank: res.RankOf[u], // 留痕：分到的是本人第几心仪
				Source:      "auto",
			})
		}
	}

	stats := AssignmentStats{
		TotalParticipants: len(cands),
		First:             res.First,
		Second:            res.Second,
		Third:             res.Third,
		Unassigned:        res.Unassigned,
		SlotCount:         len(slots),
	}
	if users, err := s.repo.ListUsersByDirection(sess.Direction); err == nil {
		for _, u := range users {
			if len(byUser[u.Username]) < 3 {
				stats.Unsubmitted++
			}
		}
	}
	statsJSON, _ := json.Marshal(stats)

	ver := &model.AssignmentVersion{
		SessionID: sessionID,
		Confirmed: 0,
		Stats:     string(statsJSON),
		CreatedBy: admin,
	}
	if err := s.repo.CreateVersion(ver, assigns); err != nil {
		return nil, err
	}
	if sess.Status != model.StatusConfirmed {
		sess.Status = model.StatusAssigned
		if err := s.repo.UpdateSession(sess); err != nil {
			return nil, err
		}
	}
	return ver, nil
}

// Get 查看分配（默认最新版本，可指定 versionId）。
// 尚未生成分配时返回 (nil, nil)，由调用方按正常空状态处理。
func (s *AssignmentService) Get(sessionID uint, versionID uint) (*AssignmentView, error) {
	sess, err := s.repo.GetSession(sessionID)
	if err != nil {
		return nil, NotFound("场次不存在")
	}
	var ver *model.AssignmentVersion
	if versionID > 0 {
		ver, err = s.repo.GetVersion(versionID)
	} else {
		ver, err = s.repo.LatestVersion(sessionID)
	}
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	assigns, err := s.repo.GetAssignments(ver.ID)
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
	names := make([]string, 0, len(assigns))
	for _, a := range assigns {
		names = append(names, a.Username)
	}
	profiles, _ := s.repo.UsersByNames(names)
	items := make([]AssignmentItem, 0, len(assigns))
	occupied := make(map[uint]int, len(assigns))
	for _, a := range assigns {
		occupied[a.SlotID]++
	}
	for _, a := range assigns {
		u := profiles[a.Username]
		items = append(items, AssignmentItem{
			Assignment:   a,
			Name:         u.Name,
			QQ:           u.QQ,
			Phone:        u.Phone,
			Slot:         slotOf[a.SlotID],
			SlotOccupied: occupied[a.SlotID],
		})
	}
	// 空闲时段：已占用人数 < 时段容量
	freeSlots := make([]model.Slot, 0)
	for _, sl := range slots {
		if occupied[sl.ID] < slotCapacityOf(sl) {
			freeSlots = append(freeSlots, sl)
		}
	}

	var stats AssignmentStats
	if ver.Stats != "" {
		_ = json.Unmarshal([]byte(ver.Stats), &stats)
	}
	// 未提交名单：曾登录用户中心仪时间未满的
	prefs, err := s.repo.AllPreferences(sessionID)
	if err != nil {
		return nil, err
	}
	countByUser := make(map[string]int)
	for _, p := range prefs {
		countByUser[p.Username]++
	}
	users, err := s.repo.ListUsersByDirection(sess.Direction)
	if err != nil {
		return nil, err
	}
	unsubmitted := make([]model.User, 0)
	unassignedUsers := make([]model.User, 0)
	assignedSet := make(map[string]bool, len(assigns))
	for _, a := range assigns {
		assignedSet[a.Username] = true
	}
	for _, u := range users {
		if countByUser[u.Username] >= 3 {
			continue // 已提交满
		}
		unsubmitted = append(unsubmitted, u)
	}
	// 提交了心仪（含不满 3 个）但本版本没有分到的人：供管理员手动安排
	for _, u := range users {
		if countByUser[u.Username] > 0 && !assignedSet[u.Username] {
			unassignedUsers = append(unassignedUsers, u)
		}
	}
	stats.Unsubmitted = len(unsubmitted)

	return &AssignmentView{
		Version:         *ver,
		Session:         *sess,
		Assignments:     items,
		Stats:           stats,
		Unsubmitted:     unsubmitted,
		UnassignedUsers: unassignedUsers,
		FreeSlots:       freeSlots,
		Partitions:      BuildPartitionViews(slots),
		ConfirmedActive: ver.Confirmed == 1,
	}, nil
}

// AssignUser 把未分配的成员手动安排到空闲时段（可追溯：记录其心仪序）
func (s *AssignmentService) AssignUser(sessionID uint, username string, slotID uint, admin string) error {
	sess, err := s.repo.GetSession(sessionID)
	if err != nil {
		return NotFound("场次不存在")
	}
	if sess.Status == model.StatusDraft {
		return BadRequest("请先发布场次")
	}
	ver, err := s.repo.LatestVersion(sessionID)
	if err != nil {
		return BadRequest("尚未生成分配，请先点击「生成分配」")
	}
	user, err := s.repo.GetUser(username)
	if err != nil {
		return err
	}
	if user == nil {
		return NotFound("用户不存在")
	}
	if exist, err := s.repo.GetUserAssignmentInVersion(ver.ID, username); err == nil && exist.ID != 0 {
		return Conflict("该成员在本版本中已有分配")
	}
	slots, err := s.repo.GetSlots(sessionID)
	if err != nil {
		return err
	}
	capacity := 0
	for _, sl := range slots {
		if sl.ID == slotID {
			capacity = slotCapacityOf(sl)
			break
		}
	}
	if capacity == 0 {
		return BadRequest("目标时段不属于本场次")
	}
	used, err := s.repo.CountAssignmentsInSlot(ver.ID, slotID)
	if err != nil {
		return err
	}
	if used >= capacity {
		return Conflict(fmt.Sprintf("目标时段已满员（%d/%d）", used, capacity))
	}
	rank, _ := s.repo.PreferenceRank(sessionID, username, slotID)
	return s.repo.CreateAssignment(&model.Assignment{
		SessionID:   sessionID,
		VersionID:   ver.ID,
		SlotID:      slotID,
		Username:    username,
		StudentID:   user.StudentID,
		MatchedRank: rank,
		Source:      "manual",
	})
}

// Reassign 改派：把某成员调到同一场次的空闲时段
func (s *AssignmentService) Reassign(assignmentID uint, newSlotID uint) error {
	a, err := s.repo.GetAssignment(assignmentID)
	if err != nil {
		return NotFound("分配记录不存在")
	}
	if newSlotID == a.SlotID {
		return BadRequest("目标时段与当前时段相同")
	}
	// 校验目标时段属于同一场次，并取它的容量
	slots, err := s.repo.GetSlots(a.SessionID)
	if err != nil {
		return err
	}
	capacity := 0
	for _, sl := range slots {
		if sl.ID == newSlotID {
			capacity = slotCapacityOf(sl)
			break
		}
	}
	if capacity == 0 {
		return BadRequest("目标时段不属于本场次")
	}
	used, err := s.repo.CountAssignmentsInSlot(a.VersionID, newSlotID)
	if err != nil {
		return err
	}
	if used >= capacity {
		return Conflict(fmt.Sprintf("目标时段已满员（%d/%d），请使用对调功能", used, capacity))
	}
	a.SlotID = newSlotID
	a.Source = "manual"
	// 可追溯：改派后重算新时段是否仍在本人心仪内（在则为对应序，否则 0）
	a.MatchedRank, _ = s.repo.PreferenceRank(a.SessionID, a.Username, newSlotID)
	return s.repo.UpdateAssignment(a)
}

// Swap 对调两名成员的时段
func (s *AssignmentService) Swap(aID, bID uint) error {
	if aID == bID {
		return BadRequest("不能与自己对调")
	}
	a, err := s.repo.GetAssignment(aID)
	if err != nil {
		return NotFound("分配记录不存在")
	}
	b, err := s.repo.GetAssignment(bID)
	if err != nil {
		return NotFound("分配记录不存在")
	}
	if a.VersionID != b.VersionID {
		return BadRequest("两条分配记录不属于同一版本")
	}
	aSlot, bSlot := a.SlotID, b.SlotID
	// 同一版本内 (slot_id) 唯一，用哨兵值 0 中转，避免更新瞬间冲突
	err = s.repo.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(a).Update("slot_id", 0).Error; err != nil {
			return err
		}
		if err := tx.Model(b).Update("slot_id", aSlot).Error; err != nil {
			return err
		}
		if err := tx.Model(a).Update("slot_id", bSlot).Error; err != nil {
			return err
		}
		return tx.Model(&model.Assignment{}).Where("id IN ?", []uint{aID, bID}).
			Update("source", "manual").Error
	})
	if err != nil {
		return err
	}
	// 可追溯：对调后重算两人新时段对应的心仪序（内存对象同步交换后的时段）
	a.SlotID, b.SlotID = bSlot, aSlot
	for _, x := range []*model.Assignment{a, b} {
		rank, _ := s.repo.PreferenceRank(x.SessionID, x.Username, x.SlotID)
		if err := s.repo.DB.Model(&model.Assignment{}).Where("id = ?", x.ID).
			Updates(map[string]interface{}{"matched_rank": rank, "source": "manual"}).Error; err != nil {
			return err
		}
	}
	return nil
}

// Confirm 确认分配（最新版本生效，成员可见）
func (s *AssignmentService) Confirm(sessionID uint) (*model.AssignmentVersion, error) {
	sess, err := s.repo.GetSession(sessionID)
	if err != nil {
		return nil, NotFound("场次不存在")
	}
	ver, err := s.repo.LatestVersion(sessionID)
	if err != nil {
		return nil, BadRequest("尚未生成分配，请先点击「生成分配」")
	}
	if err := s.repo.ConfirmVersion(ver); err != nil {
		return nil, err
	}
	sess.Status = model.StatusConfirmed
	if err := s.repo.UpdateSession(sess); err != nil {
		return nil, err
	}
	return ver, nil
}
