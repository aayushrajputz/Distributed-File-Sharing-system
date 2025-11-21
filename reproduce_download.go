package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

const (
	baseURL = "http://localhost:8080/api/v1"
	email   = "reproduce@test.com"
	pass    = "password123"
)

func main() {
	// 1. Register
	fmt.Println("1. Registering user...")
	registerBody := map[string]string{
		"email":     email,
		"password":  pass,
		"full_name": "Reproduce User",
	}
	jsonBody, _ := json.Marshal(registerBody)
	resp, err := http.Post(baseURL+"/auth/register", "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		panic(err)
	}
	resp.Body.Close()
	// Ignore error (user might exist)

	// 2. Login
	fmt.Println("2. Logging in...")
	loginBody := map[string]string{
		"email":    email,
		"password": pass,
	}
	jsonBody, _ = json.Marshal(loginBody)
	resp, err = http.Post(baseURL+"/auth/login", "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		panic(fmt.Sprintf("Login failed: %s", body))
	}

	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("Login Response:", string(bodyBytes))
	
	var loginResp struct {
		AccessToken string `json:"accessToken"`
	}
	json.Unmarshal(bodyBytes, &loginResp)
	token := loginResp.AccessToken
	fmt.Println("   Logged in. Token length:", len(token))

	// 3. Upload File
	fmt.Println("3. Uploading 450KB file...")
	fileSize := 450 * 1024
	fileContent := make([]byte, fileSize)
	for i := 0; i < fileSize; i++ {
		fileContent[i] = 'A' // Fill with 'A'
	}

	// 3a. Init Upload
	initUploadBody := map[string]interface{}{
		"name": "test_450kb.txt",
		"size": fileSize,
		"mime_type": "text/plain",
	}
	jsonBody, _ = json.Marshal(initUploadBody)
	req, _ := http.NewRequest("POST", baseURL+"/files/upload", bytes.NewBuffer(jsonBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{}
	resp, err = client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		panic(fmt.Sprintf("Init upload failed: %s", body))
	}

	bodyBytes, _ = ioutil.ReadAll(resp.Body)
	fmt.Println("Init Upload Response:", string(bodyBytes))

	var uploadResp struct {
		FileID    string `json:"fileId"`
		UploadURL string `json:"uploadUrl"`
	}
	json.Unmarshal(bodyBytes, &uploadResp)
	fileID := uploadResp.FileID
	uploadURL := uploadResp.UploadURL
	fmt.Println("   File ID:", fileID)
	fmt.Println("   Upload URL:", uploadURL)

	// 3b. Upload to MinIO (PUT)
	fmt.Println("   PUTting to MinIO...")
	req, _ = http.NewRequest("PUT", uploadURL, bytes.NewReader(fileContent))
	req.Header.Set("Content-Type", "text/plain")
	resp, err = client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		fmt.Println("PUT Error Body:", string(body))
		panic(fmt.Sprintf("PUT to MinIO failed: %d", resp.StatusCode))
	}

	// 3c. Complete Upload
	fmt.Println("   Completing upload...")
	completeBody := map[string]string{"status": "completed"}
	jsonBody, _ = json.Marshal(completeBody)
	req, _ = http.NewRequest("POST", fmt.Sprintf("%s/files/%s/complete", baseURL, fileID), bytes.NewBuffer(jsonBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err = client.Do(req)
	if err != nil {
		panic(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		panic("Complete upload failed")
	}

	// 4. Download File
	fmt.Println("4. Downloading file...")
	req, _ = http.NewRequest("GET", fmt.Sprintf("%s/files/%s/download", baseURL, fileID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	downloadedContent, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("   Downloaded size:", len(downloadedContent))
	
	if len(downloadedContent) == fileSize {
		fmt.Println("✅ SUCCESS: Downloaded file matches uploaded size!")
	} else {
		fmt.Println("❌ FAILURE: Size mismatch!")
		fmt.Printf("   Expected: %d, Got: %d\n", fileSize, len(downloadedContent))
		if len(downloadedContent) < 1000 {
			fmt.Println("   Content:", string(downloadedContent))
		}
	}
}
