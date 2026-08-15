package selfupdate

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

// NewUpdateCommand returns the shared `update` cobra command: it compares the
// running binary's version against the latest release of repo ("owner/name")
// and, with no flags, installs the update in place (over wherever the binary
// lives). name is the display name used in messages ("repoview", "gossh").
//
// Apps with a fancier surface (gdaddon's --json/--interactive self-update)
// build their own command on Check/Apply instead.
func NewUpdateCommand(repo, name, version string) *cobra.Command {
	var checkOnly bool
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Check for a newer " + name + " release and install it over this binary",
		Long: `Update compares this binary's version against the latest ` + name + ` release.
With no flags it downloads and installs the update when one is available, in place
(over wherever this binary lives).

  --check   only report whether an update is available; don't install`,
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate(cmd.Context(), repo, name, version, checkOnly)
		},
	}
	cmd.Flags().BoolVar(&checkOnly, "check", false, "only check for an update; don't download or install")
	return cmd
}

func runUpdate(ctx context.Context, repo, name, version string, checkOnly bool) error {
	info, err := Check(ctx, repo, version)
	if err != nil {
		return err
	}

	if checkOnly {
		fmt.Printf("current:  %s\nlatest:   %s\n", info.Current, info.LatestTag)
		if info.Available {
			fmt.Println("update available")
		} else {
			fmt.Println("up to date")
		}
		return nil
	}

	if !info.Available {
		if version == "dev" {
			fmt.Println("dev build, skipping update")
		} else {
			fmt.Printf("%s is up to date (%s)\n", name, info.Current)
		}
		return nil
	}

	binDir, err := BinDir()
	if err != nil {
		return err
	}
	fmt.Printf("updating %s %s → %s\n", name, info.Current, info.LatestTag)
	report := func(format string, a ...any) { fmt.Printf(format+"\n", a...) }
	if err := Apply(ctx, repo, info, binDir, report); err != nil {
		return err
	}
	fmt.Printf("updated %s to %s\n", name, info.LatestTag)
	return nil
}
