//
// Copyright 2014-2017 Cristian Maglie. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//

package testsuite // import "go.bug.st/serial.v1/testsuite"
import (
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Test is a wrapper for a single test of the testsuite.
// It handles timeouts and part of the resources allocation.
type Test struct {
	end     chan bool
	ended   chan bool
	probe   *Probe
	timeout time.Duration
	t       *testing.T
}

// StartTest begin a test with the specified timeout.
func StartTest(t *testing.T, timeout time.Duration) (*Test, *Probe) {
	test := &Test{t: t, timeout: timeout}
	test.end = make(chan bool)
	test.ended = make(chan bool)

	probe := ConnectToProbe(t)
	test.probe = probe

	go testTimeoutHandler(test)
	log.Printf("Starting test (timeout %s)", timeout)
	return test, probe
}

func testTimeoutHandler(test *Test) {
	select {
	case <-test.end:
		// Test ended before timeout
		log.Printf("Test ended before timeout")
	case <-time.After(test.timeout):
		log.Printf("Test timed-out")
		assert.Fail(test.t, "Test timed-out")
	}
	test.probe.TurnOffTarget()
	test.probe.Close()
	test.ended <- true
}

// Completed must be called when the test ends before the
// timeout. This doesn't mean that the test is successful
// but just that the test ended before the timeout and the
// used resources can be freed.
func (test *Test) Completed() {
	select {
	case <-test.ended:
		// test already timed out, do nothing
	default:
		test.end <- true
		<-test.ended
	}
}
