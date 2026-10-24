package service

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"time"

	"glimmer-interview-hub/backend/internal/model"
	"glimmer-interview-hub/backend/internal/repository"
)

// PartitionInput 一个日程：某天的一段可预约时间（一个场次可含多个日程）
// Capacity 为该日程覆盖的「每个时段可同时面试人数」，为 0 时沿用场次级设置
type PartitionInput struct {
	Date      string
	StartTime string
	EndTime   string
	Location  string
	Capacity  int
}

// SlotCapacityInput 单个时段的容量覆盖（按 日期 + 开始时间 定位，重新切分时段后依然对得上）
type SlotCapacityInput struct {
	Date      string
	StartTime string
	Capacity  int
}

// CreateSessionInput 新建/编辑场次入参
// Partitions 为空时，回退为单日程（Date/StartTime/EndTime/Location）
type CreateSessionInput struct {
	Title          string
	Direction      string
	Date           string
	StartTime      string
	EndTime        string
	SlotMinutes    int
	SlotCapacity   int // 场次级默认容量：每个时段可同时面试的人数
	Location       string
	Contact        string
	ContactQQ      string
	Partitions     []PartitionInput
	SlotCapacities []SlotCapacityInput // 逐时段容量覆盖（优先级最高）
}

// PartitionView 日程视图（由时段反推，便于前端按天展示）
type PartitionView struct {
	Date      string `json:"date"`
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
	Location  string `json:"location"`
	Capacity  int    `json:"capacity"` // 该日程每个时段可同时面试的人数
	SlotCount int    `json:"slotCount"`
}

// BuildPartitionViews 按时段反推日程列表（按日期升序）
// Capacity 取当天第一个时段的容量作为代表值（逐时段覆盖后各时段可能不同）
func BuildPartitionViews(slots []model.Slot) []PartitionView {
	out := make([]PartitionView, 0)
	idx := map[string]int{}
	for _, s := range slots {
		if i, ok := idx[s.Date]; ok {
			p := &out[i]
			p.SlotCount++
			if s.StartTime < p.StartTime {
				p.StartTime = s.StartTime
			}
			if s.EndTime > p.EndTime {
				p.EndTime = s.EndTime
			}
			continue
		}
		idx[s.Date] = len(out)
		out = append(out, PartitionView{
			Date:      s.Date,
			StartTime: s.StartTime,
			EndTime:   s.EndTime,
			Location:  s.Location,
			Capacity:  slotCapacityOf(s),
			SlotCount: 1,
		})
	}
	return out
}

// slotCapacityOf 时段容量兜底（老数据可能是 0）
func slotCapacityOf(s model.Slot) int {
	if s.Capacity > 0 {
		return s.Capacity
	}
	return 1
}

// EncodeSlotCapacities 逐时段容量覆盖 -> JSON（只保存 >0 的显式覆盖）
func EncodeSlotCapacities(in []SlotCapacityInput) string {
	m := map[string]int{}
	for _, sc := range in {
		d := strings.TrimSpace(sc.Date)
		t := strings.TrimSpace(sc.StartTime)
		if d == "" || t == "" || sc.Capacity <= 0 {
			continue
		}
		if sc.Capacity > 20 {
			sc.Capacity = 20
		}
		m[d+" "+t] = sc.Capacity
	}
	if len(m) == 0 {
		return ""
	}
	b, err := json.Marshal(m)
	if err != nil {
		return ""
	}
	return string(b)
}

// DecodeSlotCapacities JSON -> 逐时段容量覆盖
func DecodeSlotCapacities(raw string) map[string]int {
	out := map[string]int{}
	if strings.TrimSpace(raw) == "" {
		return out
	}
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}

