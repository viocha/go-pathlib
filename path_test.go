package path

import "testing"

func TestFromURL(t *testing.T) {
	// 测试从 file URL 创建 Path
	testCases := []struct {
		url      string
		expected string
	}{
		{"file:///C:/path/to/file.txt", `C:\path\to\file.txt`},
	}

	for _, tc := range testCases {
		t.Run(tc.url, func(t *testing.T) {
			path, err := FromURL(tc.url)
			if err != nil {
				t.Errorf("Failed to create path from URL: %v", err)
				return
			}
			if path.String() != tc.expected {
				t.Errorf("Expected path %s, got %s", tc.expected, path.String())
			}
		})
	}
}
