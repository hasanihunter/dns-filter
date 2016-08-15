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
	"encoding/json"
	"errors"
	"fmt"
	"github.com/miekg/dns"
	"io/ioutil"
	"os"
	"path"
)

type DNSConfig struct {
	Host       string
	Port       int64
	Forwarders []DNSForwardingServer
	Filters    []DNSFilter
	Logfile    string
}

func setupDefaultDNSForwardingServers() []DNSForwardingServer {

	// use free default dns servers
	hurricaneElectric1 := DNSForwardingServer{"74.82.42.42", 53, "udp"}
	opennic1 := DNSForwardingServer{"107.150.40.234", 53, "udp"}
	opennic2 := DNSForwardingServer{"162.211.64.20", 53, "udp"}
	opennic3 := DNSForwardingServer{"50.116.23.211", 53, "udp"}
	opennic4 := DNSForwardingServer{"50.116.40.226", 53, "udp"}
	freedns1 := DNSForwardingServer{"37.235.1.174", 53, "udp"}
	freedns2 := DNSForwardingServer{"37.235.1.177", 53, "udp"}
	google1 := DNSForwardingServer{"8.8.8.8", 53, "udp"}
	google2 := DNSForwardingServer{"8.8.4.4", 53, "udp"}

	dnsServers := []DNSForwardingServer{
		hurricaneElectric1,
		opennic1,
		opennic2,
		opennic3,
		opennic4,
		freedns1,
		freedns2,
		google1,
		google2,
	}

	return dnsServers
}

func parseConfigFile(configPath string) (*DNSConfig, error) {

	var configMap map[string]interface{}

	// try to read the config file
	bytes, configError := ioutil.ReadFile(configPath)

	if configError != nil {
		return nil, configError
	}

	err := json.Unmarshal(bytes, &configMap)

	if err != nil {
		return nil, err
	}

	config := &DNSConfig{}

	host, hostExists := configMap["host"].(string)

	if !hostExists {
		// bind to localhost interface by default
		host = "localhost"
	}

	config.Host = host

	port, portExists := configMap["port"].(float64)

	if !portExists {
		// bind to port 1234 by default
		port = 1234
	}

	config.Port = int64(port)

	forwardingSlice, forwardingMapExists := configMap["forwarders"].([]interface{})

	dnsServers := []DNSForwardingServer{}

	if !forwardingMapExists {

		// use default forwarding servers
		dnsServers = setupDefaultDNSForwardingServers()

	} else {
		for _, forwardInterface := range forwardingSlice {

			forwardMap := forwardInterface.(map[string]interface{})

			// validate that the host in the forwardMap is legit
			forwardHost, forwardHostExists := forwardMap["host"].(string)

			if !forwardHostExists {
				continue
			}

			// if the forward host is smaller than the smallest ipv4 address.. kick it
			if len(forwardHost) < len("8.8.8.8") {
				continue
			}

			forwardPort, forwardPortExists := forwardMap["port"].(int64)

			if !forwardPortExists {
				// we will just default to port 53
				forwardPort = 53
			}

			forwardProtocol, forwardProtocolExists := forwardMap["protocol"].(string)

			if !forwardProtocolExists {
				// default to udp
				forwardProtocol = "udp"
			} else {
				// make sure that the protocol is either udp or tcp
				if forwardProtocol != "udp" && forwardProtocol != "tcp" {
					// we have a misconfiguration so let the user know
					errorMessage := fmt.Sprintf("%s is an invalid protocol.  Protocol for host: %s must be either udp or tcp",
						forwardProtocol, forwardHost)

					return nil, errors.New(errorMessage)
				}

				// if we get here, then all is good
			}

			dnsServer := DNSForwardingServer{
				forwardHost,
				forwardPort,
				forwardProtocol,
			}

			dnsServers = append(dnsServers, dnsServer)

		}

		if len(dnsServers) < 1 {
			// if we make it all the way down here and we don't have any servers (due to a bad config, then use defaults)
			// use default forwarding servers
			dnsServers = setupDefaultDNSForwardingServers()
		}
	}

	config.Forwarders = dnsServers

	// process any filters
	dnsFilters := []DNSFilter{}

	filterSlice, filtersExists := configMap["filters"].([]interface{})

	if filtersExists {

		for _, filterInterface := range filterSlice {

			filterMap := filterInterface.(map[string]interface{})

			filterHost, filterHostExists := filterMap["host"].(string)

			if !filterHostExists {
				continue
			}

			// if the filter host is smaller than the smallest domain/host address.. keep on rolling
			if len(filterHost) < len("a.io") {
				continue
			}

			filterType, filterTypeExists := filterMap["type"].(string)

			// filter ALL records
			filterInt := dns.TypeANY

			if !filterTypeExists {
				filterType = "ALL"
			}

			if filterType != "ALL" {
				if filterType == "AAAA" {
					filterInt = dns.TypeAAAA
				} else if filterType == "A" {
					filterInt = dns.TypeA
				} else if filterType == "MX" {
					filterInt = dns.TypeMX
				} else if filterType == "TXT" {
					filterInt = dns.TypeTXT
				} else if filterType == "CNAME" {
					filterInt = dns.TypeCNAME
				}
			}

			var exactHostMatching bool

			// ignore matching criteria if the filterType is set to ALL
			if filterType == "ALL" {
				exactHostMatching = false
			} else {

				matchingType, matchingTypeExists := filterMap["matching"].(string)

				if matchingTypeExists {
					if matchingType == "contains" {
						exactHostMatching = false
					} else if matchingType == "exact" {
						exactHostMatching = true
					} else {
						errorMessage := fmt.Sprintf("%s is an invalid matching type.  Filter matching for host: %s must be either \"contains\" or \"exact\"",
							matchingType, filterHost)
						return nil, errors.New(errorMessage)
					}
				} else {
					errorMessage := fmt.Sprintf("Filter is required matching for host: %s must be either \"contains\" or \"exact\"",
						filterHost)
					return nil, errors.New(errorMessage)
				}

			}

			// if we get here, then we have validated the filter
			dnsFilter := DNSFilter{
				filterHost,
				filterInt,
				exactHostMatching,
			}

			dnsFilters = append(dnsFilters, dnsFilter)
		}

	}

	config.Filters = dnsFilters

	logFilePath, logFilePathExists := configMap["logfile"].(string)

	if !logFilePathExists {
		// default to filter-dns.log in the current working directory
		currentDirectory, dirErr := os.Getwd()

		if dirErr != nil {
			return nil, dirErr
		}

		logFilePath = path.Join(currentDirectory, "filter-dns.log")
	}

	setupLogging(logFilePath)

	return config, nil

}