// normalizePartitions 整理并校验日程列表：排序、查同一天时段重叠、生成时段
// 容量优先级：单个日程填写的 Capacity > 场次级 SlotCapacity > 1
func (in CreateSessionInput) normalizePartitions() ([]PartitionInput, []model.Slot, error) {
	ps := make([]PartitionInput, 0, len(in.Partitions)+1)
	if len(in.Partitions) > 0 {
		for _, p := range in.Partitions {
			p.Date = strings.TrimSpace(p.Date)
			p.StartTime = strings.TrimSpace(p.StartTime)
			p.EndTime = strings.TrimSpace(p.EndTime)
			p.Location = strings.TrimSpace(p.Location)
			if p.Date == "" || p.StartTime == "" || p.EndTime == "" {
				continue
			}
			ps = append(ps, p)
		}
	} else {
		ps = append(ps, PartitionInput{
			Date:      in.Date,
			StartTime: in.StartTime,
			EndTime:   in.EndTime,
			Location:  in.Location,
		})
	}
	if len(ps) == 0 {
		return nil, nil, BadRequest("请至少添加一个日程（日期 + 时间段）")
	}
	defaultCap := in.SlotCapacity
	if defaultCap < 1 {
		defaultCap = 1
	}
	if defaultCap > 20 {
		return nil, nil, BadRequest("每个时段最多同时面试 20 人")
	}
	for i := range ps {
		if ps[i].Capacity < 0 || ps[i].Capacity > 20 {
			return nil, nil, BadRequest("每个时段最多同时面试 20 人")
		}
		if ps[i].Capacity == 0 {
			ps[i].Capacity = defaultCap
		}
	}
	sort.Slice(ps, func(i, j int) bool {
		if ps[i].Date != ps[j].Date {
			return ps[i].Date < ps[j].Date
		}
		return ps[i].StartTime < ps[j].StartTime
	})
	slots := make([]model.Slot, 0)
	idx := 0
	// 逐时段容量覆盖：key = 日期 + 开始时间
	overCap := make(map[string]int, len(in.SlotCapacities))
	for _, sc := range in.SlotCapacities {
		d := strings.TrimSpace(sc.Date)
		t := strings.TrimSpace(sc.StartTime)
		if d == "" || t == "" || sc.Capacity <= 0 {
			continue
		}
		if sc.Capacity > 20 {
			return nil, nil, BadRequest("每个时段最多同时面试 20 人")
		}
		overCap[d+" "+t] = sc.Capacity
	}
	for pi, p := range ps {
		if err := validateDate(p.Date); err != nil {
			return nil, nil, err
		}
		loc := strings.TrimSpace(p.Location)
		if loc == "" {
			loc = strings.TrimSpace(in.Location) // 日程未单独填地点时沿用场次地点
		}
		daySlots, err := GenerateSlots(p.StartTime, p.EndTime, in.SlotMinutes, loc, p.Capacity)
		if err != nil {
			return nil, nil, err
		}
		// 逐时段覆盖优先级最高（此时 slot.Date 尚未回填，用日程日期做 key）
		for i := range daySlots {
			if c, ok := overCap[p.Date+" "+daySlots[i].StartTime]; ok {
				daySlots[i].Capacity = c
			}
		}
		for _, s := range daySlots {
			s.Date = p.Date
			s.PartitionIdx = pi
			s.Idx = idx
			idx++
			slots = append(slots, s)
		}
		// 同一天的多个日程不能重叠
		for _, q := range ps[pi+1:] {
			if q.Date != p.Date {
				break
			}
			if q.StartTime < p.EndTime {
				return nil, nil, BadRequest(p.Date + " 的日程时间段存在重叠，请调整")
			}
		}
	}
	if len(slots) == 0 {
		return nil, nil, BadRequest("所选时间段不足以容纳一场面试，请调整时长")
	}
	return ps, slots, nil
}

// sessionTimeRange 汇总日程的整体起止（写回 Session 用于列表排序与兼容展示）
func sessionTimeRange(ps []PartitionInput) (string, string, string) {
	if len(ps) == 0 {
		return "", "", ""
	}
	start, end := ps[0].StartTime, ps[0].EndTime
	for _, p := range ps[1:] {
		if p.StartTime < start {
			start = p.StartTime
		}
		if p.EndTime > end {
			end = p.EndTime
		}
	}
	return ps[0].Date, start, end
}

// Viewer 当前请求者的身份视角
type Viewer struct {
	Username   string
	Direction  string   // 首个方向（兼容旧调用）
	Directions []string // 已获邀请的全部方向
	IsAdmin    bool
}

// CanAccessDirection 是否可查看/预约该方向的场次
func (v Viewer) CanAccessDirection(direction string) bool {
	if v.IsAdmin {
		return true
	}
	if direction == "" {
		return false
	}
	for _, d := range v.Directions {
		if d == direction {
			return true
		}
	}
	// 兼容仅填充了 Direction 的调用方
	return v.Direction == direction
}

