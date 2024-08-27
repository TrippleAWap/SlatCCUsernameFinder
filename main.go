package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

func main() {
	fmt.Println("Loading usernames....")
	inputUsernames, err := os.ReadFile("./usernames.txt")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			path, _ := filepath.Abs("./")
			err = fmt.Errorf("couldn't open \"usernames.txt\", please verify usernames.txt exists in %s", path)
		}
		log.Fatal(err)
	}
	usernames := strings.Split(strings.ReplaceAll(string(inputUsernames), "\r", ""), "\n")
	fmt.Printf("Loaded %v usernames from the data...\n", len(usernames))
	cachedUsernames := make(map[string]bool)
	bytesV, err := os.ReadFile("./cached_usernames.txt")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Printf("couldnt find cached usernames.\n attempting to create file \"cached_usernames.txt\"\n")
			err = os.WriteFile("./cached_usernames.txt", nil, 0644)
			if err != nil {
				fmt.Printf("failed to create file \"cached_usernames.txt\" | %v\n", err)
			}
		} else {
			log.Fatal(err)
		}
	} else {
		values := strings.Split(strings.ReplaceAll(string(bytesV), "\r", ""), "\n")
		fmt.Printf("loaded %v cached usernames\n", len(values))
		for _, line := range values {
			entries := strings.Split(line, ",")
			cachedUsernames[entries[0]] = false
		}
	}
	fmt.Printf("( please note, the username cache is ONLY intended to cache TAKEN users as FREE or VALID usernames should be checked EVERYTIME )\n")

	batchingSize := 100
	usernamesChecked := 0
	uniqueUsernames := make([]string, 0)

	println("filtering uniqueUsernames")

	for _, username := range usernames {
		if _, ok := cachedUsernames[username]; ok {
			continue
		}
		uniqueUsernames = append(uniqueUsernames, username)
	}

	fmt.Printf("Spawned %v go routines for %v usernames...\n", math.Ceil(float64(len(uniqueUsernames))/float64(batchingSize)), len(uniqueUsernames))
	go func(usernamesChecked *int) {
		for {
			fmt.Printf("Checked %d usernames\n", *usernamesChecked)
			time.Sleep(time.Second)
		}
	}(&usernamesChecked)
	fmt.Printf("BATCH SCANNING\n")
	newUsernames := StartBatchedScanning(uniqueUsernames, batchingSize, &usernamesChecked)
	fmt.Printf("DONE\n")
	var formatted string
	foundUsernames := make([]string, 0)

	for username, valid := range newUsernames {
		if !valid {
			formatted += fmt.Sprintf("%s,%v\r\n", username, valid)
			delete(newUsernames, username)
			continue
		}

		foundUsernames = append(foundUsernames, username)
	}
	for k, v := range cachedUsernames {
		formatted += fmt.Sprintf("%s,%v\r\n", k, v)
	}
	err = os.WriteFile("./cached_usernames.txt", []byte(formatted), 0644)
	if err != nil {
		log.Fatal(err)
	}
	// append data to the end of found_usernames.txt;
	sort.Sort(ByLength(foundUsernames))
	data := strings.Join(foundUsernames, "\r\n")
	err = os.WriteFile("./found_usernames.txt", []byte(data), 0644)
	if err != nil {
		log.Fatal(err)
	}
}

func StartBatchedScanning(usernames []string, batchingInterval int, usernamesChecked *int) map[string]bool {
	result := make(map[string]bool)
	var wg sync.WaitGroup
	mu := sync.Mutex{}
	for i := 0; i < len(usernames); i += batchingInterval {
		wg.Add(1)

		go func(usernames []string) {
			defer wg.Done()
			resultB := StartScanning(usernames, usernamesChecked)
			//time.Sleep(time.Millisecond * time.Duration(rand.Intn(99))) // this wont but mutexes are slow..
			mu.Lock()
			defer mu.Unlock()
			for s, b := range resultB {
				result[s] = b
			}
		}(usernames[i:min(i+batchingInterval, len(usernames))])
	}

	wg.Wait()

	return result
}

func StartScanning(usernames []string, usernamesChecked *int) map[string]bool {
	result := make(map[string]bool)
	for i := 0; i < len(usernames); i++ {
		username := usernames[i]
		result[username] = false
		usable, err := IsUsableUsername(username)
		*usernamesChecked += 1
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		if usable {
			result[username] = true
		}
	}
	return result
}

func IsUsableUsername(username string) (bool, error) {
	// uri encode the username;
	escapedUsername := url.QueryEscape(username)
	req, _ := http.NewRequest("POST", "https://slat.cc/"+escapedUsername, nil)
	for k, v := range map[string]string{
		"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:129.0) Gecko/20100101 Firefox/129.0",
		"Accept":          "*/*",
		"Accept-Language": "en-CA,en-US;q=0.7,en;q=0.3",
		"Content-Type":    "application/json",
		"Sec-GPC":         "1",
		"Sec-Fetch-Dest":  "empty",
		"Sec-Fetch-Mode":  "cors",
		"Sec-Fetch-Site":  "same-origin",
		"Priority":        "u=0",
	} {
		req.Header.Set(k, v)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("IsUsableUsername: %v", err)
	}
	defer res.Body.Close()
	bytes, _ := io.ReadAll(res.Body)
	if len(bytes) == 0 {
		return false, fmt.Errorf("IsUsableUsername: empty response | status - %s", res.Status)
	}
	if strings.Contains(string(bytes), "<a class=\"inline-flex items-center gap-2 justify-center rounded-md duration-300 font-medium ring-offset-background transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50 border border-emerald-500/15 text-emerald-500 bg-gradient-to-r from-emerald-600/5 to-emerald-600/20 hover:bg-emerald-600/10 h-10 px-4 py-2\" href=\"/\">Go to Homepage</a>") {
		return true, nil
	}
	return false, nil
}

type ByLength []string

func (s ByLength) Len() int           { return len(s) }
func (s ByLength) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s ByLength) Less(i, j int) bool { return len(s[i]) < len(s[j]) }
