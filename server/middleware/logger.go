// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package middleware

import (
	"net/http"
	"time"

	"github.com/golang/glog"
)

var (
	green   = string([]byte{27, 91, 57, 55, 59, 52, 50, 109})
	white   = string([]byte{27, 91, 57, 48, 59, 52, 55, 109})
	yellow  = string([]byte{27, 91, 57, 55, 59, 52, 51, 109})
	red     = string([]byte{27, 91, 57, 55, 59, 52, 49, 109})
	blue    = string([]byte{27, 91, 57, 55, 59, 52, 52, 109})
	magenta = string([]byte{27, 91, 57, 55, 59, 52, 53, 109})
	cyan    = string([]byte{27, 91, 57, 55, 59, 52, 54, 109})
	reset   = string([]byte{27, 91, 48, 109})
)

type RWWrapper struct {
	http.ResponseWriter
	http.Hijacker
	status  int
	written bool
}

func (r *RWWrapper) saveStatus(status int, warn bool) {
	if !r.written {
		r.status = status
		r.written = true
	} else if warn {
		glog.Errorf("double header write (previous: %d, new: %d)", r.status, status)
	}
}

func (r *RWWrapper) WriteHeader(status int) {
	r.saveStatus(status, true)
	r.ResponseWriter.WriteHeader(status)
}

func (r *RWWrapper) Write(p []byte) (int, error) {
	r.saveStatus(http.StatusOK, false)
	return r.ResponseWriter.Write(p)
}

func MakeLogger() func(inner http.Handler) http.Handler {
	return func(inner http.Handler) http.Handler {
		mw := func(w http.ResponseWriter, r *http.Request) {
			// Start timer
			start := time.Now()
			path := r.URL.Path
			if r.URL.RawQuery != "" {
				path += "?" + r.URL.RawQuery
			}

			rwwrapper := &RWWrapper{
				ResponseWriter: w,
				Hijacker:       w.(http.Hijacker),
			}

			// Process request
			inner.ServeHTTP(rwwrapper, r)

			// Stop timer
			end := time.Now()
			latency := end.Sub(start)

			clientIP := r.RemoteAddr
			if ips, ok := r.Header["X-Real-Ip"]; ok {
				if len(ips) > 0 {
					clientIP = ips[0]
				}
			}
			method := r.Method
			statusCode := rwwrapper.status
			statusColor := colorForStatus(statusCode)
			methodColor := colorForMethod(method)

			logf := getLogf(statusCode)

			logf("%v |%s %3d %s| %13v | %s |%s  %s %-7s %s",
				//end.Format("2006/01/02 - 15:04:05"),
				end.Format("02.01.2006"),
				statusColor, statusCode, reset,
				latency,
				clientIP,
				methodColor, reset, method,
				path,
			)

			glog.Flush()
		}
		return MkMiddleware(mw)
	}
}

func colorForStatus(code int) string {
	switch {
	case code >= 200 && code < 300:
		return green
	case code >= 300 && code < 400:
		return white
	case code >= 400 && code < 500:
		return yellow
	default:
		return red
	}
}

func getLogf(code int) func(format string, args ...interface{}) {
	switch {
	case code >= 200 && code < 300:
		return glog.Infof
	case code >= 300 && code < 400:
		return glog.Warningf
	case code >= 400 && code < 500:
		return glog.Warningf
	default:
		return glog.Errorf
	}
}

func colorForMethod(method string) string {
	switch method {
	case "GET":
		return blue
	case "POST":
		return cyan
	case "PUT":
		return yellow
	case "DELETE":
		return red
	case "PATCH":
		return green
	case "HEAD":
		return magenta
	case "OPTIONS":
		return white
	default:
		return reset
	}
}
