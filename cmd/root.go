/*
Copyright © 2021 Ken'ichiro Oyama <k1lowxb@gmail.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/Songmu/prompter"
	"github.com/k1LoW/pr-bullet/gh"
	"github.com/k1LoW/pr-bullet/version"
	"github.com/labstack/gommon/color"
	"github.com/mattn/go-colorable"
	"github.com/spf13/cobra"
)

var yes bool

type Repo struct {
	Owner string
	Repo  string
}

func (r Repo) String() string {
	return fmt.Sprintf("%s/%s", r.Owner, r.Repo)
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:          "pr-bullet [PULL_REQUEST_URL] [TARGET_REPOS...]",
	Short:        "pr-bullet is a tool for copying pull request to multiple repositories",
	Long:         `pr-bullet is a tool for copying pull request to multiple repositories.`,
	Version:      version.Version,
	SilenceUsage: true,
	Args: func(cmd *cobra.Command, args []string) error {
		switch {
		case len(args) == 0:
			return fmt.Errorf("accepts > 0 arg(s), received %d", len(args))
		case len(args) == 1:
			fi, err := os.Stdin.Stat()
			if err != nil {
				return err
			}
			if (fi.Mode() & os.ModeCharDevice) != 0 {
				return fmt.Errorf("when received 1 arg, %s need STDIN", version.Name)
			}
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var t []string
		useStdin := false
		g, err := gh.New()
		if err != nil {
			return err
		}
		prURL := args[0]
		if len(args[1:]) > 0 {
			t = args[1:]
		} else {
			useStdin = true
			s, err := getStdin(os.Stdin)
			if err != nil {
				return nil
			}
			t = strings.Split(strings.TrimRight(s, "\n"), "\n")
		}

		ctx := context.Background()

		owner, repo, number, err := gh.Parse(prURL)
		if err != nil {
			return err
		}
		pr, files, err := g.GetPullRequest(ctx, owner, repo, number)
		if err != nil {
			return err
		}

		repos := []Repo{}
		s := []string{}

		for _, tr := range t {
			owner, repo, number, err := gh.Parse(tr)
			if err != nil {
				return err
			}
			if _, err := g.GetRepository(ctx, owner, repo); err != nil {
				return err
			}
			if number != 0 {
				return fmt.Errorf("invalid arg: %s", tr)
			}
			repos = append(repos, Repo{owner, repo})
			s = append(s, Repo{owner, repo}.String())
		}

		cmd.PrintErrln(color.Cyan("Original pull request:"))
		cmd.PrintErrf("  Title ... %s\n", pr.GetTitle())
		cmd.PrintErrf("  URL   ... %s\n", prURL)
		cmd.PrintErrf("  Files ... %d\n", len(files))
		cmd.PrintErrln(color.Cyan("Target repositories:"))
		cmd.PrintErrf("  %s\n", strings.Join(s, ", "))
		cmd.PrintErrln("")

		switch {
		case useStdin && !yes:
			return errors.New("when using STDIN, add the --yes option to allow the process to continue.")
		case !useStdin && !yes:
			yes = prompter.YN("Do you want to create pull requests?", true)
		}

		if !yes {
			return nil
		}

		cmd.PrintErrln("")

		for _, r := range repos {
			if err := g.CopyPullRequest(ctx, r.Owner, r.Repo, pr, files); err != nil {
				return err
			}
		}

		return nil
	},
}

func Execute() {
	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)

	log.SetOutput(ioutil.Discard)
	if env := os.Getenv("DEBUG"); env != "" {
		debug, err := os.Create(fmt.Sprintf("%s.debug", version.Name))
		if err != nil {
			rootCmd.PrintErrln(err)
			os.Exit(1)
		}
		log.SetOutput(debug)
	}

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().BoolVarP(&yes, "yes", "y", false, "automatic yes to prompts")
}

func getStdin(stdin io.Reader) (string, error) {
	in := bufio.NewReader(stdin)
	out := new(bytes.Buffer)
	nc := colorable.NewNonColorable(out)
	for {
		s, err := in.ReadBytes('\n')
		if err == io.EOF {
			break
		} else if err != nil {
			return "", err
		}
		_, err = nc.Write(s)
		if err != nil {
			return "", err
		}
	}
	return out.String(), nil
}
