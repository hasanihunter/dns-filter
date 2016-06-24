#!/bin/sh
CC=clang GOARCH=amd64 GOOS=freebsd go build -o dns-filter