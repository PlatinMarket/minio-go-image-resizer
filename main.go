package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/disintegration/imaging"
	"github.com/minio/minio-go"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
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

// Convert match result into one bool
func isMatched(matched bool, err error) bool {
	if err != nil {
		log.Fatalln(err)
		return false
	}
	return matched
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

	// Check bucket exists
	bucketExists, err := minioClient.BucketExists(*bucketName)
	if err != nil {
		log.Fatalln(err)
	}
	if !bucketExists {
		log.Fatalln("Bucket " + *bucketName + " not exists on " + *endPoint)
	}

	// Initialize handler
	thumbnailGenerator := thumbnailHandlers{
		minioClient: minioClient,
	}

	// Handler for index page
	http.HandleFunc("/", thumbnailGenerator.ProcessRequest)

	log.Println("Starting image resizer on " + *address)

	// Start http server
	log.Fatalln(http.ListenAndServe(*address, nil))
}

func CreateBackground(width, height int) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), &image.Uniform{color.White}, image.ZP, draw.Src)
	return img
}

// Request Process
func (api thumbnailHandlers) ProcessRequest(w http.ResponseWriter, r *http.Request) {

	// Get Path
	path := r.URL.Path

	// Set variables
	file := ""
	width := 0
	height := 0
	id := 0
	sourceFileName := ""
	targetFileName := ""

	// Check request
	switch true {
	case isMatched(regexp.MatchString("^/[0-9]{1,6}/pictures/thumb/[0-9]{2,3}X-[0-9]{2,3}X-.*\\.(jpg|jfif|jpeg|gif|png|webp|svg)$", path)):

		// /123/pictures/thumb/000X-000X-file.ext
		parts := regexp.MustCompile("^/([0-9]{1,6})/pictures/thumb/([0-9]{2,3})X-([0-9]{2,3})X-(.*)$").FindStringSubmatch(path)

		// Set variables
		file = parts[4]
		id, _ = strconv.Atoi(parts[1])
		width, _ = strconv.Atoi(parts[2])
		height, _ = strconv.Atoi(parts[3])

		// Set files
		sourceFileName = fmt.Sprintf("%d/pictures/%s", id, file)
		targetFileName = fmt.Sprintf("%d/pictures/thumb/%dX-%dX-%s", id, width, height, file)

	case isMatched(regexp.MatchString("^/[0-9]{1,6}/pictures/thumb/[0-9]{2,3}X-.*\\.(jpg|jfif|jpeg|gif|png|webp|svg)$", path)):

		// /123/pictures/thumb/000X-file.ext
		parts := regexp.MustCompile("^/([0-9]{1,6})/pictures/thumb/([0-9]{2,3})X-(.*)$").FindStringSubmatch(path)

		// Set variables
		file = parts[3]
		id, _ = strconv.Atoi(parts[1])
		width, _ = strconv.Atoi(parts[2])
		height = 0

		// Set files
		sourceFileName = fmt.Sprintf("%d/pictures/%s", id, file)
		targetFileName = fmt.Sprintf("%d/pictures/thumb/%dX-%s", id, width, file)

	case isMatched(regexp.MatchString("^/[0-9]{1,6}/dosyalar/_thumbs/.*\\.(jpg|jfif|jpeg|gif|png|webp|svg)$", path)):

		// /123/dosyalar/_thumbs/file.ext
		parts := regexp.MustCompile("^/([0-9]{1,6})/dosyalar/_thumbs/(.*)$").FindStringSubmatch(path)

		// Set variables
		file = parts[2]
		id, _ = strconv.Atoi(parts[1])
		width = 100
		height = 100

		// Set files
		sourceFileName = fmt.Sprintf("%d/dosyalar/%s", id, file)
		targetFileName = fmt.Sprintf("%d/dosyalar/_thumbs/%s", id, file)

	default:

		// Not found
		http.NotFound(w, r)
		return

	}

	// Equals height to width
	if height == 0 {
		height = width
	}

	// Check source file is alive
	sourceFileInfo, err := api.minioClient.StatObject(*bucketName, sourceFileName, minio.StatObjectOptions{})
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Check target file is alive so return
	targetFileInfo, err := api.minioClient.StatObject(*bucketName, targetFileName, minio.StatObjectOptions{})
	if err == nil {
		targetFile, err := api.minioClient.GetObject(*bucketName, targetFileName, minio.GetObjectOptions{})
		if err != nil {
			http.Error(w, err.Error(), 500)
			log.Println("File "+targetFileName+" read error!", err)
			return
		}
		defer targetFile.Close()
		w.Header().Add("Content-Type", targetFileInfo.ContentType)
		_, err = io.CopyN(w, targetFile, targetFileInfo.Size)
		if err != nil {
			log.Println("File response copy error!", err)
		}
		return
	}

	// Open source file
	sourceFile, err := api.minioClient.GetObject(*bucketName, sourceFileName, minio.GetObjectOptions{})
	if err != nil {
		http.Error(w, err.Error(), 500)
		log.Println("File "+sourceFileName+" read error!", err)
		return
	}
	defer sourceFile.Close()

	// Get image data
	imageData, inputFormat, err := image.Decode(sourceFile)
	if err != nil {
		http.Error(w, err.Error(), 500)
		log.Println(err, sourceFileName)
		return
	}

	// Resize image
	targetData := imaging.Fit(imageData, width, height, imaging.Lanczos)
	targetData = imaging.OverlayCenter(CreateBackground(width, height), targetData, 100)

	// Create pipes

	imageReader, imageWriter := io.Pipe()

	// Save image
	var outputFormat imaging.Format
	switch inputFormat {
	case "jpeg":
		outputFormat = imaging.JPEG
		w.Header().Add("Content-type", sourceFileInfo.ContentType)
		go func() {
			defer imageWriter.Close()
			imaging.Encode(imageWriter, targetData, outputFormat, imaging.JPEGQuality(80))
		}()
	case "gif":
		outputFormat = imaging.GIF
	case "png":
		outputFormat = imaging.PNG
	case "bmp":
		outputFormat = imaging.BMP
	default:
		err := "Unknown format " + inputFormat
		http.Error(w, err, 404)
		log.Println(err, sourceFileName)
		return
	}

	fmt.Println("2")

	var buf bytes.Buffer
	tee := io.TeeReader(imageReader, &buf)

	size, _ := io.Copy(w, tee)

	_, a := api.minioClient.PutObject(*bucketName, targetFileName, io.Reader(&buf), size, minio.PutObjectOptions{})
	if a != nil {
		log.Println(a, targetFileName)
	}
	fmt.Println("3")

	//message := ""

	// Debug
	//message = fmt.Sprintf("file:\t\t%s\nid:\t\t%d\nwidth:\t\t%d\nheight:\t\t%d\n\nsource:\t\t%s\ntarget:\t\t%s\n", file, id, width, height, sourceFileName, targetFileName)

	// Set pattern for split
	/*
		pattern := regexp.MustCompile("^pictures/thumb/([0-9]{2,3})X-([0-9]{2,3})X-(.*)$")

		var parts []string
		parts = pattern.FindStringSubmatch(r.URL.Query().Get(":file"))

		if parts == nil {
			pattern := regexp.MustCompile("^pictures/thumb/([0-9]+)X-([0-9]+)X-(.*)$")
		}
		// Get parts
		parts := pattern.FindStringSubmatch(r.URL.Query().Get(":file"))
		if parts == nil {
			api.ErrorHandler(w, r, 404)
			return
		}

		log.Println(parts)*/
	//w.Write([]byte(message))
}
