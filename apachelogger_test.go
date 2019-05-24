package apachelogger

import (
	"fmt"
	"net/url"
	"strings"
	"testing"
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
	type args struct {
		aAddress string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
		{" 1", args{"127.0.0.1"}, "127.0.0.0"},
		{" 2", args{"192.168.1.234:1234"}, "192.168.1.0"},
		{" 3", args{"[2001:9876:5432:abcd:1234:5678:90ab:cdef]"}, "2001:9876:5432:abcd:0:0:0:0"},
		{" 4", args{"[2001:4567:9876:abcd:1234:5678:90ab:cdef]:6789"}, "2001:4567:9876:abcd:0:0:0:0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getRemote(tt.args.aAddress); got != tt.want {
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
	var s string
	quit := make(chan bool, 2)
	wchan := make(chan string, 64)
	go goWrite("/dev/stderr", wchan, quit)
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		for i := 1; i < 100; i++ {
			s = fmt.Sprintf("%02d ", i)
			wchan <- strings.Repeat(s, 40) + "\n"
		}
	}
	quit <- true
} // Benchmark_goWrite()
