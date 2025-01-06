package go_chef_code_gen_utils

import (
	"bytes"
	"embed"
	"errors"
	"io/fs"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type MR func(a, b []byte) []byte
type Builder struct {
	Prefix    string
	structure embed.FS
	MergeMode bool
	MergeFn   map[string]MR
}

func NewBuilder(prefix string, structure embed.FS, mergeMode bool) *Builder {
	return &Builder{Prefix: prefix, structure: structure, MergeMode: mergeMode}
}
func (b *Builder) Build(prefix, destination string, obj any) {
	b.process(prefix, destination, obj)
}
func (b *Builder) process(prefix, destination string, obj any) {
	dir, err := b.structure.ReadDir(prefix)
	if err != nil {
		log.Fatal(err)
	}
	Mkdir(destination)
	for _, fl := range dir {
		secondPath := filepath.Join(prefix, fl.Name())
		if fl.IsDir() {
			directory := b.cleanPath(secondPath)
			newDirectory := b.RewritePath(directory, obj)
			if strings.HasSuffix(newDirectory, b.OSSlash("/")) {
				continue
			}
			newDirectoryPath := filepath.Join(destination, newDirectory)
			Mkdir(newDirectoryPath)

			if b.checkWindows() {
				secondPath = filepath.ToSlash(secondPath)
			}
			b.process(secondPath, destination, obj)
			b.CleanEmptyDir(newDirectoryPath)
			continue
		}

		b.buildFile(secondPath, destination, obj)
	}
	b.CleanEmptyDir(destination)
}

func (b *Builder) CleanEmptyDir(destination string) {
	dir, err := os.ReadDir(destination)
	if err != nil {
		slog.Error("Error reading directory", slog.Any("error", err), slog.String("destination", destination))
		os.Exit(1)
	}
	if len(dir) == 0 {
		err = os.Remove(destination)
		if err != nil {
			slog.Error("Error removing directory", slog.Any("error", err), slog.String("destination", destination))
			os.Exit(1)
		}
	}
}
func (b *Builder) checkWindows() bool {
	return os.Getenv("GOOS") == "windows" ||
		strings.Contains(strings.ToLower(os.Getenv("OS")), "windows")
}

func (b *Builder) cleanPath(path string) string {
	path = b.OSSlash(path)
	result, _ := strings.CutPrefix(path, b.Prefix)
	return result
}

func (b *Builder) OSSlash(path string) string {
	if b.checkWindows() {
		path = filepath.ToSlash(path)
	}
	return path
}

func (b *Builder) buildFile(secondPath string, folder string, obj any) {
	secondPath = b.OSSlash(secondPath)
	rF, err := b.structure.ReadFile(secondPath)
	if err != nil {
		slog.Error("Error reading file", slog.Any("error", err), slog.String("secondPath", secondPath))
		os.Exit(1)
	}
	t, err := template.New("").Parse(string(rF))
	if err != nil {
		slog.Error("Error parsing template", slog.Any("error", err), slog.String("secondPath", secondPath))
		os.Exit(1)
	}
	if strings.HasSuffix(secondPath, "}}") {
		secondPath = b.RewritePath(secondPath, obj)
	}
	parsedPath, _ := strings.CutSuffix(secondPath, ".tmpl")
	parsedPath = b.cleanPath(parsedPath)

	newPath := b.RewritePath(parsedPath, obj)
	filePath := filepath.Join(
		folder,
		newPath,
	)

	if folder == filePath {
		return
	}
	//TODO rewrite
	builder := bytes.NewBuffer([]byte{})
	err = t.ExecuteTemplate(builder, "", obj)
	if err != nil {
		slog.Error("Error ExecuteTemplate", slog.Any("error", err), slog.String("secondPath", secondPath))
		os.Exit(1)
	}
	bt := builder.Bytes()
	if b.MergeMode {
		file, err := os.ReadFile(filePath)
		if errors.Is(err, fs.ErrNotExist) {
			slog.Error("Error reading file", slog.Any("error", err), slog.String("secondPath", secondPath))
		} else {
			ext := filepath.Ext(filePath)
			if val, ok := b.MergeFn[ext]; ok {
				bt = val(bt, file)
			} else {
				if bytes.Contains(file, bt) {
					return
				}
				file = append(file, "\n"...)
				file = append(file, bt...)
				file = append(file, "\n"...)
				bt = file
			}
		}
	} else {
		_, err = os.Create(filePath)
		if err != nil {
			slog.Error("Error creating file", slog.Any("error", err), slog.String("secondPath", secondPath))
			os.Exit(1)
		}
	}
	err = os.WriteFile(filePath, bt, os.ModePerm)
	if err != nil {
		slog.Error("Write string to file", slog.Any("error", err), slog.String("secondPath", secondPath))
		return
	}
}
func (b *Builder) RewritePath(folder string, obj any) string {
	t, err := template.New("").Parse(folder)
	if err != nil {
		slog.Error("Error parsing template", slog.Any("error", err), slog.String("folder", folder))
		os.Exit(1)
	}
	buffer := bytes.NewBufferString("")
	err = t.ExecuteTemplate(buffer, "", obj)
	if err != nil {
		slog.Error("Error ExecuteTemplate", slog.Any("error", err), slog.String("folder", folder))
		os.Exit(1)
	}
	return strings.ReplaceAll(buffer.String(), " ", "")
}

func Mkdir(path string) {
	var err error
	if path == "" {
		return
	}
	_, err = os.ReadDir(path)
	if errors.Is(err, fs.ErrNotExist) {
		err := os.Mkdir(path, os.ModePerm)
		if err != nil {
			slog.Error("Error creating directory", slog.Any("error", err), slog.String("path", path))
			os.Exit(1)
		}
	}
}
