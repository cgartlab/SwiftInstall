package release

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type Version struct {
	Major      int
	Minor      int
	Patch      int
	Prerelease string
	Build      string
}

type VersionDecision struct {
	CurrentVersion   Version
	NewVersion       Version
	ChangeType       ChangeType
	Reason           string
	Confidence       float64
	RequiresApproval bool
}

type VersionRule struct {
	Name        string
	Condition   func(ChangeAnalysisResult) bool
	VersionBump ChangeType
	Priority    int
}

type VersionEngine struct {
	rules          []VersionRule
	customPatterns []*regexp.Regexp
}

func NewVersionEngine() *VersionEngine {
	engine := &VersionEngine{
		rules: make([]VersionRule, 0),
	}

	engine.addDefaultRules()

	return engine
}

func (e *VersionEngine) addDefaultRules() {
	e.rules = append(e.rules, VersionRule{
		Name: "breaking_change",
		Condition: func(r ChangeAnalysisResult) bool {
			return r.BreakingChanges > 0
		},
		VersionBump: ChangeTypeMajor,
		Priority:    100,
	})

	e.rules = append(e.rules, VersionRule{
		Name: "new_features",
		Condition: func(r ChangeAnalysisResult) bool {
			return r.NewFeatures > 0
		},
		VersionBump: ChangeTypeMinor,
		Priority:    80,
	})

	e.rules = append(e.rules, VersionRule{
		Name: "bug_fixes",
		Condition: func(r ChangeAnalysisResult) bool {
			return r.BugFixes > 0 && r.NewFeatures == 0 && r.BreakingChanges == 0
		},
		VersionBump: ChangeTypePatch,
		Priority:    60,
	})

	e.rules = append(e.rules, VersionRule{
		Name: "large_changes",
		Condition: func(r ChangeAnalysisResult) bool {
			return r.FilesModified > 20 || r.LinesAdded > 1000
		},
		VersionBump: ChangeTypeMinor,
		Priority:    50,
	})

	e.rules = append(e.rules, VersionRule{
		Name: "minor_changes",
		Condition: func(r ChangeAnalysisResult) bool {
			return r.OtherChanges > 0 && r.NewFeatures == 0 && r.BugFixes == 0
		},
		VersionBump: ChangeTypePatch,
		Priority:    40,
	})
}

func (e *VersionEngine) AddCustomRule(rule VersionRule) {
	e.rules = append(e.rules, rule)
}

func (e *VersionEngine) ParseVersion(versionStr string) (Version, error) {
	versionStr = strings.TrimSpace(versionStr)
	versionStr = strings.TrimPrefix(versionStr, "v")

	re := regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)(?:-([a-zA-Z0-9.-]+))?(?:\+([a-zA-Z0-9.-]+))?$`)
	matches := re.FindStringSubmatch(versionStr)

	if matches == nil {
		return Version{}, fmt.Errorf("invalid version format: %s", versionStr)
	}

	major, _ := strconv.Atoi(matches[1])
	minor, _ := strconv.Atoi(matches[2])
	patch, _ := strconv.Atoi(matches[3])

	return Version{
		Major:      major,
		Minor:      minor,
		Patch:      patch,
		Prerelease: matches[4],
		Build:      matches[5],
	}, nil
}

func (e *VersionEngine) DetermineNewVersion(currentVersion Version, analysis ChangeAnalysisResult) VersionDecision {
	changeType := e.evaluateRules(analysis)

	newVersion := e.bumpVersion(currentVersion, changeType)

	reason := e.generateReason(analysis, changeType)
	requiresApproval := e.needsApproval(analysis, changeType)

	return VersionDecision{
		CurrentVersion:   currentVersion,
		NewVersion:       newVersion,
		ChangeType:       changeType,
		Reason:           reason,
		Confidence:       analysis.Confidence,
		RequiresApproval: requiresApproval,
	}
}

func (e *VersionEngine) evaluateRules(analysis ChangeAnalysisResult) ChangeType {
	for i := 0; i < len(e.rules)-1; i++ {
		for j := i + 1; j < len(e.rules); j++ {
			if e.rules[i].Priority < e.rules[j].Priority {
				e.rules[i], e.rules[j] = e.rules[j], e.rules[i]
			}
		}
	}

	for _, rule := range e.rules {
		if rule.Condition(analysis) {
			return rule.VersionBump
		}
	}

	return ChangeTypeNone
}

func (e *VersionEngine) bumpVersion(current Version, changeType ChangeType) Version {
	newVersion := current

	switch changeType {
	case ChangeTypeMajor:
		newVersion.Major++
		newVersion.Minor = 0
		newVersion.Patch = 0
	case ChangeTypeMinor:
		newVersion.Minor++
		newVersion.Patch = 0
	case ChangeTypePatch:
		newVersion.Patch++
	}

	newVersion.Prerelease = ""
	newVersion.Build = ""

	return newVersion
}

func (e *VersionEngine) generateReason(analysis ChangeAnalysisResult, changeType ChangeType) string {
	var reasons []string

	if analysis.BreakingChanges > 0 {
		reasons = append(reasons, fmt.Sprintf("包含 %d 个破坏性变更", analysis.BreakingChanges))
	}
	if analysis.NewFeatures > 0 {
		reasons = append(reasons, fmt.Sprintf("新增 %d 个功能", analysis.NewFeatures))
	}
	if analysis.BugFixes > 0 {
		reasons = append(reasons, fmt.Sprintf("修复 %d 个Bug", analysis.BugFixes))
	}

	if len(reasons) == 0 {
		reasons = append(reasons, "常规维护性更新")
	}

	return fmt.Sprintf("版本升级类型: %s。原因: %s", changeType.String(), strings.Join(reasons, "，"))
}

func (e *VersionEngine) needsApproval(analysis ChangeAnalysisResult, changeType ChangeType) bool {
	if changeType == ChangeTypeMajor {
		return true
	}

	if analysis.BreakingChanges > 0 {
		return true
	}

	if analysis.FilesModified > 50 || analysis.LinesAdded > 2000 {
		return true
	}

	return false
}

func (v Version) String() string {
	version := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.Prerelease != "" {
		version += "-" + v.Prerelease
	}
	if v.Build != "" {
		version += "+" + v.Build
	}
	return version
}

func (v Version) WithPrefix() string {
	return "v" + v.String()
}

func (v Version) Compare(other Version) int {
	if v.Major != other.Major {
		if v.Major > other.Major {
			return 1
		}
		return -1
	}
	if v.Minor != other.Minor {
		if v.Minor > other.Minor {
			return 1
		}
		return -1
	}
	if v.Patch != other.Patch {
		if v.Patch > other.Patch {
			return 1
		}
		return -1
	}
	return 0
}

func (v Version) GreaterThan(other Version) bool {
	return v.Compare(other) > 0
}

func (v Version) LessThan(other Version) bool {
	return v.Compare(other) < 0
}

func (v Version) Equal(other Version) bool {
	return v.Compare(other) == 0
}
