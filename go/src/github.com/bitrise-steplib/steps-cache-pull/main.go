package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-utils/cmdex"
	"github.com/bitrise-io/go-utils/pathutil"
)

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
		log.Fatalf(" [!] Failed to create temp directory for cache infos: %s", err)
	}
	log.Printf("=> tmpCacheInfosDirPath: %#v", tmpCacheInfosDirPath)

	tarCmdParams := []string{"-xvzf", cacheFilePath}
	log.Printf(" $ tar %s", tarCmdParams)
	if fullOut, err := cmdex.RunCommandInDirAndReturnCombinedStdoutAndStderr(tmpCacheInfosDirPath, "tar", tarCmdParams...); err != nil {
		log.Printf(" [!] Failed to uncompress cache archive, full output (stdout & stderr) was: %s", fullOut)
		return "", fmt.Errorf("Failed to uncompress cache archive, error was: %s", err)
	}

	for _, aCacheContentInfo := range cacheInfo.Contents {
		log.Printf(" * aCacheContentInfo: %#v", aCacheContentInfo)
		srcPath := filepath.Join(tmpCacheInfosDirPath, aCacheContentInfo.RelativePathInArchive)
		targetPath := aCacheContentInfo.DestinationPath

		// create required target path
		targetBaseDir := filepath.Dir(targetPath)
		if err := os.MkdirAll(targetBaseDir, 0755); err != nil {
			log.Printf(" [!] Failed to create base path (%s) for cache item (%s): %s", targetBaseDir, srcPath, err)
			continue
		}

		log.Printf("   MOVE: %s => %s", srcPath, targetPath)
		if err := os.Rename(srcPath, targetPath); err != nil {
			log.Printf(" [!] Failed to move cache item (%s) to it's place: %s", srcPath, err)
			continue
		}
	}

	return tmpCacheInfosDirPath, nil
}

func main() {
	log.Println("Cache pull...")

	//
	// Download Cache Archive
	//

	// TODO: WIP
	cacheArchiveFilePath, err := pathutil.AbsPath("./cache.tar.gz")
	if err != nil {
		log.Fatalf(" [!] Failed to get absolute path of cache archive: %s", err)
	}
	log.Printf("=> cacheArchiveFilePath: %s", cacheArchiveFilePath)

	//
	// Read Cache Info from archive
	//
	cacheInfoFromArchive, err := readCacheInfoFromArchive(cacheArchiveFilePath)
	if err != nil {
		log.Fatalf(" [!] Failed to read from Archive file: %s", err)
	}
	log.Printf("=> cacheInfoFromArchive: %#v", cacheInfoFromArchive)

	//
	// Uncompress cache
	//
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

	log.Println("=> DONE")
}
