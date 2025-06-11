// SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
// SPDX-License-Identifier: Apache-2.0

package cpuinfo

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"k8s.io/utils/cpuset"
)

type CPUInfo struct {
	// CpuId is the enumerated CPU ID
	CpuId int `json:"cpuId"`

	// CoreId is the logical core ID, unique within each SocketId
	CoreId int `json:"coreId"`

	// SocketId is the physical socket ID
	SocketId int `json:"socketId"`

	// Numa Node is the NUMA node ID, unique within each SocketId
	NumaNode int `json:"numaNode"`

	// NUMA Node Affinity Mask
	NumaNodeAffinityMask string `json:"numaNodeAffinityMask"`
}

func GetCPUInfos(options ...CPUInfoOption) ([]CPUInfo, error) {
	opts := &cpuInfoOptions{}
	for _, opt := range options {
		opt(opts)
	}

	filename := HostProc("cpuinfo")
	lines, err := ReadLines(filename)
	if err != nil {
		return []CPUInfo{}, err
	}

	cpuInfos := []CPUInfo{}
	var cpuInfoLines []string
	for _, line := range lines {
		// `/proc/cpuinfo` uses empty lines to denote a new CPU block of data.
		if strings.TrimSpace(line) == "" {
			// Parse and reset CPU lines.
			cpuInfo := opts.parseCPUInfo(cpuInfoLines...)
			if cpuInfo != nil {
				cpuInfos = append(cpuInfos, *cpuInfo)
			}
			cpuInfoLines = []string{}
		} else {
			// Gather CPU info lines for later processing.
			cpuInfoLines = append(cpuInfoLines, line)
		}
	}

	return cpuInfos, nil
}

type cpuInfoOptions struct {
	noECore bool
}

type CPUInfoOption func(opts *cpuInfoOptions)

// WithoutECores will not report Intel E-Cores.
func WithoutECores() CPUInfoOption {
	return func(opts *cpuInfoOptions) {
		opts.noECore = true
	}
}

func (opts cpuInfoOptions) parseCPUInfo(lines ...string) *CPUInfo {
	cpuInfo := &CPUInfo{
		CpuId:                -1,
		SocketId:             -1,
		CoreId:               -1,
		NumaNode:             -1,
		NumaNodeAffinityMask: "",
	}

	if len(lines) == 0 {
		return nil
	}

	for _, line := range lines {
		// Within each CPU block of data, each line uses ':' to separate the
		// key-value pair (with whitespace padding).
		fields := strings.Split(line, ":")
		if len(fields) < 2 {
			continue
		}
		key := strings.TrimSpace(fields[0])
		value := strings.TrimSpace(fields[1])

		switch key {
		case "processor":
			cpuInfo.CpuId = parseInt(value)
		case "physical id":
			cpuInfo.SocketId = parseInt(value)
		case "core id":
			cpuInfo.CoreId = parseInt(value)
		}
	}

	if cpuInfo.CpuId < 0 || cpuInfo.SocketId < 0 || cpuInfo.CoreId < 0 {
		return nil
	}

	if err := populateNumaInfo(cpuInfo); err != nil {
		log.Printf("Warning: failed to populate NUMA info for CPU %d: %v", cpuInfo.CpuId, err)
	}

	if opts.avoidCPU(cpuInfo.CpuId) {
		return nil
	}
	return cpuInfo
}

func populateNumaInfo(cpuInfo *CPUInfo) error {
	nodePath := HostSys(fmt.Sprintf("devices/system/cpu/cpu%d", cpuInfo.CpuId))
	files, err := os.ReadDir(nodePath)
	if err != nil {
		return err
	}

	for _, file := range files {
		if strings.HasPrefix(file.Name(), "node") {
			nodeId, err := strconv.Atoi(strings.TrimPrefix(file.Name(), "node"))
			if err != nil {
				continue
			}
			cpuInfo.NumaNode = nodeId
			mask, err := ReadLines(HostSys(fmt.Sprintf("devices/system/node/node%d/cpumap", nodeId)))
			if err == nil {
				cpuInfo.NumaNodeAffinityMask = formatAffinityMask(mask[0])
			}
			return nil
		}
	}
	return fmt.Errorf("no numa node found for cpu %d", cpuInfo.CpuId)
}

func formatAffinityMask(mask string) string {
	newMask := strings.ReplaceAll(mask, ",", "")
	newMask = strings.TrimSpace(newMask)
	return "0x" + newMask
}

func parseInt(str string) int {
	val, err := strconv.Atoi(str)
	if err != nil {
		panic(err)
	}
	return val
}

// avoidCPU returns true when the given CPU should not be reported.
func (opts cpuInfoOptions) avoidCPU(cpuId int) bool {
	avoidECore := opts.noECore && testECore(cpuId)
	return avoidECore
}

// testECore returns true when the CPU is detected as an E-Core.
func testECore(cpuId int) bool {
	filename := HostSys("devices/cpu_atom/cpus")
	lines, err := ReadLines(filename)
	if err != nil {
		// No file, no chance of e-cores on the machine
		return false
	}
	cpuSet, err := cpuset.Parse(lines[0])
	if err != nil {
		panic(err)
	}
	if cpuSet.Contains(cpuId) {
		return true
	}
	return false
}

func GetCPUModelName() (string, error) {
	filename := HostProc("cpuinfo")
	lines, err := ReadLines(filename)
	if err != nil {
		return "", err
	}
	return parseCPUModelName(lines...), nil
}

func parseCPUModelName(lines ...string) string {
	for _, line := range lines {
		// Within each CPU block of data, each line uses ':' to separate the
		// key-value pair (with whitespace padding).
		fields := strings.Split(line, ":")
		if len(fields) < 2 {
			continue
		}
		key := strings.TrimSpace(fields[0])
		value := strings.TrimSpace(fields[1])

		switch key {
		case "model name":
			return value
		}
	}
	return ""
}

// ReadFile reads contents from a file.
func ReadFile(filename string) (string, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// ReadLines reads contents from a file and splits them by new lines.
func ReadLines(filename string) ([]string, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(data), "\n")

	return lines, nil
}

func HostRoot(combineWith ...string) string {
	return GetEnv("HOST_ROOT", "/", combineWith...)
}

func HostProc(combineWith ...string) string {
	return HostRoot(combinePath("proc", combineWith...))
}

func HostSys(combineWith ...string) string {
	return HostRoot(combinePath("sys", combineWith...))
}

// GetEnv retrieves the environment variable key, or uses the default value.
func GetEnv(key string, otherwise string, combineWith ...string) string {
	value := os.Getenv(key)
	if value == "" {
		value = otherwise
	}

	return combinePath(value, combineWith...)
}

func combinePath(value string, combineWith ...string) string {
	switch len(combineWith) {
	case 0:
		return value
	case 1:
		return filepath.Join(value, combineWith[0])
	default:
		all := make([]string, len(combineWith)+1)
		all[0] = value
		copy(all[1:], combineWith)
		return filepath.Join(all...)
	}
}
