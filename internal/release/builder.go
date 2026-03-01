package release

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type BuildStatus int

const (
	BuildStatusPending BuildStatus = iota
	BuildStatusRunning
	BuildStatusSuccess
	BuildStatusFailed
	BuildStatusCancelled
)

type BuildResult struct {
	Platform     PlatformConfig
	Status       BuildStatus
	OutputPath   string
	Size         int64
	Duration     time.Duration
	Error        error
	BuildLog     string
	ArtifactHash string
}

type TestResult struct {
	Suite    string
	Status   BuildStatus
	Coverage float64
	Passed   int
	Failed   int
	Skipped  int
	Duration time.Duration
	Error    error
	TestLog  string
}

type BuildManager struct {
	config    BuildConfig
	logger    *ReleaseLogger
	artifacts []BuildResult
}

type TestManager struct {
	config  TestConfig
	logger  *ReleaseLogger
	results []TestResult
}

func NewBuildManager(config BuildConfig, logger *ReleaseLogger) *BuildManager {
	return &BuildManager{
		config:    config,
		logger:    logger,
		artifacts: make([]BuildResult, 0),
	}
}

func (bm *BuildManager) Build(ctx context.Context, version string, projectName string) ([]BuildResult, error) {
	bm.logger.SetStage(StageBuild)
	bm.logger.Info("Starting build process", map[string]interface{}{
		"version":   version,
		"platforms": len(bm.config.Platforms),
	})

	results := make([]BuildResult, 0, len(bm.config.Platforms))
	errChan := make(chan error, len(bm.config.Platforms))
	resultChan := make(chan BuildResult, len(bm.config.Platforms))

	for _, platform := range bm.config.Platforms {
		go func(p PlatformConfig) {
			result := bm.buildPlatform(ctx, p, version, projectName)
			resultChan <- result
			if result.Error != nil {
				errChan <- result.Error
			}
		}(platform)
	}

	for i := 0; i < len(bm.config.Platforms); i++ {
		result := <-resultChan
		results = append(results, result)
	}

	close(errChan)
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	bm.artifacts = results

	if len(errors) > 0 {
		return results, fmt.Errorf("build failed for %d platform(s)", len(errors))
	}

	bm.logger.Info("Build process completed", map[string]interface{}{
		"artifacts": len(results),
	})

	return results, nil
}

func (bm *BuildManager) buildPlatform(ctx context.Context, platform PlatformConfig, version string, projectName string) BuildResult {
	startTime := time.Now()
	result := BuildResult{
		Platform: platform,
		Status:   BuildStatusRunning,
	}

	bm.logger.Debug("Building platform", map[string]interface{}{
		"goos":   platform.GOOS,
		"goarch": platform.GOARCH,
	})

	outputName := bm.generateArtifactName(projectName, version, platform)
	outputPath := filepath.Join("release", outputName)

	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		result.Status = BuildStatusFailed
		result.Error = err
		result.Duration = time.Since(startTime)
		return result
	}

	env := os.Environ()
	env = append(env, fmt.Sprintf("GOOS=%s", platform.GOOS))
	env = append(env, fmt.Sprintf("GOARCH=%s", platform.GOARCH))

	args := []string{"build", "-o", outputPath}

	for key, value := range bm.config.BuildArgs {
		args = append(args, key, value)
	}

	args = append(args, "-ldflags", fmt.Sprintf("-s -w -X main.version=%s", version))
	args = append(args, ".")

	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Env = env

	output, err := cmd.CombinedOutput()
	result.BuildLog = string(output)

	if err != nil {
		result.Status = BuildStatusFailed
		result.Error = fmt.Errorf("build command failed: %w", err)
		result.Duration = time.Since(startTime)
		bm.logger.Error("Build failed for platform", err, map[string]interface{}{
			"goos":   platform.GOOS,
			"goarch": platform.GOARCH,
			"output": string(output),
		})
		return result
	}

	if info, err := os.Stat(outputPath); err == nil {
		result.Size = info.Size()
	}

	result.Status = BuildStatusSuccess
	result.OutputPath = outputPath
	result.Duration = time.Since(startTime)

	bm.logger.Debug("Build completed for platform", map[string]interface{}{
		"goos":     platform.GOOS,
		"goarch":   platform.GOARCH,
		"output":   outputPath,
		"size":     result.Size,
		"duration": result.Duration,
	})

	return result
}

func (bm *BuildManager) generateArtifactName(projectName, version string, platform PlatformConfig) string {
	template := bm.config.ArtifactNaming

	template = strings.ReplaceAll(template, "{{.Name}}", projectName)
	template = strings.ReplaceAll(template, "{{.Version}}", version)
	template = strings.ReplaceAll(template, "{{.GOOS}}", platform.GOOS)
	template = strings.ReplaceAll(template, "{{.GOARCH}}", platform.GOARCH)
	template = strings.ReplaceAll(template, "{{.Suffix}}", platform.Suffix)

	return template
}

