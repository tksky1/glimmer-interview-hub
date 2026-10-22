package repository

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"glimmer-interview-hub/backend/internal/model"
)

// Repo 数据访问层
type Repo struct {
	DB *gorm.DB
}

func New(db *gorm.DB) *Repo {
	return &Repo{DB: db}
}

// ---------- 管理员 ----------

func (r *Repo) IsAdmin(username string) bool {
	var count int64
	r.DB.Model(&model.Admin{}).Where("username = ?", username).Count(&count)
	return count > 0
}

func (r *Repo) ListAdmins() ([]model.Admin, error) {
	var list []model.Admin
	err := r.DB.Order("created_at asc").Find(&list).Error
	return list, err
}

func (r *Repo) AddAdmin(username string) error {
	return r.DB.Create(&model.Admin{Username: username, CreatedAt: time.Now()}).Error
}

func (r *Repo) DeleteAdmin(username string) error {
	return r.DB.Where("username = ?", username).Delete(&model.Admin{}).Error
}

func (r *Repo) CountAdmins() (int64, error) {
	var n int64
	err := r.DB.Model(&model.Admin{}).Count(&n).Error
	return n, err
}

// ---------- 用户 ----------

func (r *Repo) UpsertUser(username, studentid string) error {
	now := time.Now()
	return r.DB.Exec(
		`INSERT INTO users (username, student_id, last_seen_at) VALUES (?, ?, ?)
		 ON CONFLICT(username) DO UPDATE SET student_id = excluded.student_id, last_seen_at = excluded.last_seen_at`,
		username, studentid, now,
	).Error
}

// GetUser 查询用户（不存在返回 nil, nil）
func (r *Repo) GetUser(username string) (*model.User, error) {
	var u model.User
	err := r.DB.Where("username = ?", username).First(&u).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

// SetUserDirection 绑定用户方向
func (r *Repo) SetUserDirection(username, direction string) error {
	return r.DB.Model(&model.User{}).Where("username = ?", username).
		Update("direction", direction).Error
}

// AddUserDirection 追加一个已获邀请的方向（幂等）
func (r *Repo) AddUserDirection(username, direction string) error {
	if username == "" || direction == "" {
		return nil
	}
	return r.DB.Exec(
		`INSERT INTO user_directions (username, direction, created_at) VALUES (?, ?, ?)
		 ON CONFLICT(username, direction) DO NOTHING`,
		username, direction, time.Now(),
	).Error
}

// AddInviteRedemption 记录邀请码核销；同一人对同一码只记一次，返回是否首次
func (r *Repo) AddInviteRedemption(code, username string) (bool, error) {
	res := r.DB.Exec(
		`INSERT OR IGNORE INTO invite_redemptions (code, username, created_at) VALUES (?, ?, ?)`,
		code, username, time.Now(),
	)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// ListUserDirections 查询用户全部已绑定方向（按绑定顺序）
func (r *Repo) ListUserDirections(username string) ([]string, error) {
	var list []string
	err := r.DB.Model(&model.UserDirection{}).
		Where("username = ?", username).
		Order("id asc").
		Pluck("direction", &list).Error
	return list, err
}

// UpdateUserProfile 保存成员真实姓名、QQ 与手机号
func (r *Repo) UpdateUserProfile(username, name, qq, phone string) error {
	return r.DB.Model(&model.User{}).Where("username = ?", username).
		Updates(map[string]interface{}{"name": name, "qq": qq, "phone": phone}).Error
}

// UsersByNames 批量查询用户资料（username -> user）
func (r *Repo) UsersByNames(usernames []string) (map[string]model.User, error) {
	out := make(map[string]model.User, len(usernames))
	if len(usernames) == 0 {
		return out, nil
	}
	var list []model.User
	if err := r.DB.Where("username IN ?", usernames).Find(&list).Error; err != nil {
		return nil, err
	}
	for _, u := range list {
		out[u.Username] = u
	}
	return out, nil
}

// ListUsersByDirection 查询绑定了某方向的全部用户
func (r *Repo) ListUsersByDirection(direction string) ([]model.User, error) {
	var list []model.User
	err := r.DB.
		Joins("JOIN user_directions ud ON ud.username = users.username").
		Where("ud.direction = ?", direction).
		Order("users.last_seen_at desc").
		Find(&list).Error
	return list, err
}

// CountUserPreferences 用户已提交的偏好条数（跨全部场次）
func (r *Repo) CountUserPreferences(username string) (int64, error) {
	var n int64
	err := r.DB.Model(&model.Preference{}).Where("username = ?", username).Count(&n).Error
	return n, err
}

func (r *Repo) ListUsers() ([]model.User, error) {
	var list []model.User
	err := r.DB.Order("last_seen_at desc").Find(&list).Error
	return list, err
}

// ---------- 邀请码 ----------

func (r *Repo) CreateInvite(inv *model.Invite) error {
	return r.DB.Create(inv).Error
}

func (r *Repo) GetInviteByCode(code string) (*model.Invite, error) {
	var inv model.Invite
	if err := r.DB.Where("code = ?", code).First(&inv).Error; err != nil {
		return nil, err
	}
	return &inv, nil
}

func (r *Repo) ListInvites() ([]model.Invite, error) {
	var list []model.Invite
	err := r.DB.Order("id desc").Find(&list).Error
	return list, err
}

func (r *Repo) UpdateInvite(inv *model.Invite) error {
	return r.DB.Save(inv).Error
}

func (r *Repo) DeleteInvite(id uint) error {
	return r.DB.Delete(&model.Invite{}, id).Error
}

// ---------- 场次 / 时段 ----------

func (r *Repo) CreateSessionWithSlots(s *model.Session, slots []model.Slot) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(s).Error; err != nil {
			return err
		}
		for i := range slots {
			slots[i].SessionID = s.ID
		}
		return tx.Create(&slots).Error
	})
}

