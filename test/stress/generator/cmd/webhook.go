package cmd

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var (
	webhookURL string

	httpClient = http.Client{
		Timeout:   5 * time.Second,
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
)

var webhookCmd = &cobra.Command{
	Use:   "webhook",
	Short: "Generate webhook event source messages",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		validateGlobalParameters(cmd, args)
		if webhookURL == "" {
			fmt.Printf("\nUrl is missing\n\n")
			cmd.HelpFunc()(cmd, args)
			os.Exit(1)
		}
		stressRun(runWebhook)
	},
}

func runWebhook() error {
	resp, err := httpClient.Post(webhookURL, "application/json", bytes.NewBuffer([]byte("{}")))
	if err != nil {
		return fmt.Errorf("failed to post an event: %w", err)
	}
	if resp.StatusCode == 200 {
		return nil
	}
	return fmt.Errorf("failed to post an message, response code: %d", resp.StatusCode)
}

func init() {
	webhookCmd.Flags().StringVar(&webhookURL, "url", "", "The url to post messages")
	rootCmd.AddCommand(webhookCmd)
}
