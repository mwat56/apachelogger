/*
Copyright © 2019, 2025  M.Watermann, 10247 Berlin, Germany

	    All rights reserved
	EMail : <support@mwat.de>
*/
package apachelogger

import (
	"errors"
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

//lint:file-ignore ST1017 – I prefer Yoda conditions

type (
	// `tLogWriter` embeds a `ResponseWriter` and provides log-to-file.
	tLogWriter struct {
		http.ResponseWriter           // used to construct the HTTP response
		size                int       // the size/length of the data sent
		status              int       // HTTP status code of current request
		when                time.Time // access time
	}
)

const (
	/*
		91.64.58.179 - username [25/Apr/2018:20:16:45 +0200] "GET /path/to/file?lang=en HTTP/1.1" 200 27155 "-" "Mozilla/5.0 (X11; Linux x86_64; rv:56.0) Gecko/20100101 Firefox/56.0"

		2001:4dd6:b474:0:1234:5678:90ab:cdef - - [24/Apr/2018:23:58:42 +0200] "GET /path/to/file HTTP/1.1" 200 5361 "https://www.google.de/" "Mozilla/5.0 (iPhone; CPU iPhone OS 12_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/12.1 Mobile/15E148 Safari/604.1"
	*/

	// `alApacheFormatPattern` is the format of Apache like logfile entries:
	alApacheFormatPattern = `%s - %s [%s] "%s %s %s" %d %d "%s" "%s"` + "\n"

	// `alDefaultChannelBufferSize` defines the default size for log
	// message channels.
	alDefaultChannelBufferSize = 128

	// `alFileCloseDelay` is the time to wait before closing idle log files.
	alFileCloseDelay = time.Second << 3 // eight seconds

	// `alInitDelay` is the time to wait for application initialisation.
	alInitDelay = 1234 * time.Millisecond

	// Mode of opening the logfile(s).
	alOpenFlags = os.O_CREATE | os.O_APPEND | os.O_WRONLY | os.O_SYNC
)

var (
	// `AnonymiseURLs` decides whether to anonymise the remote IP addresses
	// before writing them to the logfile (default: `true`).
	//
	// For privacy and legal reasons this variable should always
	// stay `true`.
	AnonymiseURLs = true

	// `AnonymiseErrors` decides whether to anonymise remote IP addresses
	// that cause errors with our server using this module.
	AnonymiseErrors = false

	// Channel to send access log messages to and read messages from.
	alAccessQueue = make(chan string, alDefaultChannelBufferSize)

	// RegEx to match bracketed IPv6 addresses:
	alBracketRE = regexp.MustCompile(`\[([0-9a-f:\.]+)\]`)

	// Name of current user (used by `goCustomLog()`).
	alCurrentUser string = "-"

	// Channel to send error log messages to and read messages from.
	alErrorQueue = make(chan string, alDefaultChannelBufferSize)

	// `alLastLoggingDate` stores the last day of logging
	alLastLoggingDate time.Time = time.Now()

	// Make sure to initialise the wrapper only once.
	alWrapOnce sync.Once
)

// ---------------------------------------------------------------------------
// `tLogWriter` methods:

// `Write()` writes the data to the connection as part of an HTTP reply.
//
// Part of the `http.ResponseWriter` interface.
//
// Parameters:
// - `aData`: The data to to write.
//
// Returns:
// - `int`: The number of bytes written.
// - `error`: a possible error of processing.
func (lw *tLogWriter) Write(aData []byte) (int, error) {
	if 0 == lw.status {
		lw.status = 200
	}
	// Add length of _all_ chunks of data written.
	lw.size += len(aData) // We need this value for the logfile.

	return lw.ResponseWriter.Write(aData)
} // Write()

// `WriteHeader()` sends an HTTP response header with the provided
// status code.
//
// Part of the `http.ResponseWriter` interface.
//
// Parameters:
// - `aStatus`: The request's final result code.
func (lw *tLogWriter) WriteHeader(aStatus int) {
	lw.status = aStatus
	lw.ResponseWriter.WriteHeader(aStatus)
} // WriteHeader()

// ---------------------------------------------------------------------------
// `tLogLog` methods:

type (
	// Simple structure implementing the `io.Writer` interface.
	tLogLog struct{}
)

// `Write()` sends `aMessage` from the running server to the log file.
// It returns the number of bytes written and `nil`.
//
// Implementing the `io.Writer` interface.
//
// Parameters:
// - `aMessage` The error text to log.
//
// Returns:
// - `int`: The number of bytes written.
// - `error`: `nil`
func (ll tLogLog) Write(aMessage []byte) (int, error) {
	result := len(aMessage)
	if 0 < result {
		// Write to the error logfile in background:
		go goCustomLog(`errorLogger`, string(aMessage), `ERR`, time.Now(), alErrorQueue)
	}

	return result, nil
} // Write()

// `SetErrorLog()` sets the error logger of `aServer`.
//
// Parameters:
// - `aServer` The server instance whose errlogger is to be set.
func SetErrorLog(aServer *http.Server) {
	aServer.ErrorLog = log.New(tLogLog{}, "", log.Llongfile)
} // SetErrLog()

// ---------------------------------------------------------------------------
// Internal helper functions:

// `compareDayStamps()` checks whether the current message's date differs
// from the last logging date.
//
// Returns:
// - `bool`: `true` if the day changed from the day the
// last protocol message was logged, or `false` otherwise.
func compareDayStamps() bool {
	var (
		currentLoggingDate  = time.Now()
		nYear, nMonth, nDay = currentLoggingDate.Date()
		oYear, oMonth, oDay = alLastLoggingDate.Date()
	)

	changed := (nDay != oDay) ||
		(nMonth != oMonth) ||
		(nYear != oYear)
	if changed {
		alLastLoggingDate = currentLoggingDate
	}

	return changed
} // compareDayStamps()

// `getPath()` returns the requested path (and CGI query).
//
// Parameters:
// - aURL: The HTTP request object's URL.
//
// Returns:
// - `string`: The HTTP path as a string.
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

// `getProto()` returns the HTTP protocol version from the request.
//
// If the protocol version is not specified in the request, it defaults
// to "HTTP/1.0".
//
// Parameters:
// - `aRequest`: The HTTP request object.
//
// Returns:
// - `string`: The HTTP protocol version as a string.
func getProto(aRequest *http.Request) (rProto string) {
	if rProto = aRequest.Proto; "" == rProto {
		rProto = "HTTP/1.0"
	}

	return
} // getProto()

// `getReferrer()` returns the request header's referrer field.
//
// If the referrer field is not specified in the request, it defaults
// to "-".
//
// Parameters:
// - aHeader: The HTTP request object's header.
//
// Returns:
// - `string`: The HTTP referrer field as a string.
func getReferrer(aHeader *http.Header) (rReferrer string) {
	if rReferrer = aHeader.Get("Referer"); 0 < len(rReferrer) {
		return
	}

	if rReferrer = aHeader.Get("Referrer"); 0 < len(rReferrer) {
		return
	}

	return "-"
} // getReferrer()

// `getRemote()` reads and anonymises the remote address.
//
// It takes an http.Request and the HTTP status code of the current request.
// It returns the anonymised remote address.
//
// If the request went through a proxy, the function will try to anonymise
// the remote IP address of the proxy.
//
// If the 'AnonymiseURLs' flag is set to 'true', the function will anonymise
// the remote IP addresses. If the 'AnonymiseErrors' flag is set to 'true',
// the function will anonymise the remote IP addresses of requests causing
// errors.
//
// Parameters:
// - `aRequest`: The HTTP request object.
// - `aStatus`: The HTTP status code.
//
// Returns:
// - `string`: The anonymised remote address as a string.
func getRemote(aRequest *http.Request, aStatus int) (rAddress string) {
	var err error

	addr := aRequest.RemoteAddr
	// We neither need nor want the remote port here.
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

	ip := net.ParseIP(rAddress)
	if ip == nil {
		return // Not a valid IP address
	}

	if ip4 := ip.To4(); nil != ip4 {
		// IPv4 address: Zero out last octet
		ip4[3] = 0
		rAddress = ip4.String()
	} else {
		// IPv6 address: Zero out last 8 bytes
		ip[8], ip[9], ip[10], ip[11], ip[12], ip[13], ip[14], ip[15] = 0, 0, 0, 0, 0, 0, 0, 0
		rAddress = ip.String()
	}

	return
} // getRemote()

// `getUsername()` returns the request's username (if any).
//
// Parameters:
// - `aURL`: The HTTP request object's URL.
//
// Returns:
// - `string`: The username as a string.
func getUsername(aURL *url.URL) string {
	if nil != aURL.User {
		if user := aURL.User.Username(); "" != user {
			return user
		}
	}

	return "-"
} // getUsername()

// `goCustomLog()` sends a custom log message on behalf of `Log()` and `Err()`.
//
// Parameters:
// - `aSender`: Identification of the message's sender.
// - `aMessage`: The message to write to the logfile.
// - `aMethod`: Either `LOG` or `ERR`.
// - `aTime`: The time to log.
// - `aLogChannel`: The channel to send the message to.
func goCustomLog(aSender, aMessage, aMethod string, aTime time.Time, aLogChannel chan<- string) {
	defer func() {
		_ = recover() // panic: send on closed channel
	}()
	if "" == aSender {
		aSender = filepath.Base(os.Args[0])
	}
	if "" == aMessage {
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
		aMethod,
		aMessage,
		"HTTP/intern",
		500,
		len(aMessage),
		aSender, // instead of Referer header
		"mwat56/apachelogger",
	)
} // goCustomLog()

// `goDoLogWrite()` performs the actual file write.
//
// This function runs indefinitely, handling all write requests.
//
// Parameters:
// - `aLogFile`: The name of the logfile to write to.
// - `aMsgSource`: The source of log messages to write.
func goDoLogWrite(aLogFile string, aMsgSource <-chan string) {
	var (
		cLen       int
		closeTimer *time.Timer
		err        error
		logFile    *os.File
	)
	defer func() {
		// try to avoid resource leaks
		if nil != logFile {
			if err := logFile.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
				fmt.Fprintf(os.Stderr, "Error closing logfile: %v\n", err)
			}
		}
		if nil != closeTimer {
			_ = closeTimer.Stop()
		}
	}()

	time.Sleep(alInitDelay)
	closeTimer = time.NewTimer(alFileCloseDelay)

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
					if logFile, err = os.OpenFile(aLogFile,
						alOpenFlags, 0640); /* #nosec G302 */ nil == err {
						break
					}
					time.Sleep(1234)
					closeTimer.Reset(alFileCloseDelay)
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
			closeTimer.Reset(alFileCloseDelay)

		case <-closeTimer.C:
			// Nothing logged in eight seconds => close the file.
			if nil != logFile {
				_ = logFile.Close()
				logFile = nil
			}
			closeTimer.Reset(alFileCloseDelay)
		} // select
	} // for
} // goDoLogWrite()

