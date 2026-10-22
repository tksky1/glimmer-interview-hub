package model

import "time"

// Admin 管理员名单（与旧系统解耦，启动时由 ADMIN_USERNAMES 种子化）
type Admin struct {
	Username  string    `gorm:"primaryKey;size:50" json:"username"`
	CreatedAt time.Time `json:"createdAt"`
}

// User 曾登录过本平台的用户（方向经邀请码绑定）
type User struct {
	Username   string    `gorm:"primaryKey;size:50" json:"username"`
	StudentID  string    `gorm:"size:30;index" json:"studentid"`
	Direction  string    `gorm:"size:20;index" json:"direction"` // 首个绑定方向（兼容旧逻辑），完整方向见 UserDirection
	Name       string    `gorm:"size:50" json:"name"`            // 真实姓名（成员自行填写）
	QQ         string    `gorm:"size:30" json:"qq"`              // 联系 QQ（成员自行填写）
	Phone      string    `gorm:"size:20" json:"phone"`           // 手机号（成员自行填写）
	LastSeenAt time.Time `json:"lastSeenAt"`
}

// UserDirection 用户已获邀请的方向（同一用户可绑定多个方向）
type UserDirection struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Username  string    `gorm:"size:50;not null;index" json:"username"`
	Direction string    `gorm:"size:20;not null;index" json:"direction"`
	CreatedAt time.Time `json:"createdAt"`
}

// Session 面试场次（管理员在日历上框选的一段日程）
type Session struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Title       string    `gorm:"size:100;not null" json:"title"`
	Direction   string    `gorm:"size:20;not null;default:'';index" json:"direction"` // c/java/web/ml
	Date        string    `gorm:"size:10;not null;index" json:"date"`                 // YYYY-MM-DD
	StartTime   string    `gorm:"size:5;not null" json:"startTime"`                   // HH:MM
	EndTime     string    `gorm:"size:5;not null" json:"endTime"`                     // HH:MM
	SlotMinutes   int       `gorm:"not null" json:"slotMinutes"`                        // 每场面试时长（分钟，5 的倍数）
	SlotCapacity  int       `gorm:"not null;default:1" json:"slotCapacity"`             // 每个时段可同时面试的人数（场次级默认，可被单个日程覆盖）
	SlotCapacitys string    `gorm:"type:text" json:"-"`                                // 逐时段容量覆盖 JSON：{"日期 开始时间": 人数}
	Location      string    `gorm:"size:255;not null" json:"location"`                  // 面试地点
	Contact     string    `gorm:"size:50" json:"contact"`                             // 联系人
	ContactQQ   string    `gorm:"size:30" json:"contactQQ"`                           // 联系人 QQ
	Status      string    `gorm:"size:20;not null;default:draft;index" json:"status"` // draft/published/assigned/confirmed
	CreatedBy   string    `gorm:"size:50" json:"createdBy"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// Invite 方向邀请码（成员通过邀请码/专属链接绑定方向）
type Invite struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	Code      string     `gorm:"size:32;not null;uniqueIndex" json:"code"`
	Direction string     `gorm:"size:20;not null;index" json:"direction"`
	Note      string     `gorm:"size:100" json:"note"`
	CreatedBy string     `gorm:"size:50" json:"createdBy"`
	CreatedAt time.Time  `json:"createdAt"`
	UsedBy    string     `gorm:"size:50;index" json:"usedBy"` // 最近一位核销者
	UsedAt    *time.Time `json:"usedAt"`
	UsedCount int        `gorm:"not null;default:0" json:"usedCount"` // 累计核销人数（不限次数）
}

// InviteRedemption 邀请码核销记录（同一人对同一码只计一次）
type InviteRedemption struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Code      string    `gorm:"size:32;not null;index" json:"code"`
	Username  string    `gorm:"size:50;not null;index" json:"username"`
	CreatedAt time.Time `json:"createdAt"`
}

// 方向常量与展示名
const (
	DirC    = "c"
	DirJava = "java"
	DirWeb  = "web"
	DirML   = "ml"
)

// DirectionNames 方向展示名
var DirectionNames = map[string]string{
	DirC:    "C / 计系",
	DirJava: "Java 后端",
	DirWeb:  "Web 前端",
	DirML:   "ML 算法",
}

// ValidDirection 是否为合法方向
func ValidDirection(d string) bool {
	_, ok := DirectionNames[d]
	return ok
}

// Slot 由场次切分出的具体面试时段，容量由管理员配置（默认 1 人同时面试）
// 一个场次可由多个「日程」（不同日期 / 同一天多段）组成，时段通过 Date + PartitionIdx 归属到某个日程
type Slot struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	SessionID    uint   `gorm:"not null;index" json:"sessionId"`
	Idx          int    `gorm:"not null" json:"idx"`
	Date         string `gorm:"size:10;not null;default:'';index" json:"date"` // YYYY-MM-DD，所属日程
	PartitionIdx int    `gorm:"not null;default:0" json:"partitionIdx"`        // 第几个日程
	StartTime    string `gorm:"size:5;not null" json:"startTime"`              // HH:MM
	EndTime      string `gorm:"size:5;not null" json:"endTime"`                // HH:MM
	Capacity     int    `gorm:"not null;default:1" json:"capacity"`            // 该时段可同时面试的人数
	Location     string `gorm:"size:255" json:"location"`
}

// Preference 成员的心仪时间偏好（rank: 1/2/3）
type Preference struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	SessionID uint      `gorm:"not null;index" json:"sessionId"`
	Username  string    `gorm:"size:50;not null;index" json:"username"`
	StudentID string    `gorm:"size:30" json:"studentid"`
	SlotID    uint      `gorm:"not null" json:"slotId"`
	Rank      int       `gorm:"not null" json:"rank"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// AssignmentVersion 一次自动/人工分配产生的版本
