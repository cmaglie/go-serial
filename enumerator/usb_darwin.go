//
// Copyright 2014-2026 Cristian Maglie. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//

package enumerator

// #cgo LDFLAGS: -framework CoreFoundation -framework IOKit
// #include <IOKit/IOKitLib.h>
// #include <IOKit/IOCFPlugIn.h>
// #include <IOKit/usb/IOUSBLib.h>
// #include <CoreFoundation/CoreFoundation.h>
// #include <stdlib.h>
import "C"
import (
	"errors"
	"fmt"
	"time"
	"unsafe"

	"github.com/ebitengine/purego"
)

type machPort uint32
type kernReturn int32
type ioReturn int32
type hResult int32
type uLong uint32
type ioObject uint32
type ioIterator uint32
type ioRegistryEntry ioObject
type ioService ioObject
type cfUUIDBytes struct {
	byte0  uint8
	byte1  uint8
	byte2  uint8
	byte3  uint8
	byte4  uint8
	byte5  uint8
	byte6  uint8
	byte7  uint8
	byte8  uint8
	byte9  uint8
	byte10 uint8
	byte11 uint8
	byte12 uint8
	byte13 uint8
	byte14 uint8
	byte15 uint8
}
type ioUSBDevRequest struct {
	bmRequestType uint8
	bRequest      uint8
	wValue        uint16
	wIndex        uint16
	wLength       uint16
	pData         unsafe.Pointer
	wLenDone      uint32
}
type ioUSBConfigurationDescriptor struct {
	bLength             uint8
	bDescriptorType     uint8
	wTotalLength        uint16
	bNumInterfaces      uint8
	bConfigurationValue uint8
	iConfiguration      uint8
	bmAttributes        uint8
	maxPower            uint8
}
type cfAllocatorRef uintptr
type cfIndex int64
type cfStringEncoding uint32
type cfNumberType int32
type cfNumberRef uintptr
type cfUUIDRef uintptr
type cfStringRef uintptr
type cfTypeRef uintptr
type cfMutableDictionaryRef uintptr
type cfDictionaryRef uintptr
type ioCFPlugInInterface struct {
	reserved       uintptr
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
}
type ioUSBDeviceInterface struct {
	reserved                      uintptr
	QueryInterface                uintptr
	AddRef                        uintptr
	Release                       uintptr
	CreateDeviceAsyncEventSource  uintptr
	GetDeviceAsyncEventSource     uintptr
	CreateDeviceAsyncPort         uintptr
	GetDeviceAsyncPort            uintptr
	USBDeviceOpen                 uintptr
	USBDeviceClose                uintptr
	GetDeviceClass                uintptr
	GetDeviceSubClass             uintptr
	GetDeviceProtocol             uintptr
	GetDeviceVendor               uintptr
	GetDeviceProduct              uintptr
	GetDeviceReleaseNumber        uintptr
	GetDeviceAddress              uintptr
	GetDeviceBusPowerAvailable    uintptr
	GetDeviceSpeed                uintptr
	GetNumberOfConfigurations     uintptr
	GetLocationID                 uintptr
	GetConfigurationDescriptorPtr uintptr
	GetConfiguration              uintptr
	SetConfiguration              uintptr
	GetBusFrameNumber             uintptr
	ResetDevice                   uintptr
	DeviceRequest                 uintptr
}

const (
	kIOMasterPortDefault      machPort         = 0
	kernSuccess               kernReturn       = 0
	kIOReturnSuccess          ioReturn         = 0
	sOK                       hResult          = 0
	kUSBIn                    uint8            = 1
	kUSBStandard              uint8            = 0
	kUSBDevice                uint8            = 0
	kUSBRqGetDescriptor       uint8            = 6
	kUSBStringDesc            uint8            = 3
	kCFAllocatorDefault       cfAllocatorRef   = 0
	kCFStringEncodingMacRoman cfStringEncoding = 0
	kCFStringEncodingUTF8     cfStringEncoding = 0x08000100
	kCFStringEncodingUTF16LE  cfStringEncoding = 0x14000100
	kCFNumberSInt16Type       cfNumberType     = 2
)

