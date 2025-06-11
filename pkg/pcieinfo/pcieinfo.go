package pcieinfo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pravk03/topologyutil/pkg/cpuinfo"
)

// PCIEDeviceInfo holds information about a single PCIe device.
type PCIEDeviceInfo struct {
	Address              string `json:"address"`
	VendorID             string `json:"vendor"`
	DeviceID             string `json:"device"`
	SubVendorID          string `json:"subVendor,omitempty"`
	SubDeviceID          string `json:"subDevice,omitempty"`
	Class                string `json:"class"`
	Driver               string `json:"driver"`
	PCIERootComplexID    string `json:"pcieRootComplexId"`
	NUMANode             int    `json:"numaNode"`
	NumaNodeAffinityMask string `json:"numaNodeAffinityMask"`
}

// PCIEDeviceKey defines a unique key for a PCIe device based on its IDs.
// This struct will be used as the key in the map.
type PCIEDeviceKey struct {
	VendorID    string
	DeviceID    string
	SubVendorID string
	SubDeviceID string
}

// PCIEInfo is a struct that holds the collection of all PCIe devices.
type PCIEInfo struct {
	Devices map[PCIEDeviceKey]PCIEDeviceInfo
}

// NewPCIEInfo scans the system and returns a new PCIEInfo instance
// containing a map of all found PCIe devices.
func NewPCIEInfo() (*PCIEInfo, error) {
	devices := make(map[PCIEDeviceKey]PCIEDeviceInfo)
	pciPath := cpuinfo.HostSys("bus/pci/devices")

	fmt.Printf("Reading PCIe devices from: %s\n", pciPath)

	err := filepath.Walk(pciPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.Mode()&os.ModeSymlink != 0 && path != pciPath {
			addr := filepath.Base(path)
			realDevPath, err := filepath.EvalSymlinks(path)
			if err != nil {
				return nil
			}

			vendor, _ := readFile(filepath.Join(path, "vendor"))
			device, _ := readFile(filepath.Join(path, "device"))
			subvendor, _ := readFile(filepath.Join(path, "subsystem_vendor"))
			subdevice, _ := readFile(filepath.Join(path, "subsystem_device"))

			// Create the unique key for this device, trimming the "0x" prefix for the key.
			key := PCIEDeviceKey{
				VendorID:    strings.TrimPrefix(strings.TrimSpace(vendor), "0x"),
				DeviceID:    strings.TrimPrefix(strings.TrimSpace(device), "0x"),
				SubVendorID: strings.TrimPrefix(strings.TrimSpace(subvendor), "0x"),
				SubDeviceID: strings.TrimPrefix(strings.TrimSpace(subdevice), "0x"),
			}

			class, _ := readFile(filepath.Join(path, "class"))
			driver, _ := readLink(filepath.Join(path, "driver"))
			numaNode, _ := readIntFromFile(filepath.Join(path, "numa_node"))
			numaNodeAffinityMask, _ := readFile(filepath.Join(path, "local_cpus"))

			pcieRootComplexID := addr
			tempDevPath := realDevPath
			for {
				parentPath := filepath.Dir(tempDevPath)
				parentBase := filepath.Base(parentPath)
				if filepath.Base(filepath.Dir(parentPath)) == "devices" {
					pcieRootComplexID = parentBase
					break
				}
				tempDevPath = parentPath
			}

			devices[key] = PCIEDeviceInfo{
				Address:              addr,
				VendorID:             key.VendorID,
				DeviceID:             key.DeviceID,
				SubVendorID:          key.SubVendorID,
				SubDeviceID:          key.SubDeviceID,
				Class:                strings.TrimSpace(class),
				Driver:               driver,
				NUMANode:             numaNode,
				PCIERootComplexID:    pcieRootComplexID,
				NumaNodeAffinityMask: formatAffinityMask(numaNodeAffinityMask),
			}
		}
		return nil
	})

	if err != nil && !os.IsNotExist(err) {
		fmt.Printf("Error walking PCIe devices: %v\n", err)
		return nil, err
	}

	return &PCIEInfo{Devices: devices}, nil
}

// FindDevice is now a METHOD on the PCIEInfo struct.
// It looks up a device by its IDs in the map that it already knows about.
func (p *PCIEInfo) FindDevice(vendor, device, subVendor, subDevice string) (PCIEDeviceInfo, bool) {
	key := PCIEDeviceKey{
		VendorID:    vendor,
		DeviceID:    device,
		SubVendorID: subVendor,
		SubDeviceID: subDevice,
	}
	deviceInfo, found := p.Devices[key]
	return deviceInfo, found
}

// GetAllDevices returns a slice of all found PCIEDeviceInfo objects.
// This is useful for iterating over all devices.
func (p *PCIEInfo) GetAllDevices() []PCIEDeviceInfo {
	allDevices := make([]PCIEDeviceInfo, 0, len(p.Devices))
	for _, deviceInfo := range p.Devices {
		allDevices = append(allDevices, deviceInfo)
	}
	return allDevices
}

func formatAffinityMask(mask string) string {
	newMask := strings.ReplaceAll(mask, ",", "")
	newMask = strings.TrimSpace(newMask)
	return "0x" + newMask
}

// readFile reads the content of a file and returns it as a string.
func readFile(filename string) (string, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// readLink reads the target of a symbolic link and returns the base name.
func readLink(filename string) (string, error) {
	link, err := os.Readlink(filename)
	if err != nil {
		return "", err
	}
	return filepath.Base(link), nil
}

// readIntFromFile reads an integer from a file.
func readIntFromFile(filename string) (int, error) {
	data, err := readFile(filename)
	if err != nil {
		return 0, err
	}
	var val int
	_, err = fmt.Sscanf(strings.TrimSpace(data), "%d", &val)
	if err != nil {
		return 0, err
	}
	return val, nil
}