type AssignmentVersion struct {
	ID          uint         `gorm:"primaryKey" json:"id"`
	SessionID   uint         `gorm:"not null;index" json:"sessionId"`
	Confirmed   int          `gorm:"not null;default:0" json:"confirmed"` // 0 未确认 / 1 已确认
	Stats       string       `gorm:"type:text" json:"stats"`              // JSON 统计
	CreatedBy   string       `gorm:"size:50" json:"createdBy"`
	CreatedAt   time.Time    `json:"createdAt"`
	Assignments []Assignment `gorm:"foreignKey:VersionID" json:"assignments,omitempty"`
}

// Assignment 版本内的一条分配记录（同一版本内 username 唯一；slot 可重复——一个时段可容多人）
type Assignment struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	VersionID   uint      `gorm:"not null;index" json:"versionId"`
	SessionID   uint      `gorm:"not null;index" json:"sessionId"`
	SlotID      uint      `gorm:"not null" json:"slotId"`
	Username    string    `gorm:"size:50;not null;index" json:"username"`
	StudentID   string    `gorm:"size:30" json:"studentid"`
	MatchedRank int       `gorm:"not null;default:0" json:"matchedRank"`       // 分到的是本人第几心仪（1-3，0=不在心仪内/未匹配）
	Source      string    `gorm:"size:10;not null;default:auto" json:"source"` // auto/manual
	UpdatedAt   time.Time `json:"updatedAt"`
}

// 场次状态常量
const (
	StatusDraft     = "draft"
	StatusPublished = "published"
	StatusAssigned  = "assigned"
	StatusConfirmed = "confirmed"
)

// UniqueIndexes 复合唯一约束（GORM 标签对多字段唯一索引支持不稳定，迁移后显式创建）
var UniqueIndexes = []string{
	"CREATE UNIQUE INDEX IF NOT EXISTS uniq_slot_session_idx ON slots(session_id, idx)",
	"CREATE UNIQUE INDEX IF NOT EXISTS uniq_pref_sur ON preferences(session_id, username, rank)",
	"CREATE UNIQUE INDEX IF NOT EXISTS uniq_pref_sus ON preferences(session_id, username, slot_id)",
	// 一个时段可容纳多人同时面试：assignment 的 slot 唯一约束改为普通索引
	"DROP INDEX IF EXISTS uniq_assign_vs",
	"CREATE INDEX IF NOT EXISTS idx_assign_vs ON assignments(version_id, slot_id)",
	"CREATE UNIQUE INDEX IF NOT EXISTS uniq_assign_vu ON assignments(version_id, username)",
	"CREATE UNIQUE INDEX IF NOT EXISTS uniq_user_dir ON user_directions(username, direction)",
	"CREATE UNIQUE INDEX IF NOT EXISTS uniq_invite_redemption ON invite_redemptions(code, username)",
	// 历史邀请码已使用过的，回填使用人数
	"UPDATE invites SET used_count = 1 WHERE used_by <> '' AND used_count = 0",
	// 历史用户（users.direction 单方向）回填到多方向表，保证老数据可见
	"INSERT OR IGNORE INTO user_directions (username, direction, created_at) SELECT username, direction, CURRENT_TIMESTAMP FROM users WHERE direction <> ''",
	// 历史单日场次：把场次日期回填到时段上（多日程改造后时段自带日期）
	"UPDATE slots SET date = (SELECT date FROM sessions WHERE sessions.id = slots.session_id) WHERE date IS NULL OR date = ''",
	"CREATE INDEX IF NOT EXISTS idx_slot_date ON slots(date)",
	// 容量改造：历史场次/时段默认 1 人同时面试
	"UPDATE sessions SET slot_capacity = 1 WHERE slot_capacity IS NULL OR slot_capacity < 1",
	"UPDATE slots SET capacity = 1 WHERE capacity IS NULL OR capacity < 1",
}