var kIOUSBDeviceInterfaceID = cfUUIDBytes{
	byte0: 0x5c, byte1: 0x81, byte2: 0x87, byte3: 0xd0,
	byte4: 0x9e, byte5: 0xf3, byte6: 0x11, byte7: 0xd4,
	byte8: 0x8b, byte9: 0x45, byte10: 0x00, byte11: 0x0a,
	byte12: 0x27, byte13: 0x05, byte14: 0x28, byte15: 0x61,
}

const (
	kIOUSBDeviceInterfaceIDLo uint64 = 0xd411f39ed087815c
	kIOUSBDeviceInterfaceIDHi uint64 = 0x612805270a00458b
)

var (
	ioServiceMatching             func(string) cfMutableDictionaryRef
	ioServiceGetMatchingServices  func(machPort, cfDictionaryRef, *ioIterator) kernReturn
	ioObjectRelease               func(ioObject) kernReturn
	ioObjectGetClass              func(ioObject, *byte) kernReturn
	ioIteratorIsValid             func(ioIterator) uint8
	ioIteratorReset               func(ioIterator) kernReturn
	ioIteratorNext                func(ioIterator) ioObject
	cfRelease                     func(cfTypeRef)
	cfStringGetLength             func(cfStringRef) cfIndex
	cfStringGetMaximumSize        func(cfIndex, cfStringEncoding) cfIndex
	cfStringGetCString            func(cfStringRef, *byte, cfIndex, cfStringEncoding) uint8
	cfStringGetCStringPtr         func(cfStringRef, cfStringEncoding) *byte
	cfStringCreateWithCString     func(cfAllocatorRef, string, cfStringEncoding) cfStringRef
	cfStringCreateWithBytesFunc   func(cfAllocatorRef, *uint8, cfIndex, cfStringEncoding, uint8) cfStringRef
	cfNumberGetValue              func(cfNumberRef, cfNumberType, unsafe.Pointer) uint8
	ioRegistryEntryGetParent      func(ioRegistryEntry, string, *ioRegistryEntry) kernReturn
	ioRegistryEntryCreateProperty func(ioRegistryEntry, cfStringRef, cfAllocatorRef, uint32) cfTypeRef
	ioCreatePlugInInterface       func(ioService, cfUUIDRef, cfUUIDRef, unsafe.Pointer, *int32) ioReturn
)

func init() {
	coreFoundation := mustDlopen("/System/Library/Frameworks/CoreFoundation.framework/CoreFoundation")
	ioKit := mustDlopen("/System/Library/Frameworks/IOKit.framework/IOKit")

	purego.RegisterLibFunc(&cfRelease, coreFoundation, "CFRelease")
	purego.RegisterLibFunc(&cfStringGetLength, coreFoundation, "CFStringGetLength")
	purego.RegisterLibFunc(&cfStringGetMaximumSize, coreFoundation, "CFStringGetMaximumSizeForEncoding")
	purego.RegisterLibFunc(&cfStringGetCString, coreFoundation, "CFStringGetCString")
	purego.RegisterLibFunc(&cfStringGetCStringPtr, coreFoundation, "CFStringGetCStringPtr")
	purego.RegisterLibFunc(&cfStringCreateWithCString, coreFoundation, "CFStringCreateWithCString")
	purego.RegisterLibFunc(&cfStringCreateWithBytesFunc, coreFoundation, "CFStringCreateWithBytes")
	purego.RegisterLibFunc(&cfNumberGetValue, coreFoundation, "CFNumberGetValue")
	purego.RegisterLibFunc(&ioRegistryEntryGetParent, ioKit, "IORegistryEntryGetParentEntry")
	purego.RegisterLibFunc(&ioRegistryEntryCreateProperty, ioKit, "IORegistryEntryCreateCFProperty")
	purego.RegisterLibFunc(&ioCreatePlugInInterface, ioKit, "IOCreatePlugInInterfaceForService")
	purego.RegisterLibFunc(&ioServiceMatching, ioKit, "IOServiceMatching")
	purego.RegisterLibFunc(&ioServiceGetMatchingServices, ioKit, "IOServiceGetMatchingServices")
	purego.RegisterLibFunc(&ioObjectRelease, ioKit, "IOObjectRelease")
	purego.RegisterLibFunc(&ioObjectGetClass, ioKit, "IOObjectGetClass")
	purego.RegisterLibFunc(&ioIteratorIsValid, ioKit, "IOIteratorIsValid")
	purego.RegisterLibFunc(&ioIteratorReset, ioKit, "IOIteratorReset")
	purego.RegisterLibFunc(&ioIteratorNext, ioKit, "IOIteratorNext")
}

