package compat_test

import (
	"testing"

	"google.golang.org/protobuf/proto"

	"github.com/pinealctx/gcode/testdata/compat/dao"
	"github.com/pinealctx/gcode/testdata/compat/pbgo"
)

// --- Protobuf marshal benchmarks ---

// BenchmarkMarshalBinary_Dao benchmarks our generated MarshalBinary.
func BenchmarkMarshalBinary_Dao(b *testing.B) {
	p := populatedDao()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := p.MarshalBinary(); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMarshalBinary_Pbgo benchmarks protoc-gen-go proto.Marshal.
func BenchmarkMarshalBinary_Pbgo(b *testing.B) {
	p := populatedPbgo()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := proto.Marshal(p); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkUnmarshalBinary_Dao benchmarks our generated UnmarshalBinary.
func BenchmarkUnmarshalBinary_Dao(b *testing.B) {
	wire, _ := populatedDao().MarshalBinary()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var p dao.Person
		if err := p.UnmarshalBinary(wire); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkUnmarshalBinary_Pbgo benchmarks protoc-gen-go proto.Unmarshal.
func BenchmarkUnmarshalBinary_Pbgo(b *testing.B) {
	wire, _ := proto.Marshal(populatedPbgo())
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var p pbgo.Person
		if err := proto.Unmarshal(wire, &p); err != nil {
			b.Fatal(err)
		}
	}
}

// --- Large message protobuf benchmarks ---

func populatedDaoLarge() *dao.Person {
	scores := make([]int32, 1000)
	tags := make([]string, 1000)
	for i := range scores {
		scores[i] = int32(i)
		tags[i] = "tag"
	}
	p := populatedDao()
	p.Scores = scores
	p.Tags = tags
	return p
}

func populatedPbgoLarge() *pbgo.Person {
	scores := make([]int32, 1000)
	tags := make([]string, 1000)
	for i := range scores {
		scores[i] = int32(i)
		tags[i] = "tag"
	}
	p := populatedPbgo()
	p.Scores = scores
	p.Tags = tags
	return p
}

func BenchmarkMarshalBinary_Dao_Large(b *testing.B) {
	p := populatedDaoLarge()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := p.MarshalBinary(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMarshalBinary_Pbgo_Large(b *testing.B) {
	p := populatedPbgoLarge()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := proto.Marshal(p); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUnmarshalBinary_Dao_Large(b *testing.B) {
	wire, _ := populatedDaoLarge().MarshalBinary()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var p dao.Person
		if err := p.UnmarshalBinary(wire); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUnmarshalBinary_Pbgo_Large(b *testing.B) {
	wire, _ := proto.Marshal(populatedPbgoLarge())
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var p pbgo.Person
		if err := proto.Unmarshal(wire, &p); err != nil {
			b.Fatal(err)
		}
	}
}

// --- Validate benchmark ---

// BenchmarkValidate_Person benchmarks the generated Validate() method on a fully-populated Person.
func BenchmarkValidate_Person(b *testing.B) {
	p := populatedDao()
	p.Email = "alice@example.com"
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if err := p.Validate(); err != nil {
			b.Fatal(err)
		}
	}
}
