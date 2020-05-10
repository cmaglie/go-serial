//
// Copyright 2014-2020 Cristian Maglie. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//

package testsuite

import (
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.bug.st/serial"
)

func TestConcurrentReadAndWrite(t *testing.T) {
	probe := NewProbe(t, 20*time.Second)
	defer func() {
		log.Print("T1 - Completed")
		probe.Completed()
	}()

	probe.TurnOnTarget()
	target := probe.ConnectToTarget(t)
	defer target.Close()

	// Try to send while a receive is waiting for data
	// https://github.com/bugst/go-serial/issues/15

	// Make a blocking Recv call
	done := make(chan bool)
	go func() {
		defer func() {
			log.Print("T2 - Completed")
			done <- true
		}()

		log.Print("T2 - Waiting on Read()")
		buff := make([]byte, 1024)
		n, err := target.Read(buff) // blocking read
		log.Printf("T2 - Returned from read. n=%d err=%v", n, err)

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
		log.Printf("T1 - Sending 1 byte...")
		target.Write([]byte{' '})
	}
	elapsed := time.Since(start)
	log.Printf("T1 - Done sending. elapsed/expected=%s/%s", elapsed, expected)
	require.InDelta(t, expected.Seconds(), elapsed.Seconds(), epsilon.Seconds())
	target.Close()

	// Wait for goroutines completion and cleanup
	<-done
}

func TestDisconnectingPortDetection(t *testing.T) {
	probe := NewProbe(t, 20*time.Second)
	defer func() {
		log.Print("T1 - Completed")
		probe.Completed()
	}()

	probe.TurnOnTarget()
	target := probe.ConnectToTarget(t)
	defer target.Close()

	// Disconnect target after a small delay
	done := make(chan bool)
	go func() {
		defer func() {
			log.Print("T2 - Completed")
			done <- true
		}()

		log.Printf("T2 - Delay 200ms before disconnecting target")
		time.Sleep(200 * time.Millisecond)
		log.Printf("T2 - Disconnect target")
		probe.TurnOffTarget()
	}()

	// Do a blocking Read that should return after the target disconnection
	log.Printf("T1 - Make a Read call")
	buff := make([]byte, 1024)
	n, err := target.Read(buff)

	log.Printf("T1 - Read returned: n=%d err=%v", n, err)
	require.Error(t, err, "Read returned no errors")
	require.Equal(t, 0, n, "Read has returned some bytes")

	// Wait for goroutines completion and cleanup
	<-done
}

func TestFlushRXSerialBuffer(t *testing.T) {
	probe := NewProbe(t, 20*time.Second)
	defer func() {
		log.Print("T1 - Completed")
		probe.Completed()
	}()

	probe.TurnOnTarget()
	target := probe.ConnectToTarget(t)
	defer target.Close()

	// Send a bunch of data to the Target
	log.Printf("T1 - Starting echo test and sending 'HELLO!' to the target")
	n, err := target.Write([]byte("EHELLO!")) // 'E' starts echo test, 'HELLO!' should be repeated
	require.NoError(t, err, "Error sending data to be echoed")
	require.Equal(t, 7, n, "Write sent a wrong number of bytes")

	// Wait a bit to receive data back
	log.Printf("T1 - Waiting a bit to receive data back")
	time.Sleep(100 * time.Millisecond)

	// Read the first echoed char
	log.Printf("T1 - Reading the first echoed char (should be 'H')")
	buff := make([]byte, 1)
	n, err = target.Read(buff)
	require.NoError(t, err, "Error reading echoed data")
	require.Equal(t, 1, n, "Read received less bytes than expected")
	require.Equal(t, byte('H'), buff[0], "Incorrect data received")

	// Flush buffers
	log.Printf("T1 - Flushing read buffer...")
	err = target.ResetInputBuffer()
	require.NoError(t, err, "Error flushing rx buffer")

	// Send other data
	log.Printf("T1 - Sending 'X' to target")
	n, err = target.Write([]byte("X"))
	require.NoError(t, err, "Error sending data to be echoed")
	require.Equal(t, 1, n, "Write sent a wrong number of bytes")

	// Wait a bit to receive data back
	log.Printf("T1 - Waiting a bit to receive data back")
	time.Sleep(100 * time.Millisecond)

	// Read the first echoed char
	log.Printf("T1 - Reading the first echoed char (should be 'X', and 'ELLO!' should be discarded)")
	n, err = target.Read(buff)
	require.NoError(t, err, "Error reading echoed data")
	require.Equal(t, 1, n, "Read received less bytes than expected")
	require.Equal(t, byte('X'), buff[0], "Incorrect data received")
}