// SessionWithCount 场次 + 统计（含日程列表，一个场次可跨多天）
type SessionWithCount struct {
	model.Session
	SlotCount    int64           `json:"slotCount"`
	Participants int64           `json:"participants"`
	Partitions   []PartitionView `json:"partitions"`
}

// SlotView 时段 + 首选人数 + 我的心仪时间 + 我的已确认分配 + 容量/已占用
type SlotView struct {
	model.Slot
	FirstChoiceCount int  `json:"firstChoiceCount"`
	PrefCount        int  `json:"prefCount"`   // 被选为心仪的总人数（不限 rank）
	MyRank           int  `json:"myRank"`      // 0 表示未选
	MyConfirmed      bool `json:"myConfirmed"` // 是否是我的最终面试时段
	Occupied         int  `json:"occupied"`    // 最新分配版本中已占用该时段的人数
}

// SessionDetail 场次详情（成员/管理员共用）
type SessionDetail struct {
	Session       model.Session   `json:"session"`
	Slots         []SlotView      `json:"slots"`
	Partitions    []PartitionView `json:"partitions"`
	SlotCapacitys map[string]int  `json:"slotCapacities"` // 逐时段容量覆盖（编辑页回填用）
	Participants  int64           `json:"participants"`
}

// CalendarSession 日历视图中的场次（含时段与我的选择）
type CalendarSession struct {
	Session      model.Session   `json:"session"`
	Slots        []SlotView      `json:"slots"`
	Partitions   []PartitionView `json:"partitions"`
	Participants int64           `json:"participants"`
	MyRankOf     map[uint]int    `json:"-"`
	Confirmed    bool            `json:"-"` // 我是否已确认分配到本场次
}

// SessionService 场次业务
type SessionService struct {
	repo *repository.Repo
}

func NewSessionService(repo *repository.Repo) *SessionService {
	return &SessionService{repo: repo}
}

// GenerateSlots 按起止时间与每场时长切分时段（capacity 为每个时段可同时面试的人数）
func GenerateSlots(start, end string, slotMinutes int, location string, capacity int) ([]model.Slot, error) {
	startMin, err := parseHM(start)
	if err != nil {
		return nil, BadRequest("开始时间格式应为 HH:MM")
	}
	endMin, err := parseHM(end)
	if err != nil {
		return nil, BadRequest("结束时间格式应为 HH:MM")
	}
	if slotMinutes <= 0 || slotMinutes%5 != 0 {
		return nil, BadRequest("每场时长必须为 5 的正整数倍（分钟）")
	}
	if endMin <= startMin {
		return nil, BadRequest("结束时间必须晚于开始时间")
	}
	if endMin > 24*60 {
		return nil, BadRequest("结束时间不能超过 24:00")
	}
	n := (endMin - startMin) / slotMinutes
	if n < 1 {
		return nil, BadRequest("所选时间段不足以容纳一场面试，请调整时长")
	}
	slots := make([]model.Slot, 0, n)
	cap := capacity
	if cap < 1 {
		cap = 1
	}
	for i := 0; i < n; i++ {
		s := startMin + i*slotMinutes
		e := s + slotMinutes
		slots = append(slots, model.Slot{
			Idx:       i,
			StartTime: formatHM(s),
			EndTime:   formatHM(e),
			Location:  location,
			Capacity:  cap,
		})
	}
	return slots, nil
}

// normalizeCapacity 容量兜底：至少 1 人，最多 20 人
func normalizeCapacity(c int) int {
	if c < 1 {
		return 1
	}
	if c > 20 {
		return 20
	}
	return c
}

func validateInput(in CreateSessionInput) error {
	in.Title = strings.TrimSpace(in.Title)
	in.Location = strings.TrimSpace(in.Location)
	if in.Title == "" {
		return BadRequest("请填写场次标题")
	}
	if !model.ValidDirection(in.Direction) {
		return BadRequest("请选择面试方向")
	}
	if in.Location == "" {
		return BadRequest("请填写面试地点")
	}
	if strings.TrimSpace(in.Contact) == "" {
		return BadRequest("请填写联系人")
	}
	if strings.TrimSpace(in.ContactQQ) == "" {
		return BadRequest("请填写联系人 QQ")
	}
	// 单日程（旧格式）时才校验 Session.Date；多日程由 normalizePartitions 逐个校验
	if len(in.Partitions) == 0 {
		return validateDate(in.Date)
	}
	return nil
}

