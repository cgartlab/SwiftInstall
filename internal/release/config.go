package release

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type ReleaseConfig struct {
	Versioning    VersioningConfig    `json:"versioning"`
	AutoRelease   AutoReleaseConfig   `json:"autoRelease"`
	Build         BuildConfig         `json:"build"`
	Test          TestConfig          `json:"test"`
	Deploy        DeployConfig        `json:"deploy"`
	Notifications NotificationsConfig `json:"notifications"`
	Logging       LoggingConfig       `json:"logging"`
}

type VersioningConfig struct {
	Strategy              string            `json:"strategy"`
	PrereleaseEnabled     bool              `json:"prereleaseEnabled"`
	PrereleaseIdentifier  string            `json:"prereleaseIdentifier"`
	CommitMessagePatterns []string          `json:"commitMessagePatterns"`
	VersionFilePattern    string            `json:"versionFilePattern"`
	BranchPatterns        map[string]string `json:"branchPatterns"`
}

type AutoReleaseConfig struct {
	Enabled             bool     `json:"enabled"`
	TriggerBranches     []string `json:"triggerBranches"`
	RequireApproval     bool     `json:"requireApproval"`
	ApprovalThreshold   float64  `json:"approvalThreshold"`
	MaxAutoBumpLevel    string   `json:"maxAutoBumpLevel"`
	QuietPeriodHours    int      `json:"quietPeriodHours"`
	MinCommitsThreshold int      `json:"minCommitsThreshold"`
}

type BuildConfig struct {
	Platforms      []PlatformConfig  `json:"platforms"`
	ArtifactNaming string            `json:"artifactNaming"`
	BuildTimeout   int               `json:"buildTimeout"`
	CacheEnabled   bool              `json:"cacheEnabled"`
	BuildArgs      map[string]string `json:"buildArgs"`
}

type PlatformConfig struct {
	GOOS   string `json:"goos"`
	GOARCH string `json:"goarch"`
	Suffix string `json:"suffix"`
}

type TestConfig struct {
	Enabled       bool     `json:"enabled"`
	MinCoverage   float64  `json:"minCoverage"`
	Timeout       int      `json:"timeout"`
	TestSuites    []string `json:"testSuites"`
	Parallel      bool     `json:"parallel"`
	RequiredTests []string `json:"requiredTests"`
}

type DeployConfig struct {
	Enabled            bool                `json:"enabled"`
	Environments       []EnvironmentConfig `json:"environments"`
	RollbackStrategy   string              `json:"rollbackStrategy"`
	HealthCheckPath    string              `json:"healthCheckPath"`
	HealthCheckTimeout int                 `json:"healthCheckTimeout"`
}

type EnvironmentConfig struct {
	Name           string            `json:"name"`
	Type           string            `json:"type"`
	AutoDeploy     bool              `json:"autoDeploy"`
	DeployStrategy string            `json:"deployStrategy"`
	Variables      map[string]string `json:"variables"`
}

type NotificationsConfig struct {
	Enabled  bool        `json:"enabled"`
	Channels []string    `json:"channels"`
	Webhooks []string    `json:"webhooks"`
	Slack    SlackConfig `json:"slack"`
	Email    EmailConfig `json:"email"`
}

type SlackConfig struct {
	WebhookURL string `json:"webhookURL"`
	Channel    string `json:"channel"`
	Username   string `json:"username"`
}

type EmailConfig struct {
	SMTPServer string   `json:"smtpServer"`
	SMTPPort   int      `json:"smtpPort"`
	Recipients []string `json:"recipients"`
}

type LoggingConfig struct {
	Level      string `json:"level"`
	OutputPath string `json:"outputPath"`
	MaxSize    int    `json:"maxSize"`
	MaxBackups int    `json:"maxBackups"`
	MaxAge     int    `json:"maxAge"`
	Compress   bool   `json:"compress"`
}

type ConfigManager struct {
	configPath string
	config     *ReleaseConfig
}

func NewConfigManager(configPath string) *ConfigManager {
	return &ConfigManager{
		configPath: configPath,
	}
}

func (cm *ConfigManager) Load() error {
	data, err := os.ReadFile(cm.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return cm.createDefaultConfig()
		}
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var config ReleaseConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	cm.config = &config
	return nil
}

