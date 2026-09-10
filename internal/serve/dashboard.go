// This file used to contain DashboardHTML as a big string constant.
// That string is now embedded from web/index.html via the //go:embed directive
// in server.go, which is cleaner and lets the HTML be edited as a proper file.
//
// This file is intentionally kept to document the change, but it no longer
// exports anything. The embed happens in server.go.
package serve