func mustDlopen(path string) uintptr {
	handle, err := purego.Dlopen(path, purego.RTLD_NOW|purego.RTLD_GLOBAL)
	if err != nil {
		panic(fmt.Sprintf("dlopen %s failed: %v", path, err))
	}
	return handle
}

func cStringToGo(ptr *byte) string {
	if ptr == nil {
		return ""
	}
	n := 0
	for *(*byte)(unsafe.Add(unsafe.Pointer(ptr), n)) != 0 {
		n++
	}
	return unsafe.String(ptr, n)
}

func nativeGetDetailedPortsList() ([]*PortDetails, error) {
	var ports []*PortDetails

	services, err := getAllServices("IOSerialBSDClient")
	if err != nil {
		return nil, &PortEnumerationError{causedBy: err}
	}
	defer func() {
		for _, service := range services {
			service.Release()
		}
	}()

	for _, service := range services {
		port, err := extractPortInfo(io_registry_entry_t(service))
		if err != nil {
			return nil, &PortEnumerationError{causedBy: err}
		}
		ports = append(ports, port)
	}
	return ports, nil
}

func extractPortInfo(service io_registry_entry_t) (*PortDetails, error) {
	port := &PortDetails{}
	// If called too early the port may still not be ready or fully enumerated
	// so we retry 5 times before returning error.
	for retries := 5; retries > 0; retries-- {
		name, err := service.GetStringProperty("IOCalloutDevice")
		if err == nil {
			port.Name = name
			break
		}
		if retries == 0 {
			return nil, fmt.Errorf("error extracting port info from device: %w", err)
		}
		time.Sleep(50 * time.Millisecond)
	}
	port.IsUSB = false

	validUSBDeviceClass := map[string]bool{
		"IOUSBDevice":     true,
		"IOUSBHostDevice": true,
	}
	usbDevice := service
	var searchErr error
	for !validUSBDeviceClass[usbDevice.GetClass()] {
		parent, err := usbDevice.GetParent("IOService")
		if err != nil {
			searchErr = err
			break
		}
		if usbDevice != service {
			usbDevice.Release()
		}
		usbDevice = parent
	}
	if usbDevice != service {
		defer usbDevice.Release()
	}
	if searchErr == nil {
		// It's an IOUSBDevice
		vid, _ := usbDevice.GetIntProperty("idVendor", kCFNumberSInt16Type)
		pid, _ := usbDevice.GetIntProperty("idProduct", kCFNumberSInt16Type)
		serialNumber, _ := usbDevice.GetStringProperty("kUSBSerialNumberString")
		vendor, _ := usbDevice.GetStringProperty("kUSBVendorString")
		product, _ := usbDevice.GetStringProperty("kUSBProductString")
		configuration, _ := usbDevice.GetUSBConfigurationString()

		port.IsUSB = true
		port.VID = fmt.Sprintf("%04X", vid)
		port.PID = fmt.Sprintf("%04X", pid)
		port.SerialNumber = serialNumber
		port.Manufacturer = vendor
		port.Product = product
		port.Configuration = configuration
	}
	return port, nil
}

