package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/json" // so we can ignore our non root CA on EH appliance
	"flag"
	"fmt" // for printing stuff
	"io/ioutil"
	"log"      // Output
	"net/http" // for making HTTP requests
	"os"       // for getting input from user
	"strings"
	"time"
	//	"io/ioutil"
)

var (
	APIKey  = "none"
	ehops   = make(map[string]string)
	Path    = "none"
	keyfile = flag.String("k", "keys", "location of keyfile that contains hostname and APIKey")
)

func terminate(message string) {
	log.Fatal(message)
}
func terminatef(message string, v ...interface{}) {
	log.Fatalf(message, v...)
}
func getKeys() {
	keyfile, err := ioutil.ReadFile(*keyfile)
	if err != nil {
		terminatef("Could not find keys file", err.Error())
	} else if err := json.NewDecoder(bytes.NewReader(keyfile)).Decode(&ehops); err != nil {
		terminatef("Keys file is in wrong format", err.Error())
	} else {
		for key, value := range ehops {
			APIKey = "ExtraHop apikey=" + value
			Path = "https://" + key + "/api/v1/"
			ehops[key] = value
		}
	}
}

func CreateEhopRequest(method string, call string, payload string) *http.Response {
	//Create a 'transport' object... this is necessary if we want to ignore
	//the EH insecure CA.  Similar to '--insecure' option for curl
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	//Crate a new client object... and pass him the parameters of the transport object
	//we created above

	client := http.Client{Transport: tr}
	postBody := []byte(payload)
	req, err := http.NewRequest(method, Path+call, bytes.NewBuffer(postBody))
	if err != nil {
		log.Fatalf("Failed to create HTTP request: %q", err.Error())
	}
	//Add some header stuff to make it EH friendly
	req.Header.Add("Authorization", APIKey)
	req.Header.Add("Content-Type", " application/json")
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Failed to perform HTTP request: %q", err.Error())
	}
	//	defer resp.Body.Close()
	return resp
}
func ConvertResponseToJsonArray(resp *http.Response) []map[string]interface{} {
	// Depending on the request, you may not need an array
	//var results = make(map[string]interface{})
	var mapp = make([]map[string]interface{}, 0)
	if err := json.NewDecoder(resp.Body).Decode(&mapp); err != nil {
		log.Fatalf("Could not parse results: %q", err.Error())
	}
	defer resp.Body.Close()
	return mapp
}

func main() {
	getKeys()
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Please enter the name of the file where the custom devices are listed")
	filename, _ := reader.ReadString('\n')
	fmt.Println("\nThank You")
	time.Sleep(2 * time.Second)

	//read from file
	device_file, err := os.Open(strings.TrimSpace(filename))
	if err != nil {
		log.Fatalf("Could not read file")
	}

	var devices = make(map[string]string)
	if err := json.NewDecoder(device_file).Decode(&devices); err != nil {
		log.Fatalf("Could not parse results: %q", err.Error())
	}

	body := ""
	url := ""
	index := 0
	location := ""
	for key, value := range devices {
		body = `{
			  "author": "Tony",
			  "description": "none",
			  "disabled": false,
			  "name": "` + key + `"
			}`
		fmt.Println("Adding custom device " + key)
		response := CreateEhopRequest("POST", "customdevices", body)
		index = strings.LastIndex(response.Header.Get("location"), "/")
		location = response.Header.Get("location")[index+1:]
		body = `{
		  "ipaddr": "` + value + `"
		}`
		url = "customdevices/" + location + "/criteria"
		fmt.Println("Adding criteria + " + value + " for custom device " + key)
		response = CreateEhopRequest("POST", url, body)
	}
}
