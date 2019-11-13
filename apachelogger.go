/*
   Copyright © 2019 M.Watermann, 10247 Berlin, Germany
                   All rights reserved
               EMail : <support@mwat.de>
*/

package apachelogger

//lint:file-ignore ST1017 – I prefer Yoda conditions

/*
   This package can be used to add a logfile facility to your
   `Go` web-server.
   The format of the generated logfile entries resembles that
   of the popular Apache web-server.
*/

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

var (
	// AnonymiseURLs decides whether to anonymise the remote URLs
	// (default: `true`).
	//
	// For legal reasons this variable should always stay `true`.
	AnonymiseURLs = true
)

type (
	// `tLogWriter` embeds a `ResponseWriter` and includes log-to-file.
	tLogWriter struct {
		http.ResponseWriter           // used to construct the HTTP response
		size                int       // the size/length of the data sent
		status              int       // HTTP status code of current request
		when                time.Time // access time
	}
)

// Write writes the data to the connection as part of an HTTP reply.
//
// Part of the `http.ResponseWriter` interface.
func (lw *tLogWriter) Write(aData []byte) (int, error) {
	if 0 == lw.status {
		lw.status = 200
	}
	// Add length of all chunks of data written.
	lw.size += len(aData) // We need this value for the logfile.

	return lw.ResponseWriter.Write(aData)
} // Write()

// WriteHeader sends an HTTP response header with the provided
// status code.
//
// Part of the `http.ResponseWriter` interface.
//
//	`aStatus` is the request's funal result code.
func (lw *tLogWriter) WriteHeader(aStatus int) {
	lw.status = aStatus
	lw.ResponseWriter.WriteHeader(aStatus)
} // WriteHeader()

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

type (
	// Simple structure implemening the `Writer` interface.
	tLogLog struct{}
)

// Write sends `aData` to the log file.
//
//	`aData` The error text to log.
func (ll tLogLog) Write(aData []byte) (int, error) {
	dl := len(aData)
	if 0 < dl {
		// To return fast to the caller we perform the actual
		// writing to the logfile in background:
		go goCustomLog(`errorLogger`, string(aData), time.Now())
	}

	return dl, nil
} // Write()

// SetErrLog sets the error logger of `aServer`.
//
//	`aServer` The server instance whose errlogger is to be set.
func SetErrLog(aServer *http.Server) {
	aServer.ErrorLog = log.New(tLogLog{}, "", log.Llongfile)
} // SetErrLog()

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

// `getReferrer()` returns the request header's referrer field.
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
	alIpv4RE = regexp.MustCompile(`^([0-9]{1,3})\.([0-9]{1,3})\.([0-9]{1,3})\.[0-9]{1,3}`)

	// RegEx to match IPv6 addresses:
	alIpv6RE = regexp.MustCompile(`([0-9a-f]{1,4})\:([0-9a-f]{1,4})\:([0-9a-f]{1,4})\:([0-9a-f]{1,4})\:([0-9a-f]{1,4})\:([0-9a-f]{1,4})\:([0-9a-f]{1,4})\:([0-9a-f]{1,4})`)
)