func getAllServices(serviceType string) ([]io_object_t, error) {
	i, err := getMatchingServices(serviceMatching(serviceType))
	if err != nil {
		return nil, err
	}
	defer i.Release()

	var services []io_object_t
	tries := 0
	for tries < 5 {
		// Extract all elements from iterator
		if service, ok := i.Next(); ok {
			services = append(services, service)
			continue
		}
		// If the list of services is empty or the iterator is still valid return the result
		if len(services) == 0 || i.IsValid() {
			return services, nil
		}
		// Otherwise empty the result and retry
		for _, s := range services {
			s.Release()
		}
		services = []io_object_t{}
		i.Reset()
		tries++
	}
	// Give up if the iteration continues to fail...
	return nil, fmt.Errorf("IOServiceGetMatchingServices failed, data changed while iterating")
}

// serviceMatching create a matching dictionary that specifies an IOService class match.
func serviceMatching(serviceType string) cfMutableDictionaryRef {
	return ioServiceMatching(serviceType)
}

// getMatchingServices look up registered IOService objects that match a matching dictionary.
func getMatchingServices(matcher cfMutableDictionaryRef) (io_iterator_t, error) {
	var i ioIterator
	err := ioServiceGetMatchingServices(kIOMasterPortDefault, cfDictionaryRef(matcher), &i)
	if err != kernSuccess {
		return 0, fmt.Errorf("IOServiceGetMatchingServices failed (code %d)", err)
	}
	return io_iterator_t(i), nil
}

func cfStringCreateWithString(s string) cfStringRef {
	return cfStringCreateWithCString(kCFAllocatorDefault, s, kCFStringEncodingMacRoman)
}

func cfStringCreateWithBytes(data unsafe.Pointer, len uint32, encoding cfStringEncoding) (cfStringRef, bool) {
	str := cfStringCreateWithBytesFunc(kCFAllocatorDefault, (*uint8)(data), cfIndex(len), encoding, 0)
	return cfStringRef(str), str != 0
}

func (ref cfStringRef) GetLength() uint32 {
	return uint32(cfStringGetLength(ref))
}

func (ref cfStringRef) GetMaximumSizeForEncoding(encoding cfStringEncoding) uint32 {
	return uint32(cfStringGetMaximumSize(cfIndex(ref.GetLength()), encoding))
}

func (ref cfStringRef) GetGoString() (string, bool) {
	maxSize := ref.GetMaximumSizeForEncoding(kCFStringEncodingUTF8) + 1
	buff := C.malloc(C.size_t(maxSize))
	if buff == nil {
		return "", false
	}
	defer C.free(buff)
	if cfStringGetCString(ref, (*byte)(buff), cfIndex(maxSize), kCFStringEncodingUTF8) == 0 {
		return "", false
	}
	return cStringToGo((*byte)(buff)), true
}

func (ref cfStringRef) Release() {
	cfRelease(cfTypeRef(ref))
}

func (ref cfTypeRef) Release() {
	cfRelease(ref)
}

// io_registry_entry_t

type io_registry_entry_t ioRegistryEntry

func (me *io_registry_entry_t) GetParent(plane string) (io_registry_entry_t, error) {
	var parent ioRegistryEntry
	err := ioRegistryEntryGetParent(ioRegistryEntry(*me), plane, &parent)
	if err != 0 {
		return 0, errors.New("no parent device available")
	}
	return io_registry_entry_t(parent), nil
}

func (me *io_registry_entry_t) CreateCFProperty(key string) (cfTypeRef, error) {
	k := cfStringCreateWithString(key)
	defer k.Release()
	property := ioRegistryEntryCreateProperty(ioRegistryEntry(*me), k, kCFAllocatorDefault, 0)
	if property == 0 {
		return 0, errors.New("Property not found: " + key)
	}
	return cfTypeRef(property), nil
}

