package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/bitrise-io/go-utils/cmdex"
	"github.com/bitrise-io/go-utils/pathutil"
)

var (
	gIsDebugMode = false
)

// StepParamsModel ...
type StepParamsModel struct {
	CacheAPIURL string
	IsDebugMode bool
}

// CreateStepParamsFromEnvs ...
func CreateStepParamsFromEnvs() (StepParamsModel, error) {
	stepParams := StepParamsModel{
		CacheAPIURL: os.Getenv("cache_api_url"),
		IsDebugMode: os.Getenv("is_debug_mode") == "true",
	}

	return stepParams, nil
}

// CacheContentModel ...
type CacheContentModel struct {
	DestinationPath       string `json:"destination_path"`
	RelativePathInArchive string `json:"relative_path_in_archive"`
}

// CacheInfosModel ...
type CacheInfosModel struct {
	Fingerprint string              `json:"fingerprint"`
	Contents    []CacheContentModel `json:"cache_contents"`
}

func exportEnvironmentWithEnvman(keyStr, valueStr string) error {
	envman := exec.Command("envman", "add", "--key", keyStr)
	envman.Stdin = strings.NewReader(valueStr)
	envman.Stdout = os.Stdout
	envman.Stderr = os.Stderr
	return envman.Run()
}

func readCacheInfoFromArchive(archiveFilePth string) (CacheInfosModel, error) {
	f, err := os.Open(archiveFilePth)
	if err != nil {
		return CacheInfosModel{}, fmt.Errorf("Failed to open Archive file (%s): %s", archiveFilePth, err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Printf(" [!] Failed to close Archive file (%s): %s", archiveFilePth, err)
		}
	}()

	gzf, err := gzip.NewReader(f)
	if err != nil {
		return CacheInfosModel{}, fmt.Errorf("Failed to initialize Archive gzip reader: %s", err)
	}
	defer func() {
		if err := gzf.Close(); err != nil {
			log.Printf(" [!] Failed to close Archive gzip reader(%s): %s", archiveFilePth, err)
		}
	}()

	tarReader := tar.NewReader(gzf)
	for {
		header, err := tarReader.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return CacheInfosModel{}, fmt.Errorf("Failed to read Archive, Tar error: %s", err)
		}
		filePth := header.Name
		if filePth == "./cache-info.json" {
			var cacheInfos CacheInfosModel
			if err := json.NewDecoder(tarReader).Decode(&cacheInfos); err != nil {
				return CacheInfosModel{}, fmt.Errorf("Failed to read Cache Info JSON from Archive: %s", err)
			}
			return cacheInfos, nil
		}
	}

	return CacheInfosModel{}, errors.New("Did not find the required Cache Info file in the Archive")
}

