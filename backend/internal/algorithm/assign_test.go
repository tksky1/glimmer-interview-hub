package algorithm

import "testing"

func TestAssign_PreferFirstChoice(t *testing.T) {
	// 两人都首选 slot1：一人拿第一心仪，另一人拿第二心仪
	cands := []Candidate{
		{Username: "a", SlotIDs: []uint{1, 2, 3}},
		{Username: "b", SlotIDs: []uint{1, 3, 2}},
	}
	res := Assign(cands, []uint{1, 2, 3})
	if res.First != 1 || res.Second != 1 || res.Unassigned != 0 {
		t.Fatalf("期望 1 个第一心仪 + 1 个第二心仪，得到 first=%d second=%d unassigned=%d", res.First, res.Second, res.Unassigned)
	}
	if res.SlotOf["a"] == 0 || res.SlotOf["b"] == 0 || res.SlotOf["a"] == res.SlotOf["b"] {
		t.Fatalf("两人应分到不同时段: %v", res.SlotOf)
	}
}

func TestAssign_AllFirstSatisfiable(t *testing.T) {
	cands := []Candidate{
		{Username: "a", SlotIDs: []uint{1, 2, 3}},
		{Username: "b", SlotIDs: []uint{2, 1, 3}},
		{Username: "c", SlotIDs: []uint{3, 1, 2}},
	}
	res := Assign(cands, []uint{1, 2, 3})
	if res.First != 3 || res.Unassigned != 0 {
		t.Fatalf("期望全部第一心仪达成，得到 first=%d unassigned=%d", res.First, res.Unassigned)
	}
}

func TestAssign_MoreCandidatesThanSlots(t *testing.T) {
	cands := []Candidate{
		{Username: "a", SlotIDs: []uint{1, 2}},
		{Username: "b", SlotIDs: []uint{1, 2}},
		{Username: "c", SlotIDs: []uint{1, 2}},
	}
	res := Assign(cands, []uint{1, 2})
	if res.First+res.Second+res.Third != 2 {
		t.Fatalf("只能分配 2 人，得到 %d", res.First+res.Second+res.Third)
	}
	if res.Unassigned != 1 {
		t.Fatalf("应有 1 人未分配，得到 %d", res.Unassigned)
	}
}

func TestAssign_GlobalOptimality(t *testing.T) {
	// a 首选 1；b 首选 1，次选 2；c 只想选 2（第一心仪）
	// 最优：a->1, b->2(第二心仪), c 未分配 或 c->2 且 b 未分配？
	// 费用比较：方案一 a=1(1) b=2(6) c 无 => 7；方案二 a=1(1) c=2(1) b 无 => 2 => 更优
	cands := []Candidate{
		{Username: "a", SlotIDs: []uint{1, 3}},
		{Username: "b", SlotIDs: []uint{1, 2}},
		{Username: "c", SlotIDs: []uint{2, 1}},
	}
	res := Assign(cands, []uint{1, 2})
	if res.SlotOf["a"] != 1 {
		t.Fatalf("a 应分到 slot1，得到 %d", res.SlotOf["a"])
	}
	if res.SlotOf["c"] != 2 {
		t.Fatalf("c 应分到 slot2（全局更优），得到 %d", res.SlotOf["c"])
	}
	if res.SlotOf["b"] != 0 {
		t.Fatalf("b 应未分配，得到 %d", res.SlotOf["b"])
	}
	if res.First != 2 || res.Unassigned != 1 {
		t.Fatalf("期望 first=2 unassigned=1，得到 first=%d unassigned=%d", res.First, res.Unassigned)
	}
}

func TestAssign_NoPreferences(t *testing.T) {
	cands := []Candidate{{Username: "a"}, {Username: "b"}}
	res := Assign(cands, []uint{1, 2})
	if res.Unassigned != 2 || res.Participants != 2 {
		t.Fatalf("无心仪选择应全部未分配，得到 unassigned=%d", res.Unassigned)
	}
}

func TestAssignWithCapacity_MultiplePerSlot(t *testing.T) {
	// 3 人都首选 slot1，slot1 容量 2：两人拿第一心仪，第三人落到第二心仪
	cands := []Candidate{
		{Username: "a", SlotIDs: []uint{1, 2, 3}},
		{Username: "b", SlotIDs: []uint{1, 2, 3}},
		{Username: "c", SlotIDs: []uint{1, 2, 3}},
	}
	res := AssignWithCapacity(cands, []uint{1, 2, 3}, map[uint]int{1: 2, 2: 1, 3: 1})
	if res.First != 2 || res.Unassigned != 0 {
		t.Fatalf("slot1 容量 2 时应满足 2 个第一心仪，得到 first=%d unassigned=%d", res.First, res.Unassigned)
	}
	same := 0
	for _, id := range res.SlotOf {
		if id == 1 {
			same++
		}
	}
	if same != 2 {
		t.Fatalf("slot1 上应恰好 2 人，得到 %d", same)
	}
}