func (me *io_registry_entry_t) GetStringProperty(key string) (string, error) {
	property, err := me.CreateCFProperty(key)
	if err != nil {
		return "", err
	}
	defer property.Release()

	if ptr := cfStringGetCStringPtr(cfStringRef(property), 0); ptr != nil {
		return cStringToGo(ptr), nil
	}
	// in certain circumstances CFStringGetCStringPtr may return NULL
	// and we must retrieve the string by copy
	buff := make([]byte, 1024)
	if cfStringGetCString(cfStringRef(property), &buff[0], 1024, 0) == 0 {
		return "", fmt.Errorf("property '%s' can't be converted", key)
	}
	return cStringToGo(&buff[0]), nil
}

func (me *io_registry_entry_t) GetUSBConfigurationString() (string, error) {
	configuration, err := RetrieveUSBConfigurationString(io_service_t(*me))
	if err != nil {
		return "", fmt.Errorf("USB configuration string not available: %w", err)
	}
	return configuration, nil
}

func (me *io_registry_entry_t) GetIntProperty(key string, intType cfNumberType) (int, error) {
	property, err := me.CreateCFProperty(key)
	if err != nil {
		return 0, err
	}
	defer property.Release()
	var res int
	if cfNumberGetValue(cfNumberRef(property), intType, unsafe.Pointer(&res)) == 0 {
		return res, fmt.Errorf("property '%s' can't be converted or has been truncated", key)
	}
	return res, nil
}

func (me *io_registry_entry_t) Release() {
	ioObjectRelease(ioObject(*me))
}

func (me *io_registry_entry_t) GetClass() string {
	class := make([]byte, 1024)
	ioObjectGetClass(ioObject(*me), &class[0])
	return cStringToGo(&class[0])
}

// io_iterator_t

type io_iterator_t ioIterator

// IsValid checks if an iterator is still valid.
// Some iterators will be made invalid if changes are made to the
// structure they are iterating over. This function checks the iterator
// is still valid and should be called when Next returns zero.
// An invalid iterator can be Reset and the iteration restarted.
func (me *io_iterator_t) IsValid() bool {
	return ioIteratorIsValid(ioIterator(*me)) != 0
}

func (me *io_iterator_t) Reset() {
	ioIteratorReset(ioIterator(*me))
}

func (me *io_iterator_t) Next() (io_object_t, bool) {
	res := ioIteratorNext(ioIterator(*me))
	return io_object_t(res), res != 0
}

func (me *io_iterator_t) Release() {
	ioObjectRelease(ioObject(*me))
}

// io_object_t

type io_object_t ioObject

func (me *io_object_t) Release() {
	ioObjectRelease(ioObject(*me))
}

func (me *io_object_t) GetClass() string {
	class := make([]byte, 1024)
	ioObjectGetClass(ioObject(*me), &class[0])
	return cStringToGo(&class[0])
}

// io_service_t

type io_service_t ioService

func (me *io_service_t) IOCreatePlugInInterfaceForService() (plugin *IOCFPlugIn, score int32, err error) {
	res := IOCFPlugIn{}
	var s int32
	kr := ioCreatePlugInInterface(
		ioService(*me),
		cfUUIDRef(C.kIOUSBDeviceUserClientTypeID),
		cfUUIDRef(C.kIOCFPlugInInterfaceID),
		unsafe.Pointer(&res.h),
		&s,
	)
	if kr != kIOReturnSuccess || res.h == nil {
		return nil, 0, fmt.Errorf("IOCreatePlugInInterfaceForService failed (code %d)", kr)
	}
	return &res, s, nil
}

// IOCFPlugInInterface

type IOCFPlugIn struct {
	h unsafe.Pointer
}

func (me *IOCFPlugIn) iface() *ioCFPlugInInterface {
	return (*ioCFPlugInInterface)(*(*unsafe.Pointer)(me.h))
}

