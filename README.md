# ApacheLogger

[![Golang](https://img.shields.io/badge/Language-Go-green.svg)](https://golang.org/)
[![GoDoc](https://godoc.org/github.com/mwat56/apachelogger?status.svg)](https://godoc.org/github.com/mwat56/apachelogger/)
[![Go Report](https://goreportcard.com/badge/github.com/mwat56/apachelogger)](https://goreportcard.com/report/github.com/mwat56/apachelogger)
[![Issues](https://img.shields.io/github/issues/mwat56/apachelogger.svg)](https://github.com/mwat56/apachelogger/issues?q=is%3Aopen+is%3Aissue)
[![Size](https://img.shields.io/github/repo-size/mwat56/apachelogger.svg)](https://github.com/mwat56/apachelogger/)
[![Tag](https://img.shields.io/github/tag/mwat56/apachelogger.svg)](https://github.com/mwat56/apachelogger/tags)
[![License](https://img.shields.io/github/license/mwat56/apachelogger.svg)](https://github.com/mwat56/apachelogger/blob/master/LICENSE)
[![View examples](https://img.shields.io/badge/learn%20by-examples-0077b3.svg)](https://github.com/mwat56/apachelogger/blob/master/cmd/demo.go)

- [ApacheLogger](#apachelogger)
	- [Purpose](#purpose)
	- [Installation](#installation)
	- [Usage](#usage)
	- [Special Features](#special-features)
	- [Licence](#licence)

## Purpose

This package can be used to add a logfile facility to your `Go` web-server.
The format of the generated logfile entries resemble those of the popular _Apache_ web-server (see below).

## Installation

You can use `Go` to install this package for you:

    go get -u github.com/mwat56/apachelogger

## Usage

To include the automatic logging facility you just call the `Wrap()` function (which is one of only two exported functions of this little package) as shown here:

    func main() {
        // the filename should be taken from the commandline or a config file:
        logfile := "/dev/stderr"

        pageHandler := http.NewServeMux()
        pageHandler.HandleFunc("/", myHandler)

        server := http.Server{
            Addr:    "127.0.0.1:8080",
            Handler: apachelogger.Wrap(pageHandler, logfile),
            //       ^^^^^^^^^^^^^^^^^^
        }

        if err := server.ListenAndServe(); nil != err {
            log.Fatalf("%s: %v", os.Args[0], err)
        }
    } // main()

So you just have to find a way the get/set the name of the desired `logfile` – e.g. via a commandline option, or an environment variable, or a config file, whatever suits you best.
Then you setup your `server` like shown above using the call to `apachelogger.Wrap()` to wrap your original pagehandler with the logging facility.
That's all.

The creation pattern for a logfile entry is this:

    apacheFormatPattern = `%s - %s [%s] "%s %s %s" %d %d "%s" "%s"`

All the placeholders to be seen in the pattern will be filled in with the appropriate values at runtime which are (in order of appearance):

* remote IP,
* remote user,
* date/time of request,
* request method,
* requested URL,
* request protocol,
* server status,
* served size,
* remote referrer,
* remote user agent.

It means you can now use all the logfile analysers etc. for Apache logs for your own logfiles as well.

## Special Features

As _**privacy**_ becomes a serious concern for a growing number of people (including law makers) – the IP address is definitely to be considered as _personal data_ – this logging facility _anonymises_ the requesting users by setting the host-part of the respective remote address to zero (`0`).
This option takes care of e.g. European servers who may _not without explicit consent_ of the users store personal data; this includes IP addresses in logfiles and elsewhere (eg. statistical data gathered from logfiles).

While the logging of web-requests is done fully automatic you can _manually add entries_ to the logfile by calling

    apachelogger.Log(aSender, aMessage string)

The `aSender` argument should give some indication of from where in your program you're calling the function, and `aMessage` is the text you want to write to the logfile.
To preserve the format of the log-entry neither `aSender` nor `aMessage` should contain double-quotes (`"`).
The messages are logged as coming from `127.0.0.1` with an user-agent of `mwat56/apachelogger`; this should make it easy to find these messages amongst all the 'normal' ones.

To avoid that a `panic` crashes your program this modules catches and `recover`s such situations.
The error/cause of the `panic` is written to the logfile for later inspection.

## Licence

        Copyright © 2019 M.Watermann, 10247 Berlin, Germany
                        All rights reserved
                    EMail : <support@mwat.de>

> This program is free software; you can redistribute it and/or modify it under the terms of the GNU General Public License as published by the Free Software Foundation; either version 3 of the License, or (at your option) any later version.
>
> This software is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.
>
> You should have received a copy of the GNU General Public License along with this program. If not, see the [GNU General Public License](http://www.gnu.org/licenses/gpl.html) for details.
