//
// Firmware for the Probe board.
//
// Copyright 2014-2020 Cristian Maglie. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//
// This firmware is part of the testsuite of http://go.bug.st/serial project.
//
// The Probe is able to perform some actions on the Target board:
// - Power on/off to simulate connect/disconnect.
// - TODO: Parse and report information coming from the Target board through
//   the UART (if the Target is connected).

// The Probe answers with a ">" prompt when it's ready to accept
// commands.
// The available commands are:
// '0' - Turn off the target board
// '1' - Turn on the target board
// 'V' - Prints a brief information about this firmware
// '?' - Prints a list of the available commands

const int TARGET_BOARD_CTL_PIN = 2;
extern const DeviceDescriptor USB_DeviceDescriptor;

void setup() {
  pinMode(TARGET_BOARD_CTL_PIN, OUTPUT);
  turnOffTarget();
  Serial.begin(9600);
}

void turnOnTarget() {
  digitalWrite(2, LOW);
}

void turnOffTarget() {
  digitalWrite(2, HIGH);
}

void loop() {
  Serial.print(">");
  while (true) {
    int c = Serial.peek();
    if (c != '\n' && c != '\r') {
      break;
    }
    Serial.read(); // eat new line
  }

  int c;
  do {
    c = Serial.read();
  } while (c == -1);
  Serial.println();

  if (c == 'v' || c == 'V' || c == '?') {
    Serial.println("Probe Firmware v1.0.0 [01V?]");
    if (c == 'v' || c == 'V') {
      return;
    }
    Serial.println("Copyright 2014-2020 Cristian Maglie. All rights reserved.");
    Serial.println();
    Serial.println("'0' - Turn off the target board");
    Serial.println("'1' - Turn on the target board");
    Serial.println("'V' - Prints a brief information about this firmware");
    Serial.println("'?' - Prints a list of the available commands");
    return;
  }
  if (c == '1') {
    turnOnTarget();
    Serial.println("1");
    return;
  }
  if (c == '0') {
    turnOffTarget();
    Serial.println("0");
    return;
  }
}

