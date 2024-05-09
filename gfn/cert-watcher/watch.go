// SPDX-FileCopyrightText: Copyright (c) 2024 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: LicenseRef-NvidiaProprietary
//
// NVIDIA CORPORATION, its affiliates and licensors retain all intellectual
// property and proprietary rights in and to this material, related
// documentation and any modifications thereto. Any use, reproduction,
// disclosure or distribution of this material and related documentation
// without an express license agreement from NVIDIA CORPORATION or
// its affiliates is strictly prohibited.

package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/ohler55/ojg/jp"
	"github.com/ohler55/ojg/oj"
)

const (
	exportCertErr         = "failed to export cert:"
	pollingUpdateDuration = 15 * time.Minute
	nginxReloadEndpoint   = "http://127.0.0.1:13129/reload"
)

var lastModTime time.Time

func main() {
	certJsonPath := getEnv("CERT_PATH", `$.pki['container-cache'].certificate`)
	caChainJsonPath := getEnv("CA_CHAIN_PATH", `$.pki['container-cache'].ca_chain[*]`)
	keyJsonPath := getEnv("KEY_PATH", `$.pki['container-cache'].private_key`)
	certOutputPath := getEnv("CERT_OUTPUT_FILE", "/certs/nginx-proxy.pem")
	keyOutputPath := getEnv("KEY_OUTPUT_FILE", "/certs/nginx-proxy.key")
	secretsPath := getEnv("SECRETS_FILE", "/vault/secrets.json")

	caChainJsonExpr, err := jp.ParseString(certJsonPath)
	if err != nil {
		log.Fatal("failed to parse CA_CHAIN_PATH's jsonPath value", caChainJsonPath, err)
	}

	certJsonExpr, err := jp.ParseString(certJsonPath)
	if err != nil {
		log.Fatal("failed to parse CERT_PATH's jsonPath value", certJsonPath, err)
	}
	keyJsonExpr, err := jp.ParseString(keyJsonPath)
	if err != nil {
		log.Fatal("failed to parse KEY_PATH's jsonPath value", keyJsonPath, err)
	}

	// export cert and reload nginx
	err = exportAndReload(secretsPath, certJsonExpr, caChainJsonExpr, keyJsonExpr, certOutputPath, keyOutputPath)
	if err == nil {
		log.Println("Cert has been exported and nginx has been reloaded successfully")
	}

	// start watching for updates, does not return
	watch(secretsPath, certJsonExpr, caChainJsonExpr, keyJsonExpr, certOutputPath, keyOutputPath)
}

// This function checks for updates to secretsPath file every pollingUpdateDuration or when the secrets file has an update, it exports the updated certs and reloads nginx
func watch(secretsPath string, certJsonExpr jp.Expr, caChainJsonExpr jp.Expr, keyJsonExpr jp.Expr, certOutputPath string, keyOutputPath string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()
	err = watcher.Add(secretsPath)
	if err != nil {
		log.Fatal(err)
	}

	// ticker for polling fallback
	ticker := time.NewTicker(pollingUpdateDuration)
	defer ticker.Stop()

	for {
		select {
		// watch for events
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			log.Println("event:", event)
			if event.Op == fsnotify.Remove {
				log.Println("skipping remove event since secrets file may not be there")
				continue
			}
			log.Println("File has been updated")
			// export cert and reload nginx
			err = exportAndReload(secretsPath, certJsonExpr, caChainJsonExpr, keyJsonExpr, certOutputPath, keyOutputPath)
			if err == nil {
				log.Println("Cert has been exported and nginx has been reloaded successfully")
			}
		// watch for errors
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Println("error:", err)
		// check for updates every pollingUpdateDuration
		case <-ticker.C:
			log.Println("polling based refresh triggered")
			// check if file has been updated
			if checkFileUpdated(secretsPath) {
				log.Println("File has been updated")
				// export cert and reload nginx
				err = exportAndReload(secretsPath, certJsonExpr, caChainJsonExpr, keyJsonExpr, certOutputPath, keyOutputPath)
				if err == nil {
					log.Println("Cert has been exported and nginx has been reloaded successfully")
				}
			}
		}
	}
}

// exportAndReload exports the cert and key from the secrets file and reloads nginx
func exportAndReload(secretsPath string, certJsonExpr jp.Expr, caChainJsonExpr jp.Expr, keyJsonExpr jp.Expr, certOutputPath string, keyOutputPath string) error {
	err := exportCert(secretsPath, certJsonExpr, caChainJsonExpr, keyJsonExpr, certOutputPath, keyOutputPath)
	if err != nil {
		log.Println(exportCertErr, err)
	}

	err = setLastModTime(secretsPath)
	if err != nil {
		log.Println("Error while getting file info:", err)
	}

	// reload nginx may fail if the nginx container is not started. The cert-watcher still needs to keep running.
	ok := reloadNginx()
	if !ok {
		log.Println("Error reloading nginx")
	}
	return err
}

