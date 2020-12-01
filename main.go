package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/bitrise-io/go-steputils/stepconf"
	"github.com/bitrise-io/go-utils/log"
)

const (
	stepID = "cache-pull"
)

const (
	cachePullEndTimePath = "/tmp/cache_pull_end_time"
)

// Config stores the step inputs.
type Config struct {
	CacheAPIURL           string `env:"cache_api_url"`
	DebugMode             bool   `env:"is_debug_mode,opt[true,false]"`
	AllowFallback         bool   `env:"allow_fallback,opt[true,false]"`
	ExtractToRelativePath bool   `env:"extract_to_relative_path,opt[true,false]"`

	StackID   string `env:"BITRISEIO_STACK_ID"`
	BuildSlug string `env:"BITRISE_BUILD_SLUG"`
}

// downloadCacheArchive downloads the cache archive and returns the downloaded file's path.
// If the URI points to a local file it returns the local paths.
func downloadCacheArchive(url string, buildSlug string) (string, error) {
	if strings.HasPrefix(url, "file://") {
		return strings.TrimPrefix(url, "file://"), nil
	}

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Warnf("Failed to close response body: %s", err)
		}
	}()

	if resp.StatusCode != 200 {
		responseBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}

		return "", fmt.Errorf("non success response code: %d, body: %s", resp.StatusCode, string(responseBytes))
	}

	const cacheArchivePath = "/tmp/cache-archive.tar"
	f, err := os.Create(cacheArchivePath)
	if err != nil {
		return "", fmt.Errorf("failed to open the local cache file for write: %s", err)
	}

	var bytesWritten int64
	bytesWritten, err = io.Copy(f, resp.Body)
	if err != nil {
		return "", err
	}

	data := map[string]interface{}{
		"cache_archive_size": bytesWritten,
		"build_slug":         buildSlug,
	}
	log.Debugf("Size of downloaded cache archive: %d Bytes", bytesWritten)
	log.RInfof(stepID, "cache_fallback_archive_size", data, "Size of downloaded cache archive: %d Bytes", bytesWritten)

	return cacheArchivePath, nil
}

// performRequest performs an http request and returns the response's body, if the status code is 200.
func performRequest(url string) (io.ReadCloser, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		defer func() {
			if err := resp.Body.Close(); err != nil {
				log.Warnf("Failed to close response body: %s", err)
			}
		}()

		responseBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		return nil, fmt.Errorf("non success response code: %d, body: %s", resp.StatusCode, string(responseBytes))
	}

	return resp.Body, nil
}

// getCacheDownloadURL gets the given build's cache download URL.
func getCacheDownloadURL(cacheAPIURL string) (string, error) {
	req, err := http.NewRequest("GET", cacheAPIURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %s", err)
	}

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %s", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Warnf("Failed to close response body: %s", err)
		}
	}()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("request sent, but failed to read response body (http-code: %d): %s", resp.StatusCode, body)
	}

	if resp.StatusCode < 200 || resp.StatusCode > 202 {
		return "", fmt.Errorf("build cache not found: probably cache not initialised yet (first cache push initialises the cache), nothing to worry about ;)")
	}

	var respModel struct {
		DownloadURL string `json:"download_url"`
	}
	if err := json.Unmarshal(body, &respModel); err != nil {
		return "", fmt.Errorf("failed to parse JSON response (%s): %s", body, err)
	}

	if respModel.DownloadURL == "" {
		return "", errors.New("download URL not included in the response")
	}

	return respModel.DownloadURL, nil
}

// parseStackID reads the stack id from the given json bytes.
func parseStackID(b []byte) (string, error) {
	type ArchiveInfo struct {
		StackID string `json:"stack_id,omitempty"`
	}
	var archiveInfo ArchiveInfo
	if err := json.Unmarshal(b, &archiveInfo); err != nil {
		return "", err
	}
	return archiveInfo.StackID, nil
}

// failf prints an error and terminates the step.
func failf(format string, args ...interface{}) {
	log.Errorf(format, args...)
	os.Exit(1)
}

func isBitriseCacheAPIURL(url string) bool {
	return url == os.Getenv("BITRISE_CACHE_API_URL")
}

func writeCachePullTimestamp() (err error) {
	f, err := os.Create(cachePullEndTimePath)
	if err != nil {
		return err
	}

	defer func() {
		if fErr := f.Close(); fErr != nil {
			err = fErr
		}
	}()

	_, err = f.WriteString(strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10))

	return err
}

