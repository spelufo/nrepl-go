package main

import (
	"encoding/json"
	"fmt"
	"os"

	nrepl "github.com/spelufo/nrepl-go"
	"github.com/spf13/cobra"
)

var (
	flagHost     string
	flagPort     string
	flagPortFile string
	flagNS       string
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
			host := flagHost
			if host == "" {
				host = "localhost"
			}
			addr = host + ":7888"
		} else {
			addr = a
		}
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

// convertBytes recursively converts []byte values to strings for JSON output.
func convertBytes(v any) any {
	switch val := v.(type) {
	case []byte:
		return string(val)
	case map[string]any:
		m := make(map[string]any, len(val))
		for k, v := range val {
			m[k] = convertBytes(v)
		}
		return m
	case []any:
		s := make([]any, len(val))
		for i, v := range val {
			s[i] = convertBytes(v)
		}
		return s
	default:
		return v
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
		ch, err := func() (<-chan nrepl.Response, error) {
			if flagNS != "" {
				return client.EvalIn(args[0], flagNS)
			}
			return client.Eval(args[0])
		}()
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
		data, err := json.MarshalIndent(convertBytes(map[string]any(resp.Msg)), "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
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

func main() {
	rootCmd.PersistentFlags().StringVar(&flagHost, "host", "", "nREPL host (default localhost)")
	rootCmd.PersistentFlags().StringVar(&flagPort, "port", "", "nREPL port")
	rootCmd.PersistentFlags().StringVar(&flagPortFile, "port-file", "", "path to .nrepl-port file")

	evalCmd.Flags().StringVarP(&flagNS, "namespace", "n", "", "namespace to evaluate in")
	rootCmd.AddCommand(evalCmd, describeCmd, completionsCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
