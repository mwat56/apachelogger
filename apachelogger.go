/*
   Copyright © 2019, 2022 M.Watermann, 10247 Berlin, Germany
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
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

var (
	// AnonymiseURLs decides whether to anonymise the remote IP addresses
	// before writing them to the logfile (default: `true`).
	//
	// For privacy and legal reasons this variable should always
	// stay `true`.
	AnonymiseURLs = true

	// AnonymiseErrors decides whether to anonymise remote IP addresses
	// that cause errors with our server using this module.
	AnonymiseErrors = false
)

type (
	// `tLogWriter` embeds a `ResponseWriter` and provides log-to-file.
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
	// Add length of _all_ chunks of data written.
	lw.size += len(aData) // We need this value for the logfile.

	return lw.ResponseWriter.Write(aData)
} // Write()

// WriteHeader sends an HTTP response header with the provided
// status code.
//
// Part of the `http.ResponseWriter` interface.
//
//	`aStatus` is the request's final result code.
func (lw *tLogWriter) WriteHeader(aStatus int) {
	lw.status = aStatus
	lw.ResponseWriter.WriteHeader(aStatus)
} // WriteHeader()

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

type (
	// Simple structure implementing the `io.Writer` interface.
	tLogLog struct{}
)

// Write sends `aMessage` from the running server to the log file.
// It returns the number of bytes written and `nil`.
//
// Implementing the `io.Writer` interface.
//
//	`aMessage` The error text to log.
func (ll tLogLog) Write(aMessage []byte) (int, error) {
	ml := len(aMessage)
	if 0 < ml {
		// Write to the error logfile in background:
		go goCustomLog(`errorLogger`, string(aMessage), `ERR`, time.Now(), alErrorQueue)
	}

	return ml, nil
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

	alBracketRE = regexp.MustCompile(`\[([0-9a-f:\.]+)\]`)

	// alLastLoggingDate stores the last day of logging
	alLastLoggingDate time.Time = time.Now()
)

// `getRemote()` reads and anonymises the remote address.
func getRemote(aRequest *http.Request, aStatus int) (rAddress string) {
	var err error

	addr := aRequest.RemoteAddr
	// We neither need nor want the remote port here:
	if rAddress, _, err = net.SplitHostPort(addr); nil != err {
		// err == "missing port in address"

		if matches := alBracketRE.FindStringSubmatch(addr); 1 < len(matches) {
			// Remove "[]" from address
			rAddress = matches[1]
		} else {
			rAddress = addr
		}
	}

	// Check whether the request went through a proxy.
	// X-Forwarded-For: client, proxy1, proxy2
	// Note: "proxy3" is the actual sender (i.e. aRequest.RemoteAddr).
	if xff := strings.Trim(aRequest.Header.Get("X-Forwarded-For"), ","); 0 < len(xff) {
		addrs := strings.Split(xff, ",")
		if ip := net.ParseIP(addrs[0]); nil != ip {
			rAddress = ip.String()
		}
	}

	if !AnonymiseURLs { // Bad choice generally …
		return
	}

	if (!AnonymiseErrors) && (400 <= aStatus) {
		// store full address for requests causing errors
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
	// Channel to send access log messages to and read messages from.
	alAccessQueue = make(chan string, 127)

	// Name of current user (used by `goCustomLog()`).
	alCurrentUser string = "-"

	// Channel to send error log messages to and read messages from.
	alErrorQueue = make(chan string, 127)

	// Make sure to initialise the wrapper only once.
	alWrapOnce sync.Once
)

// compareDayStamps returns whether the current message's date differs
// from the last logging date.
//
// The method returns `true` if the day/month/year changed from the
// time the last protocol messages was logged.
func compareDayStamps() (rChanged bool) {
	var (
		currentLoggingDate  = time.Now()
		nYear, nMonth, nDay = currentLoggingDate.Date()
		oYear, oMonth, oDay = alLastLoggingDate.Date()
	)

	rChanged = (nDay != oDay) ||
		(nMonth != oMonth) ||
		(nYear != oYear)
	if rChanged {
		alLastLoggingDate = currentLoggingDate
	}

	return
} // compareDayStamps()

// `goCustomLog()` sends a custom log message on behalf of `Log()`.
//
//	`aSender` Identification of the message's sender.
//	`aMessage` The message to write to the logfile.
//	`aPrefix` A prefix for the log message (either `LOG` or `ERR`).
//	`aTime` The time to log.
//	`aLogChannel` The channel to send the message to.
func goCustomLog(aSender, aMessage, aPrefix string, aTime time.Time, aLogChannel chan<- string) {
	defer func() {
		_ = recover() // panic: send on closed channel
	}()
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
	aLogChannel <- fmt.Sprintf(alApacheFormatPattern,
		"127.0.0.1",
		alCurrentUser,
		aTime.Format("02/Jan/2006:15:04:05 -0700"),
		aPrefix,
		aMessage,
		"HTTP/1.0",
		500,
		len(aMessage),
		aSender,
		"mwat56/apachelogger",
	)
} // goCustomLog()

// `goIgnoreLog()` just reads from `aMsgSource` ignoring the values.
func goIgnoreLog(aMsgSource <-chan string) {
	for range aMsgSource {
		runtime.Gosched()
	}
} // goIgnoreLog()

// `goStandardLog()` prepares the actual background logging.
//
// This function is called once for each request.
//
//	`aLogger` is the handler of log messages.
//	`aRequest` represents an HTTP request received by the server.
//	`aLogChannel` is the channel to write the message to.
func goStandardLog(aLogger *tLogWriter, aRequest *http.Request, aLogChannel chan<- string) {
	defer func() {
		_ = recover() // panic: send on closed channel
	}()
	agent := aRequest.UserAgent()
	if 0 == len(agent) {
		agent = "-"
	}

	// build the log string and send it to the channel:
	aLogChannel <- fmt.Sprintf(alApacheFormatPattern,
		getRemote(aRequest, aLogger.status),
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

const (
	alFileCloserDelay = time.Second << 3 // eight seconds

	// Mode of opening the logfile(s).
	alOpenFlags = os.O_CREATE | os.O_APPEND | os.O_WRONLY | os.O_SYNC
)

// `goWriteLog()` performs the actual file write.
//
// This function is run only once, handling all write requests.
//
//	`aMsgLog` The name of the logfile to write to.
//	`aMsgSource` The source of log messages to write.
func goWriteLog(aMsgLog string, aMsgSource <-chan string) {
	var (
		cLen       int
		closeTimer *time.Timer
		err        error
		logFile    *os.File
	)
	defer func() {
		// try to avoid resource leaks
		if nil != logFile {
			_ = logFile.Close()
		}
		if nil != closeTimer {
			_ = closeTimer.Stop()
		}
	}()

	time.Sleep(alFileCloserDelay) // let the application initialise
	closeTimer = time.NewTimer(alFileCloserDelay)

	for { // Wait for strings to log/write
		select {
		case txt, more := <-aMsgSource:
			if !more { // Channel closed
				return
			}
			if compareDayStamps() { // it's a new day …
				txt = "\n" + txt
			} // if

			if nil == logFile {
				// Loop until we actually opened the logfile:
				for {
					if logFile, err = os.OpenFile(aMsgLog, alOpenFlags, 0640); /* #nosec G302 */ nil == err {
						break
					}
					time.Sleep(1234)
					closeTimer.Reset(alFileCloserDelay)
				} // for
			} // if
			fmt.Fprint(logFile, txt)
			if cLen = len(aMsgSource); 0 < cLen {
				// Batch all waiting messages at once.
				for txt = range aMsgSource {
					fmt.Fprint(logFile, txt)
					cLen--
					if 0 < cLen {
						continue
					}
					if cLen = len(aMsgSource); 0 == cLen {
						break
					}
				} // for
			} // if
			closeTimer.Reset(alFileCloserDelay)

		case <-closeTimer.C:
			// Nothing logged in eight seconds => close the file.
			if nil != logFile {
				_ = logFile.Close()
				logFile = nil
			}
			closeTimer.Reset(alFileCloserDelay)
		} // select
	} // for
} // goWriteLog()

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

