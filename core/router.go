package core

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/gin-contrib/secure"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"gitrob/common"
)

const (
	GithubBaseURI   = "https://raw.githubusercontent.com"
	GitLabBaseURI   = "https://gitlab.com"
	CspPolicy       = "default-src 'none'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'"
	ReferrerPolicy  = "no-referrer"
	MaximumFileSize = 153600
)

var IsGithub bool

type BinaryFileSystem struct {
	fs http.FileSystem
}

func (b *BinaryFileSystem) Open(name string) (http.File, error) {
	return b.fs.Open(name)
}

func (b *BinaryFileSystem) Exists(prefix, filepath string) bool {
	if p := strings.TrimPrefix(filepath, prefix); len(p) < len(filepath) {
		if _, err := b.fs.Open(p); err != nil {
			return false
		}
		return true
	}
	return false
}

func NewBinaryFileSystem(root string) *BinaryFileSystem {
	fs := &assetfs.AssetFS{
		Asset:     Asset,
		AssetDir:  AssetDir,
		AssetInfo: AssetInfo,
		Prefix:    root,
	}
	return &BinaryFileSystem{
		fs,
	}
}

func NewRouter(s *Session) *gin.Engine {
	IsGithub = s.IsGithubSession

	if *s.Options.Debug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(static.Serve("/", NewBinaryFileSystem("static")))
	router.Use(secure.New(secure.Config{
		SSLRedirect:           false,
		IsDevelopment:         false,
		FrameDeny:             true,
		ContentTypeNosniff:    true,
		BrowserXssFilter:      true,
		ContentSecurityPolicy: CspPolicy,
		ReferrerPolicy:        ReferrerPolicy,
	}))

	router.GET("/stats", func(c *gin.Context) {
		c.JSON(http.StatusOK, s.Stats)
	})

	router.GET("/findings", func(c *gin.Context) {
		c.JSON(http.StatusOK, s.Findings)
	})

	router.GET("/users", func(c *gin.Context) {
		c.JSON(http.StatusOK, s.FoundUsers.UniqueSignatures())
	})

	router.GET("/targets", func(c *gin.Context) {
		c.JSON(http.StatusOK, s.Targets)
	})

	router.GET("/repositories", func(c *gin.Context) {
		c.JSON(http.StatusOK, s.Repositories)
	})

	router.GET("/files/:owner/:repo/:commit/*path", fetchFile)

	return router
}

func fetchFile(c *gin.Context) {
	fileURL := getFileURL(c)

	headRequest, err := http.NewRequestWithContext(c.Request.Context(), http.MethodHead, fileURL, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err,
		})
		return
	}

	resp, err := http.DefaultClient.Do(headRequest)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err,
		})
		return
	}

	if resp.StatusCode == http.StatusNotFound {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "No content",
		})
		return
	}

	if resp.ContentLength > MaximumFileSize {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"message": fmt.Sprintf("File size exceeds maximum of %d bytes", MaximumFileSize),
		})
		return
	}

	getRequest, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, fileURL, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err,
		})
		return
	}

	resp, err = http.DefaultClient.Do(getRequest)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err,
		})
		return
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err,
		})
		return
	}

	c.String(http.StatusOK, string(body))
}

func getFileURL(c *gin.Context) string {
	if IsGithub {
		return fmt.Sprintf("%s/%s/%s/%s%s", GithubBaseURI, c.Param("owner"), c.Param("repo"), c.Param("commit"), c.Param("path"))
	}
	results := common.CleanURLSpaces(c.Param("owner"), c.Param("repo"), c.Param("commit"), c.Param("path"))
	return fmt.Sprintf("%s/%s/%s/%s/%s%s", GitLabBaseURI, results[0], results[1], "/-/raw/", results[2], results[3])
}
