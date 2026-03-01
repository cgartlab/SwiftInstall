package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"swiftinstall/internal/release"
)

var (
	configPath   = flag.String("config", "release-config.json", "Path to release configuration file")
	projectName  = flag.String("project", "", "Project name")
	currentTag   = flag.String("tag", "", "Current version tag (auto-detected if not provided)")
	dryRun       = flag.Bool("dry-run", false, "Perform a dry run without making changes")
	skipTests    = flag.Bool("skip-tests", false, "Skip test execution")
	skipDeploy   = flag.Bool("skip-deploy", false, "Skip deployment")
	verbose      = flag.Bool("verbose", false, "Enable verbose output")
	outputFormat = flag.String("output", "text", "Output format (text, json)")
)

func main() {
	flag.Parse()

	if *projectName == "" {
		fmt.Fprintln(os.Stderr, "Error: project name is required")
		flag.Usage()
		os.Exit(1)
	}

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()

	pipeline, err := release.NewReleasePipeline(*configPath, *projectName)
	if err != nil {
		return fmt.Errorf("failed to create release pipeline: %w", err)
	}
	defer pipeline.Close()

	logger := pipeline.GetLogger()

	gitManager := release.NewGitManager(".", logger)

	currentVersion := *currentTag
	if currentVersion == "" {
		tag, err := gitManager.GetLatestTag()
		if err != nil {
			currentVersion = "v0.0.0"
			logger.Info("No existing tags found, starting from v0.0.0", nil)
		} else {
			currentVersion = tag
		}
	}

	commits, err := gitManager.GetCommitMessages(currentVersion)
	if err != nil {
		logger.Warn("Failed to get commits, using empty list", map[string]interface{}{
			"error": err,
		})
		commits = []string{}
	}

	diff, err := gitManager.GetDiffSinceTag(currentVersion)
	if err != nil {
		logger.Warn("Failed to get diff, using empty list", map[string]interface{}{
			"error": err,
		})
		diff = &release.GitDiff{Files: []release.FileChange{}}
	}

	if *dryRun {
		return performDryRun(pipeline, currentVersion, commits, diff.Files)
	}

	result, err := pipeline.Execute(ctx, commits, diff.Files, currentVersion)
	if err != nil {
		return fmt.Errorf("release pipeline failed: %w", err)
	}

	return outputResult(result)
}

func performDryRun(pipeline *release.ReleasePipeline, currentVersion string, commits []string, fileChanges []release.FileChange) error {
	analyzer := release.NewChangeAnalyzer()
	versionEngine := release.NewVersionEngine()

	analysis := analyzer.AnalyzeChanges(commits, fileChanges)

	currentVer, err := versionEngine.ParseVersion(currentVersion)
	if err != nil {
		return fmt.Errorf("failed to parse version: %w", err)
	}

	decision := versionEngine.DetermineNewVersion(currentVer, analysis)

	dryRunResult := map[string]interface{}{
		"dryRun":           true,
		"currentVersion":   currentVersion,
		"newVersion":       decision.NewVersion.String(),
		"changeType":       decision.ChangeType.String(),
		"reason":           decision.Reason,
		"confidence":       decision.Confidence,
		"requiresApproval": decision.RequiresApproval,
		"analysis": map[string]interface{}{
			"totalCommits":    analysis.TotalCommits,
			"breakingChanges": analysis.BreakingChanges,
			"newFeatures":     analysis.NewFeatures,
			"bugFixes":        analysis.BugFixes,
			"filesModified":   analysis.FilesModified,
			"linesAdded":      analysis.LinesAdded,
			"linesDeleted":    analysis.LinesDeleted,
		},
		"commits": commits,
	}

	if *outputFormat == "json" {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(dryRunResult)
	}

	fmt.Println("=== DRY RUN MODE ===")
	fmt.Printf("Current Version: %s\n", currentVersion)
	fmt.Printf("New Version: %s\n", decision.NewVersion.String())
	fmt.Printf("Change Type: %s\n", decision.ChangeType.String())
	fmt.Printf("Reason: %s\n", decision.Reason)
	fmt.Printf("Confidence: %.2f%%\n", decision.Confidence*100)
	fmt.Printf("Requires Approval: %v\n", decision.RequiresApproval)
	fmt.Println()
	fmt.Println("=== CHANGE ANALYSIS ===")
	fmt.Printf("Total Commits: %d\n", analysis.TotalCommits)
	fmt.Printf("Breaking Changes: %d\n", analysis.BreakingChanges)
	fmt.Printf("New Features: %d\n", analysis.NewFeatures)
	fmt.Printf("Bug Fixes: %d\n", analysis.BugFixes)
	fmt.Printf("Files Modified: %d\n", analysis.FilesModified)
	fmt.Printf("Lines Added: %d\n", analysis.LinesAdded)
	fmt.Printf("Lines Deleted: %d\n", analysis.LinesDeleted)
	fmt.Println()
	fmt.Println("=== COMMITS ===")
	for i, commit := range commits {
		fmt.Printf("%d. %s\n", i+1, commit)
	}

	return nil
}

