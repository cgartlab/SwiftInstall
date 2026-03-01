package release

import (
	"regexp"
	"strings"
)

type ChangeType int

const (
	ChangeTypeNone ChangeType = iota
	ChangeTypePatch
	ChangeTypeMinor
	ChangeTypeMajor
)

type ChangeCategory int

const (
	CategoryFeature ChangeCategory = iota
	CategoryFix
	CategoryBreaking
	CategoryDocs
	CategoryStyle
	CategoryRefactor
	CategoryPerf
	CategoryTest
	CategoryBuild
	CategoryCI
	CategoryChore
)

type FileChange struct {
	Path         string
	ChangeType   string
	AddedLines   int
	DeletedLines int
	Modified     bool
	IsNew        bool
	IsDeleted    bool
}

type CommitAnalysis struct {
	Hash           string
	Message        string
	Type           ChangeCategory
	Scope          string
	BreakingChange bool
	Files          []FileChange
}

type ChangeAnalysisResult struct {
	TotalCommits     int
	BreakingChanges  int
	NewFeatures      int
	BugFixes         int
	OtherChanges     int
	FilesModified    int
	FilesAdded       int
	FilesDeleted     int
	LinesAdded       int
	LinesDeleted     int
	SuggestedVersion ChangeType
	Confidence       float64
	AnalysisDetails  []CommitAnalysis
}

type ChangeAnalyzer struct {
	conventionalCommitPattern *regexp.Regexp
	breakingPattern           *regexp.Regexp
	featurePatterns           []*regexp.Regexp
	fixPatterns               []*regexp.Regexp
}

func NewChangeAnalyzer() *ChangeAnalyzer {
	return &ChangeAnalyzer{
		conventionalCommitPattern: regexp.MustCompile(`^(feat|fix|docs|style|refactor|perf|test|build|ci|chore|revert)(\(.+\))?!?:\s*.+`),
		breakingPattern:           regexp.MustCompile(`BREAKING\s*CHANGE:|^[^:]+!:`),
		featurePatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)^feat(\(.+\))?!?:`),
			regexp.MustCompile(`(?i)add\s+(new\s+)?(feature|functionality|support)`),
			regexp.MustCompile(`(?i)implement\s+new`),
			regexp.MustCompile(`(?i)introduce\s+`),
		},
		fixPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)^fix(\(.+\))?:`),
			regexp.MustCompile(`(?i)fix\s+(bug|issue|error)`),
			regexp.MustCompile(`(?i)resolve\s+(issue|bug)`),
			regexp.MustCompile(`(?i)patch\s+`),
		},
	}
}

func (a *ChangeAnalyzer) AnalyzeCommitMessage(message string) CommitAnalysis {
	analysis := CommitAnalysis{
		Message: message,
	}

	if matches := a.conventionalCommitPattern.FindStringSubmatch(message); len(matches) > 1 {
		analysis.Type = a.parseCommitType(matches[1])
		if len(matches) > 2 && matches[2] != "" {
			analysis.Scope = strings.Trim(matches[2], "()")
		}
		if strings.Contains(message, "!:") || a.breakingPattern.MatchString(message) {
			analysis.BreakingChange = true
		}
	} else {
		analysis.Type = a.inferCommitType(message)
		if a.breakingPattern.MatchString(message) {
			analysis.BreakingChange = true
		}
	}

	return analysis
}

func (a *ChangeAnalyzer) parseCommitType(typeStr string) ChangeCategory {
	switch strings.ToLower(typeStr) {
	case "feat":
		return CategoryFeature
	case "fix":
		return CategoryFix
	case "docs":
		return CategoryDocs
	case "style":
		return CategoryStyle
	case "refactor":
		return CategoryRefactor
	case "perf":
		return CategoryPerf
	case "test":
		return CategoryTest
	case "build":
		return CategoryBuild
	case "ci":
		return CategoryCI
	case "chore":
		return CategoryChore
	default:
		return CategoryChore
	}
}

