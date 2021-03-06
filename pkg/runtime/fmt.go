package runtime

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"

	goruntime "runtime"

	"github.com/FuzzyMonkeyCo/monkey/pkg/as"
	"github.com/bazelbuild/buildtools/build"
	"github.com/bazelbuild/buildtools/buildifier/utils"
	"github.com/bazelbuild/buildtools/warn"
	"github.com/bazelbuild/buildtools/wspace"
)

// Heavily inspired from https://github.com/bazelbuild/buildtools/blob/174cbb4ba7d15a3ad029c2e4ee4f30ea4d76edce/buildifier/buildifier.go

var (
// Pass down debug flags into build package
// build.DisableRewrites = []string{}
// build.AllowSort = []string{}
)

// Format standardizes Starlark codes
func Format(W bool) error {
	inputType := "default" // generic Starlark files
	mode := "check"
	if W {
		mode = "fix"
	}
	lint := "warn"                   // "off" or "fix"
	warningsList := warn.AllWarnings // warn.DefaultWarnings
	log.Println("[DBG] AllWarnings", warn.AllWarnings)
	log.Println("[DBG] DefaultWarnings", warn.DefaultWarnings)
	return runFormat(inputType, mode, lint, warningsList)
}

// FmtError contains fmt diagnostics
type FmtError []string

var _ error = (FmtError)(nil)

// Error makes FmtError implement the error interface
func (e FmtError) Error() string {
	var s strings.Builder
	for i := 0; i < len(e); i += 3 {
		fmt.Fprintf(&s, "%s %s %s", e[i+0], e[i+1], e[i+2])
	}
	return s.String()
}

func runFormat(inputType, mode, lint string, warningsList []string) (err error) {
	tf := &utils.TempFile{}
	defer tf.Clean()

	files := []string{localCfg}
	recursively := false
	if recursively {
		places := []string{"."}
		if files, err = utils.ExpandDirectories(&places); err != nil {
			log.Println("[ERR]", err)
			return
		}
	}
	var diagnostics *utils.Diagnostics
	if diagnostics, err = fmtFiles(inputType, mode, lint, warningsList, files, tf); err != nil {
		return
	}

	var fmtErrs []string
	for _, f := range diagnostics.Files {
		if !f.Formatted {
			fmtErrs = append(fmtErrs,
				fmt.Sprintf("%s:1:", f.Filename),
				"(fmt)",
				"use fmt with flag -w to reformat this file",
			)
		}
		for _, w := range f.Warnings {
			msg := w.Message
			switch w.Category {
			case "module-docstring":
				continue
			case "function-docstring-args":
				if strings.HasPrefix(msg, `Argument "ctx" is not documented.`) && strings.Contains(msg, "(ctx):\n") {
					// Skip warning about our special ctx positional argument
					continue
				}
			case "name-conventions":
				msg = strings.ReplaceAll(msg,
					"be lower_snake_case (for variables), UPPER_SNAKE_CASE (for constants), or UpperCamelCase ending with 'Info' (for providers).",
					"be lower_snake_case.",
				)
			}
			msg = strings.ReplaceAll(msg, "Buildifier", "`fmt`")

			catFmt := "(%s)"
			if !w.Actionable {
				catFmt = "[%s]"
			}

			fmtErrs = append(fmtErrs,
				fmt.Sprintf("%s:%d:", f.Filename, w.Start.Line),
				fmt.Sprintf(catFmt, w.Category),
				msg,
				// w.URL,
			)
		}
	}
	if len(fmtErrs) != 0 {
		err = FmtError(fmtErrs)
	}
	return
}

func fmtFiles(inputType, mode, lint string, warningsList, files []string, tf *utils.TempFile) (
	diags *utils.Diagnostics,
	err error,
) {
	// Decide how many file reads to run in parallel.
	// At most 100, and at most one per 10 input files.
	nworker := 100
	if n := (len(files) + 9) / 10; nworker > n {
		nworker = n
	}
	goruntime.GOMAXPROCS(nworker + 1)

	// Start nworker workers reading stripes of the input
	// argument list and sending the resulting data on
	// separate channels. file[k] is read by worker k%nworker
	// and delivered on ch[k%nworker].
	type result struct {
		file string
		data []byte
		err  error
	}

	ch := make([]chan result /*,*/, nworker)
	for i := 0; i < nworker; i++ {
		ch[i] = make(chan result, 1)
		go func(i int) {
			for j := i; j < len(files); j += nworker {
				file := files[j]
				data, err := ioutil.ReadFile(file)
				ch[i] <- result{file, data, err}
			}
		}(i)
	}

	fileDiagnostics := []*utils.FileDiagnostics{}
	var errs strings.Builder
	first := true

	// Process files. The processing still runs in a single goroutine
	// in sequence. Only the reading of the files has been parallelized.
	// The goal is to optimize for runs where most files are already
	// formatted correctly, so that reading is the bulk of the I/O.
	for i, file := range files {
		res := <-ch[i%nworker]
		if res.file != file {
			err = fmt.Errorf("expected file %q but received %q", file, res.file)
			log.Println("[ERR]", err)
			return
		}
		if res.err != nil {
			log.Println("[ERR]", res.err)
			if !first {
				errs.WriteString("\n")
			}
			first = false
			errs.WriteString(res.err.Error())
			continue
		}
		fd, e := fmtFile(inputType, mode, lint, file, res.data, warningsList, len(files) > 1, tf)
		if fd != nil {
			fileDiagnostics = append(fileDiagnostics, fd)
		}
		if e != nil {
			if !first {
				errs.WriteString("\n")
			}
			first = false
			errs.WriteString(e.Error())
		}
	}
	if errLines := errs.String(); errLines != "" {
		err = errors.New(errLines)
		log.Println("[ERR]", err)
		return
	}
	diags = utils.NewDiagnostics(fileDiagnostics...)
	return
}

func fmtFile(inputType, mode, lint, filename string, data []byte, warningsList []string, displayFilenames bool, tf *utils.TempFile) (
	diags *utils.FileDiagnostics,
	err error,
) {
	displayFilename := filename
	parser := utils.GetParser(inputType)

	var f *build.File
	if f, err = parser(displayFilename, data); err != nil {
		// This is a parse error. They start with file:line:
		diags = utils.InvalidFileDiagnostics(displayFilename)
		return
	}

	if absoluteFilename, e := filepath.Abs(displayFilename); e == nil {
		f.WorkspaceRoot, f.Pkg, f.Label = wspace.SplitFilePath(absoluteFilename)
	}

	verbose := true
	warnings := utils.Lint(f, lint, &warningsList, verbose)
	diags = utils.NewFileDiagnostics(f.DisplayPath(), warnings)

	ndata := build.Format(f)
	switch mode {
	case "check":
		if !bytes.Equal(data, ndata) {
			diags.Formatted = false
			return
		}
	case "fix":
		if bytes.Equal(data, ndata) {
			return
		}
		if err = ioutil.WriteFile(filename, ndata, 0666); err != nil { // TODO: use same perms as filename
			log.Println("[ERR]", err)
			return
		}
		as.ColorNFO.Printf("fixed %s\n", f.DisplayPath())
	}
	return
}
