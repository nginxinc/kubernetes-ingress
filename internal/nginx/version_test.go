package nginx_test

import (
	"testing"

	"github.com/nginxinc/kubernetes-ingress/internal/nginx"
)

func TestNginxVersionParsing(t *testing.T) {
	t.Parallel()
	type testCase struct {
		input    string
		expected nginx.Version
	}
	testCases := []testCase{
		{
			input: "nginx version: nginx/1.25.1 (nginx-plus-r30-p1)",
			expected: nginx.Version{
				Raw:    "nginx version: nginx/1.25.1 (nginx-plus-r30-p1)",
				OSS:    "1.25.1",
				IsPlus: true,
				Plus:   "nginx-plus-r30-p1",
			},
		},
		{
			input: "nginx version: nginx/1.25.3 (nginx-plus-r31)",
			expected: nginx.Version{
				Raw:    "nginx version: nginx/1.25.3 (nginx-plus-r31)",
				OSS:    "1.25.3",
				IsPlus: true,
				Plus:   "nginx-plus-r31",
			},
		},
		{
			input: "nginx version: nginx/1.25.0",
			expected: nginx.Version{
				Raw:    "nginx version: nginx/1.25.0",
				OSS:    "1.25.0",
				IsPlus: false,
				Plus:   "",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			actual := nginx.NewVersion(tc.input)
			if actual != tc.expected {
				t.Errorf("expected %v but got %v", tc.expected, actual)
			}
		})
	}
}

func TestNginxVersionFormat(t *testing.T) {
	t.Parallel()

	tt := []struct {
		input string
		want  string
	}{
		{
			input: "nginx version: nginx/1.25.1 (nginx-plus-r30-p1)",
			want:  "1.25.1-nginx-plus-r30-p1",
		},
		{
			input: "nginx version: nginx/1.25.3 (nginx-plus-r31)",
			want:  "1.25.3-nginx-plus-r31",
		},
		{
			input: "nginx version: nginx/1.25.0",
			want:  "1.25.0",
		},
	}
	for _, tc := range tt {
		t.Run(tc.input, func(t *testing.T) {
			v := nginx.NewVersion(tc.input)
			got := v.Format()
			if got != tc.want {
				t.Errorf("want %q but got %q", tc.want, got)
			}
		})
	}
}

func TestNginxVersionPlusGreaterThanOrEqualTo(t *testing.T) {
	t.Parallel()
	type testCase struct {
		version  nginx.Version
		input    string
		expected bool
	}
	testCases := []testCase{
		{
			version:  nginx.NewVersion("nginx version: nginx/1.25.1 (nginx-plus-r30-p1)"),
			input:    "nginx-plus-r30-p1",
			expected: true,
		},
		{
			version:  nginx.NewVersion("nginx version: nginx/1.25.1 (nginx-plus-r30)"),
			input:    "nginx-plus-r30",
			expected: true,
		},
		{
			version:  nginx.NewVersion("nginx version: nginx/1.25.1 (nginx-plus-r30-p1)"),
			input:    "nginx-plus-r30",
			expected: true,
		},
		{
			version:  nginx.NewVersion("nginx version: nginx/1.25.1 (nginx-plus-r30)"),
			input:    "nginx-plus-r30-p1",
			expected: false,
		},
		{
			version:  nginx.NewVersion("nginx version: nginx/1.25.1"),
			input:    "nginx-plus-r30-p1",
			expected: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			actual, _ := tc.version.PlusGreaterThanOrEqualTo(tc.input)
			if actual != tc.expected {
				t.Errorf("expected %v but got %v", tc.expected, actual)
			}
		})
	}
}