// ReplaceSlots 草稿态重新生成时段
func (r *Repo) ReplaceSlots(sessionID uint, slots []model.Slot) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("session_id = ?", sessionID).Delete(&model.Slot{}).Error; err != nil {
			return err
		}
		for i := range slots {
			slots[i].SessionID = sessionID
		}
		return tx.Create(&slots).Error
	})
}

func (r *Repo) ListSessions() ([]model.Session, error) {
	var list []model.Session
	err := r.DB.Order("date desc, id desc").Find(&list).Error
	return list, err
}

// ListSessionsInRange 查询在 [start, end] 区间内有时段、状态为 published/assigned/confirmed 的场次
// 场次可跨多天，只要任一时段落在该区间内即命中（返回该场次全部时段的归属场次）
func (r *Repo) ListSessionsInRange(start, end string) ([]model.Session, error) {
	var list []model.Session
	err := r.DB.Where("status <> ? AND id IN (SELECT DISTINCT session_id FROM slots WHERE date >= ? AND date <= ?)",
		model.StatusDraft, start, end).
		Order("date asc, id asc").Find(&list).Error
	return list, err
}

// SlotsBySessions 批量取多个场次的时段（按日期、序号排序）
func (r *Repo) SlotsBySessions(sessionIDs []uint) (map[uint][]model.Slot, error) {
	out := make(map[uint][]model.Slot, len(sessionIDs))
	if len(sessionIDs) == 0 {
		return out, nil
	}
	var list []model.Slot
	if err := r.DB.Where("session_id IN ?", sessionIDs).
		Order("session_id asc, date asc, idx asc").Find(&list).Error; err != nil {
		return nil, err
	}
	for _, s := range list {
		out[s.SessionID] = append(out[s.SessionID], s)
	}
	return out, nil
}

func (r *Repo) GetSession(id uint) (*model.Session, error) {
	var s model.Session
	if err := r.DB.First(&s, id).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *Repo) GetSlots(sessionID uint) ([]model.Slot, error) {
	var list []model.Slot
	err := r.DB.Where("session_id = ?", sessionID).Order("date asc, idx asc").Find(&list).Error
	return list, err
}

func (r *Repo) CountSlots(sessionID uint) (int64, error) {
	var n int64
	err := r.DB.Model(&model.Slot{}).Where("session_id = ?", sessionID).Count(&n).Error
	return n, err
}

func (r *Repo) UpdateSession(s *model.Session) error {
	return r.DB.Save(s).Error
}

// DeleteSession 删除场次，并级联清理其时段、心仪时间与分配结果（单事务）
func (r *Repo) DeleteSession(id uint) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("session_id = ?", id).Delete(&model.Assignment{}).Error; err != nil {
			return err
		}
		if err := tx.Where("session_id = ?", id).Delete(&model.AssignmentVersion{}).Error; err != nil {
			return err
		}
		if err := tx.Where("session_id = ?", id).Delete(&model.Preference{}).Error; err != nil {
			return err
		}
		if err := tx.Where("session_id = ?", id).Delete(&model.Slot{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.Session{}, id).Error
	})
}

