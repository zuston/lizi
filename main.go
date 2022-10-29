package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"pure/core"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/shurcooL/githubv4"
)

type Response[T any] struct {
	Code    int    `json:"code,omitempty"`
	Data    *T     `json:"data,omitempty"`
	Message string `json:"message,omitempty"`
}

var githubUserName = os.Getenv("GITHUB_USER_NAME")
var githubRepo = os.Getenv("GITHUB_REPO")
var githubAccessToken = os.Getenv("GITHUB_ACCESS_TOKEN")

var api = core.NewApi(githubUserName, githubRepo, githubAccessToken)

var storage core.Storage = *core.NewStorage()

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

func FetchPosts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	defer r.Body.Close()
	next := r.URL.Query().Get("next")
	pre := r.URL.Query().Get("pre")

	discussions, err := api.FetchPosts(pre, next)

	if err != nil {
		redirectTo404(w, Response[core.Discussions]{
			Code:    400,
			Data:    nil,
			Message: err.Error(),
		})
		return
	}

	indexTemplate, err := template.New("index.html").Funcs(funcMap).ParseFiles("templates/base/navbar.html", "templates/base/footer.html", "templates/index.html")
	if err != nil {
		redirectTo404(w, Response[core.Discussions]{
			Code:    400,
			Data:    nil,
			Message: err.Error(),
		})
		return
	}
	indexTemplate.Execute(w, discussions)
}

func redirectTo404(w http.ResponseWriter, r Response[core.Discussions]) {
	errTemplate := template.Must(template.ParseFiles("templates/error.html"))
	errTemplate.Execute(w, r)
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

func FetchPost(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	defer r.Body.Close()
	idAndTitle := strings.TrimPrefix(r.URL.Path, "/article/")
	idAndTitleArr := strings.Split(idAndTitle, "/")
	number, err := strconv.Atoi(idAndTitleArr[0])
	if err != nil {
		redirectTo404(w, Response[core.Discussions]{
			Code:    400,
			Data:    nil,
			Message: err.Error(),
		})
		return
	}

	discussion, err := api.FetchPost(number)
	if err != nil {
		redirectTo404(w, Response[core.Discussions]{
			Code:    400,
			Data:    nil,
			Message: err.Error(),
		})
		return
	}

	postTemplate, err := template.New("post.html").Funcs(funcMap).ParseFiles("templates/base/navbar.html", "templates/base/footer.html", "templates/post.html")
	if err != nil {
		redirectTo404(w, Response[core.Discussions]{
			Code:    400,
			Data:    nil,
			Message: err.Error(),
		})
		return
	}
	postTemplate.Execute(w, discussion)
}

// Cached 缓存页面
func Cached(duration string, handler func(w http.ResponseWriter, r *http.Request)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		content := storage.Get(r.RequestURI)
		if content != nil {
			w.Write(content)
		} else {
			c := httptest.NewRecorder()
			handler(c, r)

			for k, v := range c.Header() {
				w.Header()[k] = v
			}

			w.WriteHeader(c.Code)
			content := c.Body.Bytes()

			if d, err := time.ParseDuration(duration); err == nil {
				storage.Set(r.RequestURI, content, d)
			}

			w.Write(content)
		}

	})
}

func main() {
	Render()
	log.Printf("Finished rendering the html.")
	//http.Handle("/", Cached("10m", FetchPosts))
	//http.Handle("/article/", Cached("1h", FetchPost))
	//http.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir("./templates/css/"))))
	//http.Handle("/js/", http.StripPrefix("/js/", http.FileServer(http.Dir("./templates/js/"))))
	//err := http.ListenAndServe(":9000", nil)
	//log.Fatal(err)
}