// `goIgnoreLog()` is a background goroutine that reads from `aMsgSource`
// ignoring the values.
//
// Parameters:
// - `aMsgSource`: The channel to read the messages from.
func goIgnoreLog(aMsgSource <-chan string) {
	for {
		select {
		case txt := <-aMsgSource:
			// just empty the channel
			if "" != txt {
				txt = ""
			}

		default:
			runtime.Gosched()
		}
	}
} // goIgnoreLog()

// `goWebLog()` prepares the actual background logging.
//
// This function is called once for each request.
//
// Parameters:
// - `aLogger`: The handler of log messages.
// - `aRequest:` An HTTP request received by the server.
// - `aLogChannel`: The channel to write the message to.
func goWebLog(aLogger *tLogWriter, aRequest *http.Request,
	aLogChannel chan<- string) {
	defer func() {
		_ = recover() // panic: send on closed channel
	}()
	agent := aRequest.UserAgent()
	if "" == agent {
		agent = "-"
	}

	// build the log string and send it to the channel:
	aLogChannel <- fmt.Sprintf(alApacheFormatPattern,
		getRemote(aRequest, aLogger.status),
		getUsername(aRequest.URL),
		aLogger.when.Format("02/Jan/2006:15:04:05 -0700"),
		aRequest.Method,
		getPath(aRequest.URL),
		getProto(aRequest),
		aLogger.status,
		aLogger.size,
		getReferrer(&aRequest.Header),
		agent,
	)

	aLogger.status, aLogger.size = 0, 0
} // goWebLog()

