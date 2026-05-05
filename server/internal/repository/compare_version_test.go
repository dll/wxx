package repository

import "testing"

func TestCompareVersion(t *testing.T) {
	tests := []struct {
		name string
		v1   string
		v2   string
		want int
	}{
		{"相同版本三部分", "1.0.0", "1.0.0", 0},
		{"相同版本两部分", "2.0", "2.0", 0},
		{"相同版本一部分", "1", "1", 0},
		{"高主版本", "2.0.0", "1.0.0", 1},
		{"低主版本", "1.0.0", "2.0.0", -1},
		{"高次版本", "1.2.0", "1.1.0", 1},
		{"低次版本", "1.0.0", "1.1.0", -1},
		{"高补丁版本", "1.0.3", "1.0.1", 1},
		{"低补丁版本", "1.0.0", "1.0.5", -1},
		{"两部分vs三部分相等", "1.0", "1.0.0", 0},
		{"一部分vs三部分相等", "3", "3.0.0", 0},
		{"大版本号比较", "10.0.0", "2.0.0", 1},
		{"无效版本v1", "abc", "1.0.0", 0},
		{"无效版本v2", "1.0.0", "xyz", 0},
		{"两个无效版本", "abc", "xyz", 0},
		{"空字符串v1", "", "1.0.0", 0},
		{"空字符串v2", "1.0.0", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := compareVersion(tt.v1, tt.v2)
			if got != tt.want {
				t.Errorf("compareVersion(%q, %q) = %d, want %d", tt.v1, tt.v2, got, tt.want)
			}
		})
	}
}
