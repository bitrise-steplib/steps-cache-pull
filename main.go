package main

import (
	"archive/tar"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/bitrise-io/go-utils/command"
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

func downloadCacheArchive(url string) error {
	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf(" [!] Failed to close Archive download response body: %s", err)
		}
	}()

	if resp.StatusCode != 200 {
		responseBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		return fmt.Errorf("Failed to download archive - non success response code: %d, body: %s", resp.StatusCode, string(responseBytes))
	}

	out, err := os.Create("/tmp/cache-archive.tar")
	if err != nil {
		return fmt.Errorf("Failed to open the local cache file for write: %s", err)
	}

	defer func() {
		if err := out.Close(); err != nil {
			log.Printf(" [!] Failed to close Archive download file: %+v", err)
		}
	}()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func downloadAndExtractCacheArchive(url string) error {
	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf(" [!] Failed to close Archive download response body: %s", err)
		}
	}()

	if resp.StatusCode != 200 {
		responseBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		return fmt.Errorf("Failed to download archive - non success response code: %d, body: %s", resp.StatusCode, string(responseBytes))
	}

	// out, err := os.Create("/tmp/cache-archive.tar")
	// if err != nil {
	// 	return fmt.Errorf("Failed to open the local cache file for write: %s", err)
	// }

	// defer func() {
	// 	if err := out.Close(); err != nil {
	// 		log.Printf(" [!] Failed to close Archive download file: %+v", err)
	// 	}
	// }()

	// _, err = io.Copy(out, resp.Body)
	// if err != nil {
	// 	return err
	// }

	tarReader := tar.NewReader(resp.Body)

	if err := untar(tarReader); err != nil {
		return err
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

func downloadFileWithRetry(cacheAPIURL string, localPath string) error {
	downloadURL, err := getCacheDownloadURL(cacheAPIURL)
	if err != nil {
		return err
	}
	if gIsDebugMode {
		log.Printf("   [DEBUG] downloadURL: %s", downloadURL)
	}

	if err := downloadCacheArchive(downloadURL); err != nil {
		fmt.Println()
		log.Printf(" ===> (!) First download attempt failed, retrying...")
		fmt.Println()
		return downloadCacheArchive(downloadURL)
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
		log.Println(" (i) No Cache API URL specified, there's no cache to use, exiting.")
		return
	}

	//
	// Download Cache Archive
	//

	log.Println("=> Downloading Cache ...")
	startTime := time.Now()

	downloadURL, err := getCacheDownloadURL(stepParams.CacheAPIURL)
	if err != nil {
		log.Fatalf("failed to get download url")
	}

	if err := downloadAndExtractCacheArchive(downloadURL); err != nil {
		log.Fatalf("failed to download file")
	}

	log.Println("=> Downloading Cache [DONE]")
	log.Println("=> Took: " + time.Now().Sub(startTime).String())

	return

	log.Println("=> Downloading Cache ...")
	startTime = time.Now()

	cacheArchiveFilePath := "/tmp/cache-archive.tar"
	if err := downloadFileWithRetry(stepParams.CacheAPIURL, cacheArchiveFilePath); err != nil {
		log.Fatalf(" [!] Unable to download cache: %s", err)
	}

	log.Println("=> Downloading Cache [DONE]")
	log.Println("=> Took: " + time.Now().Sub(startTime).String())

	log.Println("=> Uncompressing archive ...")
	startTime = time.Now()
	// if err := untarFiles(false); err != nil {
	// 	fmt.Println()
	// 	log.Printf(" ===> (!) Uncompressing failed, retrying...")
	// 	fmt.Println()
	// 	err := untarFiles(true)
	// 	if err != nil {
	// 		log.Fatalf("Failed to uncompress archive, error: %+v", err)
	// 	}
	// }

	cmd := command.New("tar", "-xPf", "/tmp/cache-archive.tar")
	if err := cmd.Run(); err != nil {
		log.Fatalf(" [!] Unable to uncompress cache: %s", err)
	}

	log.Println("=> Uncompressing archive [DONE]")
	log.Println("=> Took: " + time.Now().Sub(startTime).String())

	log.Println("=> Finished")
}

// untar un-tarballs the contents of tr into destination.
func untar(tr *tar.Reader) error {
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		if err := untarFile(tr, header); err != nil {
			return err
		}
	}
	return nil
}

// untarFile untars a single file from tr with header header into destination.
func untarFile(tr *tar.Reader, header *tar.Header) error {
	switch header.Typeflag {
	case tar.TypeDir:
		return mkdir(header.Name)
	case tar.TypeReg:
		return writeNewFile(header.Name, tr, header.FileInfo().Mode())
	case tar.TypeSymlink:
		return writeNewSymbolicLink(header.Name, header.Linkname)
	case tar.TypeLink:
		return writeNewHardLink(header.Name, header.Linkname)
	default:
		log.Printf("%s: unknown type flag: %c", header.Name, header.Typeflag)
		return nil
	}
}

func writeNewFile(fpath string, in io.Reader, fm os.FileMode) error {
	err := os.MkdirAll(filepath.Dir(fpath), 0755)
	if err != nil {
		return fmt.Errorf("%s: making directory for file: %v", fpath, err)
	}

	out, err := os.Create(fpath)
	if err != nil {
		return fmt.Errorf("%s: creating new file: %v", fpath, err)
	}
	defer out.Close()

	err = out.Chmod(fm)
	if err != nil && runtime.GOOS != "windows" {
		return fmt.Errorf("%s: changing file mode: %v", fpath, err)
	}

	_, err = io.Copy(out, in)
	if err != nil {
		return fmt.Errorf("%s: writing file: %v", fpath, err)
	}
	return nil
}

func writeNewSymbolicLink(fpath string, target string) error {
	err := os.MkdirAll(filepath.Dir(fpath), 0755)
	if err != nil {
		return fmt.Errorf("%s: making directory for file: %v", fpath, err)
	}

	err = os.Symlink(target, fpath)
	if err != nil {
		return fmt.Errorf("%s: making symbolic link for: %v", fpath, err)
	}

	return nil
}

func writeNewHardLink(fpath string, target string) error {
	err := os.MkdirAll(filepath.Dir(fpath), 0755)
	if err != nil {
		return fmt.Errorf("%s: making directory for file: %v", fpath, err)
	}

	err = os.Link(target, fpath)
	if err != nil {
		return fmt.Errorf("%s: making hard link for: %v", fpath, err)
	}

	return nil
}

func mkdir(dirPath string) error {
	err := os.MkdirAll(dirPath, 0755)
	if err != nil {
		return fmt.Errorf("%s: making directory: %v", dirPath, err)
	}
	return nil
}
