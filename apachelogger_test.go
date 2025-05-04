/*
Copyright © 2019, 2025  M.Watermann, 10247 Berlin, Germany

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
	for _, tc := range tests {
		alLastLoggingDate = tc.prevTime
		t.Run(tc.name, func(t *testing.T) {
			if got := compareDayStamps(); got != tc.want {
				t.Errorf("%q: Test_compareDayStamps() = %v, want %v",
					tc.name, got, tc.want)
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
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := getPath(tc.url); got != tc.want {
				t.Errorf("%q: Test_getPath() = %v, want %v",
					tc.name, got, tc.want)
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
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if gotRReferrer := getReferrer(tc.header); gotRReferrer != tc.wantRReferrer {
				t.Errorf("%q: getReferrer() = %q,\nwant %q",
					tc.name, gotRReferrer, tc.wantRReferrer)
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
	req8 := httptest.NewRequest("GET", "/", nil)
	req8.RemoteAddr = "127.0.0.1:1234"

	type tArgs struct {
		aRequest *http.Request
		aStatus  int
	}
	tests := []struct {
		name string
		args tArgs
		want string
	}{
		// TODO: Add test cases.
		{" 1", tArgs{req1, 200}, "127.0.0.0"},
		{" 2", tArgs{req2, 301}, "192.168.1.0"},
		{" 3", tArgs{req2, 404}, "192.168.1.234"},
		{" 4", tArgs{req4, 200}, "2001:9876:5432:abcd::"},
		{" 5", tArgs{req4, 404}, "2001:9876:5432:abcd:1234:5678:90ab:cdef"},
		{" 6", tArgs{req6, 200}, "2001:4567:9876:abcd::"},
		{" 7", tArgs{req6, 503}, "2001:4567:9876:abcd:1234:5678:90ab:cdef"},
		{" 8", tArgs{req8, 200}, "127.0.0.0"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := getRemote(tc.args.aRequest, tc.args.aStatus); got != tc.want {
				t.Errorf("getRemote() = %v,\nwant %v", got, tc.want)
			}
		})
	}
} // Test_getRemote()

func Test_getRemote2(t *testing.T) {
	// Create test requests with different remote addresses
	req1, _ := http.NewRequest("GET", "http://example.com", nil)
	req1.RemoteAddr = "192.168.1.1:8080"

	req2, _ := http.NewRequest("GET", "http://example.com", nil)
	req2.RemoteAddr = "[2001:db8::1]:8080"

	req3, _ := http.NewRequest("GET", "http://example.com", nil)
	req3.RemoteAddr = "192.168.1.1" // No port

	req4, _ := http.NewRequest("GET", "http://example.com", nil)
	req4.RemoteAddr = "[2001:db8::1]" // IPv6 with brackets, no port

	req5, _ := http.NewRequest("GET", "http://example.com", nil)
	req5.RemoteAddr = "192.168.1.1:8080"
	req5.Header.Add("X-Forwarded-For", "10.0.0.1")

	// Save original anonymization settings
	origAnonymiseURLs := AnonymiseURLs
	origAnonymiseErrors := AnonymiseErrors
	defer func() {
		// Restore original settings
		AnonymiseURLs = origAnonymiseURLs
		AnonymiseErrors = origAnonymiseErrors
	}()

	tests := []struct {
		name           string
		request        *http.Request
		status         int
		anonymiseURLs  bool
		anonymiseErrs  bool
		expectedResult string
	}{
		{"IPv4 with port", req1, 200, false, false, "192.168.1.1"},
		{"IPv6 with port", req2, 200, false, false, "2001:db8::1"},
		{"IPv4 no port", req3, 200, false, false, "192.168.1.1"},
		{"IPv6 brackets no port", req4, 200, false, false, "2001:db8::1"},
		{"With X-Forwarded-For", req5, 200, false, false, "10.0.0.1"},
		{"Anonymised IPv4", req1, 200, true, false, "192.168.1.0"},
		{"Anonymised IPv6", req2, 200, true, false, "2001:db8::"},
		{"Error not anonymised", req1, 404, true, true, "192.168.1.0"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Set anonymization flags for this test
			AnonymiseURLs = tc.anonymiseURLs
			AnonymiseErrors = tc.anonymiseErrs

			got := getRemote(tc.request, tc.status)
			if got != tc.expectedResult {
				t.Errorf("getRemote() = %q, want %q",
					got, tc.expectedResult)
			}
		})
	}
} // Test_getRemote()

func Test_getUsername(t *testing.T) {
	var u1, u2 url.URL
	user2 := "user"
	u2.User = url.UserPassword(user2, "pwHash")

	tests := []struct {
		name string
		url  *url.URL
		want string
	}{
		// TODO: Add test cases.
		{" 1", &u1, "-"},
		{" 2", &u2, user2},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := getUsername(tc.url); got != tc.want {
				t.Errorf("getUsername() = %v, want %v", got, tc.want)
			}
		})
	}
} // Test_getUsername()

func Benchmark_goDoLogWrite(b *testing.B) {
	runtime.GOMAXPROCS(1)
	go goDoLogWrite("/dev/stdout", alAccessQueue)
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		for i := 1; i < 9; i++ {
			Log("Benchmark_goWrite", strings.Repeat(fmt.Sprintf("%02d%02d ", n, i), 20))
		}
	}
} // Benchmark_goDoLogWrite()

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