func (me *IOCFPlugIn) QueryIOUSBDeviceInterface() (*IOUSBDevice, error) {
	var queryInterface func(unsafe.Pointer, uint64, uint64, unsafe.Pointer) hResult
	purego.RegisterFunc(&queryInterface, me.iface().QueryInterface)
	device := IOUSBDevice{}
	result := queryInterface(me.h, kIOUSBDeviceInterfaceIDLo, kIOUSBDeviceInterfaceIDHi, unsafe.Pointer(&device.h))
	if result != sOK {
		return nil, fmt.Errorf("QueryInterface failed (code %d)", result)
	}
	return &device, nil
}

func (me *IOCFPlugIn) Release() {
	var release func(unsafe.Pointer) uLong
	purego.RegisterFunc(&release, me.iface().Release)
	release(me.h)
}

// IOUSBDeviceInterface

type IOUSBDevice struct {
	h unsafe.Pointer
}

func (me *IOUSBDevice) iface() *ioUSBDeviceInterface {
	return (*ioUSBDeviceInterface)(*(*unsafe.Pointer)(me.h))
}

func (me *IOUSBDevice) USBDeviceOpen() error {
	var usbDeviceOpen func(unsafe.Pointer) ioReturn
	purego.RegisterFunc(&usbDeviceOpen, me.iface().USBDeviceOpen)
	kr := usbDeviceOpen(me.h)
	if kr != kIOReturnSuccess {
		return fmt.Errorf("USBDeviceOpen failed (code %d)", kr)
	}
	return nil
}

func (me *IOUSBDevice) USBDeviceClose() error {
	var usbDeviceClose func(unsafe.Pointer) ioReturn
	purego.RegisterFunc(&usbDeviceClose, me.iface().USBDeviceClose)
	kr := usbDeviceClose(me.h)
	if kr != kIOReturnSuccess {
		return fmt.Errorf("USBDeviceClose failed (code %d)", kr)
	}
	return nil
}

func (me *IOUSBDevice) Release() {
	var release func(unsafe.Pointer) uLong
	purego.RegisterFunc(&release, me.iface().Release)
	release(me.h)
}

func (me *IOUSBDevice) GetConfiguration() (uint8, error) {
	var getConfiguration func(unsafe.Pointer, *uint8) ioReturn
	purego.RegisterFunc(&getConfiguration, me.iface().GetConfiguration)
	var config uint8
	kr := getConfiguration(me.h, &config)
	if kr != kIOReturnSuccess {
		return 0, fmt.Errorf("GetConfiguration failed (code %d)", kr)
	}
	return config, nil
}

func (me *IOUSBDevice) GetNumberOfConfigurations() (uint8, error) {
	var getNumberOfConfigurations func(unsafe.Pointer, *uint8) ioReturn
	purego.RegisterFunc(&getNumberOfConfigurations, me.iface().GetNumberOfConfigurations)
	var numConfigs uint8
	kr := getNumberOfConfigurations(me.h, &numConfigs)
	if kr != kIOReturnSuccess {
		return 0, fmt.Errorf("GetNumberOfConfigurations failed (code %d)", kr)
	}
	return numConfigs, nil
}

func (me *IOUSBDevice) GetConfigurationDescriptorPtr(index uint8) (*ioUSBConfigurationDescriptor, error) {
	var getConfigurationDescriptorPtr func(unsafe.Pointer, uint8, *unsafe.Pointer) ioReturn
	purego.RegisterFunc(&getConfigurationDescriptorPtr, me.iface().GetConfigurationDescriptorPtr)
	var configDesc unsafe.Pointer
	kr := getConfigurationDescriptorPtr(me.h, index, &configDesc)
	if kr != kIOReturnSuccess {
		return nil, fmt.Errorf("GetConfigurationDescriptorPtr failed (code %d)", kr)
	}
	return (*ioUSBConfigurationDescriptor)(unsafe.Pointer(configDesc)), nil
}

