#!/bin/sh
CC=clang GOARCH=amd64 GOOS=windows go build -o dns-filter.exe