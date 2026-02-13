package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
)

func main() {
	// Hardcoded for convenience debugging
	apiKey := "AIzaSyCyS-YhpEgGCj2BUggVhINPwfIOl4oHco4"

	proxy := "http://127.0.0.1:7890" // User's proxy

	client := &http.Client{}
	if proxy != "" {
		proxyURL, _ := url.Parse(proxy)
		client.Transport = &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		}
	}

	url := "https://generativelanguage.googleapis.com/v1beta/models?key=" + apiKey

	fmt.Println("Querying models from:", url)
	resp, err := client.Get(url)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// Save to file since user cannot copy-paste
	os.WriteFile("models_output.txt", body, 0644)
	fmt.Println("Saved model list to models_output.txt")

	fmt.Println("Response Status:", resp.Status)
	fmt.Println("Response Body:")
	fmt.Println(string(body))
}
