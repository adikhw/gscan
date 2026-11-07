package main

import "testing"

var benchmarkStatus = "## feature/login...origin/feature/login [ahead 2, behind 3]\n" +
	" M main.go\nA  added.go\n?? notes.txt\n?? scratch.txt\n"

func BenchmarkParseStatus(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = parseStatus(benchmarkStatus)
	}
}

func BenchmarkSummarize(b *testing.B) {
	st := parseStatus(benchmarkStatus)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = summarize(st)
	}
}
