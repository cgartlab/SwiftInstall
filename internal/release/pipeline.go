package release

import (
	"context"
	"fmt"
	"time"
)

type ReleaseState int

const (
	StateIdle ReleaseState = iota
	StateAnalyzing
	StateVersionDeciding
	StateBuilding
	StateTesting
	StateDeploying
	StateCompleted
	StateFailed
	StateRolledBack
)

type ReleasePipeline struct {
	configManager *ConfigManager
	analyzer      *ChangeAnalyzer
	versionEngine *VersionEngine
	buildManager  *BuildManager
	testManager   *TestManager
	deployManager *DeployManager
	logger        *ReleaseLogger
	errorHandler  *ErrorHandler

	state          ReleaseState
	currentVersion Version
	projectName    string
	releaseID      string
}

type ReleaseResult struct {
	ReleaseID       string
	Success         bool
	PreviousVersion string
	NewVersion      string
	ChangeType      ChangeType
	AnalysisResult  ChangeAnalysisResult
	BuildResults    []BuildResult
	TestResults     []TestResult
	DeployResults   []DeployResult
	Duration        time.Duration
	Error           error
}

func NewReleasePipeline(configPath string, projectName string) (*ReleasePipeline, error) {
	configManager := NewConfigManager(configPath)
	if err := configManager.Load(); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	config := configManager.GetConfig()
	releaseID := generateReleaseID()

	logger, err := NewReleaseLogger(config.Logging, releaseID)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	errorHandler := NewErrorHandler(logger)

	return &ReleasePipeline{
		configManager: configManager,
		analyzer:      NewChangeAnalyzer(),
		versionEngine: NewVersionEngine(),
		buildManager:  NewBuildManager(config.Build, logger),
		testManager:   NewTestManager(config.Test, logger),
		deployManager: NewDeployManager(config.Deploy, logger, errorHandler),
		logger:        logger,
		errorHandler:  errorHandler,
		projectName:   projectName,
		releaseID:     releaseID,
		state:         StateIdle,
	}, nil
}

func (p *ReleasePipeline) Execute(ctx context.Context, commits []string, fileChanges []FileChange, currentVersionStr string) (*ReleaseResult, error) {
	startTime := time.Now()
	result := &ReleaseResult{
		ReleaseID:       p.releaseID,
		PreviousVersion: currentVersionStr,
	}

	p.logger.Info("Starting release pipeline", map[string]interface{}{
		"releaseId":      p.releaseID,
		"projectName":    p.projectName,
		"currentVersion": currentVersionStr,
	})

	defer func() {
		result.Duration = time.Since(startTime)
		p.logger.Info("Release pipeline completed", map[string]interface{}{
			"success":  result.Success,
			"duration": result.Duration,
		})
	}()

	currentVersion, err := p.versionEngine.ParseVersion(currentVersionStr)
	if err != nil {
		return nil, p.handleError(ErrCodeVersionParse, "Failed to parse current version", err, false)
	}
	p.currentVersion = currentVersion

	analysisResult := p.analyzeChanges(commits, fileChanges)
	result.AnalysisResult = analysisResult

	versionDecision := p.decideVersion(currentVersion, analysisResult)
	result.NewVersion = versionDecision.NewVersion.String()
	result.ChangeType = versionDecision.ChangeType

	if versionDecision.RequiresApproval {
		p.logger.Warn("Version bump requires manual approval", map[string]interface{}{
			"currentVersion": currentVersion.String(),
			"newVersion":     versionDecision.NewVersion.String(),
			"changeType":     versionDecision.ChangeType.String(),
			"reason":         versionDecision.Reason,
		})
	}

	buildResults, err := p.build(ctx, versionDecision.NewVersion.String())
	if err != nil {
		result.Success = false
		result.Error = err
		return result, err
	}
	result.BuildResults = buildResults

	testResults, err := p.test(ctx)
	if err != nil {
		result.Success = false
		result.Error = err
		return result, err
	}
	result.TestResults = testResults

	deployResults, err := p.deploy(ctx, versionDecision.NewVersion.String(), buildResults)
	if err != nil {
		result.Success = false
		result.Error = err
		return result, err
	}
	result.DeployResults = deployResults

	result.Success = true
	p.state = StateCompleted
	p.logger.SetStage(StageComplete)

	return result, nil
}

