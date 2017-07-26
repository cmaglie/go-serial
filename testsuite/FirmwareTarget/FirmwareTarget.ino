//
// Firmware for the Target board.
//
// Copyright 2014-2020 Cristian Maglie. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//
// This firmware is part of the testsuite of http://go.bug.st/go-serial project.
//
// The Target board can perform various test based on the first character read:
// - 'E': perform echo test.
// - 'M': perform USB-CDC configuration report test.
//        Every change to set speed via USB-CDC is reported as well as DTR and RTS bit.
// TODO: tests will be added as needed.

void setup() {
  Serial.begin(9600);
}

void loop() {
  int c = Serial.read();
  if (c == 'E')
    echoTest();
  if (c == 'M')
    modemBitsTest();
  if (c == -1)
    return;
}

void echoTest() {
  while (true) {
    int c = Serial.read();
    if (c == -1)
      continue;
    Serial.print((char) c);
  }
}

void modemBitsTest() {
  while (true) {
    bool dtr = Serial.dtr();
    bool rts = Serial.rts();
    unsigned long baud = Serial.baud();
    Serial.print("BPS=");
    Serial.print(baud);
    Serial.print(" DTR=");
    Serial.print(dtr ? '1' : '0');
    Serial.print(" RTS=");
    Serial.println(rts ? '1' : '0');
    delay(250);
  }
}

