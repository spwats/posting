package main

import (
	"net/http"
	"path/filepath"
	"strconv"
	"text/template"

	log "github.com/sirupsen/logrus"
	"github.com/swatsoncodes/posting/db"
	"github.com/swatsoncodes/posting/models"
	"github.com/swatsoncodes/posting/upstream/imgur"
)

const postsTemplate string = "posts.html"
const badRequest, internalErr string = "🚮 bad post!", "🔥 internal error"

// Poster is the primary class of the blog.
// It holds the necessary data to communicate with 3rd party APIs and render HTML templates.
// A Poster creates new Posts by receiving them over HTTP and storing them in the DB.
// It can display those Posts by retreiving them from the DB and rendering them in a nice HTML template
type Poster struct {
	Uploader           models.Uploader    // used for uploading media to external host
	DB                 *db.PostsDB        // DB connection for storing/retrieving Posts
	PageSize           int                // number of Posts to display on a single page
	PostsTemplate      *template.Template // html template for rendering Posts
	bodySizeLimitBytes int64              // upper limit for incoming request bodies, in bytes
}

// NewPoster creates a new Poster
func NewPoster(imgurClientID, templatesPath string, pageSize int, bodySizeLimit int64, postsDB *db.PostsDB) (*Poster, error) {
	template, err := template.ParseFiles(filepath.Join(templatesPath, postsTemplate))
	if err != nil {
		return nil, err
	}
	return &Poster{imgur.Uploader{ClientID: imgurClientID}, postsDB, pageSize, template, bodySizeLimit}, nil
}

// CreatePost creates a new Post by parsing incoming HTTP request bodies and storing them in the DB.
// It will upload any attached media to a separate host.
func (poster Poster) CreatePost(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(poster.bodySizeLimitBytes)
	if err != nil {
		log.WithError(err).Warn("unable to parse form body")
		http.Error(w, badRequest, http.StatusBadRequest)
		return
	}

	post, err := models.ParsePost(*r.MultipartForm)
	if err != nil {
		log.WithError(err).Warn("got bad post")
		http.Error(w, badRequest, http.StatusBadRequest)
		return
	}

	if err := post.UploadMedia(poster.Uploader); err != nil {
		log.WithError(err).Error("failed to upload images to host")
		http.Error(w, internalErr, http.StatusInternalServerError)
		return
	}

	if err := (*poster.DB).PutPost(*post); err != nil {
		log.WithError(err).Error("failed to save post to DB")
		http.Error(w, internalErr, http.StatusInternalServerError)
		return
	}

	poster.GetPosts(w, r)
}

// GetPosts retrieves Posts from the DB and renders them using the HTML template.
// It uses the "page" URL query param to determine which Posts to display
func (poster Poster) GetPosts(w http.ResponseWriter, r *http.Request) {
	pageNum := getPageNum(r)
	posts, isMore, err := (*poster.DB).GetPosts(pageNum*poster.PageSize, poster.PageSize)
	if err != nil {
		log.WithError(err).Error("failed to get posts from db")
		http.Error(w, internalErr, http.StatusInternalServerError)
		return
	}

	// NextPage and PrevPage are used for displaying HTML navigation buttons
	// If NextPage or PrevPage are < 0, it indicates there are no older or newer posts to fetch, respectively
	nextPage := -1
	if isMore {
		nextPage = pageNum + 1
	}
	templatePayload := struct {
		Posts              []models.Post
		NextPage, PrevPage int
	}{
		*posts,
		nextPage,
		pageNum - 1,
	}

	if err = poster.PostsTemplate.Execute(w, templatePayload); err != nil {
		log.WithError(err).Error(err.Error())
		http.Error(w, internalErr, http.StatusInternalServerError)
		return
	}
}

// getPageNum determines which page number the requester wants using the "page" URL query param
// if "page" is not present, not an integer, or < 0, this function returns 0
func getPageNum(r *http.Request) (offset int) {
	if page, ok := r.URL.Query()["page"]; ok {
		if len(page) == 0 {
			return
		}
		if p, err := strconv.Atoi(page[0]); err == nil {
			if p < 0 {
				return
			}
			return p
		}
	}
	return
}
