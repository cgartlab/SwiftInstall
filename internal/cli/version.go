package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newVersionCmd(version, commit, date string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "显示版本信息",
		Run: func(cmd *cobra.Command, args []string) {
			if commit == "none" || commit == "" {
				fmt.Printf("sis %s\n", version)
			} else {
				short := commit
				if len(commit) > 7 {
					short = commit[:7]
				}
				fmt.Printf("sis %s (%s, %s)\n", version, short, date)
			}
		},
	}
}
