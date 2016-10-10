/*
Copyright (c) 2016, Hasani Hunter
All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

1. Redistributions of source code must retain the above copyright notice, this
   list of conditions and the following disclaimer.
2. Redistributions in binary form must reproduce the above copyright notice,
   this list of conditions and the following disclaimer in the documentation
   and/or other materials provided with the distribution.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE FOR
ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
(INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"runtime"
)

var (
	logger *log.Logger
)

func setupLogging(logFilePath string) {

	if len(logFilePath) > 0 {
		if logFilePath[0] != os.PathSeparator {
			// the user has set the log file path to be a relative path so pre pend the directory

			// get the parent directory
			logDirectory := path.Dir(logFilePath)

			logDirectoryInfo, logDirectoryErr := os.Stat(logDirectory)

			if logDirectoryErr != nil || logDirectoryInfo == nil {
				// directory doesn't exist.. so try to make it
				os.Mkdir(logDirectory, 0700)
			} else {
				if !logDirectoryInfo.IsDir() {
					panic("Log directory is a regular file")
				}
			}
		}

		fo, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0600)

		if err != nil {
			panic(err)
		}

		logger = log.New(fo, "", log.LstdFlags)
	} else {
		logger = log.New(os.Stdout, "", log.LstdFlags)
	}
}

func logConsole(message string, v ...interface{}) {
	logMessage := fmt.Sprintf(message, v...)
	log.Println(logMessage)
}

func logMessage(message string, v ...interface{}) error {

	if logger == nil {
		// we have no logger so return an error
		return errors.New("Logger is nil")
	} else {

		var arg_message string

		if len(v) > 0 {
			arg_message = fmt.Sprintf(message, v...)
		} else {
			arg_message = message
		}

		_, file, line, ok := runtime.Caller(2)

		var logMessage string

		if ok {
			// if the filename includes the full path, then strip it
			logMessage = fmt.Sprintf("%s:%d - %s", path.Base(file), line, arg_message)
		} else {
			logMessage = arg_message
		}

		logger.Println(logMessage)

		return nil
	}
}

func logFatal(message string, v ...interface{}) {
	var arg_message string

	if len(v) > 0 {
		arg_message = fmt.Sprintf(message, v...)
	} else {
		arg_message = message
	}

	log.Fatal(arg_message)
}