// Create 创建场次（草稿态）并切分时段
func (s *SessionService) Create(in CreateSessionInput, admin string) (*model.Session, []model.Slot, error) {
	if err := validateInput(in); err != nil {
		return nil, nil, err
	}
	ps, slots, err := in.normalizePartitions()
	if err != nil {
		return nil, nil, err
	}
	date, start, end := sessionTimeRange(ps)
	sess := &model.Session{
		Title:         strings.TrimSpace(in.Title),
		Direction:     in.Direction,
		Date:          date,
		StartTime:     start,
		EndTime:       end,
		SlotMinutes:   in.SlotMinutes,
		SlotCapacity:  normalizeCapacity(in.SlotCapacity),
		SlotCapacitys: EncodeSlotCapacities(in.SlotCapacities),
		Location:      strings.TrimSpace(in.Location),
		Contact:       strings.TrimSpace(in.Contact),
		ContactQQ:     strings.TrimSpace(in.ContactQQ),
		Status:        model.StatusDraft,
		CreatedBy:     admin,
	}
	if err := s.repo.CreateSessionWithSlots(sess, slots); err != nil {
		return nil, nil, err
	}
	return sess, slots, nil
}

// Update 编辑场次（重新切分时段）。
// 未确认（草稿/预约中/已分配待确认）的场次都可以编辑：只要被改动/删除的时段没有人选择。
// 已确认的场次锁定，需删除场次重建。
func (s *SessionService) Update(id uint, in CreateSessionInput) (*model.Session, []model.Slot, error) {
	sess, err := s.repo.GetSession(id)
	if err != nil {
		return nil, nil, NotFound("场次不存在")
	}
	if sess.Status == model.StatusConfirmed {
		return nil, nil, BadRequest("分配已确认后不能修改日程；如需调整请删除场次后重建")
	}
	if err := validateInput(in); err != nil {
		return nil, nil, err
	}
	ps, slots, err := in.normalizePartitions()
	if err != nil {
		return nil, nil, err
	}
	// 保护已有人选择的时段：被移除（改时间/改日期/删日程）的时段不能还有心仪时间
	oldSlots, err := s.repo.GetSlots(id)
	if err != nil {
		return nil, nil, err
	}
	newKeys := make(map[string]bool, len(slots))
	for _, sl := range slots {
		newKeys[sl.Date+" "+sl.StartTime+" "+sl.EndTime] = true
	}
	removed := make([]model.Slot, 0)
	for _, o := range oldSlots {
		if !newKeys[o.Date+" "+o.StartTime+" "+o.EndTime] {
			removed = append(removed, o)
		}
	}
	if len(removed) > 0 {
		prefCounts, err := s.repo.PreferenceCountsBySession(id)
		if err != nil {
			return nil, nil, err
		}
		busy := make([]string, 0)
		for _, o := range removed {
			if prefCounts[o.ID] > 0 {
				busy = append(busy, o.Date+" "+o.StartTime+"-"+o.EndTime)
			}
		}
		if len(busy) > 0 {
			sort.Strings(busy)
			return nil, nil, BadRequest("以下时段已有人选择，不能修改或删除：" + strings.Join(busy, "、") +
				"；可让成员改选其它时段，或直接删除整个场次")
		}
	}
	date, start, end := sessionTimeRange(ps)
	sess.Title = strings.TrimSpace(in.Title)
	sess.Direction = in.Direction
	sess.Date = date
	sess.StartTime = start
	sess.EndTime = end
	sess.SlotMinutes = in.SlotMinutes
	sess.SlotCapacity = normalizeCapacity(in.SlotCapacity)
	sess.SlotCapacitys = EncodeSlotCapacities(in.SlotCapacities)
	sess.Location = strings.TrimSpace(in.Location)
	sess.Contact = strings.TrimSpace(in.Contact)
	sess.ContactQQ = strings.TrimSpace(in.ContactQQ)
	if err := s.repo.UpdateSession(sess); err != nil {
		return nil, nil, err
	}
	// 未变时段沿用原 ID：已保存的心仪时间与分配记录继续有效
	if err := s.repo.ReplaceSlots(id, slots); err != nil {
		return nil, nil, err
	}
	return sess, slots, nil
}