func (a *ChangeAnalyzer) inferCommitType(message string) ChangeCategory {
	for _, pattern := range a.featurePatterns {
		if pattern.MatchString(message) {
			return CategoryFeature
		}
	}

	for _, pattern := range a.fixPatterns {
		if pattern.MatchString(message) {
			return CategoryFix
		}
	}

	lowerMsg := strings.ToLower(message)
	if strings.Contains(lowerMsg, "refactor") {
		return CategoryRefactor
	}
	if strings.Contains(lowerMsg, "performance") || strings.Contains(lowerMsg, "optimize") {
		return CategoryPerf
	}
	if strings.Contains(lowerMsg, "test") {
		return CategoryTest
	}
	if strings.Contains(lowerMsg, "doc") {
		return CategoryDocs
	}

	return CategoryChore
}

func (a *ChangeAnalyzer) AnalyzeChanges(commits []string, fileChanges []FileChange) ChangeAnalysisResult {
	result := ChangeAnalysisResult{
		FilesModified: len(fileChanges),
	}

	commitAnalyses := make([]CommitAnalysis, 0, len(commits))
	for _, commit := range commits {
		analysis := a.AnalyzeCommitMessage(commit)
		commitAnalyses = append(commitAnalyses, analysis)

		result.TotalCommits++
		if analysis.BreakingChange {
			result.BreakingChanges++
		}
		switch analysis.Type {
		case CategoryFeature:
			result.NewFeatures++
		case CategoryFix:
			result.BugFixes++
		default:
			result.OtherChanges++
		}
	}

	result.AnalysisDetails = commitAnalyses

	for _, fc := range fileChanges {
		result.LinesAdded += fc.AddedLines
		result.LinesDeleted += fc.DeletedLines
		if fc.IsNew {
			result.FilesAdded++
		}
		if fc.IsDeleted {
			result.FilesDeleted++
		}
	}

	result.SuggestedVersion = a.determineVersionBump(result)
	result.Confidence = a.calculateConfidence(result)

	return result
}

func (a *ChangeAnalyzer) determineVersionBump(result ChangeAnalysisResult) ChangeType {
	if result.BreakingChanges > 0 {
		return ChangeTypeMajor
	}

	if result.NewFeatures > 0 {
		return ChangeTypeMinor
	}

	if result.BugFixes > 0 || result.OtherChanges > 0 {
		return ChangeTypePatch
	}

	return ChangeTypeNone
}

func (a *ChangeAnalyzer) calculateConfidence(result ChangeAnalysisResult) float64 {
	confidence := 0.5

	if result.TotalCommits > 0 {
		confidence += 0.1
	}
	if result.BreakingChanges > 0 || result.NewFeatures > 0 || result.BugFixes > 0 {
		confidence += 0.2
	}
	if len(result.AnalysisDetails) > 0 {
		clearCommits := 0
		for _, ca := range result.AnalysisDetails {
			if ca.Type == CategoryFeature || ca.Type == CategoryFix || ca.BreakingChange {
				clearCommits++
			}
		}
		confidence += float64(clearCommits) / float64(len(result.AnalysisDetails)) * 0.2
	}

	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

func (ct ChangeType) String() string {
	switch ct {
	case ChangeTypeMajor:
		return "major"
	case ChangeTypeMinor:
		return "minor"
	case ChangeTypePatch:
		return "patch"
	default:
		return "none"
	}
}

func (cc ChangeCategory) String() string {
	switch cc {
	case CategoryFeature:
		return "feat"
	case CategoryFix:
		return "fix"
	case CategoryDocs:
		return "docs"
	case CategoryStyle:
		return "style"
	case CategoryRefactor:
		return "refactor"
	case CategoryPerf:
		return "perf"
	case CategoryTest:
		return "test"
	case CategoryBuild:
		return "build"
	case CategoryCI:
		return "ci"
	case CategoryChore:
		return "chore"
	default:
		return "unknown"
	}
}
