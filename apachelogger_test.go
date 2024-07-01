/*
Copyright © 2019, 2024  M.Watermann, 10247 Berlin, Germany

	    All rights reserved
	EMail : <support@mwat.de>
*/
package apachelogger

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"runtime"
	"strings"
	"testing"
	"time"
)

//lint:file-ignore ST1017 – I prefer Yoda conditions

func Test_compareDayStamps(t *testing.T) {
	ll1 := time.Now()
	ll2 := ll1.Add(-1 * (24 * time.Hour))
	ll3 := ll1.Add((24 * time.Hour))

	tests := []struct {
		name     string
		prevTime time.Time
		want     bool
	}{
		{"1", ll1, false},
		{"2", ll2, true},
		{"4", ll3, true},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		alLastLoggingDate = tt.prevTime
		t.Run(tt.name, func(t *testing.T) {
			if got := compareDayStamps(); got != tt.want {
				t.Errorf("%q: Test_compareDayStamps() = %v, want %v",
					tt.name, got, tt.want)
			}
		})
	}
} // Test_compareDayStamps

func Test_getPath(t *testing.T) {
	var u1, u2, u3, u4, u5 url.URL
	f := "id"
	p := "/page.html"
	q := "key=val"

	w1 := ""
	u2.Path = p
	w2 := p
	u3.Path = p
	u3.RawQuery = q
	w3 := u3.Path + "?" + u3.RawQuery
	u4.Path = p
	u4.Fragment = f
	w4 := u4.Path + "#" + u4.Fragment
	u5.Fragment = f
	u5.Path = p
	u5.RawQuery = q
	w5 := u5.Path + "?" + u5.RawQuery + "#" + u5.Fragment

	tests := []struct {
		name string
		url  *url.URL
		want string
	}{
		// TODO: Add test cases.
		{" 1", &u1, w1},
		{" 2", &u2, w2},
		{" 3", &u3, w3},
		{" 4", &u4, w4},
		{" 5", &u5, w5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getPath(tt.url); got != tt.want {
				t.Errorf("%q: Test_getPath() = %v, want %v",
					tt.name, got, tt.want)
			}
		})
	}
} // Test_getPath()

func prepHttpHeader() *http.Header {
	mockHeader := make(http.Header)
	mockHeader.Add("Content-Type", "application/json")
	mockHeader.Add("Authorization", "Bearer token123")
	mockHeader.Add("Custom-Header", "custom-value")

	return &mockHeader
} // prepHttpHeader()

func Test_getReferrer(t *testing.T) {
	mh := prepHttpHeader()
	wr := "-"
	mh1 := prepHttpHeader()
	wr1 := "some URL"
	mh1.Add("Referer", wr1)
	mh2 := prepHttpHeader()
	wr2 := "some other URL"
	mh2.Add("Referrer", wr2)
	mh3 := prepHttpHeader()
	mh3.Add("Referrrer", wr)

	// Create a new http.Header object
	mockHeader := make(http.Header)
	mockHeader.Add("Content-Type", "application/json")
	mockHeader.Add("Authorization", "Bearer token123")
	mockHeader.Add("Custom-Header", "custom-value")

	tests := []struct {
		name          string
		header        *http.Header
		wantRReferrer string
	}{
		{"0", mh, wr},
		{"1", mh1, wr1},
		{"2", mh2, wr2},
		{"3", mh3, wr},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotRReferrer := getReferrer(tt.header); gotRReferrer != tt.wantRReferrer {
				t.Errorf("%q: getReferrer() = %q,\nwant %q",
					tt.name, gotRReferrer, tt.wantRReferrer)
			}
		})
	}
} // Test_getReferrer()

func Test_getRemote(t *testing.T) {
	req1 := httptest.NewRequest("GET", "/", nil)
	req1.RemoteAddr = "127.0.0.1"
	req2 := httptest.NewRequest("GET", "/", nil)
	req2.RemoteAddr = "192.168.1.234:1234"
	req4 := httptest.NewRequest("GET", "/", nil)
	req4.RemoteAddr = "[2001:9876:5432:abcd:1234:5678:90ab:cdef]"
	req6 := httptest.NewRequest("GET", "/", nil)
	req6.RemoteAddr = "[2001:4567:9876:abcd:1234:5678:90ab:cdef]:6789"
	type args struct {
		aRequest *http.Request
		aStatus  int
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
		{" 1", args{req1, 200}, "127.0.0.0"},
		{" 2", args{req2, 301}, "192.168.1.0"},
		{" 3", args{req2, 404}, "192.168.1.234"},
		{" 4", args{req4, 200}, "2001:9876:5432:abcd:0:0:0:0"},
		{" 5", args{req4, 404}, "2001:9876:5432:abcd:1234:5678:90ab:cdef"},
		{" 6", args{req6, 200}, "2001:4567:9876:abcd:0:0:0:0"},
		{" 7", args{req6, 503}, "2001:4567:9876:abcd:1234:5678:90ab:cdef"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getRemote(tt.args.aRequest, tt.args.aStatus); got != tt.want {
				t.Errorf("getRemote() = %v,\nwant %v", got, tt.want)
			}
		})
	}
} // Test_getRemote()

func Test_getUsername(t *testing.T) {
	var u1, u2 url.URL
	user2 := "user"
	u2.User = url.UserPassword(user2, "pwHash")
	type args struct {
		aURL *url.URL
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
		{" 1", args{&u1}, "-"},
		{" 2", args{&u2}, user2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getUsername(tt.args.aURL); got != tt.want {
				t.Errorf("getUsername() = %v, want %v", got, tt.want)
			}
		})
	}
} // Test_getUsername()

func Benchmark_goWrite(b *testing.B) {
	runtime.GOMAXPROCS(1)
	go goDoLogWrite("/dev/stdout", alAccessQueue)
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		for i := 1; i < 9; i++ {
			Log("Benchmark_goWrite", strings.Repeat(fmt.Sprintf("%02d%02d ", n, i), 20))
		}
	}
} // Benchmark_goWrite()

func Benchmark_goCustomLog(b *testing.B) {
	runtime.GOMAXPROCS(1)
	go goDoLogWrite("/dev/stderr", alErrorQueue)
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		for i := 1; i < 9; i++ {
			go goCustomLog("Benchmark_goCustomLog", fmt.Sprintf("%02d%02d", n, i), `TEST`, time.Now(), alErrorQueue)
		}
	}
} // Benchmark_goCustomLog()

/* _EoF_ */