// mergeSlotCapacities 把新的逐时段容量覆盖并入场次已有配置（key = 日期 + 开始时间）
func mergeSlotCapacities(raw string, add map[string]int) string {
	m := DecodeSlotCapacities(raw)
	for k, v := range add {
		if v > 0 {
			m[k] = v
		}
	}
	if len(m) == 0 {
		return ""
	}
	b, err := json.Marshal(m)
	if err != nil {
		return raw
	}
	return string(b)
}

// sessionRangeOfSlots 由全部时段推算场次的整体日期区间（取最早一天）
func sessionRangeOfSlots(slots []model.Slot) (date, start, end string) {
	for _, sl := range slots {
		if date == "" || sl.Date < date {
			date = sl.Date
		}
	}
	for _, sl := range slots {
		if sl.Date != date {
			continue
		}
		if start == "" || sl.StartTime < start {
			start = sl.StartTime
		}
		if sl.EndTime > end {
			end = sl.EndTime
		}
	}
	return
}

// AppendPartitions 已发布场次追加日程：只新增时段，不重切已有时段，
// 因此已有时段的心仪时间、分配结果都保持不变（新时段为空闲）。
func (s *SessionService) AppendPartitions(id uint, in CreateSessionInput) (*model.Session, []model.Slot, []PartitionView, error) {
	sess, err := s.repo.GetSession(id)
	if err != nil {
		return nil, nil, nil, NotFound("场次不存在")
	}
	if sess.Status == model.StatusDraft {
		return nil, nil, nil, BadRequest("草稿场次请直接编辑场次后保存")
	}
	if sess.Status == model.StatusConfirmed {
		return nil, nil, nil, BadRequest("分配已确认后不能追加日程；如需调整请删除场次后重建")
	}
	ps := make([]PartitionInput, 0, len(in.Partitions))
	defCap := sess.SlotCapacity
	if defCap < 1 {
		defCap = 1
	}
	for _, p := range in.Partitions {
		p.Date = strings.TrimSpace(p.Date)
		p.StartTime = strings.TrimSpace(p.StartTime)
		p.EndTime = strings.TrimSpace(p.EndTime)
		p.Location = strings.TrimSpace(p.Location)
		if p.Date == "" || p.StartTime == "" || p.EndTime == "" {
			continue
		}
		if err := validateDate(p.Date); err != nil {
			return nil, nil, nil, err
		}
		if p.Capacity <= 0 {
			p.Capacity = defCap
		}
		if p.Capacity > 20 {
			return nil, nil, nil, BadRequest("每个时段最多同时面试 20 人")
		}
		if p.Location == "" {
			p.Location = sess.Location
		}
		ps = append(ps, p)
	}
	if len(ps) == 0 {
		return nil, nil, nil, BadRequest("请至少添加一个日程（日期 + 时间段）")
	}
	sort.Slice(ps, func(i, j int) bool {
		if ps[i].Date != ps[j].Date {
			return ps[i].Date < ps[j].Date
		}
		return ps[i].StartTime < ps[j].StartTime
	})
	// 逐时段容量覆盖（key = 日期 + 开始时间）
	overCap := make(map[string]int, len(in.SlotCapacities))
	for _, sc := range in.SlotCapacities {
		d, t := strings.TrimSpace(sc.Date), strings.TrimSpace(sc.StartTime)
		if d == "" || t == "" || sc.Capacity <= 0 {
			continue
		}
		if sc.Capacity > 20 {
			return nil, nil, nil, BadRequest("每个时段最多同时面试 20 人")
		}
		overCap[d+" "+t] = sc.Capacity
	}

	existing, err := s.repo.GetSlots(id)
	if err != nil {
		return nil, nil, nil, err
	}
	type interval struct{ start, end int }
	busy := make(map[string][]interval, len(existing))
	nextIdx, nextPart := 0, 0
	for _, sl := range existing {
		st, err := parseHM(sl.StartTime)
		if err != nil {
			return nil, nil, nil, err
		}
		en, err := parseHM(sl.EndTime)
		if err != nil {
			return nil, nil, nil, err
		}
		busy[sl.Date] = append(busy[sl.Date], interval{st, en})
		if sl.Idx >= nextIdx {
			nextIdx = sl.Idx + 1
		}
		if sl.PartitionIdx >= nextPart {
			nextPart = sl.PartitionIdx + 1
		}
	}

	newSlots := make([]model.Slot, 0)
	for i, p := range ps {
		st, err := parseHM(p.StartTime)
		if err != nil {
			return nil, nil, nil, BadRequest("开始时间格式应为 HH:MM")
		}
		en, err := parseHM(p.EndTime)
		if err != nil {
			return nil, nil, nil, BadRequest("结束时间格式应为 HH:MM")
		}
		// 与已有日程（含本次已追加的）时间重叠时拒绝：时段 ID 与心仪/分配都基于不重叠的时段
		for _, b := range busy[p.Date] {
			if st < b.end && en > b.start {
				return nil, nil, nil, BadRequest(p.Date + " " + p.StartTime + "-" + p.EndTime + " 与已有日程时间重叠，请调整")
			}
		}
		daySlots, err := GenerateSlots(p.StartTime, p.EndTime, sess.SlotMinutes, p.Location, p.Capacity)
		if err != nil {
			return nil, nil, nil, err
		}
		for j := range daySlots {
			if c, ok := overCap[p.Date+" "+daySlots[j].StartTime]; ok {
				daySlots[j].Capacity = c
			}
			daySlots[j].Date = p.Date
			daySlots[j].PartitionIdx = nextPart + i
			daySlots[j].Idx = nextIdx + len(newSlots)
			newSlots = append(newSlots, daySlots[j])
		}
		busy[p.Date] = append(busy[p.Date], interval{st, en})
	}

	sess.SlotCapacitys = mergeSlotCapacities(sess.SlotCapacitys, overCap)
	allSlots := make([]model.Slot, 0, len(existing)+len(newSlots))
	allSlots = append(allSlots, existing...)
	allSlots = append(allSlots, newSlots...)
	sess.Date, sess.StartTime, sess.EndTime = sessionRangeOfSlots(allSlots)
	if err := s.repo.AppendSlots(sess, newSlots); err != nil {
		return nil, nil, nil, err
	}
	return sess, newSlots, BuildPartitionViews(allSlots), nil
}

