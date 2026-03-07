package watcher

import (
	"context"
	"log/slog"
	"path/filepath"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fsnotify/fsnotify"
)

// FileChangedMsg is sent when the watched file is modified.
type FileChangedMsg struct{}

// FileErrorMsg is sent when a file watch error occurs.
type FileErrorMsg struct {
	Err error
}

// Watch monitors the directory containing timeLogFilePath for changes to the
// file named filename, sending FileChangedMsg or FileErrorMsg to the program.
func Watch(ctx context.Context, wg *sync.WaitGroup, program *tea.Program, timeLogFilePath string, filename string) error {
	defer wg.Done()

	slog.Debug("Starting filewatcher on", "filePath", timeLogFilePath)
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	defer func() {
		if err := w.Close(); err != nil {
			slog.Error("Failed to close watcher", "error", err.Error())
		}
	}()

	err = w.Add(filepath.Dir(timeLogFilePath))
	if err != nil {
		return err
	}

	for {
		select {
		case event, ok := <-w.Events:
			if !ok {
				return nil
			}
			if event.Op&(fsnotify.Write|
				fsnotify.Create|
				fsnotify.Rename) != 0 && filepath.Base(event.Name) == filename {
				program.Send(FileChangedMsg{})
			}
		case err, ok := <-w.Errors:
			if !ok {
				return nil
			}
			program.Send(FileErrorMsg{
				Err: err,
			})
		case <-ctx.Done():
			return nil
		}
	}
}