func main() {
	var conf Config
	if err := stepconf.Parse(&conf); err != nil {
		failf(err.Error())
	}
	stepconf.Print(conf)
	log.SetEnableDebugLog(conf.DebugMode)

	if conf.CacheAPIURL == "" {
		log.Warnf("No Cache API URL specified, there's no cache to use, exiting.")
		return
	}

	startTime := time.Now()

	var cacheReader io.Reader
	var cacheURI string

	if strings.HasPrefix(conf.CacheAPIURL, "file://") {
		cacheURI = conf.CacheAPIURL

		fmt.Println()
		log.Infof("Using local cache archive")

		pth := strings.TrimPrefix(conf.CacheAPIURL, "file://")

		var err error
		cacheReader, err = os.Open(pth)
		if err != nil {
			failf("Failed to open cache archive file: %s", err)
		}
	} else {
		fmt.Println()
		log.Infof("Downloading remote cache archive")

		var err error
		if isBitriseCacheAPIURL(conf.CacheAPIURL) {
			cacheURI, err = getCacheDownloadURL(conf.CacheAPIURL)
			if err != nil {
				failf("Failed to get cache download url: %s", err)
			}
		} else {
			cacheURI = conf.CacheAPIURL
		}

		cacheReader, err = performRequest(cacheURI)
		if err != nil {
			failf("Failed to perform cache download request: %s", err)
		}
	}

	cacheRecorderReader := NewRestoreReader(cacheReader)

	r, hdr, compressed, err := readFirstEntry(cacheRecorderReader)
	if err != nil {
		failf("Failed to get first archive entry: %s", err)
	}

	cacheRecorderReader.Restore()

	currentStackID := strings.TrimSpace(conf.StackID)
	if len(currentStackID) > 0 {
		fmt.Println()
		log.Infof("Checking archive and current stacks")
		log.Printf("current stack id: %s", currentStackID)

		if filepath.Base(hdr.Name) == "archive_info.json" {
			b, err := ioutil.ReadAll(r)
			if err != nil {
				failf("Failed to read first archive entry: %s", err)
			}

			archiveStackID, err := parseStackID(b)
			if err != nil {
				failf("Failed to parse first archive entry: %s", err)
			}
			log.Printf("archive stack id: %s", archiveStackID)

			if archiveStackID != currentStackID {
				log.Warnf("Cache was created on stack: %s, current stack: %s", archiveStackID, currentStackID)
				log.Warnf("Skipping cache pull, because of the stack has changed")

				if err := writeCachePullTimestamp(); err != nil {
					failf("Couldn't save cache pull timestamp: %s", err)
				}

				os.Exit(0)
			}
		} else {
			log.Warnf("cache archive does not contain stack information, skipping stack check")
		}
	}

	fmt.Println()
	log.Infof("Extracting cache archive")

	if err := extractCacheArchive(cacheRecorderReader, conf.ExtractToRelativePath, compressed); err != nil {
		if !conf.AllowFallback {
			failf("Failed to uncompress cache archive stream: %s", err)
		}

		log.Warnf("Failed to uncompress cache archive stream: %s", err)
		log.Warnf("Downloading the archive file and trying to uncompress using tar tool")
		data := map[string]interface{}{
			"archive_bytes_read": cacheRecorderReader.BytesRead,
			"build_slug":         conf.BuildSlug,
		}
		log.RInfof(stepID, "cache_archive_fallback", data, "Failed to uncompress cache archive stream: %s", err)

		pth, err := downloadCacheArchive(cacheURI, conf.BuildSlug)
		if err != nil {
			failf("Fallback failed, unable to download cache archive: %s", err)
		}

		if err := uncompressArchive(pth, conf.ExtractToRelativePath, compressed); err != nil {
			failf("Fallback failed, unable to uncompress cache archive file: %s", err)
		}
	} else {
		data := map[string]interface{}{
			"cache_archive_size": cacheRecorderReader.BytesRead,
			"build_slug":         conf.BuildSlug,
		}
		log.Debugf("Size of extracted cache archive: %d Bytes", cacheRecorderReader.BytesRead)
		log.RInfof(stepID, "cache_archive_size", data, "Size of extracted cache archive: %d Bytes", cacheRecorderReader.BytesRead)
	}

	if err := writeCachePullTimestamp(); err != nil {
		failf("Couldn't save cache pull timestamp: %s", err)
	}

	fmt.Println()
	log.Donef("Done")
	log.Printf("Took: " + time.Since(startTime).String())
}
