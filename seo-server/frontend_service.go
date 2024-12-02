package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"text/template"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type FrontendService struct {
	// InstanceURL is the URL of the instance, e.g. https://www.star-history.com for production.
	InstanceURL string
}

func NewFrontendService(instanceURL string) *FrontendService {
	return &FrontendService{
		InstanceURL: instanceURL,
	}
}

func (s *FrontendService) Serve(e *echo.Echo) {
	s.registerFileRoutes(e)
	s.registerBlogRoutes(e)
}

func (s *FrontendService) registerFileRoutes(e *echo.Echo) {
	defaultIndexHTML := getDefaultIndexHTML()
	blogs := getBlogForntmatters()

	// Serve built static files from frontend.
	e.Use(middleware.StaticWithConfig(middleware.StaticConfig{
		Root: "dist",
		// Skip middleware for index.html, so that we can inject head metadata into it in the next handler.
		Skipper: func(c echo.Context) bool {
			return c.Path() == "" || c.Path() == "/" || c.Path() == "/index.html"
		},
	}))

	e.GET("/robots.txt", func(c echo.Context) error {
		robotsTxt := fmt.Sprintf(`User-agent: *
Allow: /
Host: %s
Sitemap: %s/sitemap.xml`, s.InstanceURL, s.InstanceURL)
		return c.String(http.StatusOK, robotsTxt)
	})

	e.GET("/sitemap.xml", func(c echo.Context) error {
		urlsets := []string{}
		for _, blog := range blogs {
			urlsets = append(urlsets, fmt.Sprintf(`<url><loc>%s</loc></url>`, fmt.Sprintf("%s/blog/%s", s.InstanceURL, blog.Slug)))
		}
		sitemap := fmt.Sprintf(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9" xmlns:news="http://www.google.com/schemas/sitemap-news/0.9" xmlns:xhtml="http://www.w3.org/1999/xhtml" xmlns:mobile="http://www.google.com/schemas/sitemap-mobile/1.0" xmlns:image="http://www.google.com/schemas/sitemap-image/1.1" xmlns:video="http://www.google.com/schemas/sitemap-video/1.1">%s</urlset>`, strings.Join(urlsets, "\n"))
		return c.XMLBlob(http.StatusOK, []byte(sitemap))
	})

	// Serve index.html for all other routes.
	e.GET("*", func(c echo.Context) error {
		return c.HTML(http.StatusOK, defaultIndexHTML)
	})
}

func (s *FrontendService) registerBlogRoutes(e *echo.Echo) {
	rawIndexHTML := getRawIndexHTML()
	blogs := getBlogForntmatters()

	getBlogFrontmatterBySlug := func(slug string) *BlogFrontmatter {
		for _, blog := range blogs {
			if blog.Slug == slug {
				return blog
			}
		}
		return nil
	}

	e.GET("/blog/:blogSlug", func(c echo.Context) error {
		blogSlug := c.Param("blogSlug")
		blogFrontmatter := getBlogFrontmatterBySlug(blogSlug)
		if blogFrontmatter == nil {
			return c.HTML(http.StatusOK, getDefaultIndexHTML())
		}

		indexHTML := rawIndexHTML
		// Inject memo metadata into `index.html`.
		indexHTML = strings.ReplaceAll(indexHTML, "<!-- star-history.head.placeholder -->", s.generateBlogMetadata(blogFrontmatter))
		indexHTML = strings.ReplaceAll(indexHTML, "<!-- star-history.body.placeholder -->", fmt.Sprintf("<!-- star-history.blog.%s -->", blogSlug))
		return c.HTML(http.StatusOK, indexHTML)
	})
}

func getRawIndexHTML() string {
	bytes, _ := os.ReadFile("dist/index.html")
	return string(bytes)
}

func getDefaultIndexHTML() string {
	return strings.ReplaceAll(getRawIndexHTML(), "<!-- star-history.head.placeholder -->", getDefaultMetadata().String())
}

func getBlogForntmatters() []*BlogFrontmatter {
	rawBlogsJSON, _ := os.ReadFile("dist/blog/data.json")
	blogs := []*BlogFrontmatter{}
	// Skip error handling here since its default value is empty array.
	json.Unmarshal(rawBlogsJSON, &blogs)
	return blogs
}

func (s *FrontendService) generateBlogMetadata(blog *BlogFrontmatter) string {
	metadata := getDefaultMetadata()
	if blog.Title != "" {
		metadata.Title = template.HTMLEscapeString(fmt.Sprintf("%s - GitHub Star History", blog.Title))
	}
	if blog.Excerpt != "" {
		metadata.Description = template.HTMLEscapeString(blog.Excerpt)
	}
	if blog.FeatureImage != "" {
		metadata.ImageURL = template.HTMLEscapeString(fmt.Sprintf("%s%s", s.InstanceURL, blog.FeatureImage))
	}
	return metadata.String()
}

type Metadata struct {
	Title       string
	Description string
	ImageURL    string
}

func getDefaultMetadata() *Metadata {
	return &Metadata{
		Title:       "GitHub Star History",
		Description: "View and compare GitHub star history graph of open source projects.",
		ImageURL:    "https://www.star-history.com/star-history.webp",
	}
}

func (m *Metadata) String() string {
	metadataList := []string{
		fmt.Sprintf(`<title>%s</title>`, m.Title),
		fmt.Sprintf(`<meta name="description" content="%s" />`, m.Description),
		fmt.Sprintf(`<meta property="og:title" content="%s" />`, m.Title),
		fmt.Sprintf(`<meta property="og:description" content="%s" />`, m.Description),
		fmt.Sprintf(`<meta property="og:image" content="%s" />`, m.ImageURL),
		`<meta property="og:type" content="website" />`,
		// Twitter related fields.
		fmt.Sprintf(`<meta property="twitter:title" content="%s" />`, m.Title),
		fmt.Sprintf(`<meta property="twitter:description" content="%s" />`, m.Description),
		fmt.Sprintf(`<meta property="twitter:image" content="%s" />`, m.ImageURL),
		`<meta name="twitter:card" content="summary_large_image" />`,
		`<meta name="twitter:site" content="star-history.com" />`,
		`<meta name="twitter:creator" content="bytebase" />`,
	}
	return strings.Join(metadataList, "\n")
}
