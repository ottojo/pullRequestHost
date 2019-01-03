package main

import (
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

var webroot = flag.String("webroot", "/data/www", "Web server root directory")
var githubToken = flag.String("github-token", "", "Github access token")
var hostDomain = flag.String("host", "tbp.ottojo.space", "Web server domain")

func main() {
	flag.Parse()
	*webroot = strings.TrimSuffix(*webroot, "/")
	*githubToken = strings.TrimSpace(*githubToken)
	log.Printf("Parsed commandline parameters\n")
	http.HandleFunc("/", handler)
	log.Printf("Registered handlers.\n")
	log.Fatal(http.ListenAndServe(":1337", nil))
}

func handler(w http.ResponseWriter, r *http.Request) {
	var commitHash = r.FormValue("commit")         //TRAVIS_PULL_REQUEST_SHA
	var pullRequestNumber = r.FormValue("pr")      //TRAVIS_PULL_REQUEST
	var pullRequestSlug = r.FormValue("prSlug")    //TRAVIS_PULL_REQUEST_SLUG
	var targetRepoSlug = r.FormValue("targetSlug") //TRAVIS_REPO_SLUG

	log.Printf("Building PR #%s from %s at commit %s for merge into %s.\n",
		pullRequestNumber, pullRequestSlug, commitHash, targetRepoSlug)

	buildDir := "/tmp/" + commitHash
	log.Printf("Deleting build dir \"%s\"", buildDir)
	err := os.RemoveAll(buildDir)
	PrintError(err)

	hostDir := *webroot + "/" + commitHash
	log.Printf("Deleting host dir \"%s\"", hostDir)
	err = os.RemoveAll(hostDir)
	PrintError(err)

	//TODO: Build (PR merged into master), not PR

	FatalError(exec.Command("git", "lfs", "clone", "https://github.com/"+pullRequestSlug, buildDir).Run())
	checkoutCommand := exec.Command("git", "checkout", commitHash)
	checkoutCommand.Dir = buildDir
	FatalError(checkoutCommand.Run())

	log.Printf("Executing lektor build.\n")
	buildCmd := exec.Command("lektor", "build", "--output-path", hostDir)
	buildCmd.Dir = buildDir
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	FatalError(buildCmd.Run())

	log.Printf("Posting comment.\n")
	httpClient := http.Client{}

	requestBody := "{\"body\": \"View this PR on https://" + commitHash + "." + *hostDomain + "\"}"
	requestURL := "https://api.github.com/repos/" + targetRepoSlug + "/issues/" + pullRequestNumber + "/comments"
	log.Printf("Request URL: \"%s\"\n", requestURL)
	log.Printf("Request body: \"%s\"\n", requestBody)

	req, err := http.NewRequest("POST", requestURL, strings.NewReader(requestBody))

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "token "+*githubToken)
	log.Printf("Request Headers:\n")
	log.Println(req.Header)

	resp, err := httpClient.Do(req)
	FatalError(err)
	if resp.StatusCode != 200 {
		responseBody, _ := ioutil.ReadAll(resp.Body)
		log.Println(string(responseBody))
	}

	log.Printf("Deleting build dir \"%s\"", buildDir)
	err = os.RemoveAll(buildDir)
	PrintError(err)

	w.WriteHeader(200)
	log.Printf("Done.")
}

func FatalError(e error) {
	if e != nil {
		log.Fatal(e)
	}
}

func PrintError(e error) {
	if e != nil {
		log.Println(e)
	}
}