func (p *ReleasePipeline) analyzeChanges(commits []string, fileChanges []FileChange) ChangeAnalysisResult {
	p.state = StateAnalyzing
	p.logger.SetStage(StageAnalysis)

	p.logger.Info("Analyzing code changes", map[string]interface{}{
		"commits":     len(commits),
		"fileChanges": len(fileChanges),
	})

	result := p.analyzer.AnalyzeChanges(commits, fileChanges)

	p.logger.Info("Change analysis completed", map[string]interface{}{
		"breakingChanges":  result.BreakingChanges,
		"newFeatures":      result.NewFeatures,
		"bugFixes":         result.BugFixes,
		"suggestedVersion": result.SuggestedVersion.String(),
		"confidence":       result.Confidence,
	})

	return result
}

func (p *ReleasePipeline) decideVersion(currentVersion Version, analysis ChangeAnalysisResult) VersionDecision {
	p.state = StateVersionDeciding
	p.logger.SetStage(StageVersionDecision)

	p.logger.Info("Determining version bump", map[string]interface{}{
		"currentVersion": currentVersion.String(),
	})

	decision := p.versionEngine.DetermineNewVersion(currentVersion, analysis)

	p.logger.Info("Version decision made", map[string]interface{}{
		"currentVersion":   decision.CurrentVersion.String(),
		"newVersion":       decision.NewVersion.String(),
		"changeType":       decision.ChangeType.String(),
		"reason":           decision.Reason,
		"confidence":       decision.Confidence,
		"requiresApproval": decision.RequiresApproval,
	})

	return decision
}

func (p *ReleasePipeline) build(ctx context.Context, version string) ([]BuildResult, error) {
	p.state = StateBuilding
	p.logger.SetStage(StageBuild)

	results, err := p.buildManager.Build(ctx, version, p.projectName)
	if err != nil {
		return nil, p.handleError(ErrCodeBuildFailed, "Build process failed", err, false)
	}

	return results, nil
}

func (p *ReleasePipeline) test(ctx context.Context) ([]TestResult, error) {
	p.state = StateTesting
	p.logger.SetStage(StageTest)

	results, err := p.testManager.RunTests(ctx)
	if err != nil {
		return nil, p.handleError(ErrCodeTestFailed, "Test process failed", err, false)
	}

	return results, nil
}

func (p *ReleasePipeline) deploy(ctx context.Context, version string, artifacts []BuildResult) ([]DeployResult, error) {
	p.state = StateDeploying
	p.logger.SetStage(StageDeploy)

	results, err := p.deployManager.Deploy(ctx, version, artifacts)
	if err != nil {
		return nil, p.handleError(ErrCodeDeployFailed, "Deployment failed", err, true)
	}

	return results, nil
}

func (p *ReleasePipeline) handleError(code string, message string, err error, recoverable bool) error {
	releaseErr := NewReleaseError(code, p.mapStateToStage(p.state), message, err, recoverable)
	return p.errorHandler.Handle(releaseErr)
}

func (p *ReleasePipeline) mapStateToStage(state ReleaseState) ReleaseStage {
	switch state {
	case StateAnalyzing:
		return StageAnalysis
	case StateVersionDeciding:
		return StageVersionDecision
	case StateBuilding:
		return StageBuild
	case StateTesting:
		return StageTest
	case StateDeploying:
		return StageDeploy
	default:
		return StageAnalysis
	}
}

func (p *ReleasePipeline) GetState() ReleaseState {
	return p.state
}

func (p *ReleasePipeline) GetLogger() *ReleaseLogger {
	return p.logger
}

func (p *ReleasePipeline) Close() error {
	return p.logger.Close()
}

func generateReleaseID() string {
	return fmt.Sprintf("release-%d", time.Now().Unix())
}

func (s ReleaseState) String() string {
	switch s {
	case StateIdle:
		return "Idle"
	case StateAnalyzing:
		return "Analyzing"
	case StateVersionDeciding:
		return "VersionDeciding"
	case StateBuilding:
		return "Building"
	case StateTesting:
		return "Testing"
	case StateDeploying:
		return "Deploying"
	case StateCompleted:
		return "Completed"
	case StateFailed:
		return "Failed"
	case StateRolledBack:
		return "RolledBack"
	default:
		return "Unknown"
	}
}
