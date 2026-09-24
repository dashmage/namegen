package gen

import "testing"

func BenchmarkRandomName(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		RandomName(5)
	}
}

func BenchmarkAvgLogProb(b *testing.B) {
	model, err := loadDefaultModel()
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		model.AvgLogProb("lora")
	}
}
