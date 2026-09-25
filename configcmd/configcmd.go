// Package configcmd provides the shared `config` cobra command. Apps own where their
// config lives; this package owns the command-line interaction.
package configcmd

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/brohd11/goutil/executil"
	"github.com/brohd11/goutil/shellquote"
	"github.com/brohd11/goutil/sysopen"
	"github.com/spf13/cobra"
)

// Options describes an app's config target. Path and Dir are resolved when the command
// runs, so a changed HOME is seen. Ensure, when set, runs before anything is opened.
type Options struct {
	Path   func() (string, error)
	Dir    func() (string, error)
	Ensure func() error
}

var openPath = sysopen.OpenPath

// NewCommand builds the `config` command: it edits the config file in $EDITOR ($VISUAL,
// then Notepad on Windows), or opens the config directory with --dir.
func NewCommand(opts Options) *cobra.Command {
	var openDir bool
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Edit the application config",
		Long: `config opens the application's config file with $EDITOR (or $VISUAL when
$EDITOR is unset). On Windows, notepad.exe is used when neither variable is set. Use
--dir to open the containing config directory in the system file manager instead. A
missing config is materialized first when the app provides defaults.`,
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: false,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if opts.Ensure != nil {
				if err := opts.Ensure(); err != nil {
					return fmt.Errorf("ensure config: %w", err)
				}
			}
			if openDir {
				if opts.Dir == nil {
					return fmt.Errorf("config directory is not configured")
				}
				dir, err := opts.Dir()
				if err != nil {
					return err
				}
				return openPath(dir, false)
			}
			if opts.Path == nil {
				return fmt.Errorf("config path is not configured")
			}
			path, err := opts.Path()
			if err != nil {
				return err
			}
			return runEditor(cmd, path)
		},
	}
	cmd.Flags().BoolVar(&openDir, "dir", false, "open the config directory in the system file manager")
	return cmd
}

func runEditor(cmd *cobra.Command, path string) error {
	argv, err := editorArgs(runtime.GOOS)
	if err != nil {
		return err
	}
	ed, err := executil.CommandContext(cmd.Context(), append(argv, path)...)
	if err != nil {
		return fmt.Errorf("editor %s: %w", argv[0], err)
	}
	ed.Stdin = cmd.InOrStdin()
	ed.Stdout = cmd.OutOrStdout()
	ed.Stderr = cmd.ErrOrStderr()
	if err := ed.Run(); err != nil {
		return fmt.Errorf("editor %s: %w", argv[0], err)
	}
	return nil
}

// editorArgs takes goos so every platform's policy can be tested on one host.
func editorArgs(goos string) ([]string, error) {
	raw := strings.TrimSpace(os.Getenv("EDITOR"))
	name := "EDITOR"
	if raw == "" {
		raw = strings.TrimSpace(os.Getenv("VISUAL"))
		name = "VISUAL"
	}
	if raw == "" {
		if goos == "windows" {
			return []string{"notepad.exe"}, nil
		}
		return nil, fmt.Errorf("neither EDITOR nor VISUAL is set")
	}
	argv, err := shellquote.Split(raw)
	if err != nil {
		return nil, fmt.Errorf("parse $%s: %w", name, err)
	}
	if len(argv) == 0 {
		return nil, fmt.Errorf("$%s does not name an editor", name)
	}
	return argv, nil
}