func uncompressCaches(cacheFilePath string, cacheInfo CacheInfosModel) (string, error) {
	// for _, aCacheContentInfo := range cacheInfo.Contents {
	// 	log.Printf(" * aCacheContentInfo: %#v", aCacheContentInfo)
	// 	tarCmdParams := []string{"-xvzf", cacheFilePath}
	// 	log.Printf(" $ tar %s", tarCmdParams)
	// 	if fullOut, err := cmdex.RunCommandAndReturnCombinedStdoutAndStderr("tar", tarCmdParams...); err != nil {
	// 		log.Printf(" [!] Failed to uncompress cache content item (%#v), full output (stdout & stderr) was: %s", aCacheContentInfo, fullOut)
	// 		return "", fmt.Errorf("Failed to uncompress cache content item, error was: %s", err)
	// 	}
	// }

	tmpCacheInfosDirPath, err := pathutil.NormalizedOSTempDirPath("")
	if err != nil {
		return "", fmt.Errorf(" [!] Failed to create temp directory for cache infos: %s", err)
	}
	if gIsDebugMode {
		log.Printf("=> tmpCacheInfosDirPath: %#v", tmpCacheInfosDirPath)
	}

	// uncompress the archive
	{
		tarCmdParams := []string{"-xvzf", cacheFilePath}
		if gIsDebugMode {
			log.Printf(" $ tar %s", tarCmdParams)
		}
		if fullOut, err := cmdex.RunCommandInDirAndReturnCombinedStdoutAndStderr(tmpCacheInfosDirPath, "tar", tarCmdParams...); err != nil {
			log.Printf(" [!] Failed to uncompress cache archive, full output (stdout & stderr) was: %s", fullOut)
			return "", fmt.Errorf("Failed to uncompress cache archive, error was: %s", err)
		}
	}

	for _, aCacheContentInfo := range cacheInfo.Contents {
		if gIsDebugMode {
			log.Printf(" * aCacheContentInfo: %#v", aCacheContentInfo)
		}
		srcPath := filepath.Join(tmpCacheInfosDirPath, aCacheContentInfo.RelativePathInArchive)
		targetPath := aCacheContentInfo.DestinationPath

		isExist, err := pathutil.IsPathExists(targetPath)
		if err != nil {
			log.Printf(" [!] Failed to check whether target path (%s) exists: %s", targetPath, err)
			continue
		}
		if isExist {
			// use rsync instead of rename
			fileInfo, err := os.Stat(srcPath)
			if err != nil {
				log.Printf(" [!] Failed to get File Info of cache item source (%s): %s", srcPath, err)
				continue
			}

			rsyncSrcPth := filepath.Clean(srcPath)
			rsyncTargetPth := filepath.Clean(targetPath)
			if fileInfo.IsDir() {
				rsyncSrcPth = rsyncSrcPth + "/"
				rsyncTargetPth = rsyncTargetPth + "/"
			}
			rsyncCmdParams := []string{"-avh", rsyncSrcPth, rsyncTargetPth}
			if gIsDebugMode {
				log.Printf(" $ rsync %s", rsyncCmdParams)
			}

			log.Printf(" [RSYNC]: %s => %s", rsyncSrcPth, rsyncTargetPth)
			if fullOut, err := cmdex.RunCommandAndReturnCombinedStdoutAndStderr("rsync", rsyncCmdParams...); err != nil {
				log.Printf(" [!] Failed to rsync cache item (%s) to it's place (%s): %s", srcPath, targetPath, err)
				log.Printf("     Full output (stdout & stderr) was: %s", fullOut)
				continue
			}
		} else {
			// create required target path
			targetBaseDir := filepath.Dir(targetPath)
			if err := os.MkdirAll(targetBaseDir, 0755); err != nil {
				log.Printf(" [!] Failed to create base path (%s) for cache item (%s): %s", targetBaseDir, srcPath, err)
				continue
			}

			// move the file to its target path

			// NOTE: we use `mv` to move it instead of Go's `os.Rename`,
			//  because `mv` can move files between separate devices/drives, by using copy&delete.
			//  This is primarily an issue on the Docker stacks,
			//  where shared folders are treated as separate devices, and `os.Rename` would fail.
			mvCmdParams := []string{srcPath, targetPath}
			if gIsDebugMode {
				log.Printf(" $ mv %s", mvCmdParams)
			}

			log.Printf(" [MOVE]: %s => %s", srcPath, targetPath)
			if fullOut, err := cmdex.RunCommandAndReturnCombinedStdoutAndStderr("mv", mvCmdParams...); err != nil {
				log.Printf(" [!] Failed to mv cache item (%s) to it's place (%s): %s", srcPath, targetPath, err)
				log.Printf("     Full output (stdout & stderr) was: %s", fullOut)
				continue
			}
			// if err := os.Rename(srcPath, targetPath); err != nil {
			// 	log.Printf(" [!] Failed to move cache item (%s) to it's place: %s", srcPath, err)
			// 	continue
			// }
		}
	}

	return tmpCacheInfosDirPath, nil
}

func downloadFile(url string, localPath string) error {
	out, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("Failed to open the local cache file for write: %s", err)
	}
	defer func() {
		if err := out.Close(); err != nil {
			log.Printf(" [!] Failed to close Archive download file (%s): %s", localPath, err)
		}
	}()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("Failed to create cache download request: %s", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf(" [!] Failed to close Archive download response body: %s", err)
		}
	}()

	if resp.StatusCode != 200 {
		responseBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Printf(" (!) Failed to read response body: %s", err)
		}
		log.Printf(" ==> (!) Response content: %s", responseBytes)
		return fmt.Errorf("Failed to download archive - non success response code: %d", resp.StatusCode)
	}

	// Writer the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("Failed to save cache content into file: %s", err)
	}

	return nil
}

// GenerateDownloadURLRespModel ...
type GenerateDownloadURLRespModel struct {
	DownloadURL string `json:"download_url"`
}

