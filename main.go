package main

import (
	"fmt"
	"github.com/shurcooL/githubv4"
	"log"
	"os"
	"os/exec"
	"pure/core"
	"text/template"
	"time"
)

var githubUserName = os.Getenv("GITHUB_USER_NAME")
var githubRepo = os.Getenv("GITHUB_REPO")
var githubAccessToken = os.Getenv("GITHUB_ACCESS_TOKEN")

var githubPageRepo = "github.com/zuston/zuston.github.io.git"
var githubPageAuthor = "Junfan Zhang"
var githubPageEmail = "zuston@apache.org"

var api = core.NewApi(githubUserName, githubRepo, githubAccessToken)

var funcMap = template.FuncMap{
	"formatDate": func(unformated githubv4.DateTime) string {
		return unformated.Time.Format("2006-01-02")
	},
	"previewContent": func(fullContent githubv4.String) string {
		if len(fullContent) >= 100 {
			return string(fullContent)[0:100]
		}
		return string(fullContent)
	},
}

func Render() {
	// Remove the original output dir
	os.RemoveAll("./output")
	err := os.Mkdir("output", os.ModePerm)
	if err != nil {
		log.Fatal("Creating output directory failed.")
		return
	}
	cmd := exec.Command("cp", "-r", "./templates/js", "./output")
	cmd.Run()
	cmd = exec.Command("cp", "-r", "./templates/css", "./output")
	cmd.Run()
	log.Println("Successfully copy js/css resources to output directory.")

	discussions, err := api.FetchPosts("", "")
	if err != nil {
		log.Fatal("Errors on fetching all discussions records.")
		return
	}
	indexTemplate, err := template.New("index.html").Funcs(funcMap).ParseFiles(
		"templates/base/navbar.html",
		"templates/base/footer.html",
		"templates/index.html",
	)
	if err != nil {
		log.Fatal("Errors on rendering index.html")
		return
	}

	indexFile, err := os.Create("./output/index.html")
	indexTemplate.Execute(indexFile, discussions)

	// create the article dir
	err = os.Mkdir("./output/article", os.ModePerm)
	for _, metaInfo := range discussions.Nodes {
		number := int(metaInfo.Number)
		page, err := api.FetchPost(number)
		if err != nil {
			return
		}
		postTemplate, err := template.New("post.html").Funcs(funcMap).ParseFiles(
			"templates/base/navbar.html",
			"templates/base/footer.html",
			"templates/post.html")
		postFile, err := os.Create(fmt.Sprintf("./output/article/%d.html", number))
		postTemplate.Execute(postFile, page)
		log.Printf("Successfully rendered the post of [%d]", number)
	}
}

func execCommand(cmdDir string, cmdName string, cmdArgs ...string) {
	cmd := exec.Command(cmdName, cmdArgs...)
	cmd.Dir = cmdDir
	output, err := cmd.Output()
	log.Printf("%s - %v", string(output), err)
}

func Push2Github() {
	// Remove the original folder
	os.RemoveAll("/tmp/output")

	// cp the output to /tmp folder
	cmd := exec.Command("cp", "-r", "./output", "/tmp")
	err := cmd.Run()
	if err != nil {
		log.Fatal("Errors on copying output to tmp folder.")
	}
	// Initialize git repo
	execCmdWithDir := func(name string, args ...string) {
		execCommand("/tmp/output", name, args...)
	}
	execCmdWithDir("git", "init")
	execCmdWithDir("git", "remote", "add", "origin", "https://"+githubPageRepo)
	execCmdWithDir("git", "add", ".")
	execCmdWithDir("git", "config", "user.email", githubPageEmail)
	execCmdWithDir("git", "config", "user.name", githubPageAuthor)
	execCmdWithDir("git", "commit", "-m", "Publish latest post in "+time.ANSIC)
	execCmdWithDir("git", "push", "-f", fmt.Sprintf("https://%s@%s", githubAccessToken, githubPageRepo))
}

func main() {
	Render()
	log.Printf("Finished rendering the html.")

	// push to the Github page
	Push2Github()
	log.Printf("Finished pushing latest blog content to github page.")
}