// ---------------------------------------------------------------------------
// Exported functions:

// `Err()` writes `aMessage` on behalf of `aSender` to the error logfile.
//
// Parameters:
// - `aSender`: The name/designation of the sending entity.
// - `aMessage`: The text to write to the error logfile.
func Err(aSender, aMessage string) {
	go goCustomLog(aSender, aMessage, `ERR`, time.Now(), alErrorQueue)
} // Err()

// 'Log()' writes `aMessage` on behalf of `aSender` to the access logfile.
//
// Parameters:
// - `aSender`: The name/designation of the sending entity.
// - `aMessage`: The text to write to the access logfile.
func Log(aSender, aMessage string) {
	go goCustomLog(aSender, aMessage, `LOG`, time.Now(), alAccessQueue)
} // Log()

// `Wrap()` returns a handler function that includes logging, wrapping
// the given `aHandler`, and calling it internally.
//
// The logfile entries written to `aAccessLog` resemble the combined
// log file messages generated by the Apache web-server.
//
// In case the provided `aAccessLog` can't be opened `Wrap()` terminates
// the program with an appropriate error-message.
//
// Parameters:
// - `aHandler`: Responds to the actual HTTP request.
// - `aAccessLog`: The name of the file to use for access log messages.
// - `aErrorLog`: The name of the file to use for error log messages.
//
// Returns:
// - `http.Handler`:The (augmented) `aHandler`.
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
			go goDoLogWrite(aAccessLog, alAccessQueue)
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
				go goDoLogWrite(aErrorLog, alErrorQueue)
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
					Err("ApacheLogger/catchPanic",
						fmt.Sprintf("caught panic: %v - %s",
							err, debug.Stack()))
				}
			}()
			lw := &tLogWriter{aWriter, 0, 0, time.Now()}
			aHandler.ServeHTTP(lw, aRequest)

			// run the log-entry formatter:
			go goWebLog(lw, aRequest, alAccessQueue)
		})
} // Wrap()

/* _EoF_ */
