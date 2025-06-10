// SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"gitlab.com/SchedMD/slinky-dev/cpuutil/pkg/cpuinfo"
	"gitlab.com/SchedMD/slinky-dev/cpuutil/pkg/cpumap"
)

var noECore bool

var rootCmd = &cobra.Command{
	Use:   "cpuinfo",
	Short: "Report the cpuinfo of this machine",
	RunE: func(cmd *cobra.Command, args []string) error {
		var data []byte
		var err error

		modelName, err := cpuinfo.GetCPUModelName()
		if err != nil {
			return err
		}
		println("===== CPU Model =====")
		println(modelName)
		println("")

		// Show CPU Info
		opts := []cpuinfo.CPUInfoOption{}
		if noECore {
			opts = append(opts, cpuinfo.WithoutECores())
		}
		cpuInfos, err := cpuinfo.GetCPUInfos(opts...)
		if err != nil {
			return err
		}
		data, err = json.MarshalIndent(cpuInfos, "", "  ")
		if err != nil {
			return err
		}
		println("===== CPU Info =====")
		println(string(data))
		println("")

		// Show CPU Map
		cpuMap := cpumap.NewCPUMap(cpuInfos)
		data, err = cpuMap.MarshalJSONIndent("", "  ")
		if err != nil {
			return err
		}
		println("===== CPU Map =====")
		println(string(data))
		println("")

		return nil
	},
}

func init() {
	rootCmd.Flags().BoolVar(&noECore, "no-ecores", false, "Avoid E-Cores")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to execute command: %v\n", err)
		os.Exit(1)
	}
}
