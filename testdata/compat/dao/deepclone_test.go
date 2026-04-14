package dao_test

import (
	"testing"

	"github.com/pinealctx/gcode/testdata/compat/dao"
)

// TestDeepCloneNilReceiver verifies that DeepClone on a nil pointer returns nil.
func TestDeepCloneNilReceiver(t *testing.T) {
	t.Parallel()

	var p *dao.Person
	if got := p.DeepClone(); got != nil {
		t.Errorf("nil.DeepClone() = %v, want nil", got)
	}

	var addr *dao.Address
	if got := addr.DeepClone(); got != nil {
		t.Errorf("nil Address.DeepClone() = %v, want nil", got)
	}

	var node *dao.TreeNode
	if got := node.DeepClone(); got != nil {
		t.Errorf("nil TreeNode.DeepClone() = %v, want nil", got)
	}
}

// TestDeepCloneScalarFields verifies that scalar fields are copied correctly
// and that modifying the clone does not affect the original.
func TestDeepCloneScalarFields(t *testing.T) {
	t.Parallel()

	orig := &dao.Person{
		Name:      "Alice",
		Age:       30,
		Active:    true,
		Status:    dao.Status_STATUS_ACTIVE,
		CreatedAt: 1000,
		Rating:    3.14,
		Email:     "alice@example.com",
	}

	clone := orig.DeepClone()

	if clone.Name != orig.Name || clone.Age != orig.Age || clone.Active != orig.Active {
		t.Errorf("scalar fields not copied correctly")
	}

	// Mutate clone; original must be unchanged.
	clone.Name = "Bob"
	clone.Age = 99
	if orig.Name != "Alice" || orig.Age != 30 {
		t.Errorf("mutating clone affected original scalar fields")
	}
}

// TestDeepCloneOptionalFields verifies that optional (pointer) fields get
// independent copies — modifying through the clone's pointer must not affect
// the original.
func TestDeepCloneOptionalFields(t *testing.T) {
	t.Parallel()

	nickname := "nick"
	level := int32(5)
	verified := true
	score := float32(9.9)
	updatedAt := int64(2000)
	prevStatus := dao.Status_STATUS_INACTIVE

	orig := &dao.Person{
		Nickname:   &nickname,
		Level:      &level,
		Verified:   &verified,
		Score:      &score,
		UpdatedAt:  &updatedAt,
		PrevStatus: &prevStatus,
	}

	clone := orig.DeepClone()

	// Pointers must be different.
	if clone.Nickname == orig.Nickname {
		t.Error("Nickname pointer shared between clone and original")
	}
	if clone.Level == orig.Level {
		t.Error("Level pointer shared between clone and original")
	}
	if clone.Verified == orig.Verified {
		t.Error("Verified pointer shared between clone and original")
	}
	if clone.Score == orig.Score {
		t.Error("Score pointer shared between clone and original")
	}
	if clone.UpdatedAt == orig.UpdatedAt {
		t.Error("UpdatedAt pointer shared between clone and original")
	}
	if clone.PrevStatus == orig.PrevStatus {
		t.Error("PrevStatus pointer shared between clone and original")
	}

	// Values must be equal.
	if *clone.Nickname != nickname {
		t.Errorf("Nickname value mismatch: got %q, want %q", *clone.Nickname, nickname)
	}

	// Mutate through clone; original must be unchanged.
	*clone.Nickname = "changed"
	if *orig.Nickname != "nick" {
		t.Errorf("mutating clone.Nickname affected original")
	}
	*clone.Level = 99
	if *orig.Level != 5 {
		t.Errorf("mutating clone.Level affected original")
	}
}

