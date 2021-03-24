package cmd

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/spf13/cobra"
)

const (
	Success = "success"
	Failure = "failure"

	RootCommand = "go run ./test/stress/generator/main.go"
)

var (
	duration         time.Duration
	requestPerSecond int
	totalRequests    int64
)

var rootCmd = &cobra.Command{
	Use:   RootCommand,
	Short: "Events generator for stress testing.",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.HelpFunc()(cmd, args)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	defaultDuration, _ := time.ParseDuration("5m")
	rootCmd.PersistentFlags().DurationVarP(&duration, "duration", "d", defaultDuration, "How long it will run, e.g. 5m, 60s")
	rootCmd.PersistentFlags().IntVar(&requestPerSecond, "rps", 0, "Requests per second")
	rootCmd.PersistentFlags().Int64Var(&totalRequests, "total", 0, "Total requests")
	rootCmd.AddCommand(NewWebhookCommand())
	rootCmd.AddCommand(NewSqsCommand())
}

func validateGlobalParameters(cmd *cobra.Command, args []string) {
	if requestPerSecond <= 0 {
		fmt.Printf("\nrps needs to be specified with a number greater than 0\n\n")
		cmd.HelpFunc()(cmd, args)
		os.Exit(1)
	}
	if totalRequests < 0 {
		fmt.Printf("\ntotal needs to be a number greater than 0\n\n")
		cmd.HelpFunc()(cmd, args)
		os.Exit(1)
	}
}

func stressRun(ctx context.Context, f func() error) {
	var mutex = &sync.Mutex{}
	results := map[string]int64{Success: 0, Failure: 0}
	counter := int64(0)
	startTime := time.Now()

	defer func(start time.Time) {
		fmt.Printf("\n----------------- Summary --------------\n")
		fmt.Printf("Total generated events  : %d\n", counter)
		fmt.Printf("Succeeded events        : %d\n", results[Success])
		fmt.Printf("Failed events           : %d\n", results[Failure])
		fmt.Printf("Time taken              : %v\n", time.Since(start))
	}(startTime)

	wg := &sync.WaitGroup{}
loop:
	for {
		if time.Since(startTime).Seconds() >= duration.Seconds() {
			break loop
		}
		for i := 0; i < requestPerSecond; i++ {
			select {
			case <-ctx.Done():
				break loop
			default:
			}
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := f(); err != nil {
					fmt.Printf("Error: %v\n", err)
					mutex.Lock()
					results[Failure]++
					mutex.Unlock()
				} else {
					mutex.Lock()
					results[Success]++
					mutex.Unlock()
				}
			}()
			counter++
			if totalRequests > 0 && counter >= totalRequests {
				break loop
			}
		}
		time.Sleep(1 * time.Second)
	}

	wg.Wait()
}
