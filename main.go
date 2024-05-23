package main

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/pflag"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"go.abhg.dev/goldmark/toc"
)

var (
	inputName       string
	outputName      string
	frontMatterName string
	minDepth        int
	maxDepth        int
)

func init() {
	pflag.StringVarP(&inputName, "input", "i", "-",
		"name of output file")
	pflag.StringVarP(&outputName, "output", "o", "-",
		"name of output file")
	pflag.StringVarP(&frontMatterName, "front-matter", "f", "",
		"name of front matter file")
	pflag.IntVarP(&minDepth, "min-depth", "M", 1,
		"min depth to be included in table of contents")
	pflag.IntVarP(&maxDepth, "max-depth", "m", 2,
		"min depth to be included in table of contents")
}

func croak(s string, args ...any) {
	msg := fmt.Sprintf(s, args...)
	fmt.Fprintf(os.Stderr, "%s: %s\n", os.Args[0], msg)
	os.Exit(1)
}

func readFile(name string) ([]byte, error) {
	if name == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("(stdin): %v", err)
		}

		return data, nil
	}

	in, err := os.Open(name)
	if err != nil {
		return nil, fmt.Errorf("file %q: %v", name, err)
	}
	defer in.Close()
	if data, err := io.ReadAll(in); err != nil {
		return nil, fmt.Errorf("file %q: %v", name, err)
	} else {
		return data, nil
	}
}

func outputHandle() (io.Writer, func()) {
	if outputName == "-" {
		return os.Stdout, func() {}
	}

	fh, err := os.Create(outputName)
	if err != nil {
		croak("can't open %q for writing: %v", outputName, err)
	}

	return fh, func() { fh.Close() }
}

func readFileOrDie(name string) []byte {
	if name == "" {
		return nil
	}
	bs, err := readFile(name)
	if err != nil {
		croak("can't read: %v", err)
	}
	return bs
}

func main() {
	pflag.Parse()

	if inputName == "" {
		croak("input file name is required")
	}
	if outputName == "" {
		croak("output file name is required")
	}

	fmd := goldmark.New()

	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM, extension.Footnote),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		))

	fmSource := readFileOrDie(frontMatterName)

	fmDoc := fmd.Parser().Parse(text.NewReader(fmSource))
	source := readFileOrDie(inputName)
	if len(source) == 0 {
		croak("no data read from input")
	}
	doc := md.Parser().Parse(text.NewReader(source))
	tocTree, err := toc.Inspect(doc, source, toc.MinDepth(minDepth), toc.MaxDepth(maxDepth))
	if err != nil {
		croak("while preparing Table of Contents", err)
	}

	out, closer := outputHandle()
	defer closer()

	fmd.Renderer().Render(out, fmSource, fmDoc)
	md.Renderer().Render(out, source, toc.RenderList(tocTree))
	md.Renderer().Render(out, source, doc)
}