// TestDeepCloneMessageField verifies that nested message fields are recursively cloned.
func TestDeepCloneMessageField(t *testing.T) {
	t.Parallel()

	orig := &dao.Person{
		Address: &dao.Address{Street: "Main St", City: "Springfield"},
	}

	clone := orig.DeepClone()

	if clone.Address == orig.Address {
		t.Error("Address pointer shared between clone and original")
	}
	if clone.Address.Street != "Main St" || clone.Address.City != "Springfield" {
		t.Errorf("Address fields not copied correctly: %+v", clone.Address)
	}

	// Mutate clone's nested message; original must be unchanged.
	clone.Address.Street = "Other St"
	if orig.Address.Street != "Main St" {
		t.Errorf("mutating clone.Address.Street affected original")
	}
}

// TestDeepCloneRepeatedScalar verifies that repeated scalar slices are independently copied.
func TestDeepCloneRepeatedScalar(t *testing.T) {
	t.Parallel()

	orig := &dao.Person{
		Scores: []int32{1, 2, 3},
		Tags:   []string{"a", "b"},
	}

	clone := orig.DeepClone()

	if &clone.Scores[0] == &orig.Scores[0] {
		t.Error("Scores slice shares backing array with original")
	}

	// Mutate clone; original must be unchanged.
	clone.Scores[0] = 99
	if orig.Scores[0] != 1 {
		t.Errorf("mutating clone.Scores[0] affected original")
	}
	clone.Tags[0] = "z"
	if orig.Tags[0] != "a" {
		t.Errorf("mutating clone.Tags[0] affected original")
	}
}

// TestDeepCloneBytesField verifies that bytes fields (HasPresence and regular) are independently copied.
func TestDeepCloneBytesField(t *testing.T) {
	t.Parallel()

	orig := &dao.Person{
		Avatar:      []byte{1, 2, 3},
		Fingerprint: []byte{4, 5, 6},
	}

	clone := orig.DeepClone()

	if &clone.Avatar[0] == &orig.Avatar[0] {
		t.Error("Avatar slice shares backing array with original")
	}
	if &clone.Fingerprint[0] == &orig.Fingerprint[0] {
		t.Error("Fingerprint slice shares backing array with original")
	}

	clone.Avatar[0] = 99
	if orig.Avatar[0] != 1 {
		t.Errorf("mutating clone.Avatar[0] affected original")
	}
	clone.Fingerprint[0] = 99
	if orig.Fingerprint[0] != 4 {
		t.Errorf("mutating clone.Fingerprint[0] affected original")
	}
}

// TestDeepCloneRepeatedMessage verifies that repeated message slices are recursively cloned.
func TestDeepCloneRepeatedMessage(t *testing.T) {
	t.Parallel()

	orig := &dao.AllRepeated{
		RMessage: []*dao.Address{
			{Street: "First", City: "CityA"},
			nil,
			{Street: "Second", City: "CityB"},
		},
	}

	clone := orig.DeepClone()

	if clone.RMessage[0] == orig.RMessage[0] {
		t.Error("RMessage[0] pointer shared between clone and original")
	}
	if clone.RMessage[1] != nil {
		t.Error("nil element in RMessage should remain nil after DeepClone")
	}

	clone.RMessage[0].Street = "Changed"
	if orig.RMessage[0].Street != "First" {
		t.Errorf("mutating clone.RMessage[0].Street affected original")
	}
}

// TestDeepCloneRepeatedBytes verifies that repeated bytes fields deep-copy each element.
func TestDeepCloneRepeatedBytes(t *testing.T) {
	t.Parallel()

	orig := &dao.AllRepeated{
		RBytes: [][]byte{{1, 2}, nil, {3, 4}},
	}

	clone := orig.DeepClone()

	if &clone.RBytes[0][0] == &orig.RBytes[0][0] {
		t.Error("RBytes[0] shares backing array with original")
	}
	if clone.RBytes[1] != nil {
		t.Error("nil element in RBytes should remain nil after DeepClone")
	}

	clone.RBytes[0][0] = 99
	if orig.RBytes[0][0] != 1 {
		t.Errorf("mutating clone.RBytes[0][0] affected original")
	}
}

