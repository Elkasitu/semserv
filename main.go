package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/gin-gonic/gin"
	"github.com/google/go-github/v46/github"
)

type ruleset struct {
	Project string `json:"project"`
	Owner   string `json:"owner"`
	Repo    string `json:"repo"`
}

type blob struct {
	Sha      string `json:"sha"`
	NodeId   string `json:"node_id"`
	Size     int64  `json:"size"`
	Url      string `json:"url"`
	Content  string `json:"content"`
	Encoding string `json:"encoding"`
}

const GH_API_BASE_URL = "https://api.github.com"
const GH_API_COMMITS_URL = "%s/repos/%s/commits"

// TODO: Load from yaml/json/env/cli ?
var rulesets = []ruleset{
	{Project: "rhps", Owner: "RedHatProductSecurity", Repo: "django-migration-rules"},
}

func (r *ruleset) getLatestSHA(client *github.Client) (string, error) {
	opt := &github.ReferenceListOptions{}
	refs, _, err := client.Git.ListMatchingRefs(context.Background(), r.Owner, r.Repo, opt)
	if err != nil {
		log.Default().Printf("Error fetching matching refs: %s\n", err)
		return "", err
	}
	if len(refs) < 1 {
		log.Default().Printf("Target repository does not have any commits: %s\n", err)
		return "", err
	}
	return refs[0].GetObject().GetSHA(), nil
}

func getBlob(file *github.TreeEntry) ([]byte, error) {
	if file.GetType() != "blob" || !strings.HasSuffix(file.GetPath(), ".yaml") {
		return nil, nil
	}
	// TODO: Do not treat **all** YAML files as valid, probably only those that contain
	// a `rules:` section
	response, err := http.Get(file.GetURL())
	if err != nil {
		log.Default().Printf("Failed to fetch blob: %s\n", err)
		// TODO: no need to stop execution here, we can just return as many
		// rules as we can and silently ignore (or log) those that cannot be
		// fetched
		return nil, err
	}
	blob := &blob{}
	json.NewDecoder(response.Body).Decode(blob)
	defer response.Body.Close()
	// The GitHub API guarantees that the "content" key of any blob objects
	// are base64-encoded, so we can safely decode as base64 even if we ignore
	// the "encoding" key sent by the API
	payload, err := base64.StdEncoding.DecodeString(blob.Content)
	if err != nil {
		log.Default().Printf("Failed to base64-decode blob: %s\n", err)
		return nil, err
	}
	// strip the `rules:` part as we will be building our own ruleset
	payload = bytes.Replace(payload, []byte("rules:\n"), []byte(""), 1)
	return payload, nil
}

func (r *ruleset) getRules() ([]byte, error) {
	client := github.NewClient(nil)
	// need to get a the latest SHA in order to be able to query GitHub for a tree
	sha, _ := r.getLatestSHA(client)
	// get repository tree with recurse=true, which means that the GitHub API kindly
	// flattens the entire repository file structure so that we can simply go and get
	// what we need (.yaml files)
	tree, _, err := client.Git.GetTree(context.Background(), r.Owner, r.Repo, sha, true)
	if err != nil {
		log.Default().Printf("Error fetching tree: %s\n", err)
		return nil, err
	}
	// prepare building the YAML byte-string
	yamlRaw := []byte("rules:\n")
	for _, entry := range tree.Entries {
		blob, _ := getBlob(entry)
		if blob != nil {
			yamlRaw = append(yamlRaw, blob...)
		}
	}
	return yamlRaw, nil
}

func forwardRulesetByProjectID(c *gin.Context) {
	project := c.Param("project")
	c.Redirect(http.StatusMovedPermanently, fmt.Sprintf("/c/p/%s", project))
}

func getRulesetByProjectID(c *gin.Context) {
	project := c.Param("project")

	for _, ruleset := range rulesets {
		if ruleset.Project == project {
			var rulesObj map[string]interface{}
			rules, err := ruleset.getRules()
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"message": "Repo information could not be retrieved"})
				return
			}
			yaml.Unmarshal(rules, &rulesObj)
			c.YAML(http.StatusOK, rulesObj)
			return
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"message": "project not found"})
}

func main() {
	router := gin.Default()
	router.GET("/p/:project", forwardRulesetByProjectID)
	router.GET("/c/p/:project", getRulesetByProjectID)

	router.Run("localhost:8069")
}
