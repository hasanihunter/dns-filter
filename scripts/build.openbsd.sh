#!/bin/sh
CC=clang GOARCH=amd64 GOOS=openbsd go build -o dns-filter