func (me *IOUSBDevice) DeviceRequest(request *ioUSBDevRequest) error {
	var deviceRequest func(unsafe.Pointer, *ioUSBDevRequest) ioReturn
	purego.RegisterFunc(&deviceRequest, me.iface().DeviceRequest)
	kr := deviceRequest(me.h, request)
	if kr != kIOReturnSuccess {
		return fmt.Errorf("DeviceRequest failed (code %d)", kr)
	}
	return nil
}

func RetrieveUSBConfigurationString(service io_service_t) (string, error) {
	plugin, _, err := service.IOCreatePlugInInterfaceForService()
	if err != nil {
		return "", err
	}
	defer plugin.Release()

	device, err := plugin.QueryIOUSBDeviceInterface()
	if err != nil {
		return "", fmt.Errorf("QueryInterface for IOUSBDeviceInterface failed: %w", err)
	}
	if device == nil {
		return "", errors.New("IOUSBDeviceInterface not found")
	}
	defer device.Release()

	if err := device.USBDeviceOpen(); err != nil {
		return "", fmt.Errorf("USBDeviceOpen failed: %w", err)
	}
	defer device.USBDeviceClose()

	currentConfig, err := device.GetConfiguration()
	if err != nil || currentConfig == 0 {
		return "", fmt.Errorf("GetConfiguration failed or returned 0: %w", err)
	}

	numConfigs, err := device.GetNumberOfConfigurations()
	if err != nil {
		return "", fmt.Errorf("GetNumberOfConfigurations failed: %w", err)
	}

	var stringIndex uint8
	for index := range numConfigs {
		configDesc, err := device.GetConfigurationDescriptorPtr(index)
		if err == nil && configDesc != nil && uint8(configDesc.bConfigurationValue) == currentConfig {
			stringIndex = uint8(configDesc.iConfiguration)
			break
		}
	}
	if stringIndex == 0 {
		return "", errors.New("configuration string index not found")
	}

	pData := C.malloc(1024)
	if pData == nil {
		return "", errors.New("failed to allocate memory for USB request")
	}
	buffer := unsafe.Slice((*uint8)(pData), 1024)
	defer C.free(pData)
	request1 := ioUSBDevRequest{
		bmRequestType: (kUSBIn << 7) | (kUSBStandard << 5) | kUSBDevice,
		bRequest:      kUSBRqGetDescriptor,
		wValue:        uint16(kUSBStringDesc) << 8,
		wIndex:        0,
		wLength:       1024,
		pData:         pData,
	}
	if err := device.DeviceRequest(&request1); err != nil {
		return "", fmt.Errorf("DeviceRequest failed: %w", err)
	}
	var langID uint16 = 0x0409
	if request1.wLenDone >= 4 {
		langID = uint16(buffer[2]) | (uint16(buffer[3]) << 8)
	}

	request2 := ioUSBDevRequest{
		bmRequestType: (kUSBIn << 7) | (kUSBStandard << 5) | kUSBDevice,
		bRequest:      kUSBRqGetDescriptor,
		wValue:        uint16(kUSBStringDesc)<<8 | uint16(stringIndex),
		wIndex:        uint16(langID),
		wLength:       1024,
		pData:         pData,
	}
	if err := device.DeviceRequest(&request2); err != nil {
		return "", fmt.Errorf("DeviceRequest failed: %w", err)
	}
	if request2.wLenDone < 2 {
		return "", errors.New("invalid response length for configuration string")
	}

	descriptorLength := min(uint32(buffer[0]), uint32(request2.wLenDone))
	if descriptorLength <= 2 {
		return "", errors.New("descriptor length too short for configuration string")
	}

	cfConfiguration, ok := cfStringCreateWithBytes(unsafe.Add(pData, 2), descriptorLength-2, kCFStringEncodingUTF16LE)
	if !ok {
		return "", errors.New("failed to create CFString from bytes")
	}
	defer cfConfiguration.Release()
	configuration, ok := cfConfiguration.GetGoString()
	if !ok {
		return "", errors.New("failed to convert CFString to Go string")
	}
	return configuration, nil
}
