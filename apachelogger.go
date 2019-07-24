/*
   Copyright © 2019 M.Watermann, 10247 Berlin, Germany
                   All rights reserved
               EMail : <support@mwat.de>
*/

package apachelogger

/*
   This package can be used to add a logfile facility to your
   `Go` web-server.
   The format of the generated logfile entries resemble those of
   the popular Apache web-server.
*/

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sync"
	"time"
)

var (
	// AnonymiseURLs decides wether to anonymise the remote URLs
	// (default: `true`).
	//
	// For legal reasons this variable should always stay `true`.
	AnonymiseURLs = true
)

type (
	// `tLogWriter` embeds a `ResponseWriter` and includes log-to-file.
	tLogWriter struct {
		http.ResponseWriter     // used to construct the HTTP response
		size                int // the size/length of the data sent
		status              int // HTTP status code of the current request
	}
)

// Write writes the data to the connection as part of an HTTP reply.
//
// Part of the `http.ResponseWriter` interface.
func (lw *tLogWriter) Write(aData []byte) (int, error) {
	if 0 == lw.status {
		lw.status = 200
	}
	lw.size = len(aData) // we need this for the logfile

	return lw.ResponseWriter.Write(aData)
} // Write()

// WriteHeader sends an HTTP response header with the provided
// status code.
//
// Part of the `http.ResponseWriter` interface.
func (lw *tLogWriter) WriteHeader(aStatus int) {
	lw.status = aStatus
	lw.ResponseWriter.WriteHeader(aStatus)
} // WriteHeader()

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

// `getPath()` returns the requested path (and CGI query).
func getPath(aURL *url.URL) (rURL string) {
	rURL = aURL.Path
	if 0 < len(aURL.RawQuery) {
		rURL += "?" + aURL.RawQuery
	}
	if 0 < len(aURL.Fragment) {
		rURL += "#" + aURL.Fragment
	}

	return
} // getPath()

// `getReferrer()` returns the request's referrer field
func getReferrer(aHeader *http.Header) (rReferrer string) {
	if rReferrer = aHeader.Get("Referer"); 0 < len(rReferrer) {
		return
	}
	if rReferrer = aHeader.Get("Referrer"); 0 < len(rReferrer) {
		return
	}

	return "-"
} // getReferrer()

var (
	// RegEx to match IPv4 addresses:
	ipv4RE = regexp.MustCompile(`^([0-9]{1,3})\.([0-9]{1,3})\.([0-9]{1,3})\.[0-9]{1,3}`)

	// RegEx to match IPv6 addresses:
	ipv6RE = regexp.MustCompile(`([0-9a-f]{1,4})\:([0-9a-f]{1,4})\:([0-9a-f]{1,4})\:([0-9a-f]{1,4})\:([0-9a-f]{1,4})\:([0-9a-f]{1,4})\:([0-9a-f]{1,4})\:([0-9a-f]{1,4})`)
)

// `getRemote()` reads and anonymises the remote address.
func getRemote(aAddress string) (rAddress string) {
	var err error
	// we neither need nor want the remote port here:
	if rAddress, _, err = net.SplitHostPort(aAddress); err != nil {
		// usually no port available
		rAddress = aAddress
	}
	if !AnonymiseURLs { // Bad Choice!
		return
	}
	if matches := ipv4RE.FindStringSubmatch(rAddress); 3 < len(matches) {
		// anonymise the remote IPv4 address:
		rAddress = fmt.Sprintf("%s.%s.%s.0",
			matches[1], matches[2], matches[3])
	} else if matches := ipv6RE.FindStringSubmatch(rAddress); 8 < len(matches) {
		// anonymise the remote IPv6 address:
		rAddress = fmt.Sprintf("%s:%s:%s:%s:0:0:0:0",
			matches[1], matches[2], matches[3], matches[4])
	}

	return
} // getRemote()

// `getUsername()` returns the request's username (if any).
func getUsername(aURL *url.URL) (rUser string) {
	if nil != aURL.User {
		if rUser = aURL.User.Username(); 0 < len(rUser) {
			return
		}
	}

	return "-"
} // getUsername()

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

