package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"dns-manager/models"
)

var client = http.Client{
	Timeout: time.Second * 10,
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "dns-manager",
		Short: "Управляйте DNS серверами с помощью этого CLI!",
	}

	rootCmd.AddCommand(newAddCmd(), newRemoveCmd(), newListCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func newAddCmd() *cobra.Command {
	return &cobra.Command{
		Use: "add <ip> <domain>",
		Long: `
Добавление серверов
		
Пример:
	dns-manager add 8.8.8.8 google.com
	dns-manager add 8.8.4.4 google.com
		`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, _ := json.Marshal(models.DNS{IP: args[0], Domain: args[1]})
			req, err := http.NewRequest(http.MethodPost, "http://127.0.0.1:8080/add", bytes.NewBuffer(body))
			if err != nil {
				return err
			}

			resp, err := client.Do(req)
			if err != nil {
				return fmt.Errorf("request failed: %w", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				return decodeError(resp)
			}

			fmt.Printf("Added DNS server %s\n", args[0])

			return nil
		},
	}
}

func newRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use: "remove <ip>",
		Long: `
Удаление серверов
		
Примеры:
	dns-manager remove 8.8.8.8
	dns-manager remove 8.8.4.4
		`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, _ := json.Marshal(models.DNS{IP: args[0]})
			req, err := http.NewRequest(http.MethodDelete, "http://127.0.0.1:8080/remove", bytes.NewBuffer(body))
			if err != nil {
				return err
			}

			resp, err := client.Do(req)
			if err != nil {
				return fmt.Errorf("request failed: %w", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusNoContent {
				return decodeError(resp)
			}

			fmt.Printf("Removed DNS server %s\n", args[0])

			return nil
		},
	}
}

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use: "list",
		Long: `
Просмотр всех серверов
		
Пример:
	dns-manager list
		`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			req, err := http.NewRequest(http.MethodGet, "http://127.0.0.1:8080/list", nil)
			if err != nil {
				return err
			}

			resp, err := client.Do(req)
			if err != nil {
				return fmt.Errorf("request failed: %w", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return decodeError(resp)
			}

			if _, err = io.Copy(cmd.OutOrStdout(), resp.Body); err != nil {
				return fmt.Errorf("read response: %w", err)
			}

			return nil
		},
	}
}

func decodeError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	var e models.ErrorResponse
	if err := json.Unmarshal(body, &e); err == nil && e.Error != "" {
		return fmt.Errorf("server: %s (HTTP %d)", e.Error, resp.StatusCode)
	}
	return fmt.Errorf("server returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
}
