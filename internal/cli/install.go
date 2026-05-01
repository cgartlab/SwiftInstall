package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/cgartlab/SwiftInstall/internal/backend"
	"github.com/cgartlab/SwiftInstall/internal/config"
	"github.com/cgartlab/SwiftInstall/internal/engine"
	"github.com/cgartlab/SwiftInstall/internal/mirror"
	"github.com/cgartlab/SwiftInstall/internal/proxy"
	"github.com/spf13/cobra"
)

func newInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install packages from a manifest file",
		Long: `Install all packages listed in a YAML or TXT manifest file.
Uses winget on Windows and Homebrew on macOS.

Failing one package does not stop the batch — errors are collected
and reported in a summary at the end.

Examples:
  sis install
  sis install -f software_list.txt
  sis install --dry-run
  sis install --proxy http://127.0.0.1:10809`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			manifestPath, _ := cmd.Flags().GetString("file")
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			proxyFlag, _ := cmd.Flags().GetString("proxy")
			mirrorFlag, _ := cmd.Flags().GetString("mirror")
			skipExisting, _ := cmd.Flags().GetBool("skip-existing")
			skipChecks, _ := cmd.Flags().GetBool("skip-checks")

			if manifestPath == "" {
				manifestPath = resolveManifest(cfg.DefaultManifest)
			}
			if manifestPath == "" {
				return fmt.Errorf("no manifest file found; use --file or create software_list.txt or sis.yaml")
			}

			proxyAddr := proxyFlag
			if proxyAddr == "" {
				proxyAddr = cfg.Proxy
			}
			if proxyAddr == "" && config.BoolVal(cfg.ProxyAutoDetect) {
				if p, err := proxy.Detect(); err == nil && p.Running {
					proxyAddr = p.Address
				}
			}

			if mirrorFlag != "" {
				if err := mirror.Set(mirrorFlag); err != nil {
					return fmt.Errorf("mirror: %w", err)
				}
			}

			winget := backend.NewWingetBackend()
			if err := winget.Detect(); err != nil {
				return fmt.Errorf("winget: %w", err)
			}

			if !skipChecks {
				suite := engine.RunChecks(context.Background(), &engine.CheckConfig{
					ManifestPath: manifestPath,
					Backend:      winget,
				})
				printChecks(suite)
				if !suite.AllPass() {
					return fmt.Errorf("pre-flight checks failed; use --skip-checks to bypass")
				}
			}

			manifest, err := engine.ParseManifest(manifestPath)
			if err != nil {
				return fmt.Errorf("parse manifest: %w", err)
			}

			eng := engine.New(winget, &engine.Config{
				DryRun:       dryRun,
				SkipExisting: skipExisting,
				RetryCount:   config.IntVal(cfg.RetryCount, 2),
				RetryDelay:   time.Duration(config.IntVal(cfg.RetryDelaySec, 3)) * time.Second,
				Proxy:        proxyAddr,
			})

			eng.SetProgressHook(func(r engine.InstallResult) {
				icon := "✓"
				if r.Status == engine.StatusFailed {
					icon = "✗"
				} else if r.Status == engine.StatusSkipped {
					icon = "⚠"
				}
				fmt.Fprintf(os.Stderr, "  %s %s\n", icon, r.Package.ID)
			})

			fmt.Fprintf(os.Stderr, "Installing %d packages from %s\n", len(manifest.Packages), manifestPath)
			if dryRun {
				fmt.Fprintf(os.Stderr, "DRY-RUN — no changes will be made\n")
			}

			summary, err := eng.Install(context.Background(), manifest)
			if err != nil {
				return err
			}

			if summary != nil {
				printSummary(summary)
			}
			if summary != nil && summary.Failed > 0 {
				return fmt.Errorf("%d package(s) failed", summary.Failed)
			}
			return nil
		},
	}

	cmd.Flags().StringP("file", "f", "", "path to manifest file")
	cmd.Flags().BoolP("dry-run", "n", false, "preview installation without making changes")
	cmd.Flags().StringP("proxy", "p", "", "HTTP proxy URL")
	cmd.Flags().StringP("mirror", "m", "", "package mirror source")
	cmd.Flags().BoolP("skip-existing", "s", false, "skip already installed packages")
	cmd.Flags().Bool("skip-checks", false, "skip pre-flight checks")
	return cmd
}

func resolveManifest(defaultPath string) string {
	if defaultPath != "" {
		if _, err := os.Stat(defaultPath); err == nil {
			return defaultPath
		}
	}
	for _, name := range []string{"sis.yaml", "software_list.txt", "packages.txt"} {
		if _, err := os.Stat(name); err == nil {
			return name
		}
	}
	return ""
}

func printChecks(suite *engine.CheckSuite) {
	for _, r := range suite.Results {
		icon := "✓"
		if r.Status == engine.CheckFail {
			icon = "✗"
		} else if r.Status == engine.CheckWarn {
			icon = "⚠"
		}
		fmt.Fprintf(os.Stderr, "[%s] %s — %s\n", icon, r.Name, r.Message)
	}
}

func printSummary(s *engine.Summary) {
	fmt.Fprintf(os.Stderr, "\nSummary: %d total, %d succeeded, %d skipped, %d failed (%.1fs)\n",
		s.Total, s.Succeeded, s.Skipped, s.Failed, s.Duration.Seconds())

	for _, r := range s.Results {
		icon := "✓"
		if r.Status == engine.StatusFailed {
			icon = "✗"
		} else if r.Status == engine.StatusSkipped {
			icon = "⚠"
		}
		desc := string(r.Status)
		if len(r.Errors) > 0 {
			desc = r.Errors[0]
		}
		fmt.Fprintf(os.Stderr, "  %s %s — %s\n", icon, r.Package.ID, desc)
	}
}
