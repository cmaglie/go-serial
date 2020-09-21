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

	"github.com/arduino/go-properties-orderedmap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.bug.st/serial"
)

// Probe is a wrapper for a single test of the testsuite.
// It handles timeouts and part of the resources allocation.
type Probe struct {
	end       chan bool
	ended     chan bool
	port      serial.Port
	targetVid string
	targetPid string
	timeout   time.Duration
	t         *testing.T
}

// NewProbe begin a test with the specified timeout.
func NewProbe(t *testing.T, timeout time.Duration) *Probe {
	log.Println("Starting test using probe")
	config, err := properties.Load("testsuite.config")
	require.NoError(t, err, "Loading testsuite configuration")

	log.Println("PR - Connecting to Probe")
	portName, err := FindPortWithVIDPID(config.Get("probe.vid"), config.Get("probe.pid"))
	require.NoError(t, err, "Could not search for probe")
	require.NotEmpty(t, portName, "Probe not found")

	port, err := serial.Open(portName, &serial.Mode{})
	if portErr, ok := err.(*serial.PortError); ok && (portErr.Code() == serial.PermissionDenied || portErr.Code() == serial.PortBusy) {
		log.Println("PR - Port busy... waiting 1 sec and retry")
		time.Sleep(time.Second)
		port, err = serial.Open(portName, &serial.Mode{})
	}
	require.NoError(t, err, "Could not connect to probe")

	//time.Sleep(time.Millisecond * 2000)
	test := &Probe{
		t:         t,
		timeout:   timeout,
		end:       make(chan bool),
		ended:     make(chan bool),
		port:      port,
		targetVid: config.Get("target.vid"),
		targetPid: config.Get("target.pid"),
	}

	go test.testTimeoutHandler()
	log.Printf("Starting test (timeout %s)", timeout)

	return test
}

func (test *Probe) testTimeoutHandler() {
	select {
	case <-test.end:
		// Test ended before timeout
		log.Printf("Test ended before timeout")
	case <-time.After(test.timeout):
		log.Printf("Test timed-out")
		assert.Fail(test.t, "Test timed-out")
	}
	test.TurnOffTarget()
	log.Println("PR - Disconnecting Probe")
	test.port.Close()
	test.ended <- true
}

// TurnOnTarget turns on the Target board.
func (test *Probe) TurnOnTarget() error {
	log.Println("PR - Turn ON target")
	return test.sendCommand('1')
}

// TurnOffTarget turns off the Target board.
func (test *Probe) TurnOffTarget() error {
	log.Println("PR - Turn OFF target")
	err := test.sendCommand('0')
	if err != nil {
		// give some time to the Target to fully disconnect
		time.Sleep(100 * time.Millisecond)
	}
	return err
}

func (test *Probe) sendCommand(cmd byte) error {
	if n, err := test.port.Write([]byte{cmd}); n != 1 || err != nil {
		return fmt.Errorf("Communication error: %s", err)
	}
	buff := make([]byte, 1)
	if n, err := test.port.Read(buff); err != nil {
		return fmt.Errorf("Communication error: %s", err)
	} else if n != 1 || buff[0] != cmd {
		return fmt.Errorf("Communication error")
	}
	return nil
}

// ConnectToTarget attempts to connect to the Target board.
func (test *Probe) ConnectToTarget(t *testing.T) serial.Port {
	log.Println("TR - Connecting to Target")

	portName, err := PollToFindPortWithVIDPID(test.targetVid, test.targetPid, 15*time.Second, 500*time.Millisecond)
	require.NoError(t, err, "Could not search for target")
	require.NotEmpty(t, portName, "Target not found")
	port, err := serial.Open(portName, &serial.Mode{})
	if portErr, ok := err.(*serial.PortError); ok && (portErr.Code() == serial.PermissionDenied || portErr.Code() == serial.PortBusy) {
		log.Println("TR - Port busy... waiting 1 sec and retry")
		time.Sleep(time.Second)
		port, err = serial.Open(portName, &serial.Mode{})
	}
	require.NoError(t, err, "Could not connect to target")
	return port
}

// Completed must be called when the test ends before the
// timeout. This doesn't mean that the test is successful
// but just that the test ended before the timeout and the
// used resources can be freed.
func (test *Probe) Completed() {
	select {
	case <-test.ended:
		// test already timed out, do nothing
	default:
		test.end <- true
		<-test.ended
	}
	log.Println("Test ended")
}
