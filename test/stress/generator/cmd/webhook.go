package cmd

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

func NewWebhookCommand() *cobra.Command {
	var (
		url  string
		data string

		httpClient = http.Client{
			Timeout:   5 * time.Second,
			Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
	)

	command := &cobra.Command{
		Use:   "webhook",
		Short: "Generate webhook event source messages",
		Long:  ``,
		Run: func(cmd *cobra.Command, args []string) {
			validateGlobalParameters(cmd, args)
			if url == "" {
				fmt.Printf("\nUrl is required\n\n")
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}
			stressRun(signals.SetupSignalHandler(), func() error {
				resp, err := httpClient.Post(url, "application/json", bytes.NewBuffer([]byte(data)))
				if err != nil {
					return fmt.Errorf("failed to post an event: %w", err)
				}
				if resp.StatusCode == 200 {
					return nil
				}
				return fmt.Errorf("failed to post an message, response code: %d", resp.StatusCode)
			})
		},
	}
	command.Flags().StringVar(&url, "url", "", "The url to post messages")
	command.Flags().StringVar(&data, "data", "{}", "The data to post")
	return command
}
