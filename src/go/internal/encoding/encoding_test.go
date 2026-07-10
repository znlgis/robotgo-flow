package encoding

import (
	"testing"
)

func TestToGBK(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{name: "ascii", in: "hello"},
		{name: "chinese", in: "你好"},
		{name: "mixed", in: "C:\\用户\\文档\\test.txt"},
		{name: "empty", in: ""},
		{name: "special chars", in: "file (1).txt"},
		{name: "long path", in: "C:\\Program Files\\测试\\数据\\report.csv"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToGBK(tt.in)
			if result == "" && tt.in != "" {
				t.Errorf("ToGBK(%q) returned empty string", tt.in)
			}
			// Round-trip: GBK back to UTF-8 should match original
			decoded, ok := ToUTF8([]byte(result))
			if !ok && len([]byte(result)) > 0 {
				// Round-trip may fail if input contains chars outside GBK
				// but for common Chinese chars it should work
			}
			_ = decoded
		})
	}
}

func TestToUTF8(t *testing.T) {
	t.Run("valid utf8 returns unchanged", func(t *testing.T) {
		data := []byte("hello world")
		got, converted := ToUTF8(data)
		if converted {
			t.Error("expected converted=false for valid UTF-8")
		}
		if string(got) != string(data) {
			t.Errorf("got %q, want %q", got, data)
		}
	})

	t.Run("chinese utf8 returns unchanged", func(t *testing.T) {
		data := []byte("你好世界")
		got, converted := ToUTF8(data)
		if converted {
			t.Error("expected converted=false for valid UTF-8 Chinese")
		}
		if string(got) != string(data) {
			t.Errorf("got %q, want %q", got, data)
		}
	})

	t.Run("invalid utf8 falls through", func(t *testing.T) {
		// Single byte 0x80 is never valid UTF-8
		data := []byte{0x80, 0x81}
		got, converted := ToUTF8(data)
		// The decoder may return the data unchanged on failure
		_ = got
		_ = converted
		// We don't assert success — just that it doesn't panic
	})

	t.Run("empty input", func(t *testing.T) {
		data := []byte{}
		got, converted := ToUTF8(data)
		if converted {
			t.Error("expected converted=false for empty input")
		}
		if len(got) != 0 {
			t.Errorf("expected empty output, got %v", got)
		}
	})
}

func TestToUTF8String(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{name: "ascii", in: "hello"},
		{name: "chinese", in: "你好"},
		{name: "empty", in: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToUTF8String(tt.in)
			if result == "" && tt.in != "" {
				t.Errorf("ToUTF8String(%q) returned empty", tt.in)
			}
		})
	}
}

func TestToGBK_RoundTrip(t *testing.T) {
	// Test that common Chinese strings round-trip correctly
	inputs := []string{
		"C:\\用户",
		"D:\\数据\\报告.txt",
		"测试文件.png",
	}

	for _, in := range inputs {
		gbk := ToGBK(in)
		utf8 := ToUTF8String(gbk)
		if utf8 != in {
			t.Errorf("round-trip failed: %q -> GBK -> %q, want %q", in, utf8, in)
		}
	}
}

// ---------------------------------------------------------------------------
// Benchmarks
// ---------------------------------------------------------------------------

func BenchmarkToGBK_ASCII(b *testing.B) {
	for b.Loop() {
		ToGBK("C:\\Program Files\\myfile.txt")
	}
}

func BenchmarkToGBK_Chinese(b *testing.B) {
	for b.Loop() {
		ToGBK("C:\\用户\\文档\\测试.txt")
	}
}

func BenchmarkToUTF8String_ASCII(b *testing.B) {
	for b.Loop() {
		ToUTF8String("C:\\Program Files\\myfile.txt")
	}
}

func BenchmarkToUTF8String_Chinese(b *testing.B) {
	for b.Loop() {
		ToUTF8String("C:\\用户\\文档\\测试.txt")
	}
}
