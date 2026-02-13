package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

func main() {
	apiKey := "AIzaSyCyS-YhpEgGCj2BUggVhINPwfIOl4oHco4"
	proxy := "http://127.0.0.1:7890"

	client := &http.Client{}
	if proxy != "" {
		proxyURL, _ := url.Parse(proxy)
		client.Transport = &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		}
	}

	url := "https://generativelanguage.googleapis.com/v1beta/models?key=" + apiKey

	resp, err := client.Get(url)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Models []struct {
			Name                       string   `json:"name"`
			SupportedGenerationMethods []string `json:"supportedGenerationMethods"`
		} `json:"models"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Println("Error parsing JSON:", err)
		return
	}

	fmt.Println("\n=== AVAILABLE EMBEDDING MODELS ===")
	found := false
	for _, m := range result.Models {
		isEmbed := false
		for _, method := range m.SupportedGenerationMethods {
			if strings.Contains(method, "embed") {
				isEmbed = true
				break
			}
		}
		if isEmbed {
			fmt.Println(m.Name)
			found = true
		}
	}
	if !found {
		fmt.Println("No models support 'embed' method found!")
	}
	fmt.Println("==================================\n")
}