func (bm *BuildManager) GetArtifacts() []BuildResult {
	return bm.artifacts
}

func (bm *BuildManager) CleanArtifacts() error {
	if err := os.RemoveAll("release"); err != nil {
		return fmt.Errorf("failed to clean artifacts: %w", err)
	}
	return os.MkdirAll("release", 0755)
}

func NewTestManager(config TestConfig, logger *ReleaseLogger) *TestManager {
	return &TestManager{
		config:  config,
		logger:  logger,
		results: make([]TestResult, 0),
	}
}

func (tm *TestManager) RunTests(ctx context.Context) ([]TestResult, error) {
	if !tm.config.Enabled {
		tm.logger.Info("Tests are disabled in configuration", nil)
		return nil, nil
	}

	tm.logger.SetStage(StageTest)
	tm.logger.Info("Starting test execution", map[string]interface{}{
		"suites":      tm.config.TestSuites,
		"minCoverage": tm.config.MinCoverage,
	})

	results := make([]TestResult, 0)

	for _, suite := range tm.config.TestSuites {
		result := tm.runTestSuite(ctx, suite)
		results = append(results, result)

		if result.Status == BuildStatusFailed && tm.isRequiredTest(suite) {
			return results, fmt.Errorf("required test suite %s failed", suite)
		}
	}

	tm.results = results

	if err := tm.validateCoverage(); err != nil {
		return results, err
	}

	tm.logger.Info("Test execution completed", map[string]interface{}{
		"totalSuites": len(results),
		"passed":      tm.countPassed(results),
		"failed":      tm.countFailed(results),
	})

	return results, nil
}

func (tm *TestManager) runTestSuite(ctx context.Context, suite string) TestResult {
	startTime := time.Now()
	result := TestResult{
		Suite:  suite,
		Status: BuildStatusRunning,
	}

	tm.logger.Debug("Running test suite", map[string]interface{}{
		"suite": suite,
	})

	args := []string{"test", "-v", "-coverprofile=coverage.out"}

	if tm.config.Parallel {
		args = append(args, "-parallel", "4")
	}

	args = append(args, suite)

	timeout := time.Duration(tm.config.Timeout) * time.Minute
	testCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(testCtx, "go", args...)
	output, err := cmd.CombinedOutput()
	result.TestLog = string(output)

	if err != nil {
		result.Status = BuildStatusFailed
		result.Error = err
		result.Duration = time.Since(startTime)
		tm.logger.Error("Test suite failed", err, map[string]interface{}{
			"suite":  suite,
			"output": string(output),
		})
		return result
	}

	result.Status = BuildStatusSuccess
	result.Coverage = tm.parseCoverage(string(output))
	result.Duration = time.Since(startTime)

	tm.logger.Debug("Test suite completed", map[string]interface{}{
		"suite":    suite,
		"coverage": result.Coverage,
		"duration": result.Duration,
	})

	return result
}

func (tm *TestManager) parseCoverage(output string) float64 {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "coverage:") {
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "coverage:" && i+1 < len(parts) {
					coverageStr := strings.TrimSuffix(parts[i+1], "%")
					var coverage float64
					fmt.Sscanf(coverageStr, "%f", &coverage)
					return coverage / 100
				}
			}
		}
	}
	return 0
}

func (tm *TestManager) isRequiredTest(suite string) bool {
	for _, required := range tm.config.RequiredTests {
		if strings.Contains(suite, required) {
			return true
		}
	}
	return false
}

func (tm *TestManager) validateCoverage() error {
	totalCoverage := 0.0
	validResults := 0

	for _, result := range tm.results {
		if result.Status == BuildStatusSuccess && result.Coverage > 0 {
			totalCoverage += result.Coverage
			validResults++
		}
	}

	if validResults == 0 {
		return nil
	}

	avgCoverage := totalCoverage / float64(validResults)

	if avgCoverage < tm.config.MinCoverage {
		return fmt.Errorf("coverage %.2f%% is below minimum threshold %.2f%%",
			avgCoverage*100, tm.config.MinCoverage*100)
	}

	return nil
}

func (tm *TestManager) countPassed(results []TestResult) int {
	count := 0
	for _, r := range results {
		if r.Status == BuildStatusSuccess {
			count++
		}
	}
	return count
}

func (tm *TestManager) countFailed(results []TestResult) int {
	count := 0
	for _, r := range results {
		if r.Status == BuildStatusFailed {
			count++
		}
	}
	return count
}

func (tm *TestManager) GetResults() []TestResult {
	return tm.results
}
