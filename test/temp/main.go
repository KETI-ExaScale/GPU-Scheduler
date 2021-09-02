package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	corev1 "k8s.io/api/core/v1"
)

func main() {
	var podList corev1.PodList

	v := url.Values{}
	v.Add("fieldSelector", "status.phase=Running")
	v.Add("fieldSelector", "status.phase=Pending")

	request := &http.Request{
		Header: make(http.Header),
		Method: http.MethodGet,
		URL: &url.URL{
			Host:     "127.0.0.1:8001",
			Path:     "/api/v1/pods",
			RawQuery: v.Encode(),
			Scheme:   "http",
		},
	}
	request.Header.Set("Accept", "application/json, */*")

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		fmt.Println("error")
	}
	err = json.NewDecoder(resp.Body).Decode(&podList)
	if err != nil {
		fmt.Println("error")
	}

	fmt.Println(podList)
}
