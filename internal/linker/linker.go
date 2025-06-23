package linker

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"reflect"
	"terralink/internal/ignore"

	log "github.com/sirupsen/logrus"
)

// LoadedModules is a type alias for a slice of strings representing module names.
type LoadedModules []string

// Linker is the main struct responsible for orchestrating the linking,
// unlinking, and checking of Terraform modules. It uses an IgnoreMatcher
// to determine which files and directories to skip.
type Linker struct {
	matcher *ignore.IgnoreMatcher
}

// NewLinker creates and returns a new Linker instance.
func NewLinker(matcher *ignore.IgnoreMatcher) *Linker {
	return &Linker{
		matcher: matcher,
	}
}

// fileProcessor defines the function signature for processing a single HCL file.
// This uses generics to allow different return types from the processor.
type fileProcessor[T any] func(file *HCLFile) (T, error)

// processFiles is a generic function that walks the directory starting from scanPath.
// It applies the given processor function to each non-ignored Terraform file it finds
// and aggregates the results into a map where the key is the file path.
func processFiles[T any](scanPath string, matcher *ignore.IgnoreMatcher, processor fileProcessor[T]) (map[string]T, error) {
	results := make(map[string]T)

	walkErr := filepath.WalkDir(scanPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err // Propagate errors from walking, e.g., permission denied
		}

		// Check if the individual file should be ignored.
		if matcher.ShouldIgnore(path) {
			return nil
		}

		// Read and parse the HCL file.
		hclFile, err := NewHCLFile(path)
		if err != nil {
			log.Errorf("Warning: skipping file due to parsing error: %v\n", err)
			return nil
		}

		// Apply the specific processing logic to the file.
		result, err := processor(hclFile)
		if err != nil {
			return fmt.Errorf("error processing file %s: %w", path, err)
		}

		// Only add to results if it's a non-zero value (e.g., changes > 0 or modules found)
		//var zero T
		//if any(result) != any(zero) {
		//	results[path] = result
		//}
		if !reflect.ValueOf(result).IsZero() {
			results[path] = result
		}

		return nil
	})

	if walkErr != nil {
		return nil, fmt.Errorf("error walking directories: %w", walkErr)
	}

	return results, nil
}

// Check scans the given path for Terraform files and reports which modules
// in each file are currently in a "loaded" (dev) state.
func (l *Linker) Check(scanPath string) (map[string]LoadedModules, error) {
	return processFiles(scanPath, l.matcher, func(hclFile *HCLFile) (LoadedModules, error) {
		var loadedModules LoadedModules
		for _, module := range hclFile.Modules() {
			if module.IsLoaded() {
				loadedModules = append(loadedModules, module.Name())
			}
		}
		if len(loadedModules) == 0 {
			return loadedModules, nil // Return nil so it's skipped in results map
		}
		return loadedModules, nil
	})
}

// DevLoad scans for Terraform files and modifies module blocks that have a
// terralink dev annotation, switching them to use a local path.
func (l *Linker) DevLoad(scanPath string) (map[string]int, error) {
	return processFiles(scanPath, l.matcher, func(hclFile *HCLFile) (int, error) {
		changes := 0
		for _, module := range hclFile.Modules() {
			loaded, err := module.Load()
			if err != nil {
				return 0, fmt.Errorf("in module '%s': %w", module.Name(), err)
			}
			if loaded {
				changes++
			}
		}

		if changes > 0 {
			if err := hclFile.Write(); err != nil {
				return 0, err
			}
		}
		return changes, nil
	})
}

// DevUnload scans for Terraform files and reverts module blocks from a
// local dev state back to their original source and version.
func (l *Linker) DevUnload(scanPath string) (map[string]int, error) {
	return processFiles(scanPath, l.matcher, func(hclFile *HCLFile) (int, error) {
		changes := 0
		for _, module := range hclFile.Modules() {
			unloaded, err := module.Unload()
			if err != nil {
				return 0, fmt.Errorf("in module '%s': %w", module.Name(), err)
			}
			if unloaded {
				changes++
			}
		}

		if changes > 0 {
			if err := hclFile.Write(); err != nil {
				return 0, err
			}
		}
		return changes, nil
	})
}
