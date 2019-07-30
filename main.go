package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bitrise-io/go-steputils/stepconf"
	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/log"
)

const cacheArchivePath = "/tmp/cache-archive.tar"

// GenerateDownloadURLRespModel ...
type GenerateDownloadURLRespModel struct {
	DownloadURL string `json:"download_url"`
}

// Config stores the step inputs
type Config struct {
	CacheAPIURL string `env:"cache_api_url"`
	DebugMode   bool   `env:"is_debug_mode,opt[true,false]"`
	StackID     string `env:"BITRISE_STACK_ID"`
}

func downloadCacheArchive(url string) (string, error) {
	if strings.HasPrefix(url, "file://") {
		return strings.TrimPrefix(url, "file://"), nil
	}

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf(" [!] Failed to close Archive download response body: %s", err)
		}
	}()

	if resp.StatusCode != 200 {
		responseBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}

		return "", fmt.Errorf("non success response code: %d, body: %s", resp.StatusCode, string(responseBytes))
	}

	f, err := os.Create(cacheArchivePath)
	if err != nil {
		return "", fmt.Errorf("failed to open the local cache file for write: %s", err)
	}

	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return "", err
	}

	return cacheArchivePath, nil
}

func uncompressArchive(pth string) error {
	cmd := command.New("tar", "-xPf", pth)
	return cmd.Run()
}

func extractCacheArchive(r io.Reader) error {
	cmd := command.New("tar", "-xPf", "/dev/stdin")
	cmd.SetStdin(r)
	if output, err := cmd.RunAndReturnTrimmedCombinedOutput(); err != nil {
		return fmt.Errorf("failed to extract tar archive, output: %s, error: %s", output, err)
	}

	if rc, ok := r.(io.ReadCloser); ok {
		return rc.Close()
	}
	return nil
}

func performRequest(url string) (io.ReadCloser, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		defer func() {
			if err := resp.Body.Close(); err != nil {
				log.Warnf("Failed to close response body, error: %s", err)
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

func getCacheDownloadURL(cacheAPIURL string) (string, error) {
	req, err := http.NewRequest("GET", cacheAPIURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %s", err)
	}

	client := &http.Client{
		Timeout: 20 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %s", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf(" [!] Exception: Failed to close response body, error: %s", err)
		}
	}()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("request sent, but failed to read response body (http-code:%d): %s", resp.StatusCode, body)
	}

	if resp.StatusCode < 200 || resp.StatusCode > 202 {
		return "", fmt.Errorf("Build cache not found. Probably cache not initialised yet (first cache push initialises the cache), nothing to worry about ;)")
	}

	var respModel GenerateDownloadURLRespModel
	if err := json.Unmarshal(body, &respModel); err != nil {
		return "", fmt.Errorf("Request sent, but failed to parse JSON response (http-code:%d): %s", resp.StatusCode, body)
	}

	if respModel.DownloadURL == "" {
		return "", fmt.Errorf("Request sent, but Download URL is empty (http-code:%d): %s", resp.StatusCode, body)
	}

	return respModel.DownloadURL, nil
}

func readFirstEntry(r io.Reader) (*tar.Reader, *tar.Header, error) {
	replayReader := NewRecorderReader(r)
	replayReader.Record()

	var archive io.Reader
	var err error

	log.Debugf("attempt to read archive as .gzip")

	archive, err = gzip.NewReader(replayReader)
	if err != nil {
		// might the archive is not compressed
		log.Debugf("failed to open the archive as .gzip: %s", err)

		replayReader.Replay()
		archive = replayReader
	}

	tr := tar.NewReader(archive)
	hdr, err := tr.Next()
	if err == io.EOF {
		// no entries in the archive
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}

	return tr, hdr, nil
}

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

func logErrorfAndExit(format string, args ...interface{}) {
	log.Errorf(format, args...)
	os.Exit(1)
}

func main() {
	var conf Config
	if err := stepconf.Parse(&conf); err != nil {
		logErrorfAndExit(err.Error())
	}
	stepconf.Print(conf)
	log.SetEnableDebugLog(conf.DebugMode)

	if conf.CacheAPIURL == "" {
		log.Warnf("No Cache API URL specified, there's no cache to use, exiting.")
		return
	}

	startTime := time.Now()

	var r io.Reader
	var cacheURI string
	if strings.HasPrefix(conf.CacheAPIURL, "file://") {
		cacheURI = conf.CacheAPIURL

		fmt.Println()
		log.Infof("Using local cache archive")

		pth := strings.TrimPrefix(conf.CacheAPIURL, "file://")

		var err error
		r, err = os.Open(pth)
		if err != nil {
			logErrorfAndExit("Failed to open cache archive: %s", err)
		}
	} else {
		fmt.Println()
		log.Infof("Downloading remote cache archive")

		downloadURL, err := getCacheDownloadURL(conf.CacheAPIURL)
		if err != nil {
			logErrorfAndExit("Failed to get download url, error: %s", err)
		}
		cacheURI = downloadURL

		fmt.Println("=> Downloading and extracting cache archive ...")

		r, err = performRequest(downloadURL)
		if err != nil {
			logErrorfAndExit("Failed to perform cache download request, error: %s", err)
		}
	}

	cacheReader := NewRecorderReader(r)

	if currentStackID := os.Getenv("BITRISE_STACK_ID"); len(currentStackID) > 0 {
		fmt.Println()
		log.Infof("Checking archive and current stacks")
		log.Printf("current stack id: %s", currentStackID)

		cacheReader.Record()

		r, hdr, err := readFirstEntry(cacheReader)
		if err != nil {
			logErrorfAndExit("Failed to get first archive entry, error: %s", err)
		}

		if filepath.Base(hdr.Name) == "archive_info.json" {
			b, err := ioutil.ReadAll(r)
			if err != nil {
				logErrorfAndExit("Failed to read first archive entry, error: %s", err)
			}

			cacheReader.Replay()

			archiveStackID, err := parseStackID(b)
			if err != nil {
				logErrorfAndExit("Failed to parse first archive entry, error: %s", err)
			}
			log.Printf("archive stack id: %s", archiveStackID)

			if len(archiveStackID) > 0 && archiveStackID != currentStackID {
				log.Warnf("Cache was created on stack: %s, current stack: %s", archiveStackID, currentStackID)
				log.Warnf("Skipping cache pull, because of the stack has changed")
				os.Exit(0)
			}
		} else {
			cacheReader.Replay()
			log.Warnf("cache archive does not contain stack information, skipping stack check")
		}
	}

	fmt.Println()
	log.Infof("Extracting cache archive")

	if err := extractCacheArchive(cacheReader); err != nil {
		pth, err := downloadCacheArchive(cacheURI)
		if err != nil {
			log.Printf("Retry failed, unable to download cache archive, error: %s", err)
			return
		}

		if err := uncompressArchive(pth); err != nil {
			log.Printf("Retry failed, unable to uncompress cache archive, error: %s", err)
			return
		}
		logErrorfAndExit("Failed to uncompress cache archive: %s", err)
	}

	fmt.Println()
	log.Donef("Done")
	log.Printf("Took: " + time.Now().Sub(startTime).String())
}