// Publish 发布场次（草稿 -> 已发布，成员可见）
func (s *SessionService) Publish(id uint) error {
	sess, err := s.repo.GetSession(id)
	if err != nil {
		return NotFound("场次不存在")
	}
	if sess.Status != model.StatusDraft {
		return BadRequest("仅草稿状态的场次可以发布")
	}
	sess.Status = model.StatusPublished
	return s.repo.UpdateSession(sess)
}

// Delete 删除场次：任何状态都可删除，成员的心仪时间与分配结果一并清理
func (s *SessionService) Delete(id uint) error {
	if _, err := s.repo.GetSession(id); err != nil {
		return NotFound("场次不存在")
	}
	return s.repo.DeleteSession(id)
}

// List 场次列表（管理员看全部；成员只看自己方向的已发布场次）
func (s *SessionService) List(v Viewer) ([]SessionWithCount, error) {
	sessions, err := s.repo.ListSessions()
	if err != nil {
		return nil, err
	}
	out := make([]SessionWithCount, 0, len(sessions))
	keep := make([]model.Session, 0, len(sessions))
	for _, sess := range sessions {
		if !v.IsAdmin {
			if sess.Status == model.StatusDraft {
				continue
			}
			if !v.CanAccessDirection(sess.Direction) {
				continue
			}
		}
		keep = append(keep, sess)
	}
	ids := make([]uint, 0, len(keep))
	for _, sess := range keep {
		ids = append(ids, sess.ID)
	}
	slotsMap, err := s.repo.SlotsBySessions(ids)
	if err != nil {
		return nil, err
	}
	for _, sess := range keep {
		participants, err := s.repo.CountParticipants(sess.ID)
		if err != nil {
			return nil, err
		}
		slots := slotsMap[sess.ID]
		out = append(out, SessionWithCount{
			Session:      sess,
			SlotCount:    int64(len(slots)),
			Participants: participants,
			Partitions:   BuildPartitionViews(slots),
		})
	}
	return out, nil
}

// Detail 场次详情：时段 + 首选人数 + 我的心仪时间 + 我的已确认分配
func (s *SessionService) Detail(id uint, v Viewer) (*SessionDetail, error) {
	sess, err := s.repo.GetSession(id)
	if err != nil {
		return nil, NotFound("场次不存在")
	}
	if !v.IsAdmin {
		if sess.Status == model.StatusDraft {
			return nil, NotFound("场次不存在")
		}
		if !v.CanAccessDirection(sess.Direction) {
			return nil, NotFound("场次不存在")
		}
	}
	return s.buildDetail(sess, v.Username)
}

