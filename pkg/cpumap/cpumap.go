// SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
// SPDX-License-Identifier: Apache-2.0

package cpumap

import (
	"encoding/json"
	"slices"
	"sort"

	"github.com/kelindar/bitmap"
	"k8s.io/utils/cpuset"

	"github.com/pravk03/topologyutil/pkg/bitmaputil"
	"github.com/pravk03/topologyutil/pkg/cpuinfo"
)

type CPUMap struct {
	// AbstractToMachine is a map of abstract to machine Core
	AbstractToMachine []cpuset.CPUSet `json:"abstractToMachine"`

	// MachineToAbstract is a map of machine to abstract Core Id
	MachineToAbstract map[int]int `json:"machineToAbstract"`
}

func (cpuMap *CPUMap) MarshalJSON() ([]byte, error) {
	type Alias CPUMap
	return json.Marshal(&struct {
		AbstractToMachine []string `json:"abstractToMachine"`
		*Alias
	}{
		AbstractToMachine: func() []string {
			absToMac := make([]string, len(cpuMap.AbstractToMachine))
			for i, cpuSet := range cpuMap.AbstractToMachine {
				absToMac[i] = cpuSet.String()
			}
			return absToMac
		}(),
		Alias: (*Alias)(cpuMap),
	})
}

func (cpuMap *CPUMap) MarshalJSONIndent(prefix, indent string) ([]byte, error) {
	type Alias CPUMap
	return json.MarshalIndent(&struct {
		AbstractToMachine []string `json:"abstractToMachine"`
		*Alias
	}{
		AbstractToMachine: func() []string {
			absToMac := make([]string, len(cpuMap.AbstractToMachine))
			for i, cpuSet := range cpuMap.AbstractToMachine {
				absToMac[i] = cpuSet.String()
			}
			return absToMac
		}(),
		Alias: (*Alias)(cpuMap),
	}, prefix, indent)
}

// GetOnesBitmap returns a bitmap with all available bits set to one.
func (cpuMap CPUMap) GetOnesBitmap() bitmap.Bitmap {
	absCpus := make([]int, len(cpuMap.AbstractToMachine))
	for i := range len(cpuMap.AbstractToMachine) {
		absCpus[i] = i
	}
	return bitmaputil.New(absCpus...)
}

// ToAbstractCPUs converts the machine CPU set into an abstract CPU Bitmap.
func (cpuMap CPUMap) ToAbstractCPUs(macCpuSet cpuset.CPUSet) bitmap.Bitmap {
	absCpus := []int{}
	for _, idx := range macCpuSet.List() {
		absCpus = append(absCpus, cpuMap.MachineToAbstract[idx])
	}
	return bitmaputil.New(absCpus...)
}

// ToMachineCPUs converts the abstract CPU Bitmap into a machine CPU set.
func (cpuMap CPUMap) ToMachineCPUs(absBitmap bitmap.Bitmap) cpuset.CPUSet {
	macCpus := []int{}
	absBitmap.Range(func(idx uint32) {
		macCpus = append(macCpus, cpuMap.AbstractToMachine[idx].List()...)
	})
	return cpuset.New(macCpus...)
}

func NewCPUMap(cpuInfos []cpuinfo.CPUInfo) CPUMap {
	sort.SliceStable(cpuInfos, func(i, j int) bool {
		// Align sockets
		if cpuInfos[i].SocketId != cpuInfos[j].SocketId {
			return cpuInfos[i].SocketId < cpuInfos[j].SocketId
		}
		// Align core siblings
		if cpuInfos[i].CoreId != cpuInfos[j].CoreId {
			return cpuInfos[i].CoreId < cpuInfos[j].CoreId
		}
		return cpuInfos[i].CpuId < cpuInfos[j].CpuId
	})

	abstractToMachine := make([]cpuset.CPUSet, 0)
	for _, cpuInfo := range cpuInfos {
		coreSiblings := findCoreSiblings(cpuInfos, cpuInfo)
		if !slices.ContainsFunc(abstractToMachine, coreSiblings.Equals) {
			abstractToMachine = append(abstractToMachine, coreSiblings)
		}
	}

	machineToAbstract := make(map[int]int, len(abstractToMachine))
	for absIdx, coreSiblings := range abstractToMachine {
		for _, macIdx := range coreSiblings.List() {
			machineToAbstract[macIdx] = absIdx
		}
	}

	return CPUMap{
		AbstractToMachine: abstractToMachine,
		MachineToAbstract: machineToAbstract,
	}
}

func findCoreSiblings(cpuInfos []cpuinfo.CPUInfo, sibling cpuinfo.CPUInfo) cpuset.CPUSet {
	siblings := []int{}
	for _, cpuInfo := range cpuInfos {
		if sibling.SocketId == cpuInfo.SocketId && sibling.CoreId == cpuInfo.CoreId {
			siblings = append(siblings, cpuInfo.CpuId)
		}
	}
	return cpuset.New(siblings...)
}
