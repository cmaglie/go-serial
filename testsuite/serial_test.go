//
// Copyright 2014-2017 Cristian Maglie. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//

package testsuite // import "go.bug.st/serial.v1/testsuite"

import (
	"testing"
	"time"

	"go.bug.st/serial.v1"

	"log"

	"github.com/stretchr/testify/require"
)

func startTest(t *testing.T, timeout time.Duration, probe *Probe) {
	go func() {
		time.Sleep(timeout)
		probe.TurnOffTarget()
		probe.Close()
		require.FailNow(t, "Test timed-out")
	}()
	log.Printf("Starting test (timeout %s)", timeout)
}

func errString(err error) string {
	if err == nil {
		return "nil"
	}
	return err.Error()
}

func TestConcurrentReadAndWrite(t *testing.T) {
	// https://github.com/bugst/go-serial/issues/15

	probe := connectToProbe(t)
	defer probe.Close()

	probe.TurnOnTarget()
	target := probe.ConnectToTarget(t)
	defer target.Close()

	startTest(t, 10*time.Second, probe)

	// Try to send while a receive is waiting for data

	// Make a blocking Recv call
	go func() {
		log.Printf("T1 - Waiting on Read()")
		buff := make([]byte, 1024)
		_, err := target.Read(buff) // blocking read
		log.Printf("T1 - Returned from read. err=%s", err.Error())

		// if there are no errors then the Read call completed successfully
		// and did not block
		require.NotNil(t, err, "Read did not block")

		// fail if an error different from PortClosed happens
		portError, ok := err.(*serial.PortError)
		require.True(t, ok, "Unexpected error during read: %s", err.Error())
		require.Equal(t, serial.PortClosed, portError.Code(), "Unexpected error during read: %s", err.Error())
	}()

	// Try to send a byte each `delay` milliseconds and check if the
	// total elapsed time is in the expected range (with a `delay` ms margin)
	delay := time.Millisecond * 20
	expected := delay * 5
	epsilon := delay

	start := time.Now()
	for i := 0; i < 5; i++ {
		time.Sleep(delay)
		log.Printf("T2 - Sending 1 byte...")
		target.Write([]byte{' '})
	}
	elapsed := time.Since(start)
	log.Printf("T2 - Done sending. elapsed/expected=%s/%s", elapsed, expected)
	require.InDelta(t, expected.Seconds(), elapsed.Seconds(), epsilon.Seconds())
}

func TestDisconnectingPortDetection(t *testing.T) {
	probe := connectToProbe(t)
	defer probe.Close()

	probe.TurnOnTarget()
	target := probe.ConnectToTarget(t)
	defer target.Close()

	startTest(t, 10*time.Second, probe)

	// Disconnect target after a small delay
	go func() {
		log.Printf("T1 - Delay 500ms before disconnecting target")
		time.Sleep(500 * time.Millisecond)
		log.Printf("T1 - Disconnect target")
		probe.TurnOffTarget()
	}()

	// Do a blocking Read that should return after the target disconnection
	log.Printf("T2 - Make a Read call")
	buff := make([]byte, 1024)
	n, err := target.Read(buff)
	log.Printf("T2 - Read returned: n=%d err=%s", n, errString(err))

	log.Printf("T2 - Make another Read call")
	n, err = target.Read(buff)
	log.Printf("T2 - Read returned: n=%d err=%s", n, errString(err))
	require.NotNil(t, err, "Read did not block")

	portError, ok := err.(*serial.PortError)
	require.True(t, ok, "Unexpected error during read: %s", err.Error())
	require.Equal(t, serial.PortClosed, portError.Code(), "Unexpected error during read: %s", err.Error())
}
