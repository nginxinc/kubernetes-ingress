package metrics

import (
	"errors"
	"net"

	"github.com/golang/glog"

	"github.com/nginxinc/kubernetes-ingress/internal/metrics/collectors"
)

// SyslogListener is an interface for syslog metrics listener
// that reads syslog metrics logged by nginx
type SyslogListener interface {
	Run()
	Stop()
}

// LatencyMetricsListener implements the SyslogListener interface
type LatencyMetricsListener struct {
	conn      *net.UnixConn
	addr      string
	collector collectors.LatencyCollector
}

// NewLatencyMetricsListener returns a LatencyMetricsListener that listens over a unix socket
// for syslog messages from nginx.
func NewLatencyMetricsListener(sockPath string, c collectors.LatencyCollector) SyslogListener {
	glog.Infof("Starting latency metrics server listening on: %s", sockPath)
	conn, err := net.ListenUnixgram("unixgram", &net.UnixAddr{
		Name: sockPath,
		Net:  "unixgram",
	})
	if err != nil {
		glog.Errorf("Failed to create latency metrics listener: %v. Latency metrics will not be collected.", err)
		return NewSyslogFakeServer()
	}
	return &LatencyMetricsListener{conn: conn, addr: sockPath, collector: c}
}

// Run reads from the unix connection until an unrecoverable error occurs or the connection is closed.
func (l LatencyMetricsListener) Run() {
	buffer := make([]byte, 1024)
	for {
		n, err := l.conn.Read(buffer)
		if err != nil {
			if !isErrorRecoverable(err) {
				glog.Info("Stopping latency metrics listener")
				return
			}
		}
		go l.collector.RecordLatency(string(buffer[:n]))
	}
}

// Stop closes the unix connection of the listener.
func (l LatencyMetricsListener) Stop() {
	err := l.conn.Close()
	if err != nil {
		glog.Errorf("error closing latency metrics unix connection: %v", err)
	}
}

func isErrorRecoverable(err error) bool {
	var nerr *net.OpError
	return errors.As(err, &nerr) && nerr.Temporary()
}

// SyslogFakeListener is a fake implementation of the SyslogListener interface
type SyslogFakeListener struct{}

// NewSyslogFakeServer returns a SyslogFakeListener
func NewSyslogFakeServer() *SyslogFakeListener {
	return &SyslogFakeListener{}
}

// Run is a fake implementation of SyslogListener Run
func (s SyslogFakeListener) Run() {}

// Stop is a fake implementation of SyslogListener Stop
func (s SyslogFakeListener) Stop() {}
