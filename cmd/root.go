/*
Copyright © 2020 Luke Reed

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

var (
	captureBody bool
	rps         int
)

type Stats struct {
	mutex        *sync.Mutex
	errorCount   int
	successCount int
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&captureBody, "capture-body", false, "Whether to capture the body of the response (default false")
	rootCmd.PersistentFlags().IntVar(&rps, "rps", 1, "Requests per second (default 1)")
}

var rootCmd = &cobra.Command{
	Use:   "apicheck",
	Short: "Fancy curl with error reporting",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("no url provided")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		run(fmt.Sprintf("%s", args[0]))
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run(url string) {
	mutex := new(sync.Mutex)
	stats := Stats{
		mutex: mutex,
	}
	signals := make(chan os.Signal, 1)
	defer close(signals)

	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-signals
		fmt.Println("\n================")
		fmt.Println("End Report")
		fmt.Println("================")
		report(nil, nil, stats, true)
		os.Exit(0)
	}()
	for {
		for i := 0; i < rps; i++ {
			go doRequest(url, &stats)
		}
		time.Sleep(time.Second * 1)
	}
}

func doRequest(url string, s *Stats) {
	client := http.DefaultClient
	client.Timeout = time.Second * 2
	start := time.Now()
	resp, err := http.Get(url)
	ttlb := time.Since(start)
	if err != nil {
		s.mutex.Lock()
		s.errorCount += 1
		s.mutex.Unlock()
		report(nil, &ttlb, *s, false)
		return
	}
	if resp.StatusCode >= 300 {
		s.mutex.Lock()
		s.errorCount += 1
		s.mutex.Unlock()
		report(resp, &ttlb, *s, false)
		return
	}
	s.mutex.Lock()
	s.successCount += 1
	s.mutex.Unlock()
	report(resp, &ttlb, *s, false)
}

func report(resp *http.Response, ttlb *time.Duration, s Stats, endReport bool) {
	if !endReport && resp != nil {
		// var bod interface{}
		fmt.Println("current attempt:", resp.StatusCode)
		if captureBody {
			b, _ := ioutil.ReadAll(resp.Body)
			fmt.Println("body:", strings.Trim(string(b), "\n"))
		}
		fmt.Printf("TTLB: %dms\n", ttlb.Milliseconds())
	}
	fmt.Println("successes:", s.successCount)
	fmt.Println("errors:", s.errorCount)
	fmt.Println("================")
}