func outputResult(result *release.ReleaseResult) error {
	if *outputFormat == "json" {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(result)
	}

	fmt.Println("=== RELEASE RESULT ===")
	fmt.Printf("Release ID: %s\n", result.ReleaseID)
	fmt.Printf("Success: %v\n", result.Success)
	fmt.Printf("Previous Version: %s\n", result.PreviousVersion)
	fmt.Printf("New Version: %s\n", result.NewVersion)
	fmt.Printf("Change Type: %s\n", result.ChangeType.String())
	fmt.Printf("Duration: %v\n", result.Duration)

	if len(result.BuildResults) > 0 {
		fmt.Println()
		fmt.Println("=== BUILD RESULTS ===")
		for _, build := range result.BuildResults {
			status := "SUCCESS"
			if build.Status != release.BuildStatusSuccess {
				status = "FAILED"
			}
			fmt.Printf("Platform %s/%s: %s (%v)\n",
				build.Platform.GOOS, build.Platform.GOARCH, status, build.Duration)
		}
	}

	if len(result.TestResults) > 0 {
		fmt.Println()
		fmt.Println("=== TEST RESULTS ===")
		for _, test := range result.TestResults {
			status := "PASSED"
			if test.Status != release.BuildStatusSuccess {
				status = "FAILED"
			}
			fmt.Printf("Suite %s: %s (Coverage: %.2f%%)\n",
				test.Suite, status, test.Coverage*100)
		}
	}

	if len(result.DeployResults) > 0 {
		fmt.Println()
		fmt.Println("=== DEPLOY RESULTS ===")
		for _, deploy := range result.DeployResults {
			status := "SUCCESS"
			if deploy.Status != release.DeployStatusSuccess {
				status = "FAILED"
			}
			fmt.Printf("Environment %s: %s (%v)\n",
				deploy.Environment, status, deploy.Duration)
			if deploy.RollbackInfo != nil {
				fmt.Printf("  Rollback: %v (Previous: %s)\n",
					deploy.RollbackInfo.Success, deploy.RollbackInfo.PreviousVersion)
			}
		}
	}

	if result.Error != nil {
		fmt.Println()
		fmt.Printf("Error: %v\n", result.Error)
	}

	return nil
}

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "Automated version release and deployment tool")
		fmt.Fprintln(os.Stderr, "Options:")
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr, "\nExamples:")
		fmt.Fprintln(os.Stderr, "  # Dry run to see what would happen")
		fmt.Fprintf(os.Stderr, "  %s -project myapp -dry-run\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "  # Perform actual release")
		fmt.Fprintf(os.Stderr, "  %s -project myapp\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "  # Skip tests and deployment")
		fmt.Fprintf(os.Stderr, "  %s -project myapp -skip-tests -skip-deploy\n", os.Args[0])
	}
}
