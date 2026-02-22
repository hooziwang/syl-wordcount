package app

import (
	"testing"

	"syl-wordcount/internal/config"
	"syl-wordcount/internal/textutil"
)

func TestEvaluateRulesMaxLineWidth(t *testing.T) {
	text := "123456\n"
	m := textutil.ComputeMetrics(text)
	fc := FileContent{Path: "/tmp/a.txt", Data: []byte(text), Text: text, Metrics: m}
	max := 4
	vs, errs := EvaluateRules(fc, config.Rules{MaxLineWidth: &max})
	if len(errs) != 0 {
		t.Fatalf("unexpected errs: %v", errs)
	}
	if len(vs) == 0 {
		t.Fatalf("expected violation")
	}
	if vs[0].OverflowStartColumn != 5 {
		t.Fatalf("unexpected overflow column: %d", vs[0].OverflowStartColumn)
	}
}