func TestAssignWithCapacity_CapacityOneUnchanged(t *testing.T) {
	cands := []Candidate{
		{Username: "a", SlotIDs: []uint{1, 2, 3}},
		{Username: "b", SlotIDs: []uint{1, 2, 3}},
	}
	res := AssignWithCapacity(cands, []uint{1, 2, 3}, map[uint]int{1: 1, 2: 1, 3: 1})
	if res.First != 1 || res.Second != 1 || res.Unassigned != 0 {
		t.Fatalf("容量 1 应与旧行为一致，得到 first=%d second=%d unassigned=%d", res.First, res.Second, res.Unassigned)
	}
}

func TestAssignWithCapacity_OverCapacityStillBounded(t *testing.T) {
	// 5 人抢 1 个容量 2 的时段，另有两个容量 1 的时段：最多 4 人分配成功
	cands := make([]Candidate, 0, 5)
	for i := 0; i < 5; i++ {
		cands = append(cands, Candidate{Username: string(rune('a' + i)), SlotIDs: []uint{1, 2, 3}})
	}
	res := AssignWithCapacity(cands, []uint{1, 2, 3}, map[uint]int{1: 2, 2: 1, 3: 1})
	got := res.First + res.Second + res.Third
	if got != 4 || res.Unassigned != 1 {
		t.Fatalf("应分配 4 人、1 人未分配，得到 assigned=%d unassigned=%d", got, res.Unassigned)
	}
}

func TestAssignWithCapacity_MixedCapacities(t *testing.T) {
	// 混合容量：slot1=1 人、slot2=3 人、slot3=2 人
	// 4 个人的第一心仪都是 slot2：应 3 人进 slot2（容量 3），第 4 人落到第二心仪
	cands := []Candidate{
		{Username: "a", SlotIDs: []uint{2, 1, 3}},
		{Username: "b", SlotIDs: []uint{2, 1, 3}},
		{Username: "c", SlotIDs: []uint{2, 1, 3}},
		{Username: "d", SlotIDs: []uint{2, 3, 1}},
	}
	res := AssignWithCapacity(cands, []uint{1, 2, 3}, map[uint]int{1: 1, 2: 3, 3: 2})
	if res.First != 3 || res.Second != 1 || res.Unassigned != 0 {
		t.Fatalf("期望 3 个第一心仪 + 1 个第二心仪，得到 first=%d second=%d unassigned=%d", res.First, res.Second, res.Unassigned)
	}
	inSlot2 := 0
	for _, id := range res.SlotOf {
		if id == 2 {
			inSlot2++
		}
	}
	if inSlot2 != 3 {
		t.Fatalf("容量 3 的 slot2 上应恰好 3 人，得到 %d", inSlot2)
	}
	// 容量 1 的 slot1 最多 1 人
	inSlot1 := 0
	for _, id := range res.SlotOf {
		if id == 1 {
			inSlot1++
		}
	}
	if inSlot1 > 1 {
		t.Fatalf("容量 1 的 slot1 上不应超过 1 人，得到 %d", inSlot1)
	}
}

func TestAssignWithCapacity_PartialMultiStillOptimal(t *testing.T) {
	// 部分时段多人、部分单人混在一起：
	// a 首选 slot1(容量1)，b 首选 slot2(容量2)，c 首选 slot2，d 首选 slot3(容量1)
	// 四个人都该拿到第一心仪（1+2+1 = 4 个位子）
	cands := []Candidate{
		{Username: "a", SlotIDs: []uint{1, 2, 3}},
		{Username: "b", SlotIDs: []uint{2, 1, 3}},
		{Username: "c", SlotIDs: []uint{2, 3, 1}},
		{Username: "d", SlotIDs: []uint{3, 2, 1}},
	}
	res := AssignWithCapacity(cands, []uint{1, 2, 3}, map[uint]int{1: 1, 2: 2, 3: 1})
	if res.First != 4 || res.Unassigned != 0 {
		t.Fatalf("混合容量下应全部第一心仪达成，得到 first=%d unassigned=%d", res.First, res.Unassigned)
	}
	if res.SlotOf["a"] != 1 || res.SlotOf["d"] != 3 {
		t.Fatalf("单人时段应各分 1 人: %v", res.SlotOf)
	}
}
