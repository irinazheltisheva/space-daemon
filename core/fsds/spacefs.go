package fsds

import (
	"context"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/FleekHQ/space-daemon/log"

	"github.com/FleekHQ/space-daemon/core/space/domain"

	"github.com/FleekHQ/space-daemon/core/space"
)

// EntryNotFound error when a directory is not found
var EntryNotFound = syscall.ENOENT // errors.New("Directory entry not found")
var baseDir = NewDirEntryWithMode(
	domain.DirEntry{
		Path:  "/",
		IsDir: true,
		Name:  "",
	},
	RestrictedDirAccessMode,
)

// SpaceFSDataSource is an implementation of the FSDataSource
// It interacts with the Space Service Layer to provide data
type SpaceFSDataSource struct {
	service    space.Service
	tlfSources []*TLFDataSource
}

func NewSpaceFSDataSource(service space.Service, configOptions ...FSDataSourceConfig) *SpaceFSDataSource {
	config := dataSourceConfig{}
	for _, configure := range configOptions {
		configure(&config)
	}

	return &SpaceFSDataSource{
		service:    service,
		tlfSources: config.tlfSources,
	}
}

// Get returns the DirEntry information for item at path
func (d *SpaceFSDataSource) Get(ctx context.Context, path string) (*DirEntry, error) {
	log.Debug("FSDS.Get", "path:"+path)
	// handle quick lookup of home directory
	if isBaseDirectory(path) {
		return baseDir, nil
	}

	dataSource := d.findTLFDataSource(path)
	if dataSource == nil {
		return nil, EntryNotFound
	}

	result, err := dataSource.Get(ctx, strings.TrimPrefix(path, dataSource.basePath))
	if result != nil {
		result.entry.Path = d.prefixBasePath(dataSource.basePath, result.entry.Path)
	}

	return result, err
}

func (d *SpaceFSDataSource) findTLFDataSource(path string) *TLFDataSource {
	for _, i := range d.tlfSources {
		if strings.HasPrefix(path, i.basePath) {
			return i
		}
	}

	return nil
}

// GetChildren returns list of entries in a path
func (d *SpaceFSDataSource) GetChildren(ctx context.Context, path string) ([]*DirEntry, error) {
	log.Debug("FSDS.GetChildren", "path:"+path)
	if isBaseDirectory(path) {
		return d.getTopLevelDirectories(), nil
	}

	dataSource := d.findTLFDataSource(path)
	if dataSource == nil {
		return nil, EntryNotFound
	}

	result, err := dataSource.GetChildren(ctx, strings.TrimPrefix(path, dataSource.basePath))

	// format results
	if result != nil {
		for _, entry := range result {
			entry.entry.Path = d.prefixBasePath(dataSource.basePath, entry.entry.Path)
		}
	}

	return result, err
}

// Open is invoked to read the content of a file
func (d *SpaceFSDataSource) Open(ctx context.Context, path string) (ReadSeekCloser, error) {
	log.Debug("FSDS.Open", "path:"+path)
	dataSource := d.findTLFDataSource(path)
	if dataSource == nil {
		return nil, EntryNotFound
	}

	return dataSource.Open(ctx, strings.TrimPrefix(path, dataSource.basePath))
}

// CreateEntry creates a directory or file based on the mode at the path
func (d *SpaceFSDataSource) CreateEntry(ctx context.Context, path string, mode os.FileMode) (*DirEntry, error) {
	log.Debug("FSDS.CreateEntry", "path:"+path)
	dataSource := d.findTLFDataSource(path)
	if dataSource == nil {
		return nil, EntryNotFound
	}

	result, err := dataSource.CreateEntry(ctx, strings.TrimPrefix(path, dataSource.basePath), mode)
	if result != nil {
		result.entry.Path = d.prefixBasePath(dataSource.basePath, result.entry.Path)
	}

	return result, err
}

// Returns list of top level entry
// For now we only return the files directory
func (d *SpaceFSDataSource) getTopLevelDirectories() []*DirEntry {
	var directories []*DirEntry

	for _, ds := range d.tlfSources {
		directories = append(directories, NewDirEntryWithMode(
			domain.DirEntry{
				Path:  ds.basePath,
				IsDir: true,
				Name:  ds.name,
				//Created:       "",
				//Updated:       "",
			},
			RestrictedDirAccessMode,
		))
	}
	return directories
}

// returns the path with the parent base path prefixed
func (d *SpaceFSDataSource) prefixBasePath(basePath, path string) string {
	return fmt.Sprintf("%s%s", basePath, path)
}
