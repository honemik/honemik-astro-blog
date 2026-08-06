package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type commonOptions struct {
	Root        string
	JSON        bool
	NoColor     bool
	Yes         bool
	Interactive bool
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		printHelp()
		return nil
	}

	switch args[0] {
	case "help", "-h", "--help":
		if len(args) > 1 {
			printCommandHelp(args[1])
			return nil
		}
		printHelp()
		return nil
	case "import", "add", "new", "i":
		return runImport(args[1:], false)
	case "inspect":
		return runInspect(args[1:])
	case "list", "ls", "posts":
		return runList(args[1:])
	case "view", "show", "cat", "v":
		return runView(args[1:])
	case "draft":
		return runDraftPublish(args[1:], true)
	case "publish":
		return runDraftPublish(args[1:], false)
	case "delete", "rm", "trash":
		return runDelete(args[1:])
	case "clean-images", "orphans":
		return runCleanImages(args[1:])
	default:
		if strings.HasPrefix(args[0], "-") {
			return runImport(args, true)
		}
		if strings.HasSuffix(strings.ToLower(args[0]), ".odt") {
			return runImport(args, false)
		}
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func printHelp() {
	fmt.Print(`odt2post - ODT import and post management CLI

Common flows:
  odt2post add note.odt
  odt2post ls
  odt2post show 2026-05-01
  odt2post show "post title" -images

Usage:
  odt2post import <file.odt> [flags]
  odt2post inspect <file.odt> [flags]
  odt2post list [flags]
  odt2post view <post-selector> [flags]
  odt2post draft <post-selector> [flags]
  odt2post publish <post-selector> [flags]
  odt2post delete <post-selector> [flags]
  odt2post clean-images [flags]

Legacy import still works:
  odt2post -f note.odt -title "My Title"

Common flags:
  -root <path>       Blog root. Auto-detected when omitted.
  -json              Print machine-readable JSON where supported.
  -yes               Confirm write/delete actions.
  -no-color          Disable styled output. Reserved for scripts.
`)
}

func printCommandHelp(command string) {
	switch command {
	case "import", "add", "new", "i":
		fmt.Print(`Import an ODT file:
  odt2post add note.odt
  odt2post add note.odt -tags Blog,Note -categories Life
  odt2post inspect note.odt

Useful flags:
  -title, -desc, -tags, -categories, -series, -date, -draft
  -image-format webp|original, -image-quality 82, -max-width 1600
  -dry-run, -json, -root
`)
	case "list", "ls", "posts":
		fmt.Print(`List posts:
  odt2post ls
  odt2post ls -images
  odt2post ls -drafts
  odt2post ls -missing-images
  odt2post ls -json
`)
	case "view", "show", "cat", "v":
		fmt.Print(`View a post:
  odt2post show 2026-05-01
  odt2post show "title text" -images
  odt2post show filename.md -raw

Selectors can be a date, title text, slug text, or exact filename.
The normal view wraps long paragraphs for terminal reading; use -raw for Markdown.
`)
	case "delete", "rm", "trash":
		fmt.Print(`Delete safely:
  odt2post rm 2026-05-01
  odt2post rm 2026-05-01 -with-images -yes

Without -yes this only previews. Files are moved into post_tool/.trash/.
`)
	default:
		printHelp()
	}
}

func addCommonFlags(fs *flag.FlagSet, opts *commonOptions) {
	fs.StringVar(&opts.Root, "root", "", "Blog root. Auto-detected when omitted.")
	fs.BoolVar(&opts.JSON, "json", false, "Print JSON output.")
	fs.BoolVar(&opts.NoColor, "no-color", false, "Disable colored/styled output.")
	fs.BoolVar(&opts.Yes, "yes", false, "Confirm write/delete actions.")
	fs.BoolVar(&opts.Interactive, "interactive", false, "Allow interactive prompts.")
}

func parseFlagSet(fs *flag.FlagSet, args []string) (map[string]bool, error) {
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(normalizeFlagArgs(fs, args)); err != nil {
		return nil, err
	}
	seen := make(map[string]bool)
	fs.Visit(func(f *flag.Flag) {
		seen[f.Name] = true
	})
	return seen, nil
}

type boolFlag interface {
	IsBoolFlag() bool
}

func normalizeFlagArgs(fs *flag.FlagSet, args []string) []string {
	var flags []string
	var positional []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			positional = append(positional, args[i+1:]...)
			break
		}
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			positional = append(positional, arg)
			continue
		}

		flags = append(flags, arg)
		name := strings.TrimLeft(arg, "-")
		if idx := strings.Index(name, "="); idx >= 0 {
			name = name[:idx]
		}
		f := fs.Lookup(name)
		if f == nil {
			continue
		}
		if bf, ok := f.Value.(boolFlag); ok && bf.IsBoolFlag() {
			continue
		}
		if strings.Contains(arg, "=") {
			continue
		}
		if i+1 < len(args) {
			flags = append(flags, args[i+1])
			i++
		}
	}

	return append(flags, positional...)
}

func resolveBlogRoot(root string) (string, error) {
	if root != "" {
		abs, err := filepath.Abs(root)
		if err != nil {
			return "", err
		}
		if !isBlogRoot(abs) {
			return "", fmt.Errorf("%s does not look like this Astro blog root", abs)
		}
		return abs, nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	dir := cwd
	for {
		if isBlogRoot(dir) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	exe, err := os.Executable()
	if err == nil {
		dir = filepath.Dir(exe)
		for {
			if isBlogRoot(dir) {
				return dir, nil
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	return "", errors.New("could not auto-detect blog root; pass -root")
}

func isBlogRoot(dir string) bool {
	if _, err := os.Stat(filepath.Join(dir, "astro.config.mjs")); err != nil {
		return false
	}
	if _, err := os.Stat(filepath.Join(dir, "src", "content", "config.ts")); err != nil {
		return false
	}
	return true
}

func postsDir(root string) string {
	return filepath.Join(root, "src", "content", "posts")
}

func postImagesDir(root string) string {
	return filepath.Join(root, "public", "img", "post")
}

func trashDir(root string) string {
	return filepath.Join(root, "post_tool", ".trash")
}
