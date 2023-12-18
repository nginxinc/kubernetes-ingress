package nginx

import (
	"testing"
)

func TestNginxVersionParsing(t *testing.T) {
	t.Parallel()
	type testCase struct {
		input    string
		expected Version
	}
	testCases := []testCase{
		{
			input: "nginx version: nginx/1.25.1 (nginx-plus-r30-p1)",
			expected: Version{
				raw:    "nginx version: nginx/1.25.1 (nginx-plus-r30-p1)",
				OSS:    "1.25.1",
				IsPlus: true,
				Plus:   "nginx-plus-r30-p1",
			},
		},
		{
			input: "nginx version: nginx/1.25.3 (nginx-plus-r31)",
			expected: Version{
				raw:    "nginx version: nginx/1.25.3 (nginx-plus-r31)",
				OSS:    "1.25.3",
				IsPlus: true,
				Plus:   "nginx-plus-r31",
			},
		},
		{
			input: "nginx version: nginx/1.25.0",
			expected: Version{
				raw:    "nginx version: nginx/1.25.0",
				OSS:    "1.25.0",
				IsPlus: false,
				Plus:   "",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			actual := parseNginxVersion(tc.input)
			if actual != tc.expected {
				t.Errorf("expected %v but got %v", tc.expected, actual)
			}
		})
	}
}
