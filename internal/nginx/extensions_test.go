package nginx

import "testing"

func TestParseLBMethod(t *testing.T) {
	var testsWithValidInput = []struct {
		input    string
		expected string
	}{
		{"least_conn", "least_conn"},
		{"round_robin", ""},
		{"ip_hash", "ip_hash"},
		{"hash $request_id", "hash $request_id"},
		{"hash $request_id consistent", "hash $request_id consistent"},
	}

	var invalidInput = []string{
		"",
		"blabla",
		"least_time header",
		"hash123",
		"hash $request_id conwrongspelling",
	}

	for _, test := range testsWithValidInput {
		result, err := ParseLBMethod(test.input)
		if err != nil {
			t.Errorf("TestParseLBMethod(%q) returned an error for valid input", test.input)
		}

		if result != test.expected {
			t.Errorf("TestParseLBMethod(%q) returned %q expected %q", test.input, result, test.expected)
		}
	}

	for _, input := range invalidInput {
		_, err := ParseLBMethod(input)
		if err == nil {
			t.Errorf("TestParseLBMethod(%q) does not return an error for invalid input", input)
		}
	}
}

func TestParseLBMethodForPlus(t *testing.T) {
	var testsWithValidInput = []struct {
		input    string
		expected string
	}{
		{"least_conn", "least_conn"},
		{"round_robin", ""},
		{"ip_hash", "ip_hash"},
		{"hash $request_id", "hash $request_id"},
		{"least_time header", "least_time header"},
		{"least_time last_byte", "least_time last_byte"},
		{"least_time header inflight", "least_time header inflight"},
		{"least_time last_byte inflight", "least_time last_byte inflight"},
	}

	var invalidInput = []string{
		"",
		"blabla",
		"hash123",
		"least_time inflight header",
	}

	for _, test := range testsWithValidInput {
		result, err := ParseLBMethodForPlus(test.input)
		if err != nil {
			t.Errorf("TestParseLBMethod(%q) returned an error for valid input", test.input)
		}

		if result != test.expected {
			t.Errorf("TestParseLBMethod(%q) returned %q expected %q", test.input, result, test.expected)
		}
	}

	for _, input := range invalidInput {
		_, err := ParseLBMethodForPlus(input)
		if err == nil {
			t.Errorf("TestParseLBMethod(%q) does not return an error for invalid input", input)
		}
	}
}


func TestParseSlowStart(t *testing.T) {
	var testsWithValidInput = []string{"1", "1m10s", "11 11", "5m 30s", "1s", "100m", "5w", "15m", "11M", "3h", "100y", "600"}
	var invalidInput = []string{"ss", "rM", "m0m", "s1s", "-5s", "", "1L"}
	for _, test := range testsWithValidInput {
		result, err := ParseSlowStart(test)
		if err != nil {
			t.Errorf("TestParseSlowStart(%q) returned an error for valid input", test)
		}
		if test != result {
			t.Errorf("TestParseSlowStart(%q) returned %q expected %q", test, result, test)
		}
	}
	for _, test := range invalidInput {
		result, err := ParseSlowStart(test)
		if err == nil {
			t.Errorf("TestParseSlowStart(%q) didn't return error. Returned: %q", test, result)
		}
	}
}