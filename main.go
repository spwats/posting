package main

import (
	"net/http"
	"os"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/swatsoncodes/posting/db"
	"github.com/swatsoncodes/posting/middleware"
)

const bodySizeLimit middleware.RequestBodyLimitBytes = 32 * 1024 // 32KiB
const collectionName = "posts"                                   // TODO: make this configurable

func main() {
	var sender, twilioToken, gcloudID string
	var ok bool
	templatesPath := "templates"
	log.SetFormatter(&log.JSONFormatter{DisableHTMLEscape: true})
	log.Info("hello")

	if sender, ok = os.LookupEnv("ALLOWED_SENDER"); !ok {
		log.Fatal("env var ALLOWED_SENDER not set")
	}
	if twilioToken, ok = os.LookupEnv("TWILIO_AUTH_TOKEN"); !ok {
		log.Fatal("env var TWILIO_AUTH_TOKEN not set")
	}
	if gcloudID, ok = os.LookupEnv("GCLOUD_PROJECT_ID"); !ok {
		log.Fatal("env var GCLOUD_PROJECT_ID not set")
	}

	postsDB, err := db.NewFirestoreClient(gcloudID, collectionName)
	if err != nil {
		log.WithError(err).Fatal("failed to initialize db")
	}
	var pdb db.PostsDB = postsDB
	poster, err := NewPoster(sender, twilioToken, templatesPath, &pdb)
	router := mux.NewRouter()

	router.Handle("/posts",
		bodySizeLimit.LimitRequestBody( // guard against giant posts
			middleware.AuthChecker(poster.IsRequestAuthorized).CheckAuth( // make sure posters are authorized
				http.HandlerFunc(poster.CreatePost)))).
		Methods(http.MethodPost).
		Headers("Content-Type", "application/x-www-form-urlencoded")
	router.HandleFunc("/posts", poster.GetPosts).Methods(http.MethodGet)
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/posts", http.StatusMovedPermanently)
	}).Methods(http.MethodGet)

	router.NotFoundHandler = http.HandlerFunc(GoAway)

	router.Use(middleware.LogRequest)
	if env := os.Getenv("POSTER_ENV"); env == "DEV" {
		router.Use(middleware.LogRequestBody)
	}
	log.Info("serving on port 8080")
	http.ListenAndServe(":8080", router)
}
