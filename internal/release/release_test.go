package release

import (
	"testing"
)

func TestChangeAnalyzer_AnalyzeCommitMessage(t *testing.T) {
	analyzer := NewChangeAnalyzer()

	tests := []struct {
		name             string
		message          string
		expectedType     ChangeCategory
		expectedBreaking bool
	}{
		{
			name:             "feature commit",
			message:          "feat: add new user authentication",
			expectedType:     CategoryFeature,
			expectedBreaking: false,
		},
		{
			name:             "breaking feature",
			message:          "feat!: breaking API change",
			expectedType:     CategoryFeature,
			expectedBreaking: true,
		},
		{
			name:             "fix commit",
			message:          "fix: resolve login issue",
			expectedType:     CategoryFix,
			expectedBreaking: false,
		},
		{
			name:             "breaking change in body",
			message:          "feat: new feature\nBREAKING CHANGE: API changed",
			expectedType:     CategoryFeature,
			expectedBreaking: true,
		},
		{
			name:             "docs commit",
			message:          "docs: update README",
			expectedType:     CategoryDocs,
			expectedBreaking: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.AnalyzeCommitMessage(tt.message)
			if result.Type != tt.expectedType {
				t.Errorf("expected type %v, got %v", tt.expectedType, result.Type)
			}
			if result.BreakingChange != tt.expectedBreaking {
				t.Errorf("expected breaking %v, got %v", tt.expectedBreaking, result.BreakingChange)
			}
		})
	}
}

func TestChangeAnalyzer_AnalyzeChanges(t *testing.T) {
	analyzer := NewChangeAnalyzer()

	commits := []string{
		"feat: add new feature",
		"fix: resolve bug",
		"docs: update documentation",
	}

	fileChanges := []FileChange{
		{Path: "main.go", AddedLines: 50, DeletedLines: 10, Modified: true},
		{Path: "test.go", AddedLines: 30, DeletedLines: 5, IsNew: true},
	}

	result := analyzer.AnalyzeChanges(commits, fileChanges)

	if result.TotalCommits != 3 {
		t.Errorf("expected 3 commits, got %d", result.TotalCommits)
	}

	if result.NewFeatures != 1 {
		t.Errorf("expected 1 new feature, got %d", result.NewFeatures)
	}

	if result.BugFixes != 1 {
		t.Errorf("expected 1 bug fix, got %d", result.BugFixes)
	}

	if result.SuggestedVersion != ChangeTypeMinor {
		t.Errorf("expected minor version bump, got %v", result.SuggestedVersion)
	}
}

func TestVersionEngine_ParseVersion(t *testing.T) {
	engine := NewVersionEngine()

	tests := []struct {
		input       string
		expected    Version
		shouldError bool
	}{
		{"1.0.0", Version{Major: 1, Minor: 0, Patch: 0}, false},
		{"v2.1.3", Version{Major: 2, Minor: 1, Patch: 3}, false},
		{"1.0.0-alpha", Version{Major: 1, Minor: 0, Patch: 0, Prerelease: "alpha"}, false},
		{"invalid", Version{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := engine.ParseVersion(tt.input)
			if tt.shouldError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestVersionEngine_DetermineNewVersion(t *testing.T) {
	engine := NewVersionEngine()

	tests := []struct {
		name          string
		current       Version
		analysis      ChangeAnalysisResult
		expectedMajor int
		expectedMinor int
		expectedPatch int
	}{
		{
			name:    "breaking change bumps major",
			current: Version{Major: 1, Minor: 0, Patch: 0},
			analysis: ChangeAnalysisResult{
				BreakingChanges: 1,
			},
			expectedMajor: 2,
			expectedMinor: 0,
			expectedPatch: 0,
		},
		{
			name:    "new feature bumps minor",
			current: Version{Major: 1, Minor: 0, Patch: 0},
			analysis: ChangeAnalysisResult{
				NewFeatures: 1,
			},
			expectedMajor: 1,
			expectedMinor: 1,
			expectedPatch: 0,
		},
		{
			name:    "bug fix bumps patch",
			current: Version{Major: 1, Minor: 0, Patch: 0},
			analysis: ChangeAnalysisResult{
				BugFixes: 1,
			},
			expectedMajor: 1,
			expectedMinor: 0,
			expectedPatch: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := engine.DetermineNewVersion(tt.current, tt.analysis)
			if decision.NewVersion.Major != tt.expectedMajor {
				t.Errorf("expected major %d, got %d", tt.expectedMajor, decision.NewVersion.Major)
			}
			if decision.NewVersion.Minor != tt.expectedMinor {
				t.Errorf("expected minor %d, got %d", tt.expectedMinor, decision.NewVersion.Minor)
			}
			if decision.NewVersion.Patch != tt.expectedPatch {
				t.Errorf("expected patch %d, got %d", tt.expectedPatch, decision.NewVersion.Patch)
			}
		})
	}
}

func TestVersion_Compare(t *testing.T) {
	tests := []struct {
		name     string
		v1       Version
		v2       Version
		expected int
	}{
		{
			name:     "equal versions",
			v1:       Version{Major: 1, Minor: 0, Patch: 0},
			v2:       Version{Major: 1, Minor: 0, Patch: 0},
			expected: 0,
		},
		{
			name:     "v1 greater major",
			v1:       Version{Major: 2, Minor: 0, Patch: 0},
			v2:       Version{Major: 1, Minor: 0, Patch: 0},
			expected: 1,
		},
		{
			name:     "v1 greater minor",
			v1:       Version{Major: 1, Minor: 1, Patch: 0},
			v2:       Version{Major: 1, Minor: 0, Patch: 0},
			expected: 1,
		},
		{
			name:     "v1 less patch",
			v1:       Version{Major: 1, Minor: 0, Patch: 0},
			v2:       Version{Major: 1, Minor: 0, Patch: 1},
			expected: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.v1.Compare(tt.v2)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestVersion_String(t *testing.T) {
	tests := []struct {
		version  Version
		expected string
	}{
		{Version{Major: 1, Minor: 0, Patch: 0}, "1.0.0"},
		{Version{Major: 2, Minor: 1, Patch: 3}, "2.1.3"},
		{Version{Major: 1, Minor: 0, Patch: 0, Prerelease: "alpha"}, "1.0.0-alpha"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if result := tt.version.String(); result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}
