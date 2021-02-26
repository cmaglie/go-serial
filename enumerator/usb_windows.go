//
// Copyright 2014-2020 Cristian Maglie. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//

package enumerator

import (
	"fmt"
	"regexp"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

func parseDeviceID(deviceID string, details *PortDetails) {
	// Windows stock USB-CDC driver
	if len(deviceID) >= 3 && deviceID[:3] == "USB" {
		re := regexp.MustCompile("VID_(....)&PID_(....)(\\\\(\\w+)$)?").FindAllStringSubmatch(deviceID, -1)
		if re == nil || len(re[0]) < 2 {
			// Silently ignore unparsable strings
			return
		}
		details.IsUSB = true
		details.VID = re[0][1]
		details.PID = re[0][2]
		if len(re[0]) >= 4 {
			details.SerialNumber = re[0][4]
		}
		return
	}

	// FTDI driver
	if len(deviceID) >= 7 && deviceID[:7] == "FTDIBUS" {
		re := regexp.MustCompile("VID_(....)\\+PID_(....)(\\+(\\w+))?").FindAllStringSubmatch(deviceID, -1)
		if re == nil || len(re[0]) < 2 {
			// Silently ignore unparsable strings
			return
		}
		details.IsUSB = true
		details.VID = re[0][1]
		details.PID = re[0][2]
		if len(re[0]) >= 4 {
			details.SerialNumber = re[0][4]
		}
		return
	}

	// Other unidentified device type
}

// setupapi based
// --------------

//sys setupDiClassGuidsFromNameInternal(class string, guid *guid, guidSize uint32, requiredSize *uint32) (err error) = setupapi.SetupDiClassGuidsFromNameW
//sys setupDiGetClassDevs(guid *guid, enumerator *string, hwndParent uintptr, flags uint32) (set devicesSet, err error) = setupapi.SetupDiGetClassDevsW
//sys setupDiDestroyDeviceInfoList(set devicesSet) (err error) = setupapi.SetupDiDestroyDeviceInfoList
//sys setupDiEnumDeviceInfo(set devicesSet, index uint32, info *devInfoData) (err error) = setupapi.SetupDiEnumDeviceInfo
//sys setupDiGetDeviceInstanceId(set devicesSet, devInfo *devInfoData, devInstanceId unsafe.Pointer, devInstanceIdSize uint32, requiredSize *uint32) (err error) = setupapi.SetupDiGetDeviceInstanceIdW
//sys setupDiOpenDevRegKey(set devicesSet, devInfo *devInfoData, scope dicsScope, hwProfile uint32, keyType uint32, samDesired regsam) (hkey syscall.Handle, err error) = setupapi.SetupDiOpenDevRegKey
//sys setupDiGetDeviceRegistryProperty(set devicesSet, devInfo *devInfoData, property deviceProperty, propertyType *uint32, outValue *byte, bufSize uint32, reqSize *uint32) (res bool) = setupapi.SetupDiGetDeviceRegistryPropertyW
//sys setupDiGetDeviceProperty(set devicesSet, devInfo *devInfoData, propKey *devPropKey, propType *devPropType, propBuff unsafe.Pointer, proBuffSite uint32, reqSize *uint32, flags uint32) (res bool) = setupapi.SetupDiGetDevicePropertyW

type devPropType uint32

const (
	devPropTypeEmpty                    devPropType = 0x00000000                               // nothing, no property data
	devPropTypeNull                                 = 0x00000001                               // null property data
	devPropTypeSBYTE                                = 0x00000002                               // 8-bit signed int (SBYTE)
	devPropTypeBYTE                                 = 0x00000003                               // 8-bit unsigned int (BYTE)
	devPropTypeINT16                                = 0x00000004                               // 16-bit signed int (SHORT)
	devPropTypeUINT16                               = 0x00000005                               // 16-bit unsigned int (USHORT)
	devPropTypeINT32                                = 0x00000006                               // 32-bit signed int (LONG)
	devPropTypeUINT32                               = 0x00000007                               // 32-bit unsigned int (ULONG)
	devPropTypeINT64                                = 0x00000008                               // 64-bit signed int (LONG64)
	devPropTypeUINT64                               = 0x00000009                               // 64-bit unsigned int (ULONG64)
	devPropTypeFloat                                = 0x0000000A                               // 32-bit floating-point (FLOAT)
	devPropTypeDouble                               = 0x0000000B                               // 64-bit floating-point (DOUBLE)
	devPropTypeDecimal                              = 0x0000000C                               // 128-bit data (DECIMAL)
	devPropTypeGUID                                 = 0x0000000D                               // 128-bit unique identifier (GUID)
	devPropTypeCurrency                             = 0x0000000E                               // 64 bit signed int currency value (CURRENCY)
	devPropTypeDate                                 = 0x0000000F                               // date (DATE)
	devPropTypeFiletime                             = 0x00000010                               // file time (FILETIME)
	devPropTypeBoolean                              = 0x00000011                               // 8-bit boolean (DEVPROP_BOOLEAN)
	devPropTypeString                   devPropType = 0x00000012                               // null-terminated string
	devPropTypeStringList                           = (devPropTypeString | devPropTypeModList) // multi-sz string list
	devPropTypeSecutiryDescriptor                   = 0x00000013                               // self-relative binary SECURITY_DESCRIPTOR
	devPropTypeSecurityDescriptorString             = 0x00000014                               // security descriptor string (SDDL format)
	devPropTypeDevPropKey                           = 0x00000015                               // device property key (DEVPROPKEY)
	devPropTypeDevPropType                          = 0x00000016                               // device property type (DEVPROPTYPE)
	devPropTypeBinary                               = (devPropTypeBYTE | devPropTypeModArray)  // custom binary data
	devPropTypeError                                = 0x00000017                               // 32-bit Win32 system error code
	devPropTypeNTStatus                             = 0x00000018                               // 32-bit NTSTATUS code
	devPropTypeStringIndirect                       = 0x00000019                               // string resource (@[path\]<dllname>,-<strId>)

	devPropTypeModArray = 0x00001000 // array of fixed-sized data elements
	devPropTypeModList  = 0x00002000 // list of variable-sized data elements
)

type devPropGUID guid
type devPropPid uint32

type devPropKey struct {
	fmtid devPropGUID
	pid   devPropPid
}

// (full list here: https://github.com/tpn/winsdk-10/blob/master/Include/10.0.16299.0/shared/devpkey.h ...)
var devPropKeyDeviceBusReportedDeviceDesc = devPropKey{devPropGUID{0x540b947e, 0x8b40, 0x45bc, [8]byte{0xa8, 0xa2, 0x6a, 0x0b, 0x89, 0x4c, 0xbd, 0xa2}}, 4} // DEVPROP_TYPE_STRING

//sys cmGetParent(outParentDev *devInstance, dev devInstance, flags uint32) (cmErr cmError) = cfgmgr32.CM_Get_Parent
//sys cmGetDeviceIDSize(outLen *uint32, dev devInstance, flags uint32) (cmErr cmError) = cfgmgr32.CM_Get_Device_ID_Size
//sys cmGetDeviceID(dev devInstance, buffer unsafe.Pointer, bufferSize uint32, flags uint32) (err cmError) = cfgmgr32.CM_Get_Device_IDW
//sys cmGetDevNodeRegistryProperty(dev devInstance, prop cmDrpProp, outRegDataType *uint32, buffer unsafe.Pointer, length *uint32, flags uint32) (cmErr cmError) = cfgmgr32.CM_Get_DevNode_Registry_PropertyW
//sys cmMapCrToWin32Err(cmErr cmError, defaultErr uint32) (err uint32) = cfgmgr32.CM_MapCrToWin32Err

//
// Registry properties (specified in call to CM_Get_DevInst_Registry_Property or CM_Get_Class_Registry_Property,
// some are allowed in calls to CM_Set_DevInst_Registry_Property and CM_Set_Class_Registry_Property)
// CM_DRP_xxxx values should be used for CM_Get_DevInst_Registry_Property / CM_Set_DevInst_Registry_Property
// CM_CRP_xxxx values should be used for CM_Get_Class_Registry_Property / CM_Set_Class_Registry_Property
// DRP/CRP values that overlap must have a 1:1 correspondence with each other
//

type cmDrpProp uint32

var (
	cmDrpDeviceDesc               cmDrpProp = (0x00000001) // DeviceDesc REG_SZ property (RW)
	cmDrpHardwareID               cmDrpProp = (0x00000002) // HardwareID REG_MULTI_SZ property (RW)
	cmDrpCompatibleIDs            cmDrpProp = (0x00000003) // CompatibleIDs REG_MULTI_SZ property (RW)
	cmDrpService                  cmDrpProp = (0x00000005) // Service REG_SZ property (RW)
	cmDrpClass                    cmDrpProp = (0x00000008) // Class REG_SZ property (RW)
	cmDrpClassGUID                cmDrpProp = (0x00000009) // ClassGUID REG_SZ property (RW)
	cmDrpDriver                   cmDrpProp = (0x0000000A) // Driver REG_SZ property (RW)
	cmDrpConfigFlahs              cmDrpProp = (0x0000000B) // ConfigFlags REG_DWORD property (RW)
	cmDrpMFG                      cmDrpProp = (0x0000000C) // Mfg REG_SZ property (RW)
	cmDrpFriendlyName             cmDrpProp = (0x0000000D) // FriendlyName REG_SZ property (RW)
	cmDrpLocationInformation      cmDrpProp = (0x0000000E) // LocationInformation REG_SZ property (RW)
	cmDrpPhysicalDeviceObjectName cmDrpProp = (0x0000000F) // PhysicalDeviceObjectName REG_SZ property (R)
	cmDrpCapabilities             cmDrpProp = (0x00000010) // Capabilities REG_DWORD property (R)
	cmDrpUINumber                 cmDrpProp = (0x00000011) // UiNumber REG_DWORD property (R)
	cmDrpUpperFilters             cmDrpProp = (0x00000012) // UpperFilters REG_MULTI_SZ property (RW)
	cmDrpLowerFilters             cmDrpProp = (0x00000013) // LowerFilters REG_MULTI_SZ property (RW)
	cmDrpBusTypeGUID              cmDrpProp = (0x00000014) // Bus Type Guid, GUID, (R)
	cmDrpLegacyBusType            cmDrpProp = (0x00000015) // Legacy bus type, INTERFACE_TYPE, (R)
	cmDrpBusNumber                cmDrpProp = (0x00000016) // Bus Number, DWORD, (R)
	cmDrpEnumeratorName           cmDrpProp = (0x00000017) // Enumerator Name REG_SZ property (R)
	cmDrpSecurity                 cmDrpProp = (0x00000018) // Security - Device override (RW)
	cmDrpSecuritySDS              cmDrpProp = (0x00000019) // Security - Device override (RW)
	cmDrpDevType                  cmDrpProp = (0x0000001A) // Device Type - Device override (RW)
	cmDrpExclusive                cmDrpProp = (0x0000001B) // Exclusivity - Device override (RW)
	cmDrpCharacteristics          cmDrpProp = (0x0000001C) // Characteristics - Device Override (RW)
	cmDrpAddress                  cmDrpProp = (0x0000001D) // Device Address (R)
	cmDrpUINumberDescFormat       cmDrpProp = (0x0000001E) // UINumberDescFormat REG_SZ property (RW)
	cmDrpDevicePowerData          cmDrpProp = (0x0000001F) // CM_POWER_DATA REG_BINARY property (R)
	cmDrpRemovalPolicy            cmDrpProp = (0x00000020) // CM_DEVICE_REMOVAL_POLICY REG_DWORD (R)
	cmDrpRemovalPolicyHWDefault   cmDrpProp = (0x00000021) // CM_DRP_REMOVAL_POLICY_HW_DEFAULT REG_DWORD (R)
	cmDrpRemovalPolicyOverride    cmDrpProp = (0x00000022) // CM_DRP_REMOVAL_POLICY_OVERRIDE REG_DWORD (RW)
	cmDrpInstallState             cmDrpProp = (0x00000023) // CM_DRP_INSTALL_STATE REG_DWORD (R)
	cmDrpLocationPaths            cmDrpProp = (0x00000024) // CM_DRP_LOCATION_PATHS REG_MULTI_SZ (R)
	cmDrpBaseContainerID          cmDrpProp = (0x00000025) // Base ContainerID REG_SZ property (R)
	cmDrpMin                      cmDrpProp = (0x00000001) // First device register
	cmDrpMax                      cmDrpProp = (0x00000025) // Last device register
)

var (
	cmCrpUpperFilters    = cmDrpUpperFilters    // UpperFilters REG_MULTI_SZ property (RW)
	cmCrpLowerFilters    = cmDrpLowerFilters    // LowerFilters REG_MULTI_SZ property (RW)
	cmCrpSecurity        = cmDrpSecurity        // Class default security (RW)
	cmCrpSecuritySDS     = cmDrpSecuritySDS     // Class default security (RW)
	cmCrpDevType         = cmDrpDevType         // Class default Device-type (RW)
	cmCrpExclusive       = cmDrpExclusive       // Class default (RW)
	cmCrpCharacteristics = cmDrpCharacteristics // Class default (RW)
	cmCrpMin             = cmDrpMin             // First class register
	cmCrpMax             = cmDrpMax             // Last class register
)

// Device registry property codes
// (Codes marked as read-only (R) may only be used for
// SetupDiGetDeviceRegistryProperty)
//
// These values should cover the same set of registry properties
// as defined by the CM_DRP codes in cfgmgr32.h.
//
// Note that SPDRP codes are zero based while CM_DRP codes are one based!
type deviceProperty uint32

const (
	spdrpDeviceDesc               deviceProperty = 0x00000000 // DeviceDesc = R/W
	spdrpHardwareID                              = 0x00000001 // HardwareID = R/W
	spdrpCompatibleIDS                           = 0x00000002 // CompatibleIDs = R/W
	spdrpUnused0                                 = 0x00000003 // Unused
	spdrpService                                 = 0x00000004 // Service = R/W
	spdrpUnused1                                 = 0x00000005 // Unused
	spdrpUnused2                                 = 0x00000006 // Unused
	spdrpClass                                   = 0x00000007 // Class = R--tied to ClassGUID
	spdrpClassGUID                               = 0x00000008 // ClassGUID = R/W
	spdrpDriver                                  = 0x00000009 // Driver = R/W
	spdrpConfigFlags                             = 0x0000000A // ConfigFlags = R/W
	spdrpMFG                                     = 0x0000000B // Mfg = R/W
	spdrpFriendlyName                            = 0x0000000C // FriendlyName = R/W
	spdrpLocationIinformation                    = 0x0000000D // LocationInformation = R/W
	spdrpPhysicalDeviceObjectName                = 0x0000000E // PhysicalDeviceObjectName = R
	spdrpCapabilities                            = 0x0000000F // Capabilities = R
	spdrpUINumber                                = 0x00000010 // UiNumber = R
	spdrpUpperFilters                            = 0x00000011 // UpperFilters = R/W
	spdrpLowerFilters                            = 0x00000012 // LowerFilters = R/W
	spdrpBusTypeGUID                             = 0x00000013 // BusTypeGUID = R
	spdrpLegactBusType                           = 0x00000014 // LegacyBusType = R
	spdrpBusNumber                               = 0x00000015 // BusNumber = R
	spdrpEnumeratorName                          = 0x00000016 // Enumerator Name = R
	spdrpSecurity                                = 0x00000017 // Security = R/W, binary form
	spdrpSecuritySDS                             = 0x00000018 // Security = W, SDS form
	spdrpDevType                                 = 0x00000019 // Device Type = R/W
	spdrpExclusive                               = 0x0000001A // Device is exclusive-access = R/W
	spdrpCharacteristics                         = 0x0000001B // Device Characteristics = R/W
	spdrpAddress                                 = 0x0000001C // Device Address = R
	spdrpUINumberDescFormat                      = 0x0000001D // UiNumberDescFormat = R/W
	spdrpDevicePowerData                         = 0x0000001E // Device Power Data = R
	spdrpRemovalPolicy                           = 0x0000001F // Removal Policy = R
	spdrpRemovalPolicyHWDefault                  = 0x00000020 // Hardware Removal Policy = R
	spdrpRemovalPolicyOverride                   = 0x00000021 // Removal Policy Override = RW
	spdrpInstallState                            = 0x00000022 // Device Install State = R
	spdrpLocationPaths                           = 0x00000023 // Device Location Paths = R
	spdrpBaseContainerID                         = 0x00000024 // Base ContainerID = R

	spdrpMaximumProperty = 0x00000025 // Upper bound on ordinals
)

// Values specifying the scope of a device property change
type dicsScope uint32

const (
	dicsFlagGlobal          dicsScope = 0x00000001 // make change in all hardware profiles
	dicsFlagConfigSspecific           = 0x00000002 // make change in specified profile only
	dicsFlagConfigGeneral             = 0x00000004 // 1 or more hardware profile-specific
)

// https://msdn.microsoft.com/en-us/library/windows/desktop/ms724878(v=vs.85).aspx
type regsam uint32

const (
	keyAllAccess        regsam = 0xF003F
	keyCreateLink              = 0x00020
	keyCreateSubKey            = 0x00004
	keyEnumerateSubKeys        = 0x00008
	keyExecute                 = 0x20019
	keyNotify                  = 0x00010
	keyQueryValue              = 0x00001
	keyRead                    = 0x20019
	keySetValue                = 0x00002
	keyWOW64_32key             = 0x00200
	keyWOW64_64key             = 0x00100
	keyWrite                   = 0x20006
)

// KeyType values for SetupDiCreateDevRegKey, SetupDiOpenDevRegKey, and
// SetupDiDeleteDevRegKey.
const (
	diregDev  = 0x00000001 // Open/Create/Delete device key
	diregDrv  = 0x00000002 // Open/Create/Delete driver key
	diregBoth = 0x00000004 // Delete both driver and Device key
)

// https://msdn.microsoft.com/it-it/library/windows/desktop/aa373931(v=vs.85).aspx
type guid struct {
	data1 uint32
	data2 uint16
	data3 uint16
	data4 [8]byte
}

func (g guid) String() string {
	return fmt.Sprintf("%08x-%04x-%04x-%02x%02x-%02x%02x%02x%02x%02x%02x",
		g.data1, g.data2, g.data3,
		g.data4[0], g.data4[1], g.data4[2], g.data4[3],
		g.data4[4], g.data4[5], g.data4[6], g.data4[7])
}

func classGuidsFromName(className string) ([]guid, error) {
	// Determine the number of GUIDs for className
	n := uint32(0)
	if err := setupDiClassGuidsFromNameInternal(className, nil, 0, &n); err != nil {
		// ignore error: UIDs array size too small
	}

	res := make([]guid, n)
	err := setupDiClassGuidsFromNameInternal(className, &res[0], n, &n)
	return res, err
}

const (
	digcfDefault         = 0x00000001 // only valid with digcfDeviceInterface
	digcfPresent         = 0x00000002
	digcfAllClasses      = 0x00000004
	digcfProfile         = 0x00000008
	digcfDeviceInterface = 0x00000010
)

type devicesSet syscall.Handle

func (g *guid) getDevicesSet() (devicesSet, error) {
	return setupDiGetClassDevs(g, nil, 0, digcfPresent)
}

func (set devicesSet) destroy() {
	setupDiDestroyDeviceInfoList(set)
}

type cmError uint32

// https://msdn.microsoft.com/en-us/library/windows/hardware/ff552344(v=vs.85).aspx
type devInfoData struct {
	size     uint32
	guid     guid
	devInst  devInstance
	reserved uintptr
}

type devInstance uint32

func cmConvertError(cmErr cmError) error {
	if cmErr == 0 {
		return nil
	}
	winErr := cmMapCrToWin32Err(cmErr, 0)
	return fmt.Errorf("error %d", winErr)
}

func (dev devInstance) getParent() (devInstance, error) {
	var res devInstance
	errN := cmGetParent(&res, dev, 0)
	return res, cmConvertError(errN)
}

func (dev devInstance) GetDeviceID() (string, error) {
	var size uint32
	cmErr := cmGetDeviceIDSize(&size, dev, 0)
	if err := cmConvertError(cmErr); err != nil {
		return "", err
	}
	buff := make([]uint16, size)
	cmErr = cmGetDeviceID(dev, unsafe.Pointer(&buff[0]), uint32(len(buff)), 0)
	if err := cmConvertError(cmErr); err != nil {
		return "", err
	}
	return windows.UTF16ToString(buff[:]), nil
}

type deviceInfo struct {
	set  devicesSet
	data devInfoData
}

func (set devicesSet) getDeviceInfo(index int) (*deviceInfo, error) {
	result := &deviceInfo{set: set}

	result.data.size = uint32(unsafe.Sizeof(result.data))
	err := setupDiEnumDeviceInfo(set, uint32(index), &result.data)
	return result, err
}

func (dev *deviceInfo) getInstanceID() (string, error) {
	n := uint32(0)
	setupDiGetDeviceInstanceId(dev.set, &dev.data, nil, 0, &n)
	buff := make([]uint16, n)
	if err := setupDiGetDeviceInstanceId(dev.set, &dev.data, unsafe.Pointer(&buff[0]), uint32(len(buff)), &n); err != nil {
		return "", err
	}
	return windows.UTF16ToString(buff[:]), nil
}

func (dev *deviceInfo) openDevRegKey(scope dicsScope, hwProfile uint32, keyType uint32, samDesired regsam) (syscall.Handle, error) {
	return setupDiOpenDevRegKey(dev.set, &dev.data, scope, hwProfile, keyType, samDesired)
}

func nativeGetDetailedPortsList() ([]*PortDetails, error) {
	guids, err := classGuidsFromName("Ports")
	if err != nil {
		return nil, &PortEnumerationError{causedBy: err}
	}

	var res []*PortDetails
	for _, g := range guids {
		devsSet, err := g.getDevicesSet()
		if err != nil {
			return nil, &PortEnumerationError{causedBy: err}
		}
		defer devsSet.destroy()

		for i := 0; ; i++ {
			device, err := devsSet.getDeviceInfo(i)
			if err != nil {
				break
			}
			details := &PortDetails{}
			portName, err := retrievePortNameFromDevInfo(device)
			if err != nil {
				continue
			}
			if len(portName) < 3 || portName[0:3] != "COM" {
				// Accept only COM ports
				continue
			}
			details.Name = portName

			if err := retrievePortDetailsFromDevInfo(device, details); err != nil {
				return nil, &PortEnumerationError{causedBy: err}
			}
			res = append(res, details)
		}
	}
	return res, nil
}

func retrievePortNameFromDevInfo(device *deviceInfo) (string, error) {
	h, err := device.openDevRegKey(dicsFlagGlobal, 0, diregDev, keyRead)
	if err != nil {
		return "", err
	}
	defer syscall.RegCloseKey(h)

	var name [1024]uint16
	nameP := (*byte)(unsafe.Pointer(&name[0]))
	nameSize := uint32(len(name) * 2)
	if err := syscall.RegQueryValueEx(h, syscall.StringToUTF16Ptr("PortName"), nil, nil, nameP, &nameSize); err != nil {
		return "", err
	}
	return syscall.UTF16ToString(name[:]), nil
}

func retrievePortDetailsFromDevInfo(device *deviceInfo, details *PortDetails) error {
	deviceID, err := device.getInstanceID()
	if err != nil {
		return err
	}
	parseDeviceID(deviceID, details)

	// On composite USB devices the serial number is usually reported on the parent
	// device, so let's navigate up one level and see if we can get this information
	if details.IsUSB && details.SerialNumber == "" {
		if parentInfo, err := device.data.devInst.getParent(); err == nil {
			if parentDeviceID, err := parentInfo.GetDeviceID(); err == nil {
				d := &PortDetails{}
				parseDeviceID(parentDeviceID, d)
				if details.VID == d.VID && details.PID == d.PID {
					details.SerialNumber = d.SerialNumber
				}
			}
		}
	}

	size := uint32(1024)
	var s uint32
	buf := make([]uint16, size)

	//fmt.Println()
	for i := cmDrpMin; i <= cmDrpMax; i++ {
		size = 1024
		cmGetDevNodeRegistryProperty(device.data.devInst, i, nil, unsafe.Pointer(&buf[0]), &size, 0)
		//fmt.Println(i, ">", windows.UTF16ToString(buf[:]))
	}

	size = 1024
	propType := devPropTypeString
	setupDiGetDeviceProperty(device.set, &device.data, &devPropKeyDeviceBusReportedDeviceDesc, &propType, unsafe.Pointer(&buf[0]), uint32(len(buf)), &s, 0)
	fmt.Println("BusReportedDeviceDesc>", windows.UTF16ToString(buf[:]))

	/*	spdrpDeviceDesc returns a generic name, e.g.: "CDC-ACM", which will be the same for 2 identical devices attached
		while spdrpFriendlyName returns a specific name, e.g.: "CDC-ACM (COM44)",
		the result of spdrpFriendlyName is therefore unique and suitable as an alternative string to for a port choice */
	n := uint32(0)
	setupDiGetDeviceRegistryProperty(device.set, &device.data, spdrpFriendlyName /* spdrpDeviceDesc */, nil, nil, 0, &n)
	buff := make([]uint16, n*2)
	buffP := (*byte)(unsafe.Pointer(&buff[0]))
	if setupDiGetDeviceRegistryProperty(device.set, &device.data, spdrpFriendlyName /* spdrpDeviceDesc */, nil, buffP, n, &n) {
		details.Product = syscall.UTF16ToString(buff[:])
	}

	return nil
}
