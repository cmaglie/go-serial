//
// Copyright 2014-2017 Cristian Maglie. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//

package testsuite // import "go.bug.st/serial.v1/testsuite"

import (
	"fmt"
	"log"
	"testing"

	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.bug.st/serial.v1"
	"go.bug.st/serial.v1/enumerator"
)

type Probe struct {
	port serial.Port
}

func connectToProbe(t *testing.T) *Probe {
	log.Println("PR - Connecting to Probe")
	portName, err := FindPortWithVIDPID("2341", "8037")
	require.NoError(t, err, "Could not search for probe")
	require.NotEmpty(t, portName, "Probe not found")

	port, err := serial.Open(portName, &serial.Mode{})
	require.NoError(t, err, "Could not connect to probe")

	//time.Sleep(time.Millisecond * 2000)
	return &Probe{port: port}
}

func (_ *Probe) ConnectToTarget(t *testing.T) serial.Port {
	log.Println("TR - Connecting to Target")
	for i := 0; i < 10; i++ {
		portName, err := FindPortWithVIDPID("2341", "8036")
		require.NoError(t, err, "Could not search for target")
		if portName == "" {
			time.Sleep(time.Millisecond * 500)
			continue
		}
		port, err := serial.Open(portName, &serial.Mode{})
		require.NoError(t, err, "Could not connect to target")
		return port
	}
	assert.FailNow(t, "Target not found")
	return nil // Should never be reached...
}

func (probe *Probe) Close() error {
	probe.TurnOffTarget()
	log.Println("PR - Disconnecting Probe")
	return probe.port.Close()
}

func (probe *Probe) TurnOnTarget() error {
	log.Println("PR - Turn ON target")
	return probe.sendCommand('1')
}

func (probe *Probe) TurnOffTarget() error {
	log.Println("PR - Turn OFF target")
	return probe.sendCommand('0')
}

func (probe *Probe) sendCommand(cmd byte) error {
	_, err := probe.port.Write([]byte{cmd})
	if err != nil {
		return fmt.Errorf("Communication error: %s", err)
	}
	buff := make([]byte, 1)
	n, err := probe.port.Read(buff)
	if err != nil {
		return fmt.Errorf("Communication error: %s", err)
	}
	if n != 1 || buff[0] != cmd {
		return fmt.Errorf("Communication error")
	}
	return nil
}

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