const (
	/*
		91.64.58.179 - username [25/Apr/2018:20:16:45 +0200] "GET /path/to/file?lang=en HTTP/1.1" 200 27155 "-" "Mozilla/5.0 (X11; Linux x86_64; rv:56.0) Gecko/20100101 Firefox/56.0"

		2001:4dd6:b474:0:1234:5678:90ab:cdef - - [24/Apr/2018:23:58:42 +0200] "GET /path/to/file HTTP/1.1" 200 5361 "https://www.google.de/" "Mozilla/5.0 (iPhone; CPU iPhone OS 12_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/12.1 Mobile/15E148 Safari/604.1"
	*/

	// `apacheFormatPattern` is the format of Apache like logfile entries:
	apacheFormatPattern = `%s - %s [%s] "%s %s %s" %d %d "%s" "%s"` + "\n"
)

// `goLog()` does the actual background logging.
//
// This function is called once for each request.
//
// `aLogger` is the handler of log messages.
//
// `aRequest` represents an HTTP request received by a server
//
// `aTime` is the actual time of the request served.
//
// `aDestination` is the channel to the logfile writer.
func goLog(aLogger *tLogWriter, aRequest *http.Request, aTime time.Time, aDestination chan<- string) {
	agent := aRequest.UserAgent()
	if "" == agent {
		agent = "-"
	}

	// build the log string and send it to the channel:
	aDestination <- fmt.Sprintf(apacheFormatPattern,
		getRemote(aRequest.RemoteAddr),
		getUsername(aRequest.URL),
		aTime.Format("02/Jan/2006:15:04:05 -0700"),
		aRequest.Method,
		getPath(aRequest.URL),
		aRequest.Proto,
		aLogger.status,
		aLogger.size,
		getReferrer(&aRequest.Header),
		agent,
	)
} // goLog()

const (
	// Half a second to sleep in `goWrite()`.
	halfSecond = 500 * time.Millisecond
)

// `goWrite()` performs the actual file write.
//
// This function is called only once, handling all write requests.
//
// `aLogfile` the name of the logfile to write to.
//
// `aSource` the source of log messages to write.
func goWrite(aLogfile string, aSource <-chan string) {
	var (
		err  error
		file *os.File
		more bool
		txt  string
	)
	defer func() {
		if nil != file {
			file.Close()
			file = nil
		}
	}()

	// let the application initialise:
	time.Sleep(halfSecond)

	for { // wait for strings to write
		select {
		case txt, more = <-aSource:
			if !more { // channel closed
				return
			}
			if nil == file {
				if file, err = os.OpenFile(aLogfile,
					os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); nil != err {
					file = os.Stderr // a last resort
				}
			}
			fmt.Fprint(file, txt)

			// let's handle waiting messages
			cCap := cap(aSource)
			for txt = range aSource {
				fmt.Fprint(file, txt)
				cCap--
				if 0 == cCap {
					break // give a chance to close the file
				}
			}

		default:
			if nil == file {
				time.Sleep(halfSecond)
			} else {
				if file != os.Stderr {
					file.Close()
				}
				file = nil
			}
		}
	}
} // goWrite()

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

// Wrap returns a handler function that includes logging, wrapping
// the given `aHandler` and calling it internally.
//
// The logfile entries written to `aLogfile` resemble the combined
// log file messages generated by the Apache web-server.
//
// `aHandler` responds to the actual HTTP request.
//
// `aLogfile` is the name of the file to use for writing the log messages.
func Wrap(aHandler http.Handler, aLogfile string) http.Handler {
	var (
		doOnce    sync.Once
		messenger chan string
	)
	doOnce.Do(func() {
		file, err := os.OpenFile(aLogfile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		file.Close()
		if nil != err {
			log.Fatalf("%s can't open logfile: %v", os.Args[0], err)
		}
		messenger = make(chan string, 64)

		// start the background writer:
		go goWrite(aLogfile, messenger)
	})

	return http.HandlerFunc(
		func(aWriter http.ResponseWriter, aRequest *http.Request) {
			lw := &tLogWriter{aWriter, 0, 0}
			aHandler.ServeHTTP(lw, aRequest)

			// run the log-entry formatter:
			go goLog(lw, aRequest, time.Now(), messenger)
		})
} // Wrap()

/* _EoF_ */