// FirstChoiceCounts 统计各时段被首选（rank=1）的人数
func (r *Repo) FirstChoiceCounts(sessionID uint) (map[uint]int, error) {
	type row struct {
		SlotID uint
		Cnt    int
	}
	var rows []row
	err := r.DB.Model(&model.Preference{}).
		Select("slot_id, count(*) as cnt").
		Where("session_id = ? and rank = 1", sessionID).
		Group("slot_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[uint]int, len(rows))
	for _, x := range rows {
		out[x.SlotID] = x.Cnt
	}
	return out, nil
}

// CountParticipants 统计已提交至少一条偏好的成员数
func (r *Repo) CountParticipants(sessionID uint) (int64, error) {
	var n int64
	err := r.DB.Model(&model.Preference{}).
		Where("session_id = ?", sessionID).
		Distinct("username").Count(&n).Error
	return n, err
}

// ---------- 偏好 ----------

func (r *Repo) MyPreferences(sessionID uint, username string) ([]model.Preference, error) {
	var list []model.Preference
	err := r.DB.Where("session_id = ? and username = ?", sessionID, username).
		Order("rank asc").Find(&list).Error
	return list, err
}

func (r *Repo) AllPreferences(sessionID uint) ([]model.Preference, error) {
	var list []model.Preference
	err := r.DB.Where("session_id = ?", sessionID).Order("username asc, rank asc").Find(&list).Error
	return list, err
}

// ---------- 方向互斥：同一方向只能预约一个场次 ----------

// ConfirmedSessionsWithPreferences 用户在某方向已确认且有心仪的其它场次
func (r *Repo) ConfirmedSessionsWithPreferences(username, direction string, exclude uint) ([]uint, error) {
	var ids []uint
	err := r.DB.Model(&model.Preference{}).
		Select("sessions.id").
		Joins("JOIN sessions ON sessions.id = preferences.session_id").
		Where("preferences.username = ? AND sessions.direction = ? AND sessions.status = ? AND sessions.id <> ?",
			username, direction, model.StatusConfirmed, exclude).
		Scan(&ids).Error
	return ids, err
}

// ClearPreferencesInDirection 清空用户在某方向其它场次的心仪时间（含未确认版本里的分配），
// 返回被清空的场次 ID。已确认场次不在此列（调用方会先拦截）。
func (r *Repo) ClearPreferencesInDirection(username, direction string, keep uint) ([]uint, error) {
	var ids []uint
	err := r.DB.Model(&model.Preference{}).
		Select("DISTINCT sessions.id").
		Joins("JOIN sessions ON sessions.id = preferences.session_id").
		Where("preferences.username = ? AND sessions.direction = ? AND sessions.id <> ?", username, direction, keep).
		Scan(&ids).Error
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return ids, nil
	}
	return ids, r.DB.Transaction(func(tx *gorm.DB) error {
		// 未确认版本里的分配记录一并撤下，避免管理员看到“没有心仪时间却占着时段”
		if err := tx.Exec(
			`DELETE FROM assignments WHERE username = ? AND session_id IN ?
			 AND version_id IN (SELECT id FROM assignment_versions WHERE confirmed = 0)`,
			username, ids).Error; err != nil {
			return err
		}
		return tx.Where("username = ? AND session_id IN ?", username, ids).
			Delete(&model.Preference{}).Error
	})
}

// PreferenceRank 查询某人心仪列表中某时段的序号（1-3；不在其心仪内返回 0）
func (r *Repo) PreferenceRank(sessionID uint, username string, slotID uint) (int, error) {
	var p model.Preference
	err := r.DB.Where("session_id = ? and username = ? and slot_id = ?", sessionID, username, slotID).
		First(&p).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, err
	}
	return p.Rank, nil
}

// ReplacePreferences 整体替换某成员在某场次下的心仪时间
func (r *Repo) ReplacePreferences(sessionID uint, username, studentid string, slotIDs []uint) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("session_id = ? and username = ?", sessionID, username).
			Delete(&model.Preference{}).Error; err != nil {
			return err
		}
		now := time.Now()
		for i, sid := range slotIDs {
			p := model.Preference{
				SessionID: sessionID,
				Username:  username,
				StudentID: studentid,
				SlotID:    sid,
				Rank:      i + 1,
				UpdatedAt: now,
			}
			if err := tx.Create(&p).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// MyAllPreferences 某成员全部偏好（跨场次）
func (r *Repo) MyAllPreferences(username string) ([]model.Preference, error) {
	var list []model.Preference
	err := r.DB.Where("username = ?", username).Order("session_id asc, rank asc").Find(&list).Error
	return list, err
}

// ---------- 分配 ----------

func (r *Repo) CreateVersion(v *model.AssignmentVersion, assigns []model.Assignment) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(v).Error; err != nil {
			return err
		}
		for i := range assigns {
			assigns[i].VersionID = v.ID
		}
		if len(assigns) > 0 {
			return tx.Create(&assigns).Error
		}
		return nil
	})
}