// TestDeepCloneRepeatedEnum verifies that repeated enum slices are independently copied.
func TestDeepCloneRepeatedEnum(t *testing.T) {
	t.Parallel()

	orig := &dao.AllRepeated{
		REnum: []dao.Status{dao.Status_STATUS_ACTIVE, dao.Status_STATUS_INACTIVE},
	}

	clone := orig.DeepClone()

	if &clone.REnum[0] == &orig.REnum[0] {
		t.Error("REnum slice shares backing array with original")
	}

	clone.REnum[0] = dao.Status_STATUS_UNSPECIFIED
	if orig.REnum[0] != dao.Status_STATUS_ACTIVE {
		t.Errorf("mutating clone.REnum[0] affected original")
	}
}

// TestDeepCloneNestedMessage verifies that recursive message structures (TreeNode) are fully cloned.
func TestDeepCloneNestedMessage(t *testing.T) {
	t.Parallel()

	orig := &dao.TreeNode{
		Value: "root",
		Child: &dao.TreeNode{
			Value: "child",
			Child: &dao.TreeNode{Value: "leaf"},
		},
	}

	clone := orig.DeepClone()

	if clone.Child == orig.Child {
		t.Error("Child pointer shared between clone and original")
	}
	if clone.Child.Child == orig.Child.Child {
		t.Error("Child.Child pointer shared between clone and original")
	}

	clone.Child.Value = "changed"
	if orig.Child.Value != "child" {
		t.Errorf("mutating clone.Child.Value affected original")
	}
}

// TestDeepCloneNilSlices verifies that nil slices remain nil after cloning.
func TestDeepCloneNilSlices(t *testing.T) {
	t.Parallel()

	orig := &dao.Person{}
	clone := orig.DeepClone()

	if clone.Scores != nil {
		t.Error("nil Scores should remain nil after DeepClone")
	}
	if clone.Tags != nil {
		t.Error("nil Tags should remain nil after DeepClone")
	}
	if clone.Avatar != nil {
		t.Error("nil Avatar should remain nil after DeepClone")
	}
}

// TestDeepCloneNilOptionalFields verifies that nil optional fields remain nil after cloning.
func TestDeepCloneNilOptionalFields(t *testing.T) {
	t.Parallel()

	orig := &dao.Person{}
	clone := orig.DeepClone()

	if clone.Nickname != nil {
		t.Error("nil Nickname should remain nil after DeepClone")
	}
	if clone.Level != nil {
		t.Error("nil Level should remain nil after DeepClone")
	}
	if clone.Address != nil {
		t.Error("nil Address should remain nil after DeepClone")
	}
}

// BenchmarkDeepClonePerson measures the allocation cost of cloning a Person
// with all field types populated (optional, repeated, nested message).
func BenchmarkDeepClonePerson(b *testing.B) {
	nickname := "nick"
	level := int32(5)
	orig := &dao.Person{
		Name:      "Alice",
		Age:       30,
		Nickname:  &nickname,
		Level:     &level,
		Scores:    []int32{1, 2, 3, 4, 5},
		Tags:      []string{"a", "b", "c"},
		Avatar:    []byte{1, 2, 3, 4},
		Address:   &dao.Address{Street: "Main St", City: "Springfield"},
		Status:    dao.Status_STATUS_ACTIVE,
	}
	b.ReportAllocs()
	for range b.N {
		_ = orig.DeepClone()
	}
}

