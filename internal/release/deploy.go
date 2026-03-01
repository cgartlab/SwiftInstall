package release

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

type DeployStatus int

const (
	DeployStatusPending DeployStatus = iota
	DeployStatusRunning
	DeployStatusSuccess
	DeployStatusFailed
	DeployStatusRolledBack
)

type DeployStrategy int

const (
	StrategyRolling DeployStrategy = iota
	StrategyBlueGreen
	StrategyCanary
)

type DeployResult struct {
	Environment    string
	Status         DeployStatus
	Version        string
	StartTime      time.Time
	EndTime        time.Time
	Duration       time.Duration
	HealthCheckURL string
	HealthStatus   bool
	Error          error
	RollbackInfo   *RollbackInfo
}

type RollbackInfo struct {
	PreviousVersion string
	RollbackTime    time.Time
	Reason          string
	Success         bool
}

type DeployManager struct {
	config       DeployConfig
	logger       *ReleaseLogger
	errorHandler *ErrorHandler
	deployments  []DeployResult
}

type HealthChecker struct {
	config DeployConfig
	logger *ReleaseLogger
}

func NewDeployManager(config DeployConfig, logger *ReleaseLogger, errorHandler *ErrorHandler) *DeployManager {
	return &DeployManager{
		config:       config,
		logger:       logger,
		errorHandler: errorHandler,
		deployments:  make([]DeployResult, 0),
	}
}

func (dm *DeployManager) Deploy(ctx context.Context, version string, artifacts []BuildResult) ([]DeployResult, error) {
	if !dm.config.Enabled {
		dm.logger.Info("Deployment is disabled in configuration", nil)
		return nil, nil
	}

	dm.logger.SetStage(StageDeploy)
	dm.logger.Info("Starting deployment process", map[string]interface{}{
		"version":      version,
		"environments": len(dm.config.Environments),
	})

	results := make([]DeployResult, 0)

	for _, env := range dm.config.Environments {
		if !env.AutoDeploy {
			dm.logger.Info("Skipping environment (auto-deploy disabled)", map[string]interface{}{
				"environment": env.Name,
			})
			continue
		}

		result := dm.deployToEnvironment(ctx, env, version, artifacts)
		results = append(results, result)

		if result.Status == DeployStatusFailed {
			if dm.config.RollbackStrategy == "automatic" {
				rollbackResult := dm.rollback(ctx, env, result)
				result.RollbackInfo = rollbackResult
			}

			return results, fmt.Errorf("deployment failed for environment %s", env.Name)
		}
	}

	dm.deployments = results

	dm.logger.Info("Deployment process completed", map[string]interface{}{
		"deployments": len(results),
	})

	return results, nil
}

func (dm *DeployManager) deployToEnvironment(ctx context.Context, env EnvironmentConfig, version string, artifacts []BuildResult) DeployResult {
	startTime := time.Now()
	result := DeployResult{
		Environment: env.Name,
		Status:      DeployStatusRunning,
		Version:     version,
		StartTime:   startTime,
	}

	dm.logger.Info("Deploying to environment", map[string]interface{}{
		"environment": env.Name,
		"version":     version,
		"strategy":    env.DeployStrategy,
	})

	switch env.DeployStrategy {
	case "rolling":
		err := dm.rollingDeploy(ctx, env, version, artifacts)
		if err != nil {
			result.Status = DeployStatusFailed
			result.Error = err
			result.EndTime = time.Now()
			result.Duration = time.Since(startTime)
			return result
		}
	case "blue-green":
		err := dm.blueGreenDeploy(ctx, env, version, artifacts)
		if err != nil {
			result.Status = DeployStatusFailed
			result.Error = err
			result.EndTime = time.Now()
			result.Duration = time.Since(startTime)
			return result
		}
	case "canary":
		err := dm.canaryDeploy(ctx, env, version, artifacts)
		if err != nil {
			result.Status = DeployStatusFailed
			result.Error = err
			result.EndTime = time.Now()
			result.Duration = time.Since(startTime)
			return result
		}
	default:
		err := dm.rollingDeploy(ctx, env, version, artifacts)
		if err != nil {
			result.Status = DeployStatusFailed
			result.Error = err
			result.EndTime = time.Now()
			result.Duration = time.Since(startTime)
			return result
		}
	}

	healthChecker := NewHealthChecker(dm.config, dm.logger)
	healthURL := dm.buildHealthCheckURL(env)
	result.HealthCheckURL = healthURL

	healthCtx, cancel := context.WithTimeout(ctx, time.Duration(dm.config.HealthCheckTimeout)*time.Second)
	defer cancel()

	healthy := healthChecker.Check(healthCtx, healthURL)
	result.HealthStatus = healthy

	if !healthy {
		result.Status = DeployStatusFailed
		result.Error = fmt.Errorf("health check failed")
		result.EndTime = time.Now()
		result.Duration = time.Since(startTime)
		return result
	}

	result.Status = DeployStatusSuccess
	result.EndTime = time.Now()
	result.Duration = time.Since(startTime)

	dm.logger.Info("Deployment successful", map[string]interface{}{
		"environment": env.Name,
		"version":     version,
		"duration":    result.Duration,
	})

	return result
}

