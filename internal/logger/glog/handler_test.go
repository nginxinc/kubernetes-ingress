package glog

import (
	"bytes"
	"log/slog"
	"regexp"
	"testing"
)

func TestGlogFormat(t *testing.T) {
	var buf bytes.Buffer
	l := slog.New(New(&buf, nil))
	l.Info("hello")
	got := buf.String()
	wantre := `^\w\d{8}\s\d+:\d+:\d+.\d{6}\s+\d+\s\w+\.go:\d+\]\s.*\s$`
	re := regexp.MustCompile(wantre)
	if !re.MatchString(got) {
		t.Errorf("\ngot:\n%q\nwant:\n%q", got, wantre)
	}

	buf.Reset()
	l.Debug("test")
	if got := buf.Len(); got != 0 {
		t.Errorf("got buf.Len() = %d, want 0", got)
	}
}
