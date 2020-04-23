/*
Package workload provides a Workload API Client implementation.
It allows a workload to get automatically rotated X.509 certificates from a
Workload API Server.
The watcher interface must be implemented to get notifications about SVIDs
rotation and errors.

A full example is available at:
https://github.com/spiffe/go-spiffe/tree/master/examples/svid-watcher
*/
package workload