// setLastModTime sets the lastModTime to the last modified time of the file at filePath
func setLastModTime(filePath string) error {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		log.Println("Error while getting file info:", err)
		return err
	}
	lastModTime = fileInfo.ModTime()
	return nil
}

// checkFileUpdated checks if the file at filePath has been updated since the last check
func checkFileUpdated(filePath string) bool {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		log.Println("Error while getting file info:", err)
		return false
	}

	modTime := fileInfo.ModTime()
	if !modTime.Equal(lastModTime) {
		lastModTime = modTime
		return true
	}

	return false
}

// reloadNginx sends a request to the nginx reload endpoint and returns true if the request was successful
// reload nginx may fail if the nginx container is not started. The cert-watcher still needs to keep running.
func reloadNginx() bool {
	resp, err := http.Get(nginxReloadEndpoint)
	if err != nil {
		log.Println("Error making request:", err)
	}
	if resp != nil {
		defer resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}
	return false
}

// exportCert exports the full ca chain cert and key from the secrets file and writes them to the output paths
func exportCert(secretsPath string, certJsonExpr jp.Expr, caChainJsonExpr jp.Expr, keyJsonExpr jp.Expr, certOutputPath string, keyOutputPath string) error {
	log.Println("exporting cert")
	fullCaChain, key, err := readDataFromSecretsFile(secretsPath, certJsonExpr, caChainJsonExpr, keyJsonExpr, certOutputPath, keyOutputPath)
	if err != nil {
		return err
	}

	fullCaChainBytes, err := base64.StdEncoding.DecodeString(fullCaChain)
	if err != nil {
		return err
	}

	keyBytes, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return err
	}

	err = replaceCertBundle(fullCaChainBytes, certOutputPath)
	if err == nil {
		return replaceCertBundle(keyBytes, keyOutputPath)
	}
	return err
}

// replaceCertBundle uses rename with a temp file for atomic updates in case nginx reads the file while
// it's being truncated and written to.
func replaceCertBundle(dataBytes []byte, outputPath string) error {
	tempFile, err := os.CreateTemp(filepath.Dir(outputPath), "temp-bundle-*")
	if err != nil {
		return err
	}
	defer tempFile.Close()
	defer os.Remove(tempFile.Name()) // in case there is an error before rename, the tempfile should be deleted
	_, err = io.Copy(tempFile, bytes.NewReader(dataBytes))
	if err != nil {
		return err
	}
	err = tempFile.Chmod(0644)
	if err != nil {
		return err
	}
	return os.Rename(tempFile.Name(), outputPath)
}

// readDataFromSecretsFile reads the full CA chain and key from the secrets file and returns them as strings
func readDataFromSecretsFile(secretsPath string, certJsonExpr jp.Expr, caChainJsonExpr jp.Expr, keyJsonExpr jp.Expr, certOutputPath string, keyOutputPath string) (string, string, error) {
	secretsJsonFile, err := os.Open(secretsPath)
	if err != nil {
		return "", "", err
	}
	log.Println("reading secrets file", secretsJsonFile)
	defer secretsJsonFile.Close()
	secretParsed, err := oj.Load(secretsJsonFile)
	log.Println("reading secrets file", secretParsed)
	if err != nil {
		return "", "", err
	}

	// get CA Chain data
	caChain := caChainJsonExpr.Get(secretParsed)
	var caChainValues []string
	for _, val := range caChain {
		caChainValues = append(caChainValues, fmt.Sprint(val))
	}
	caChainStr := strings.Join(caChainValues, "\n")

	if caChainStr != "" {
		// Get cert and key data
		cert := certJsonExpr.First(secretParsed)
		if certStr, ok := cert.(string); ok && certStr != "" {
			fullCaChain := fmt.Sprintf("%s\n%s", certStr, caChainStr)
			key := keyJsonExpr.First(secretParsed)
			if keyStr, ok := key.(string); ok && keyStr != "" {
				return fullCaChain, keyStr, nil
			}
		}
	}
	return "", "", fmt.Errorf("no cert data found")
}

// getEnv returns the value of the environment variable at path or defaultValue if the environment variable is not set
func getEnv(path, defaultValue string) string {
	value := os.Getenv(path)
	if value == "" {
		return defaultValue
	}
	return value
}
