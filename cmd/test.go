package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/nyamage/skraft/internal/runner"
	"github.com/nyamage/skraft/internal/skill"
	"github.com/nyamage/skraft/internal/testcase"
	"github.com/spf13/cobra"
)

var testCmd = &cobra.Command{
	Use:          "test [skill]",
	Short:        "Verify skill behavior against test cases",
	Args:         cobra.MaximumNArgs(1),
	SilenceUsage: true,
	RunE:         runTest,
}

func init() {
	rootCmd.AddCommand(testCmd)
}

func runTest(cmd *cobra.Command, args []string) error {
	repoRoot, err := findRepoRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(2)
	}

	claudeBin, err := exec.LookPath("claude")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: claude not found in PATH")
		os.Exit(2)
	}

	claudeVersion, err := getClaudeVersion(claudeBin)
	if err != nil {
		claudeVersion = "unknown"
	}

	skills, err := skill.Discover(repoRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(2)
	}

	if len(args) > 0 {
		s := skill.Find(skills, args[0])
		if s == nil {
			fmt.Fprintf(os.Stderr, "Error: skill %q not found\n", args[0])
			os.Exit(2)
		}
		skills = []skill.Skill{*s}
	}

	type skillWithCases struct {
		s     skill.Skill
		cases []testcase.TestCase
	}
	var allWork []skillWithCases
	for _, s := range skills {
		files, err := skill.DiscoverTestFiles(s.Dir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: discover tests for %s: %v\n", s.DirName, err)
			os.Exit(2)
		}
		var cases []testcase.TestCase
		for _, f := range files {
			tc, err := testcase.Load(f)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(2)
			}
			cases = append(cases, tc)
		}
		if len(cases) > 0 {
			allWork = append(allWork, skillWithCases{s: s, cases: cases})
		}
	}

	if len(allWork) == 0 {
		fmt.Println("No test cases found.")
		return nil
	}

	fmt.Printf("Running tests for %d skill(s) (%s)\n\n", len(allWork), claudeVersion)

	checkMark := color.New(color.FgGreen).Sprint("✓")
	crossMark := color.New(color.FgRed).Sprint("✗")
	warnMark := color.New(color.FgYellow).Sprint("⚠")

	var passed, failed, skipped int
	wallStart := time.Now()

	for _, work := range allWork {
		fmt.Printf("%s\n", work.s.DirName)
		for _, tc := range work.cases {
			res, err := runner.Run(work.s, tc, claudeBin)
			if err != nil {
				fmt.Fprintf(os.Stderr, "  error: %s: %v\n", tc.ID, err)
				failed++
				continue
			}
			if res.Skipped {
				fmt.Printf("  %s %s\n", warnMark, tc.ID)
				fmt.Printf("      skipped: %s\n", res.SkipReason)
				skipped++
				continue
			}
			if res.Pass {
				fmt.Printf("  %s %-28s (%.1fs)\n", checkMark, tc.ID, res.Duration.Seconds())
				passed++
			} else {
				fmt.Printf("  %s %-28s (%.1fs)\n", crossMark, tc.ID, res.Duration.Seconds())
				printTestFailures(res)
				failed++
			}
		}
		fmt.Println()
	}

	elapsed := time.Since(wallStart)
	fmt.Println(strings.Repeat("─", 40))

	var parts []string
	parts = append(parts, fmt.Sprintf("%d passed", passed))
	if failed > 0 {
		parts = append(parts, fmt.Sprintf("%d failed", failed))
	}
	if skipped > 0 {
		parts = append(parts, fmt.Sprintf("%d skipped", skipped))
	}
	fmt.Printf("%s in %.1fs\n", strings.Join(parts, ", "), elapsed.Seconds())

	if failed > 0 {
		os.Exit(1)
	}
	return nil
}

func printTestFailures(res runner.Result) {
	if len(res.Failures) == 1 {
		f := res.Failures[0]
		fmt.Printf("      %s: %s\n", f.Field, f.Message)
	} else {
		fmt.Printf("      failures:\n")
		for _, f := range res.Failures {
			fmt.Printf("        - %s: %s\n", f.Field, f.Message)
		}
	}
}

func getClaudeVersion(claudeBin string) (string, error) {
	out, err := exec.Command(claudeBin, "--version").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