func getCacheDownloadURL(cacheAPIURL string) (string, error) {
	req, err := http.NewRequest("GET", cacheAPIURL, nil)
	if err != nil {
		return "", fmt.Errorf("Failed to create request: %s", err)
	}
	// req.Header.Set("Content-Type", "application/json")
	// req.Header.Set("Api-Token", apiToken)
	// req.Header.Set("X-Bitrise-Event", "hook")

	client := &http.Client{
		Timeout: 20 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Failed to send request: %s", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf(" [!] Exception: Failed to close response body, error: %s", err)
		}
	}()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("Request sent, but failed to read response body (http-code:%d): %s", resp.StatusCode, body)
	}

	if resp.StatusCode < 200 || resp.StatusCode > 202 {
		return "", fmt.Errorf("Download URL was rejected (http-code:%d): %s", resp.StatusCode, body)
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

func downloadFileWithRetry(cacheAPIURL string, localPath string) error {
	downloadURL, err := getCacheDownloadURL(cacheAPIURL)
	if err != nil {
		return fmt.Errorf("Failed to generate Download URL: %s", err)
	}
	log.Printf("   downloadURL: %s", downloadURL)

	if err := downloadFile(downloadURL, localPath); err != nil {
		fmt.Println()
		log.Printf(" ===> (!) First download attempt failed, retrying...")
		fmt.Println()
		time.Sleep(3000 * time.Millisecond)
		return downloadFile(downloadURL, localPath)
	}
	return nil
}

func main() {
	log.Println("Cache pull...")

	stepParams, err := CreateStepParamsFromEnvs()
	if err != nil {
		log.Fatalf(" [!] Input error : %s", err)
	}
	gIsDebugMode = stepParams.IsDebugMode
	if gIsDebugMode {
		log.Printf("=> stepParams: %#v", stepParams)
	}
	if stepParams.CacheAPIURL == "" {
		log.Println(" (i) No Cache Download URL specified, there's no cache to use, exiting.")
		return
	}

	//
	// Download Cache Archive
	//

	log.Println("=> Downloading Cache ...")
	cacheTempDir, err := pathutil.NormalizedOSTempDirPath("bitrise-cache")
	if err != nil {
		log.Fatalf(" [!] Failed to create temp directory for cache download: %s", err)
	}
	if gIsDebugMode {
		log.Printf("=> cacheTempDir: %s", cacheTempDir)
	}
	cacheArchiveFilePath := filepath.Join(cacheTempDir, "cache.tar.gz")
	if err := downloadFileWithRetry(stepParams.CacheAPIURL, cacheArchiveFilePath); err != nil {
		log.Fatalf(" [!] Failed to download cache archive: %s", err)
	}

	if gIsDebugMode {
		log.Printf("=> cacheArchiveFilePath: %s", cacheArchiveFilePath)
	}
	log.Println("=> Downloading Cache [DONE]")

	//
	// Read Cache Info from archive
	//
	cacheInfoFromArchive, err := readCacheInfoFromArchive(cacheArchiveFilePath)
	if err != nil {
		log.Fatalf(" [!] Failed to read from Archive file: %s", err)
	}
	if gIsDebugMode {
		log.Printf("=> cacheInfoFromArchive: %#v", cacheInfoFromArchive)
	}

	//
	// Uncompress cache
	//
	log.Println("=> Uncompressing Cache ...")
	cacheDirPth, err := uncompressCaches(cacheArchiveFilePath, cacheInfoFromArchive)
	if err != nil {
		log.Fatalf(" [!] Failed to uncompress caches: %s", err)
	}
	cacheInfoJSONFilePath := filepath.Join(cacheDirPth, "cache-info.json")
	if isExist, err := pathutil.IsPathExists(cacheInfoJSONFilePath); err != nil {
		log.Fatalf(" [!] Failed to check Cache Info JSON in uncompressed cache data: %s", err)
	} else if !isExist {
		log.Fatalln(" [!] Cache Info JSON not found in uncompressed cache data")
	}
	log.Println("=> Uncompressing Cache [DONE]")

	//
	// Save & expose the Cache Info JSON
	//

	// tmpCacheInfosDirPath, err := pathutil.NormalizedOSTempDirPath("")
	// if err != nil {
	// 	log.Fatalf(" [!] Failed to create temp directory for cache infos: %s", err)
	// }
	// log.Printf("=> tmpCacheInfosDirPath: %#v", tmpCacheInfosDirPath)

	// cacheInfoJSONFilePath := filepath.Join(tmpCacheInfosDirPath, "cache-info.json")
	// jsonBytes, err := json.Marshal(cacheInfoFromArchive)
	// if err != nil {
	// 	log.Fatalf(" [!] Failed to generate Cache Info JSON: %s", err)
	// }

	// if err := fileutil.WriteBytesToFile(cacheInfoJSONFilePath, jsonBytes); err != nil {
	// 	log.Fatalf(" [!] Failed to write Cache Info YML into file (%s): %s", cacheInfoJSONFilePath, err)
	// }

	if err := exportEnvironmentWithEnvman("BITRISE_CACHE_INFO_PATH", cacheInfoJSONFilePath); err != nil {
		log.Fatalf(" [!] Failed to export Cache Info YML path with envman: %s", err)
	}
	if gIsDebugMode {
		log.Printf(" (i) $BITRISE_CACHE_INFO_PATH=%s", cacheInfoJSONFilePath)
	}

	log.Println("=> Finished")
}
