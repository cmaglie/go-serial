//
// Copyright 2014-2020 Cristian Maglie. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//

package testsuite

import (
	"log"
	"time"

	"go.bug.st/serial/enumerator"
)

// PollToFindPortWithVIDPID attempts to retrieve the port with the
// specified USB ID. Many attempts are made every pollInterval.
// If the timeout passes and a port is not found an empty string
// string is returned.
func PollToFindPortWithVIDPID(vid, pid string, timeout, pollInterval time.Duration) (string, error) {
	log.Printf("     > Searching for port %s:%s\n", vid, pid)
	for ; timeout > pollInterval; timeout -= pollInterval {
		portName, err := FindPortWithVIDPID(vid, pid)
		if err != nil {
			return "", err
		}
		if portName != "" {
			return portName, nil
		}
		time.Sleep(pollInterval)
	}
	return "", nil
}

// FindPortWithVIDPID attempts to retrieve the port with the
// specified USB ID. If the port is not found an empty string
// string is returned.
func FindPortWithVIDPID(vid, pid string) (string, error) {
	ports, err := enumerator.GetDetailedPortsList()
	if err != nil {
		return "", err
	}
	for _, port := range ports {
		if port.IsUSB {
			if port.VID == vid && port.PID == pid {
				log.Printf("       Detected port '%s' %s:%s\n", port.Name, port.VID, port.PID)
				return port.Name, nil
			}
		}
	}
	return "", nil
}