// TestDeepCloneAllScalarsCreate verifies that AllScalarsCreate (which contains
// sint32/sfixed32/fixed32/uint32/float32 optional fields) is correctly deep-cloned.
func TestDeepCloneAllScalarsCreate(t *testing.T) {
	t.Parallel()

	fSint32 := int32(1)
	fSint64 := int64(2)
	fSfixed32 := int32(3)
	fSfixed64 := int64(4)
	fFixed32 := uint32(5)
	fFixed64 := uint64(6)
	fUint32 := uint32(7)
	fUint64 := uint64(8)
	fFloat := float32(9.9)

	orig := &dao.AllScalarsCreate{
		FSint32:   &fSint32,
		FSint64:   &fSint64,
		FSfixed32: &fSfixed32,
		FSfixed64: &fSfixed64,
		FFixed32:  &fFixed32,
		FFixed64:  &fFixed64,
		FUint32:   &fUint32,
		FUint64:   &fUint64,
		FFloat:    &fFloat,
		FBytes:    []byte{1, 2, 3},
	}

	clone := orig.DeepClone()

	// All optional pointers must be independent.
	if clone.FSint32 == orig.FSint32 {
		t.Error("FSint32 pointer shared")
	}
	if clone.FSint64 == orig.FSint64 {
		t.Error("FSint64 pointer shared")
	}
	if clone.FSfixed32 == orig.FSfixed32 {
		t.Error("FSfixed32 pointer shared")
	}
	if clone.FSfixed64 == orig.FSfixed64 {
		t.Error("FSfixed64 pointer shared")
	}
	if clone.FFixed32 == orig.FFixed32 {
		t.Error("FFixed32 pointer shared")
	}
	if clone.FFixed64 == orig.FFixed64 {
		t.Error("FFixed64 pointer shared")
	}
	if clone.FUint32 == orig.FUint32 {
		t.Error("FUint32 pointer shared")
	}
	if clone.FUint64 == orig.FUint64 {
		t.Error("FUint64 pointer shared")
	}
	if clone.FFloat == orig.FFloat {
		t.Error("FFloat pointer shared")
	}
	if &clone.FBytes[0] == &orig.FBytes[0] {
		t.Error("FBytes shares backing array")
	}

	// Mutate clone; original must be unchanged.
	*clone.FSint32 = 99
	if *orig.FSint32 != 1 {
		t.Errorf("mutating clone.FSint32 affected original")
	}
	clone.FBytes[0] = 99
	if orig.FBytes[0] != 1 {
		t.Errorf("mutating clone.FBytes[0] affected original")
	}
}

// TestDeepCloneEmptySlice verifies that a non-nil empty slice is cloned as a
// non-nil empty slice (distinct from nil-slice behaviour).
func TestDeepCloneEmptySlice(t *testing.T) {
	t.Parallel()

	orig := &dao.Person{
		Scores: []int32{},
		Tags:   []string{},
		Avatar: []byte{},
	}

	clone := orig.DeepClone()

	if clone.Scores == nil {
		t.Error("empty Scores should remain non-nil after DeepClone")
	}
	if clone.Tags == nil {
		t.Error("empty Tags should remain non-nil after DeepClone")
	}
	if clone.Avatar == nil {
		t.Error("empty Avatar should remain non-nil after DeepClone")
	}
}

// TestDeepCloneAllScalarsUpdate verifies that AllScalarsUpdate (which contains
// float64 and other optional scalar fields) is correctly deep-cloned.
func TestDeepCloneAllScalarsUpdate(t *testing.T) {
	t.Parallel()

	fSint64 := int64(1)
	fSfixed32 := int32(2)
	fSfixed64 := int64(3)
	fDouble := float64(4.4)
	fFixed32 := uint32(5)
	fFixed64 := uint64(6)
	fUint32 := uint32(7)
	fUint64 := uint64(8)
	fFloat := float32(9.9)

	orig := &dao.AllScalarsUpdate{
		FSint32:   10,
		FSint64:   &fSint64,
		FSfixed32: &fSfixed32,
		FSfixed64: &fSfixed64,
		FDouble:   &fDouble,
		FFixed32:  &fFixed32,
		FFixed64:  &fFixed64,
		FUint32:   &fUint32,
		FUint64:   &fUint64,
		FFloat:    &fFloat,
		FBytes:    []byte{1, 2, 3},
	}

	clone := orig.DeepClone()

	if clone.FSint64 == orig.FSint64 {
		t.Error("FSint64 pointer shared")
	}
	if clone.FSfixed32 == orig.FSfixed32 {
		t.Error("FSfixed32 pointer shared")
	}
	if clone.FSfixed64 == orig.FSfixed64 {
		t.Error("FSfixed64 pointer shared")
	}
	if clone.FDouble == orig.FDouble {
		t.Error("FDouble pointer shared")
	}
	if clone.FFixed32 == orig.FFixed32 {
		t.Error("FFixed32 pointer shared")
	}
	if clone.FFixed64 == orig.FFixed64 {
		t.Error("FFixed64 pointer shared")
	}
	if clone.FUint32 == orig.FUint32 {
		t.Error("FUint32 pointer shared")
	}
	if clone.FUint64 == orig.FUint64 {
		t.Error("FUint64 pointer shared")
	}
	if clone.FFloat == orig.FFloat {
		t.Error("FFloat pointer shared")
	}
	if &clone.FBytes[0] == &orig.FBytes[0] {
		t.Error("FBytes shares backing array")
	}

	*clone.FDouble = 99.9
	if *orig.FDouble != 4.4 {
		t.Errorf("mutating clone.FDouble affected original")
	}
}

