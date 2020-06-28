/*
   Copyright © 2019, 2020 M.Watermann, 10247 Berlin, Germany
                   All rights reserved
               EMail : <support@mwat.de>
*/

package apachelogger

//lint:file-ignore ST1017 – I prefer Yoda conditions

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

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
	type args struct {
		aURL *url.URL
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
		{" 1", args{&u1}, w1},
		{" 2", args{&u2}, w2},
		{" 3", args{&u3}, w3},
		{" 4", args{&u4}, w4},
		{" 5", args{&u5}, w5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getPath(tt.args.aURL); got != tt.want {
				t.Errorf("getPath() = %v, want %v", got, tt.want)
			}
		})
	}
} // Test_getPath()

func Test_getRemote(t *testing.T) {
	req1 := httptest.NewRequest("GET", "/", nil)
	req1.RemoteAddr = "127.0.0.1"
	req2 := httptest.NewRequest("GET", "/", nil)
	req2.RemoteAddr = "192.168.1.234:1234"
	req3 := httptest.NewRequest("GET", "/", nil)
	req3.RemoteAddr = "[2001:9876:5432:abcd:1234:5678:90ab:cdef]"
	req4 := httptest.NewRequest("GET", "/", nil)
	req4.RemoteAddr = "[2001:4567:9876:abcd:1234:5678:90ab:cdef]:6789"
	type args struct {
		aRequest *http.Request
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
		{" 1", args{req1}, "127.0.0.0"},
		{" 2", args{req2}, "192.168.1.0"},
		{" 3", args{req3}, "2001:9876:5432:abcd:0:0:0:0"},
		{" 4", args{req4}, "2001:4567:9876:abcd:0:0:0:0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getRemote(tt.args.aRequest); got != tt.want {
				t.Errorf("getRemote() = %v, want %v", got, tt.want)
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
	go goWriteLog("/dev/stdout", alAccessQueue)
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		for i := 1; i < 100; i++ {
			Log("Benchmark_goWrite", strings.Repeat(fmt.Sprintf("%02d%02d ", n, i), 20))
		}
	}
} // Benchmark_goWrite()

func Benchmark_goCustomLog(b *testing.B) {
	go goWriteLog("/dev/stderr", alErrorQueue)
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		for i := 1; i < 100; i++ {
			go goCustomLog("Benchmark_goCustomLog", fmt.Sprintf("%02d%02d", n, i), `TEST`, time.Now(), alErrorQueue)
		}
	}
} // Benchmark_goCustomLog()
