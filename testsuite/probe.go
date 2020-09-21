//
// Copyright 2014-2020 Cristian Maglie. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//

package testsuite

import (
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.bug.st/serial"
)

// Probe is a board that can perform some actions to a Target
// board that is under testing
type Probe struct {
	port      serial.Port
	targetVid string
	targetPid string
}

// ConnectToProbe attempts to connect to a Probe board. The
// Probe is identified through VID and PID. Default USB ID is
// 2341:8037.
func ConnectToProbe(t *testing.T, vid, pid, targetVid, targetPid string) *Probe {
	log.Println("PR - Connecting to Probe")
	portName, err := FindPortWithVIDPID(vid, pid)
	require.NoError(t, err, "Could not search for probe")
	require.NotEmpty(t, portName, "Probe not found")

	port, err := serial.Open(portName, &serial.Mode{})
	require.NoError(t, err, "Could not connect to probe")

	//time.Sleep(time.Millisecond * 2000)
	return &Probe{
		port:      port,
		targetVid: targetVid,
		targetPid: targetPid,
	}
}

// ConnectToTarget attempts to connect to the Target board.
func (probe *Probe) ConnectToTarget(t *testing.T) serial.Port {
	log.Println("TR - Connecting to Target")

	portName, err := PollToFindPortWithVIDPID(probe.targetVid, probe.targetPid, 15*time.Second, 500*time.Millisecond)
	require.NoError(t, err, "Could not search for target")
	require.NotEmpty(t, portName, "Target not found")
	port, err := serial.Open(portName, &serial.Mode{})
	require.NoError(t, err, "Could not connect to target")
	return port
}

// Close terminates the connection to the Probe. The Target
// board is turned off if it was previously turned on.
func (probe *Probe) Close() error {
	probe.TurnOffTarget()
	log.Println("PR - Disconnecting Probe")
	return probe.port.Close()
}

// TurnOnTarget turns on the Target board.
func (probe *Probe) TurnOnTarget() error {
	log.Println("PR - Turn ON target")
	return probe.sendCommand('1')
}

// TurnOffTarget turns off the Target board.
func (probe *Probe) TurnOffTarget() error {
	log.Println("PR - Turn OFF target")
	err := probe.sendCommand('0')
	if err != nil {
		// give some time to the Target to fully disconnect
		time.Sleep(100 * time.Millisecond)
	}
	return err
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
