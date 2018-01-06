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

	"github.com/arduino/go-properties-orderedmap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	config, err := properties.Load("testsuite.config")
	require.NoError(t, err, "Loading testsuite configuration")

	test := &Test{t: t, timeout: timeout}
	test.end = make(chan bool)
	test.ended = make(chan bool)
	test.probe = ConnectToProbe(t, config.Get("probe.vid"), config.Get("probe.pid"), config.Get("target.vid"), config.Get("target.pid"))

	go testTimeoutHandler(test)
	log.Printf("Starting test (timeout %s)", timeout)

	return test, test.probe
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