func (s *SessionService) buildDetail(sess *model.Session, username string) (*SessionDetail, error) {
	slots, err := s.repo.GetSlots(sess.ID)
	if err != nil {
		return nil, err
	}
	firstCounts, err := s.repo.FirstChoiceCounts(sess.ID)
	if err != nil {
		return nil, err
	}
	myPrefs, err := s.repo.MyPreferences(sess.ID, username)
	if err != nil {
		return nil, err
	}
	rankOf := make(map[uint]int, len(myPrefs))
	for _, p := range myPrefs {
		rankOf[p.SlotID] = p.Rank
	}
	myAssign, _ := s.repo.MyConfirmedAssignment(sess.ID, username)
	participants, err := s.repo.CountParticipants(sess.ID)
	if err != nil {
		return nil, err
	}
	occupied, err := s.repo.AssignmentCountsInLatestVersion(sess.ID)
	if err != nil {
		return nil, err
	}
	prefCounts, err := s.repo.PreferenceCountsBySession(sess.ID)
	if err != nil {
		return nil, err
	}
	views := make([]SlotView, 0, len(slots))
	for _, slot := range slots {
		if slot.Capacity < 1 {
			slot.Capacity = 1 // 老数据兜底
		}
		views = append(views, SlotView{
			Slot:             slot,
			FirstChoiceCount: firstCounts[slot.ID],
			MyRank:           rankOf[slot.ID],
			MyConfirmed:      myAssign != nil && myAssign.SlotID == slot.ID,
			PrefCount:        prefCounts[slot.ID],
			Occupied:         occupied[slot.ID],
		})
	}
	return &SessionDetail{
		Session:       *sess,
		Slots:         views,
		Partitions:    BuildPartitionViews(slots),
		SlotCapacitys: DecodeSlotCapacities(sess.SlotCapacitys),
		Participants:  participants,
	}, nil
}

// Calendar 日历视图：区间内各场次及其时段（成员仅已获邀请的方向；direction 非空时只取该方向）
func (s *SessionService) Calendar(start, end, direction string, v Viewer) ([]CalendarSession, error) {
	if err := validateDate(start); err != nil {
		return nil, err
	}
	if err := validateDate(end); err != nil {
		return nil, err
	}
	sessions, err := s.repo.ListSessionsInRange(start, end)
	if err != nil {
		return nil, err
	}
	out := make([]CalendarSession, 0, len(sessions))
	for _, sess := range sessions {
		if !v.CanAccessDirection(sess.Direction) {
			continue
		}
		if direction != "" && sess.Direction != direction {
			continue
		}
		detail, err := s.buildDetail(&sess, v.Username)
		if err != nil {
			return nil, err
		}
		rankOf := make(map[uint]int, len(detail.Slots))
		confirmed := false
		for _, sl := range detail.Slots {
			if sl.MyRank > 0 {
				rankOf[sl.ID] = sl.MyRank
			}
			if sl.MyConfirmed {
				confirmed = true
			}
		}
		out = append(out, CalendarSession{
			Session:      detail.Session,
			Slots:        detail.Slots,
			Partitions:   detail.Partitions,
			Participants: detail.Participants,
			MyRankOf:     rankOf,
			Confirmed:    confirmed,
		})
	}
	return out, nil
}

// ---------- 时间工具 ----------

func parseHM(s string) (int, error) {
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return 0, BadRequest("时间格式应为 HH:MM")
	}
	h, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, err
	}
	m, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, err
	}
	if h < 0 || h > 24 || m < 0 || m > 59 {
		return 0, BadRequest("时间超出范围")
	}
	return h*60 + m, nil
}

func formatHM(minutes int) string {
	return fmtHM(minutes/60) + ":" + fmtHM(minutes%60)
}

func fmtHM(v int) string {
	if v < 10 {
		return "0" + strconv.Itoa(v)
	}
	return strconv.Itoa(v)
}

func validateDate(s string) error {
	if _, err := time.Parse("2006-01-02", s); err != nil {
		return BadRequest("日期格式应为 YYYY-MM-DD")
	}
	return nil
}
