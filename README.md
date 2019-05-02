# ApacheLogger

## Purpose

This package can be used to add a logfile facility to your `Go` web-server.
The format of the generated logfile entries resemble those of the popular Apache web-server.
The pattern for a logfile entry is this:

    apacheFormatPattern = "%s - %s [%s] \"%s %s %s\" %d %d \"%s\" \"%s\"\n"

All the placeholders to be seen in the pattern will be filled in with the appropriate values at runtime.
That means you can now use all the logfile analysers etc. for Apache logs for your own logfiles as well.

## Installation

You can use `Go` to install this package for you:

    go get go.mwat.de/apachelogger

## Usage

To include the automatic logging facility you just call the `Wrap()` function (which is the only exported function of this little package) as shown here:

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

## Special Features

As **privacy** becomes a serious concern for a growing number of people (including law makers) – and the IP address is definitely to be considered as _personal data_ – this logging facility _anonymises_ the requesting URLs by setting the host-part of the remote address to zero (`0`).
This option takes care of e.g. European servers who may _not without explicit consent_ of the users store personal data; this includes IP addresses in logfiles and elsewhere (eg. statistical data gathered from logfiles).

## Licence

    Copyright (C) 2019  M.Watermann, 10247 Berlin, FRG
                All rights reserved
            EMail : <support@mwat.de>

> This program is free software; you can redistribute it and/or modify it under the terms of the GNU General Public License as published by the Free Software Foundation; either version 3 of the License, or (at your option) any later version.
>
> This software is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.
>
> You should have received a copy of the GNU General Public License along with this program.  If not, see the [GNU General Public License](http://www.gnu.org/licenses/gpl.html) for details.
