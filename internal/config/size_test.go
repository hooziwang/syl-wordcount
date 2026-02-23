package config

import "testing"

func TestParseSizeToBytes(t *testing.T) {
	cases := []struct {
		in   string
		want int64
	}{
		{"", 0},
		{"100", 100},
		{"1KB", 1024},
		{"2MB", 2 * 1024 * 1024},
	}
	for _, c := range cases {
		got, err := ParseSizeToBytes(c.in)
		if err != nil {
			t.Fatalf("parse %s failed: %v", c.in, err)
		}
		if got != c.want {
			t.Fatalf("parse %s got %d want %d", c.in, got, c.want)
		}
	}
	if _, err := ParseSizeToBytes("A1"); err == nil {
		t.Fatalf("expected error")
	}
}
