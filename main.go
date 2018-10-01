package main

import (
	"flag"
	"github.com/gorilla/pat"
	"github.com/minio/minio-go"
	"log"
	"net/http"
	"net/url"
	"os"
)

// Cli Flags
var (
	address    = flag.String("a", "0.0.0.0:2222", "Server address")
	bucketName = flag.String("b", "", "Bucket name")
	endPoint   = flag.String("e", "http://minio1.servers.platinbox.org:9000", "Minio server endpoint")
)

// Thumbnail Handler
type thumbnailHandlers struct {
	minioClient *minio.Client
}

// Finds out whether the url is http(insecure) or https(secure).
func isSecure(urlStr string) bool {
	u, err := url.Parse(urlStr)
	if err != nil {
		panic(err)
	}
	return u.Scheme == "https"
}

// Find the Host of the given url.
func findHost(urlStr string) string {
	u, err := url.Parse(urlStr)
	if err != nil {
		panic(err)
	}
	return u.Host
}

// Get keys
func mustGetAccessKeys() (accessKey, secretKey string) {
	accessKey = os.Getenv("ACCESS_KEY")
	if accessKey == "" {
		log.Fatalln("Env variable 'ACCESS_KEY' not set")
	}
	secretKey = os.Getenv("SECRET_KEY")
	if secretKey == "" {
		log.Fatalln("Env variable 'SECRET_KEY' not set")
	}
	return accessKey, secretKey
}

func main() {
	flag.Parse()

	// Check bucket name
	if *bucketName == "" {
		log.Fatalln("Bucket name cannot be empty.")
	}

	// Get access keys
	accessKey, secretKey := mustGetAccessKeys()

	// Initialize minio client.
	minioClient, err := minio.New(findHost(*endPoint), accessKey, secretKey, isSecure(*endPoint))
	if err != nil {
		log.Fatalln(err)
	}

	// Initialize handler
	thumbnailGenerator := thumbnailHandlers{
		minioClient: minioClient,
	}

	router := pat.New()
	router.Get("/{id:[0-9]+}/pictures/{(thumb|ada)}/{file:.*}", thumbnailGenerator.ResizeImage)

	// Handler for index page
	http.Handle("/", router)

	log.Println("Starting image resizer on " + *address)

	// Start http server
	log.Fatalln(http.ListenAndServe(*address, nil))
}

// Image Resizer
func (api thumbnailHandlers) ResizeImage(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL.Path)
	bucketExists, err := api.minioClient.BucketExists(*bucketName)
	if err != nil {
		log.Fatalln(err)
	}
	message := ""
	if bucketExists {
		message = "Bucket " + *bucketName + " Exists!"
	} else {
		message = "Bucket " + *bucketName + " Not Exists!"
	}

	log.Println("id", r.URL.Query().Get(":id"))
	log.Println("file", r.URL.Query().Get(":file"))

	w.Write([]byte(message))
}
