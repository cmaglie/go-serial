//
// Copyright 2014-2017 Cristian Maglie. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//

package testsuite // import "go.bug.st/serial.v1/testsuite"

import (
	"log"

	"go.bug.st/serial.v1/enumerator"
)

// FindPortWithVIDPID attempts to retrieve the port with the
// specified USB ID. If the port is not found an empty string
// string is returned.
func FindPortWithVIDPID(vid, pid string) (string, error) {
	ports, err := enumerator.GetDetailedPortsList()
	if err != nil {
		return "", err
	}
	log.Printf("     > Searching for port %s:%s\n", vid, pid)
	for _, port := range ports {
		if port.IsUSB {
			log.Printf("       Detected port '%s' %s:%s\n", port.Name, port.VID, port.PID)
			if port.VID == vid && port.PID == pid {
				log.Printf("       Using '%s'", port.Name)
				return port.Name, nil
			}
		}
	}
	return "", nil
}
