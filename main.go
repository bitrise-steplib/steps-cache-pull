package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/bitrise-io/go-utils/command"
)

const cacheArchivePath = "/tmp/cache-archive.tar"

var (
	gIsDebugMode = false
)

// StepParamsModel ...
type StepParamsModel struct {
	CacheAPIURL string
	IsDebugMode bool
}

// GenerateDownloadURLRespModel ...
type GenerateDownloadURLRespModel struct {
	DownloadURL string `json:"download_url"`
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

		return fmt.Errorf("non success response code: %d, body: %s", resp.StatusCode, string(responseBytes))
	}

	out, err := os.Create(cacheArchivePath)
	if err != nil {
		return fmt.Errorf("failed to open the local cache file for write: %s", err)
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

func uncompressArchive() error {
	cmd := command.New("tar", "-xPf", cacheArchivePath)
	return cmd.Run()
}

func downloadAndExtractCacheArchive(url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		responseBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		return fmt.Errorf("non success response code: %d, body: %s", resp.StatusCode, string(responseBytes))
	}

	cmd := command.New("tar", "-xPf", "/dev/stdin")
	cmd.SetStdin(resp.Body)
	if output, err := cmd.RunAndReturnTrimmedCombinedOutput(); err != nil {
		return fmt.Errorf("failed to extract tar archive, output: %s, error: %s", output, err)
	}

	return resp.Body.Close()
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

	downloadURL, err := getCacheDownloadURL(stepParams.CacheAPIURL)
	if err != nil {
		log.Fatalf("Failed to get download url, error: %+v", err)
	}

	log.Println("=> Downloading and extracting cache archive ...")
	startTime := time.Now()

	if err := downloadAndExtractCacheArchive(downloadURL); err != nil {
		log.Printf(" [!] Unable to download or uncompress cache: %s, retrying...", err)

		if err := downloadCacheArchive(downloadURL); err != nil {
			log.Printf("Retry failed, unable to download cache archive, error: %s", err)
			return
		}

		if err := uncompressArchive(); err != nil {
			log.Printf("Retry failed, unable to uncompress cache archive, error: %s", err)
			return
		}
	}

	log.Println("=> [DONE]")
	log.Println("=> Took: " + time.Now().Sub(startTime).String())

	log.Println("=> Finished")
}