// TestDeepCloneAllRepeatedEmptySlice verifies that non-nil empty slices in AllRepeated
// remain non-nil after cloning, including the nested [][]byte type.
func TestDeepCloneAllRepeatedEmptySlice(t *testing.T) {
	t.Parallel()

	orig := &dao.AllRepeated{
		RSint32:   []int32{},
		RSfixed32: []int32{},
		RDouble:   []float64{},
		RBytes:    [][]byte{},
		RMessage:  []*dao.Address{},
		REnum:     []dao.Status{},
	}

	clone := orig.DeepClone()

	if clone.RSint32 == nil {
		t.Error("empty RSint32 should remain non-nil after DeepClone")
	}
	if clone.RSfixed32 == nil {
		t.Error("empty RSfixed32 should remain non-nil after DeepClone")
	}
	if clone.RDouble == nil {
		t.Error("empty RDouble should remain non-nil after DeepClone")
	}
	if clone.RBytes == nil {
		t.Error("empty RBytes should remain non-nil after DeepClone")
	}
	if clone.RMessage == nil {
		t.Error("empty RMessage should remain non-nil after DeepClone")
	}
	if clone.REnum == nil {
		t.Error("empty REnum should remain non-nil after DeepClone")
	}
}

// TestDeepCloneAllValidate verifies that AllValidate is correctly deep-cloned,
// covering optional enum pointer (OStatus), singular bytes (BMinmax), and
// repeated scalar (RItems) — a combination not covered by other tests.
func TestDeepCloneAllValidate(t *testing.T) {
	t.Parallel()

	oStatus := dao.Status_STATUS_ACTIVE

	orig := &dao.AllValidate{
		OStatus: &oStatus,
		BMinmax: []byte{1, 2, 3},
		RItems:  []int32{10, 20, 30},
	}

	clone := orig.DeepClone()

	// Optional enum pointer must be independent.
	if clone.OStatus == orig.OStatus {
		t.Error("OStatus pointer shared between clone and original")
	}
	if *clone.OStatus != dao.Status_STATUS_ACTIVE {
		t.Errorf("OStatus value mismatch: got %v", *clone.OStatus)
	}

	// Singular bytes must be independently copied.
	if &clone.BMinmax[0] == &orig.BMinmax[0] {
		t.Error("BMinmax shares backing array with original")
	}

	// Repeated scalar must be independently copied.
	if &clone.RItems[0] == &orig.RItems[0] {
		t.Error("RItems shares backing array with original")
	}

	// Mutate clone; original must be unchanged.
	*clone.OStatus = dao.Status_STATUS_INACTIVE
	if *orig.OStatus != dao.Status_STATUS_ACTIVE {
		t.Errorf("mutating clone.OStatus affected original")
	}
	clone.BMinmax[0] = 99
	if orig.BMinmax[0] != 1 {
		t.Errorf("mutating clone.BMinmax[0] affected original")
	}
	clone.RItems[0] = 99
	if orig.RItems[0] != 10 {
		t.Errorf("mutating clone.RItems[0] affected original")
	}
}
