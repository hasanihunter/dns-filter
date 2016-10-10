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
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/miekg/dns"
)

var (
	config      DNSConfig
	pidFilePath string
)

type DNSFilter struct {
	Host          string
	QueryType     uint16
	ExactMatching bool
}

type DNSForwardingServer struct {
	IPAddress string
	Port      int64
	Protocol  string
}

func performLookup(host string, resultType uint16) ([]dns.RR, error) {

	processLookupRequest := true

	results := []dns.RR{}

	var dnsServerErr error

	// process any filters and exclude them from the lookup
	for _, filter := range config.Filters {

		if !filter.ExactMatching {

			// apply to the entire domain
			if strings.Contains(host, filter.Host) && filter.QueryType == resultType {

				// we have a match so don't forward the lookup
				processLookupRequest = false
				break
			}
		} else {
			// we have to match exactly to *not* process the lookup request
			// add the "." to the end of the filter.HOST or the exact matching doesn't work
			if fmt.Sprintf("%s.", filter.Host) == host && filter.QueryType == resultType {
				processLookupRequest = false
				break
			}
		}
	}

	if processLookupRequest {

		// create the dns client and message
		client := new(dns.Client)

		// setup the query
		query := new(dns.Msg)
		query.SetQuestion(host, resultType)

		/* Loop through our list of forwarding dns servers and break on the
		   first one that doesn't error out */
		for _, server := range config.Forwarders {

			/* Combine the dns server ip address with the port number so that the
			   client can send the query properly */
			serverString := fmt.Sprintf("%s:%d", server.IPAddress, server.Port)

			/* We aren't interested in caching the result so we aren't going to worry
			   about the ttl in the response */
			reply, _, replyErr := client.Exchange(query, serverString)

			if replyErr == nil {
				// got a reply and no error so we are safe to bounce out
				results = reply.Answer

				/* Clearing this error in the event that a previous dns server didn't work out
				   but we are good here */
				dnsServerErr = nil
				break
			} else {
				dnsServerErr = replyErr

				logMessage("Error communicating with: %s - error: %v\n", serverString, replyErr)

				/* We aren't breaking here so we can continue to loop through the rest of the
				   dns servers. If we get to the end of our list and still have an error then
				   we just fall through to the end with an empty results slice and the last error
				   stored in dnsServerErr */
			}
		}

	}

	return results, dnsServerErr

}

func processLookupQuery(req *dns.Msg) error {

	answers := []dns.RR{}

	for _, q := range req.Question {

		results, resultsErr := performLookup(q.Name, q.Qtype)

		if resultsErr != nil {
			return resultsErr
		}

		if results == nil {
			return errors.New("Empty results should never be nil!!!!")
		}

		for _, result := range results {
			if result.Header().Name == q.Name {
				answers = append(answers, result)
			}
		}
	}

	// set the answers to the request
	req.Answer = answers
	return nil
}

func handleDNSRequest(rw dns.ResponseWriter, req *dns.Msg) {

	response := new(dns.Msg)
	response.SetReply(req)
	response.Compress = false

	// we only handle queries, not updates, etc
	switch req.Opcode {
	case dns.OpcodeQuery:
		queryErr := processLookupQuery(response)
		if queryErr != nil {
			logMessage("Error during query: %v\n", queryErr)
		}
	}

	// send the response back
	rw.WriteMsg(response)
}

func serveDNSRequests() {

	server := &dns.Server{}
	if config.Port > 0 {
		server.Addr = fmt.Sprintf("%s:%d", config.Host, config.Port)
	} else if config.Host != "" {
		server.Addr = fmt.Sprintf("%s:domain", config.Host)
	}

	listenErr := server.ListenAndServe()

	defer server.Shutdown()

	if listenErr != nil {
		logFatal("Unable to listen and serve: %s\n ", listenErr.Error())
	}
}

func createPidFile(pidPath string) {

	pidFile, pidFileErr := os.OpenFile(pidPath, os.O_RDWR|os.O_CREATE, 0666)
	if pidFileErr != nil {
		logFatal("Couldn't create pid file:  ", pidFileErr)
	} else {
		pidFile.Write([]byte(strconv.Itoa(syscall.Getpid())))
		defer pidFile.Close()
	}
}

func main() {

	configFilePath := flag.String("c", "config.json", "path to config file ")

	pidFilePath := flag.String("pid ", "dns-filter.pid ", "path pid file")

	flag.Parse()

	dns.HandleFunc(".", handleDNSRequest)

	dnsConfig, configErr := parseConfigFile(*configFilePath)

	if configErr != nil {
		log.Fatalf("Error processing configuration: %v", configErr)
	}

	if dnsConfig == nil {
		log.Fatalf("config is nil!")
	}

	config = *dnsConfig

	// setup the pid file
	createPidFile(*pidFilePath)

	// Start server
	go serveDNSRequests()

	terminateSignal := make(chan os.Signal)
	signal.Notify(terminateSignal, syscall.SIGINT, syscall.SIGTERM)
forever:
	for {
		select {
		case s := <-terminateSignal:
			logMessage("Signal (%d) received, stopping\n ", s)
			os.Remove(*pidFilePath)
			break forever
		}
	}
}
