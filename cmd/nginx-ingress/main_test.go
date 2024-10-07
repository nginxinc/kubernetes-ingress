package main

import (
	"bytes"
	"regexp"
	"testing"

	nl "github.com/nginxinc/kubernetes-ingress/internal/logger"
	"github.com/nginxinc/kubernetes-ingress/internal/logger/levels"
)

func TestLogFormats(t *testing.T) {
	testCases := []struct {
		name   string
		format string
		wantre string
	}{
		{
			name:   "glog format message",
			format: "glog",
			wantre: `^I\d{8}\s\d+:\d+:\d+.\d{6}\s+\d+\s\w+\.go:\d+\]\s.*\s$`,
		},
		{
			name:   "json format message",
			format: "json",
			wantre: `^{"time":"\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.\d+.*","level":"INFO","msg":".*}`,
		},
		{
			name:   "text format message",
			format: "text",
			wantre: `^time=\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.\d+.*level=\w+\smsg=\w+`,
		},
	}
	t.Parallel()
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			ctx := initLogger(tc.format, levels.LevelInfo, &buf)
			l := nl.LoggerFromContext(ctx)
			l.Log(ctx, levels.LevelInfo, "test")
			got := buf.String()
			re := regexp.MustCompile(tc.wantre)
			if !re.MatchString(got) {
				t.Errorf("\ngot:\n%q\nwant:\n%q", got, tc.wantre)
			}
		})
	}
}
