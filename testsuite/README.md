# Testsuite for Golang library go.bug.st/serial

This repository contains a set of regression tests that runs on real hardware serial ports.

Some tests require phisical disconnection of the hardware to check if the USB port disconnection is correctly detected by the library: a special testing harness is required to perform this tests.

## Testing harness

The testing harness is composed by two microcontroller boards and a power switch (relay).

The first board is called the "Probe" board, it runs a firmware to control the switch and is always connected to the host PC.

The second board is called the "Target" board, it is connected to the host PC via a USB cable whose power wire goes trough the switch controlled by the Probe.

![Harness Diagram](harness_diagram.png)

With this setup we can programmatically control the connection/disconnection of the Target.

Here a picture of my setup:

![Harness Picture](harness_picture.png)

It's really simple and made with cheap components readily available.

## Testing libraries

The Probe and the Target firmwares are available inside the folders `FirmwareProbe` and `FirmwareTarget`, they are Arduino sketchtes that should be loaded respectively in an Arduino Micro board and an Arduino Leonardo board (but any other Arduino-compatible board should works as well).

Two different boards have been choosen so they can be uniquely identified via USB VID/PID. The VID/PID of the Probe and the Target can be configured in the `testsuite.config` file.

## Runnning the testsuite

Just run `go test` as usual.
