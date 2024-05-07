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
	certJsonPath := getEnv("CERT_PATH", `$.pki['container-cache'].ca_chain[*]`)
	keyJsonPath := getEnv("KEY_PATH", `$.pki['container-cache'].private_key`)
	certOutputPath := getEnv("CERT_OUTPUT_FILE", "/certs/nginx-proxy.pem")
	keyOutputPath := getEnv("KEY_OUTPUT_FILE", "/certs/nginx-proxy.key")
	secretsPath := getEnv("SECRETS_FILE", "/vault/secrets.json")

	certJsonExpr, err := jp.ParseString(certJsonPath)
	if err != nil {
		log.Fatal("failed to parse CERT_PATH's jsonPath value", certJsonPath, err)
	}
	keyJsonExpr, err := jp.ParseString(keyJsonPath)
	if err != nil {
		log.Fatal("failed to parse KEY_PATH's jsonPath value", keyJsonPath, err)
	}
	// rotate cert on boot
	err = exportCert(secretsPath, certJsonExpr, keyJsonExpr, certOutputPath, keyOutputPath)
	if err != nil {
		log.Fatal(exportCertErr, err)
	}

	// set last mod time for maual checks on updates
	err = setLastModTime(secretsPath)
	if err != nil {
		log.Fatal("Error while getting file info:", err)
	}

	// start watching for updates, does not return
	watch(secretsPath, certJsonExpr, keyJsonExpr, certOutputPath, keyOutputPath)
}

// export cert at least every 15 minutes, or when the secrets file has an update
func watch(secretsPath string, certJsonExpr jp.Expr, keyJsonExpr jp.Expr, certOutputPath string, keyOutputPath string) {
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
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			log.Println("event:", event)
			if event.Op == fsnotify.Remove {
				log.Println("skipping remove event since secrets file may not be there")
				continue
			}
			err = exportCert(secretsPath, certJsonExpr, keyJsonExpr, certOutputPath, keyOutputPath)
			if err != nil {
				log.Println(exportCertErr, err)
			}
			err = setLastModTime(secretsPath)
			if err != nil {
				log.Fatal("Error while getting file info:", err)
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Println("error:", err)
		case <-ticker.C:
			log.Println("polling based refresh triggered")
			if checkFileUpdated(secretsPath) {
				log.Println("File has been updated")
				err = exportAndReload(secretsPath, certJsonExpr, keyJsonExpr, certOutputPath, keyOutputPath)
				if err == nil {
					log.Println("Cert has been exported and ngin has been reloaded successfully")
				}
			}
		}
	}
}

func exportAndReload(secretsPath string, certJsonExpr jp.Expr, keyJsonExpr jp.Expr, certOutputPath string, keyOutputPath string) error {
	err := exportCert(secretsPath, certJsonExpr, keyJsonExpr, certOutputPath, keyOutputPath)
	if err != nil {
		log.Println(exportCertErr, err)
	}
	err = setLastModTime(secretsPath)
	if err != nil {
		log.Fatal("Error while getting file info:", err)
	}
	ok := reloadNginx()
	if !ok {
		log.Fatal("Error reloading nginx")
	}
	return err
}

func setLastModTime(filePath string) error {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		log.Println("Error while getting file info:", err)
		return err
	}
	lastModTime = fileInfo.ModTime()
	return nil
}

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

func reloadNginx() bool {
	resp, err := http.Get(nginxReloadEndpoint)
	if err != nil {
		log.Fatalln("Error making request:", err)
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func exportCert(secretsPath string, certJsonExpr jp.Expr, keyJsonExpr jp.Expr, certOutputPath string, keyOutputPath string) error {
	log.Println("exporting cert")
	cert, key, err := readDataFromSecretsFile(secretsPath, certJsonExpr, keyJsonExpr, certOutputPath, keyOutputPath)
	if err != nil {
		return err
	}
	err = replaceCertBundle([]byte(cert), certOutputPath)
	if err == nil {
		return replaceCertBundle([]byte(key), keyOutputPath)
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

func readDataFromSecretsFile(secretsPath string, certJsonExpr jp.Expr, keyJsonExpr jp.Expr, certOutputPath string, keyOutputPath string) (string, string, error) {
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
	cert := certJsonExpr.Get(secretParsed)
	var caChainValues []string
	for _, val := range cert {
		caChainValues = append(caChainValues, fmt.Sprint(val))
	}
	caChainStr := strings.Join(caChainValues, "")

	if caChainStr != "" {
		key := keyJsonExpr.First(secretParsed)
		if keyStr, ok := key.(string); ok && keyStr != "" {
			return caChainStr, keyStr, nil
		}
	}
	return "", "", fmt.Errorf("no cert data found")
}

func getEnv(path, defaultValue string) string {
	value := os.Getenv(path)
	if value == "" {
		return defaultValue
	}
	return value
}
