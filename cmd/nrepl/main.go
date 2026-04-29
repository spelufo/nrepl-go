package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	nrepl "github.com/spelufo/nrepl-go"
	"github.com/spf13/cobra"
)

var (
	flagHost     string
	flagPort     string
	flagPortFile string
)

func connect() (*nrepl.Client, error) {
	var addr string
	if flagPort != "" {
		host := flagHost
		if host == "" {
			host = "localhost"
		}
		addr = host + ":" + flagPort
	} else if flagPortFile != "" {
		a, err := nrepl.ReadPortFile(flagPortFile)
		if err != nil {
			return nil, err
		}
		addr = a
	} else {
		dir, _ := os.Getwd()
		a, err := nrepl.FindPortFile(dir)
		if err != nil {
			return nil, err
		}
		addr = a
	}
	conn, err := nrepl.Dial(addr)
	if err != nil {
		return nil, err
	}
	return nrepl.NewClient(conn)
}

func printResponses(ch <-chan nrepl.Response) {
	for resp := range ch {
		if out, ok := resp.Out(); ok {
			fmt.Print(out)
		}
		if errMsg, ok := resp.Err(); ok {
			fmt.Fprint(os.Stderr, errMsg)
		}
		if val, ok := resp.Value(); ok {
			fmt.Println(val)
		}
		if ex, ok := resp.Ex(); ok {
			fmt.Fprintf(os.Stderr, "Exception: %s\n", ex)
		}
	}
}

var rootCmd = &cobra.Command{
	Use:   "nrepl",
	Short: "nREPL command-line client",
}

var evalCmd = &cobra.Command{
	Use:   "eval <code>",
	Short: "Evaluate Clojure code",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := connect()
		if err != nil {
			return err
		}
		defer client.Close()
		ch, err := client.Eval(args[0])
		if err != nil {
			return err
		}
		printResponses(ch)
		return nil
	},
}

var describeCmd = &cobra.Command{
	Use:   "describe",
	Short: "Describe the nREPL server",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := connect()
		if err != nil {
			return err
		}
		defer client.Close()
		resp, err := client.Describe()
		if err != nil {
			return err
		}
		for k, v := range resp.Msg {
			fmt.Printf("%s: %v\n", k, v)
		}
		return nil
	},
}

var completionsCmd = &cobra.Command{
	Use:   "completions <prefix>",
	Short: "Get completions for a prefix",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := connect()
		if err != nil {
			return err
		}
		defer client.Close()
		resp, err := client.Completions(args[0])
		if err != nil {
			return err
		}
		if comps, ok := resp.Msg["completions"]; ok {
			if list, ok := comps.([]any); ok {
				for _, item := range list {
					if m, ok := item.(map[string]any); ok {
						if c, ok := m["candidate"]; ok {
							switch v := c.(type) {
							case []byte:
								fmt.Println(string(v))
							default:
								fmt.Println(v)
							}
						}
					}
				}
			}
		}
		return nil
	},
}

var replCmd = &cobra.Command{
	Use:   "repl",
	Short: "Interactive nREPL session",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := connect()
		if err != nil {
			return err
		}
		defer client.Close()

		scanner := bufio.NewScanner(os.Stdin)
		fmt.Print("user=> ")
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				fmt.Print("user=> ")
				continue
			}
			ch, err := client.Eval(line)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				break
			}
			printResponses(ch)
			fmt.Print("user=> ")
		}
		fmt.Println()
		return scanner.Err()
	},
}

func main() {
	rootCmd.PersistentFlags().StringVar(&flagHost, "host", "", "nREPL host (default localhost)")
	rootCmd.PersistentFlags().StringVar(&flagPort, "port", "", "nREPL port")
	rootCmd.PersistentFlags().StringVar(&flagPortFile, "port-file", "", "path to .nrepl-port file")

	rootCmd.AddCommand(evalCmd, describeCmd, completionsCmd, replCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