// `getRemote()` reads and anonymises the remote address.
func getRemote(aRequest *http.Request) (rAddress string) {
	var err error

	addr := aRequest.RemoteAddr
	// We neither need nor want the remote port here:
	if rAddress, _, err = net.SplitHostPort(addr); err != nil {
		// err == "missing port in address"
		rAddress = addr
	}
	// Check whether the request went through a proxy.
	// X-Forwarded-For: client, proxy1, proxy2
	// Note: "proxy3" is the actual sender (i.e. aRequest.RemoteAddr).
	if xff := strings.Trim(aRequest.Header.Get("X-Forwarded-For"), ","); 0 < len(xff) {
		addrs := strings.Split(xff, ",")
		if ip := net.ParseIP(addrs[0]); ip != nil {
			rAddress = ip.String()
		}
	}

	if !AnonymiseURLs { // Bad Choice!
		return
	}

	if matches := alIpv4RE.FindStringSubmatch(rAddress); 3 < len(matches) {
		// anonymise the remote IPv4 address:
		rAddress = fmt.Sprintf("%s.%s.%s.0",
			matches[1], matches[2], matches[3])
	} else if matches := alIpv6RE.FindStringSubmatch(rAddress); 8 < len(matches) {
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

	// `alApacheFormatPattern` is the format of Apache like logfile entries:
	alApacheFormatPattern = `%s - %s [%s] "%s %s %s" %d %d "%s" "%s"` + "\n"
)

var (
	// Name of current user (used by `goCustomLog()`).
	alCurrentUser string = "-"

	// Channel to send log-messages to and read messages from.
	alMsgQueue = make(chan string, 127)
)

// `goCustomLog()` sends a custom log message on behalf of `Log()`.
func goCustomLog(aSender, aMessage string, aTime time.Time) {
	if 0 == len(aSender) {
		aSender = filepath.Base(os.Args[0])
	}
	if 0 == len(aMessage) {
		aMessage = "PING"
	} else {
		aMessage = strings.Replace(aMessage, "\n", "; ", -1)
		aMessage = strings.Replace(aMessage, "\t", " ", -1)
		aMessage = strings.TrimSpace(strings.Replace(aMessage, "  ", " ", -1))
	}

	// build the log string and send it to the channel:
	alMsgQueue <- fmt.Sprintf(alApacheFormatPattern,
		"127.0.0.1",
		alCurrentUser,
		aTime.Format("02/Jan/2006:15:04:05 -0700"),
		"LOG",
		aMessage,
		"HTTP/1.0",
		500,
		len(aMessage),
		aSender,
		"mwat56/apachelogger",
	)
} // goCustomLog()

// `goStandardLog()` prepares the actual background logging.
//
// This function is called once for each request.
//
//	`aLogger` is the handler of log messages.
//	`aRequest` represents an HTTP request received by the server.
func goStandardLog(aLogger *tLogWriter, aRequest *http.Request) {
	agent := aRequest.UserAgent()
	if 0 == len(agent) {
		agent = "-"
	}

	// build the log string and send it to the channel:
	alMsgQueue <- fmt.Sprintf(alApacheFormatPattern,
		getRemote(aRequest),
		getUsername(aRequest.URL),
		aLogger.when.Format("02/Jan/2006:15:04:05 -0700"),
		aRequest.Method,
		getPath(aRequest.URL),
		aRequest.Proto,
		aLogger.status,
		aLogger.size,
		getReferrer(&aRequest.Header),
		agent,
	)
	aLogger.status, aLogger.size = 0, 0
} // goStandardLog()

var (
	// Mode of opening the logfile(s).
	alOpenFlags = os.O_CREATE | os.O_APPEND | os.O_WRONLY
)

// `goWrite()` performs the actual file write.
//
// This function is run only once, handling all write requests.
//
//	`aAccessLog` The name of the access logfile to write to.
//	`aSource` The source of log messages to write.
func goWrite(aAccessLog string, aSource <-chan string) {
	var (
		err  error
		file *os.File
	)
	defer func() {
		if nil != file {
			_ = file.Close()
		}
	}()

	time.Sleep(time.Second) // let the application initialise
	timer := time.NewTimer(time.Minute)

	for { // wait for strings to write
		select {
		case txt, more := <-aSource:
			if !more { // channel closed
				return
			}
			if nil == file {
				for {
					if file, err = os.OpenFile(aAccessLog, alOpenFlags, 0640); /* #nosec G302 */ nil == err {
						break
					}
					time.Sleep(time.Millisecond)
				}
				// fmt.Fprintln(file, `OPEN -`, time.Now().Format("[02/Jan/2006:15:04:05 -0700]")) //TODO REMOVE
			}
			fmt.Fprint(file, txt)

			// handle the waiting messsages
			for cLen := len(aSource); 0 < cLen; cLen-- {
				fmt.Fprint(file, <-aSource)
			}

		case <-timer.C:
			if nil != file {
				// fmt.Fprintln(file, `CLOSE -`, time.Now().Format("[02/Jan/2006:15:04:05 -0700]")) //TODO REMOVE
				_ = file.Close()
				file = nil
			}
			timer.Reset(time.Minute)

		}
	}
} // goWrite()

func init() {
	if usr, err := user.Current(); (nil == err) && (0 < len(usr.Username)) {
		alCurrentUser = usr.Username
	}
} // init()

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

// Log writes `aMessage` on behalf of `aSender` to the logfile.
//
//	`aSender` The name/designation of the sending entity.
//	`aMessage` The text to write to the logfile.
func Log(aSender, aMessage string) {
	// To return fast to the caller we perform the actual
	// writing to the logfile in background:
	go goCustomLog(aSender, aMessage, time.Now())
} // Log()

// Wrap returns a handler function that includes logging, wrapping
// the given `aHandler` and calling it internally.
//
// The logfile entries written to `aAccessLog` resemble the combined
// log file messages generated by the Apache web-server.
//
// In case the provided `aAccessLog` can't be opened `Wrap()` terminates
// the program with an appropriate error-message.
//
//	`aHandler` responds to the actual HTTP request.
//	`aAccessLog` is the name of the file to use for access log messages.
func Wrap(aHandler http.Handler, aAccessLog string) http.Handler {
	var (
		doOnce sync.Once
	)
	doOnce.Do(func() {
		file, err := os.OpenFile(aAccessLog, alOpenFlags, 0640) // #nosec G302
		_ = file.Close()
		if nil != err {
			log.Fatalf("%s can't open logfile: %v", os.Args[0], err)
		}

		// start the background writer:
		go goWrite(aAccessLog, alMsgQueue)
	})

	return http.HandlerFunc(
		func(aWriter http.ResponseWriter, aRequest *http.Request) {
			defer func() {
				// make sure a `panic` won't kill the program
				if err := recover(); err != nil {
					go goCustomLog("errorLogger",
						fmt.Sprintf("caught panic: %v", err), time.Now())
				}
			}()
			lw := &tLogWriter{aWriter, 0, 0, time.Now()}
			aHandler.ServeHTTP(lw, aRequest)

			// run the log-entry formatter:
			go goStandardLog(lw, aRequest)
		})
} // Wrap()

/* _EoF_ */
