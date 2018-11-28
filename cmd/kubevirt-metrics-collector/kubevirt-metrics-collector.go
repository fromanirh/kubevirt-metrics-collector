/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2018 Red Hat, Inc.
 */

package main

import (
	"github.com/davecgh/go-spew/spew"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	flag "github.com/spf13/pflag"

	"github.com/fromanirh/kubevirt-metrics-collector/pkg/monitoring/processes"
	promlocal "github.com/fromanirh/kubevirt-metrics-collector/pkg/monitoring/processes/prometheus"
	"github.com/fromanirh/kubevirt-metrics-collector/pkg/procscanner"

	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s /path/to/kubevirt-metrics-collector.json\n", os.Args[0])
		flag.PrintDefaults()
	}
	intervalString := flag.StringP("interval", "I", processes.DefaultInterval, "metrics collection interval")
	debugMode := flag.BoolP("debug", "D", false, "enable pod resolution debug mode")
	dumpMode := flag.BoolP("dump-metrics", "M", false, "dump the available metrics and exit")
	checkMode := flag.BoolP("check-config", "C", false, "validate (and dump) configuration and exit")
	flag.Parse()

	args := flag.Args()

	var err error

	if *dumpMode {
		err = promlocal.DumpMetrics(os.Stderr)
		if err != nil {
			log.Fatalf("error dumping: %v", err)
		}
		return
	}

	if len(args) < 1 {
		flag.Usage()
		return
	}

	conf, err := processes.NewConfigFromFile(args[0])
	if err != nil {
		log.Fatalf("error reading the configuration file %s: %v", args[0], err)
	}

	conf.Interval = *intervalString
	conf.DebugMode = *debugMode
	conf.Validate()

	interval, err := time.ParseDuration(conf.Interval)
	if err != nil {
		log.Fatalf("error getting the polling interval: %s", err)
	}

	scanner := procscanner.ProcScanner{
		Targets: conf.Targets,
	}

	if *debugMode {
		spew.Fdump(os.Stderr, scanner)
	}

	// here because this way the debug mode can emit both conf and scanner content
	if *checkMode {
		spew.Fdump(os.Stderr, conf)
		return
	}

	log.Printf("kubevirt-metrics-collector started")
	defer log.Printf("kubevirt-metrics-collector stopped")

	go processes.Collect(conf, scanner, interval)

	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(conf.ListenAddress, nil))
}