func (cm *ConfigManager) Save() error {
	if cm.config == nil {
		return fmt.Errorf("no config to save")
	}

	data, err := json.MarshalIndent(cm.config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(cm.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func (cm *ConfigManager) GetConfig() *ReleaseConfig {
	return cm.config
}

func (cm *ConfigManager) SetConfig(config *ReleaseConfig) {
	cm.config = config
}

func (cm *ConfigManager) createDefaultConfig() error {
	defaultConfig := &ReleaseConfig{
		Versioning: VersioningConfig{
			Strategy:             "semantic",
			PrereleaseEnabled:    false,
			PrereleaseIdentifier: "rc",
			CommitMessagePatterns: []string{
				`^feat(\(.+\))?!?:`,
				`^fix(\(.+\))?:`,
				`^BREAKING CHANGE:`,
			},
			VersionFilePattern: "VERSION",
			BranchPatterns: map[string]string{
				"main":      "release",
				"develop":   "prerelease",
				"feature/*": "none",
			},
		},
		AutoRelease: AutoReleaseConfig{
			Enabled:             true,
			TriggerBranches:     []string{"main"},
			RequireApproval:     true,
			ApprovalThreshold:   0.8,
			MaxAutoBumpLevel:    "minor",
			QuietPeriodHours:    2,
			MinCommitsThreshold: 1,
		},
		Build: BuildConfig{
			Platforms: []PlatformConfig{
				{GOOS: "windows", GOARCH: "amd64", Suffix: ".exe"},
				{GOOS: "windows", GOARCH: "arm64", Suffix: ".exe"},
				{GOOS: "linux", GOARCH: "amd64", Suffix: ""},
				{GOOS: "linux", GOARCH: "arm64", Suffix: ""},
				{GOOS: "darwin", GOARCH: "amd64", Suffix: ""},
				{GOOS: "darwin", GOARCH: "arm64", Suffix: ""},
			},
			ArtifactNaming: "{{.Name}}-{{.Version}}-{{.GOOS}}-{{.GOARCH}}{{.Suffix}}",
			BuildTimeout:   30,
			CacheEnabled:   true,
			BuildArgs: map[string]string{
				"-ldflags": "-s -w",
			},
		},
		Test: TestConfig{
			Enabled:       true,
			MinCoverage:   0.8,
			Timeout:       10,
			TestSuites:    []string{"./..."},
			Parallel:      true,
			RequiredTests: []string{"unit", "integration"},
		},
		Deploy: DeployConfig{
			Enabled:            true,
			RollbackStrategy:   "automatic",
			HealthCheckPath:    "/health",
			HealthCheckTimeout: 30,
			Environments: []EnvironmentConfig{
				{
					Name:           "staging",
					Type:           "testing",
					AutoDeploy:     true,
					DeployStrategy: "rolling",
					Variables:      map[string]string{},
				},
				{
					Name:           "production",
					Type:           "production",
					AutoDeploy:     false,
					DeployStrategy: "blue-green",
					Variables:      map[string]string{},
				},
			},
		},
		Notifications: NotificationsConfig{
			Enabled:  false,
			Channels: []string{"slack", "email"},
			Webhooks: []string{},
			Slack: SlackConfig{
				WebhookURL: "",
				Channel:    "#releases",
				Username:   "Release Bot",
			},
			Email: EmailConfig{
				SMTPServer: "smtp.example.com",
				SMTPPort:   587,
				Recipients: []string{"team@example.com"},
			},
		},
		Logging: LoggingConfig{
			Level:      "info",
			OutputPath: "./logs/release.log",
			MaxSize:    100,
			MaxBackups: 3,
			MaxAge:     7,
			Compress:   true,
		},
	}

	cm.config = defaultConfig
	return cm.Save()
}

func (cm *ConfigManager) GetVersioningConfig() VersioningConfig {
	if cm.config == nil {
		return VersioningConfig{}
	}
	return cm.config.Versioning
}

func (cm *ConfigManager) GetAutoReleaseConfig() AutoReleaseConfig {
	if cm.config == nil {
		return AutoReleaseConfig{}
	}
	return cm.config.AutoRelease
}

func (cm *ConfigManager) GetBuildConfig() BuildConfig {
	if cm.config == nil {
		return BuildConfig{}
	}
	return cm.config.Build
}

func (cm *ConfigManager) GetTestConfig() TestConfig {
	if cm.config == nil {
		return TestConfig{}
	}
	return cm.config.Test
}

func (cm *ConfigManager) GetDeployConfig() DeployConfig {
	if cm.config == nil {
		return DeployConfig{}
	}
	return cm.config.Deploy
}

func (cm *ConfigManager) GetNotificationsConfig() NotificationsConfig {
	if cm.config == nil {
		return NotificationsConfig{}
	}
	return cm.config.Notifications
}

func (cm *ConfigManager) GetLoggingConfig() LoggingConfig {
	if cm.config == nil {
		return LoggingConfig{}
	}
	return cm.config.Logging
}

func (cm *ConfigManager) ShouldAutoRelease(branch string) bool {
	autoConfig := cm.GetAutoReleaseConfig()
	if !autoConfig.Enabled {
		return false
	}

	for _, triggerBranch := range autoConfig.TriggerBranches {
		if branch == triggerBranch {
			return true
		}
	}

	return false
}

func (cm *ConfigManager) GetBranchVersioningStrategy(branch string) string {
	versioningConfig := cm.GetVersioningConfig()

	for pattern, strategy := range versioningConfig.BranchPatterns {
		matched, err := filepath.Match(pattern, branch)
		if err == nil && matched {
			return strategy
		}
	}

	return "none"
}

func (cm *ConfigManager) Validate() error {
	if cm.config == nil {
		return fmt.Errorf("config not loaded")
	}

	if cm.config.AutoRelease.Enabled && len(cm.config.AutoRelease.TriggerBranches) == 0 {
		return fmt.Errorf("auto release enabled but no trigger branches configured")
	}

	if cm.config.Test.Enabled && cm.config.Test.MinCoverage < 0 || cm.config.Test.MinCoverage > 1 {
		return fmt.Errorf("test coverage must be between 0 and 1")
	}

	return nil
}