func (dm *DeployManager) rollingDeploy(ctx context.Context, env EnvironmentConfig, version string, artifacts []BuildResult) error {
	dm.logger.Debug("Executing rolling deployment", map[string]interface{}{
		"environment": env.Name,
	})

	time.Sleep(2 * time.Second)

	return nil
}

func (dm *DeployManager) blueGreenDeploy(ctx context.Context, env EnvironmentConfig, version string, artifacts []BuildResult) error {
	dm.logger.Debug("Executing blue-green deployment", map[string]interface{}{
		"environment": env.Name,
	})

	time.Sleep(3 * time.Second)

	return nil
}

func (dm *DeployManager) canaryDeploy(ctx context.Context, env EnvironmentConfig, version string, artifacts []BuildResult) error {
	dm.logger.Debug("Executing canary deployment", map[string]interface{}{
		"environment": env.Name,
	})

	time.Sleep(2 * time.Second)

	return nil
}

func (dm *DeployManager) rollback(ctx context.Context, env EnvironmentConfig, failedDeploy DeployResult) *RollbackInfo {
	dm.logger.SetStage(StageRollback)
	dm.logger.Warn("Initiating rollback", map[string]interface{}{
		"environment":   env.Name,
		"failedVersion": failedDeploy.Version,
	})

	rollbackInfo := &RollbackInfo{
		PreviousVersion: "previous-version",
		RollbackTime:    time.Now(),
		Reason:          "Deployment failed",
	}

	dm.logger.Debug("Executing rollback procedure", map[string]interface{}{
		"environment": env.Name,
	})

	time.Sleep(2 * time.Second)

	healthChecker := NewHealthChecker(dm.config, dm.logger)
	healthURL := dm.buildHealthCheckURL(env)

	healthCtx, cancel := context.WithTimeout(ctx, time.Duration(dm.config.HealthCheckTimeout)*time.Second)
	defer cancel()

	healthy := healthChecker.Check(healthCtx, healthURL)
	rollbackInfo.Success = healthy

	if healthy {
		dm.logger.Info("Rollback successful", map[string]interface{}{
			"environment":     env.Name,
			"previousVersion": rollbackInfo.PreviousVersion,
		})
	} else {
		dm.logger.Error("Rollback failed", fmt.Errorf("health check failed after rollback"), map[string]interface{}{
			"environment": env.Name,
		})
	}

	return rollbackInfo
}

func (dm *DeployManager) buildHealthCheckURL(env EnvironmentConfig) string {
	baseURL := env.Variables["BASE_URL"]
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	return baseURL + dm.config.HealthCheckPath
}

func (dm *DeployManager) GetDeployments() []DeployResult {
	return dm.deployments
}

func NewHealthChecker(config DeployConfig, logger *ReleaseLogger) *HealthChecker {
	return &HealthChecker{
		config: config,
		logger: logger,
	}
}

func (hc *HealthChecker) Check(ctx context.Context, url string) bool {
	hc.logger.Debug("Performing health check", map[string]interface{}{
		"url": url,
	})

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		hc.logger.Error("Failed to create health check request", err, nil)
		return false
	}

	resp, err := client.Do(req)
	if err != nil {
		hc.logger.Error("Health check request failed", err, nil)
		return false
	}
	defer resp.Body.Close()

	healthy := resp.StatusCode >= 200 && resp.StatusCode < 300

	hc.logger.Debug("Health check completed", map[string]interface{}{
		"url":     url,
		"status":  resp.StatusCode,
		"healthy": healthy,
	})

	return healthy
}

func (hc *HealthChecker) CheckWithRetry(ctx context.Context, url string, maxRetries int, interval time.Duration) bool {
	for i := 0; i < maxRetries; i++ {
		if hc.Check(ctx, url) {
			return true
		}

		if i < maxRetries-1 {
			hc.logger.Debug("Health check failed, retrying", map[string]interface{}{
				"attempt":  i + 1,
				"maxTries": maxRetries,
				"interval": interval,
			})

			select {
			case <-ctx.Done():
				return false
			case <-time.After(interval):
				continue
			}
		}
	}

	return false
}