func (r *Repo) LatestVersion(sessionID uint) (*model.AssignmentVersion, error) {
	var v model.AssignmentVersion
	err := r.DB.Where("session_id = ?", sessionID).Order("id desc").First(&v).Error
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *Repo) GetVersion(id uint) (*model.AssignmentVersion, error) {
	var v model.AssignmentVersion
	if err := r.DB.First(&v, id).Error; err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *Repo) ListVersions(sessionID uint) ([]model.AssignmentVersion, error) {
	var list []model.AssignmentVersion
	err := r.DB.Where("session_id = ?", sessionID).Order("id desc").Find(&list).Error
	return list, err
}

func (r *Repo) GetAssignments(versionID uint) ([]model.Assignment, error) {
	var list []model.Assignment
	err := r.DB.Where("version_id = ?", versionID).Order("id asc").Find(&list).Error
	return list, err
}

func (r *Repo) GetAssignment(id uint) (*model.Assignment, error) {
	var a model.Assignment
	if err := r.DB.First(&a, id).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

// GetAssignmentInSlot 某版本内占据某时段的分配记录
func (r *Repo) GetAssignmentInSlot(versionID uint, slotID uint) (*model.Assignment, error) {
	var a model.Assignment
	err := r.DB.Where("version_id = ? and slot_id = ?", versionID, slotID).First(&a).Error
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// GetUserAssignmentInVersion 某版本内某成员的分配记录
func (r *Repo) GetUserAssignmentInVersion(versionID uint, username string) (*model.Assignment, error) {
	var a model.Assignment
	err := r.DB.Where("version_id = ? and username = ?", versionID, username).First(&a).Error
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// AssignmentCountsInLatestVersion 场次最新分配版本内，各时段已占用的人数（一个时段可容多人）
func (r *Repo) AssignmentCountsInLatestVersion(sessionID uint) (map[uint]int, error) {
	type row struct {
		SlotID uint
		Cnt    int
	}
	var rows []row
	err := r.DB.Model(&model.Assignment{}).
		Select("assignments.slot_id as slot_id, count(*) as cnt").
		Joins("JOIN assignment_versions v ON v.id = assignments.version_id").
		Where("assignments.session_id = ? AND v.id = (SELECT MAX(id) FROM assignment_versions WHERE session_id = ?)",
			sessionID, sessionID).
		Group("assignments.slot_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[uint]int, len(rows))
	for _, x := range rows {
		out[x.SlotID] = x.Cnt
	}
	return out, nil
}

// CountAssignmentsInSlot 某版本内占据某时段的人数
func (r *Repo) CountAssignmentsInSlot(versionID uint, slotID uint) (int, error) {
	var n int64
	err := r.DB.Model(&model.Assignment{}).
		Where("version_id = ? and slot_id = ?", versionID, slotID).Count(&n).Error
	return int(n), err
}

func (r *Repo) UpdateAssignment(a *model.Assignment) error {
	return r.DB.Save(a).Error
}

func (r *Repo) DeleteAssignment(id uint) error {
	return r.DB.Delete(&model.Assignment{}, id).Error
}

func (r *Repo) CreateAssignment(a *model.Assignment) error {
	return r.DB.Create(a).Error
}

// ConfirmVersion 确认某版本（同场次其他版本取消确认）
func (r *Repo) ConfirmVersion(version *model.AssignmentVersion) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.AssignmentVersion{}).
			Where("session_id = ? and id <> ?", version.SessionID, version.ID).
			Update("confirmed", 0).Error; err != nil {
			return err
		}
		version.Confirmed = 1
		return tx.Save(version).Error
	})
}

// MyConfirmedAssignment 某成员在已确认版本中的分配
func (r *Repo) MyConfirmedAssignment(sessionID uint, username string) (*model.Assignment, error) {
	var a model.Assignment
	err := r.DB.Joins("JOIN assignment_versions v ON v.id = assignments.version_id").
		Where("v.session_id = ? and v.confirmed = 1 and assignments.username = ?", sessionID, username).
		First(&a).Error
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// MyAllConfirmedAssignments 某成员全部已确认分配（跨场次）
func (r *Repo) MyAllConfirmedAssignments(username string) ([]model.Assignment, error) {
	var list []model.Assignment
	err := r.DB.Joins("JOIN assignment_versions v ON v.id = assignments.version_id").
		Where("v.confirmed = 1 and assignments.username = ?", username).
		Find(&list).Error
	return list, err
}