// Err writes `aMessage` on behalf of `aSender` to the error logfile.
//
//	`aSender` The name/designation of the sending entity.
//	`aMessage` The text to write to the error logfile.
func Err(aSender, aMessage string) {
	go goCustomLog(aSender, aMessage, `ERR`, time.Now(), alErrorQueue)
} // Err()

// Log writes `aMessage` on behalf of `aSender` to the access logfile.
//
//	`aSender` The name/designation of the sending entity.
//	`aMessage` The text to write to the access logfile.
func Log(aSender, aMessage string) {
	go goCustomLog(aSender, aMessage, `LOG`, time.Now(), alAccessQueue)
} // Log()

// Wrap returns a handler function that includes logging, wrapping
// the given `aHandler`, and calling it internally.
//
// The logfile entries written to `aAccessLog` resemble the combined
// log file messages generated by the Apache web-server.
//
// In case the provided `aAccessLog` can't be opened `Wrap()` terminates
// the program with an appropriate error-message.
//
//	`aHandler` responds to the actual HTTP request.
//	`aAccessLog` is the name of the file to use for access log messages.
//	`aErrorLog` is the name of the file to use for error log messages.
//
// The function returns the (augmented) `http.Handler`.
func Wrap(aHandler http.Handler, aAccessLog, aErrorLog string) http.Handler {
	alWrapOnce.Do(func() {
		if usr, err := user.Current(); (nil == err) && (0 < len(usr.Username)) {
			alCurrentUser = usr.Username
		}

		if 0 < len(aAccessLog) {
			absFile, _ := filepath.Abs(aAccessLog)
			aAccessLog = absFile
		}
		if 0 < len(aAccessLog) {
			accessFile, err := os.OpenFile(aAccessLog, alOpenFlags, 0640) // #nosec G302
			_ = accessFile.Close()
			if nil != err {
				log.Fatalf("%s can't open access logfile: %v", os.Args[0], err)
			}
			go goWriteLog(aAccessLog, alAccessQueue)
		} else {
			go goIgnoreLog(alAccessQueue)
		}

		if 0 < len(aErrorLog) {
			absFile, _ := filepath.Abs(aErrorLog)
			aErrorLog = absFile
		}
		if 0 < len(aErrorLog) {
			if aErrorLog == aAccessLog {
				close(alErrorQueue)
				alErrorQueue = alAccessQueue
			} else {
				errorFile, err := os.OpenFile(aErrorLog, alOpenFlags, 0640) // #nosec G302
				_ = errorFile.Close()
				if nil != err {
					log.Fatalf("%s can't open error logfile: %v", os.Args[0], err)
				}
				go goWriteLog(aErrorLog, alErrorQueue)
			}
		} else {
			go goIgnoreLog(alErrorQueue)
		}
	})

	return http.HandlerFunc(
		func(aWriter http.ResponseWriter, aRequest *http.Request) {
			defer func() {
				// make sure a `panic` won't kill the program
				if err := recover(); nil != err {
					go goCustomLog("errorLogger", fmt.Sprintf("caught panic: %v – %s", err, debug.Stack()), `ERR`, time.Now(), alErrorQueue)
				}
			}()
			lw := &tLogWriter{aWriter, 0, 0, time.Now()}
			aHandler.ServeHTTP(lw, aRequest)

			// run the log-entry formatter:
			go goStandardLog(lw, aRequest, alAccessQueue)
		})
} // Wrap()

/* _EoF_ */
