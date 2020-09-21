//
// Copyright 2014-2020 Cristian Maglie. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//

package testsuite

import (
	"errors"
	"log"
	"regexp"
	"strconv"
	"strings"
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

func TestModemBitsAndPortSpeedChange(t *testing.T) {
	probe := NewProbe(t, 20*time.Second)
	defer func() {
		log.Print("T1 - Completed")
		probe.Completed()
	}()

	probe.TurnOnTarget()
	target := probe.ConnectToTarget(t)
	defer target.Close()

	// Modem bit test
	assertTargetSerialStatus := func(exBps int, exDtr, exRts bool) {
		log.Printf("T1 - Acquire port config from target")
		n, err := target.Write([]byte("M")) // 'M' ask the target to report modem bit status and serial speed
		require.NoError(t, err, "sending command to report modem bit")
		require.Equal(t, 1, n, "number of bytes sent")
		time.Sleep(100 * time.Millisecond) // wait 100 ms to get a response

		buff := make([]byte, 1024)
		n, err = target.Read(buff)
		require.NoError(t, err)
		bps, dtr, rts, err := parseTargetSerialStatus(buff[:n])
		require.NoError(t, err)
		require.Equal(t, exBps, bps)
		require.Equal(t, exDtr, dtr)
		require.Equal(t, exRts, rts)
	}

	log.Printf("T1 - Set target DTR=1 and RTS=1")
	require.NoError(t, target.SetDTR(true))
	require.NoError(t, target.SetRTS(true))
	assertTargetSerialStatus(9600, true, true)
	require.NoError(t, target.SetDTR(false))
	assertTargetSerialStatus(9600, false, true)
	require.NoError(t, target.SetDTR(true))
	assertTargetSerialStatus(9600, true, true)
	require.NoError(t, target.SetRTS(false))
	assertTargetSerialStatus(9600, true, false)
	require.NoError(t, target.SetRTS(true))
	assertTargetSerialStatus(9600, true, true)
	require.NoError(t, target.SetMode(&serial.Mode{BaudRate: 115200}))
	assertTargetSerialStatus(115200, true, true)
	target.SetMode(&serial.Mode{BaudRate: 38400})
	assertTargetSerialStatus(38400, true, true)
}

var serialReportRE = regexp.MustCompile("BPS=([0-9]+) DTR=([01]) RTS=([01])")

func parseTargetSerialStatus(buff []byte) (bps int, dtr, rts bool, err error) {
	line := strings.TrimSpace(string(buff))
	match := serialReportRE.FindAllStringSubmatch(line, 1)
	if len(match) == 0 || len(match[0]) != 4 {
		err = errors.New("invalid serial status report from target")
		return
	}
	fields := match[0]
	bps, err = strconv.Atoi(fields[1])
	if err != nil {
		err = errors.New("invalid BPS report from target: " + err.Error())
		return
	}
	dtr = (fields[2][0] == '1')
	rts = (fields[3][0] == '1')
	return
}
