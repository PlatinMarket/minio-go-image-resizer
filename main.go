package main

import (
	"bytes"
	"flag"
	"fmt"
	_ "github.com/Soreil/svg"
	"github.com/disintegration/imaging"
	"github.com/minio/minio-go"
	_ "golang.org/x/image/webp"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	_ "image/jpeg"
	"image/png"
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

func CreateBackground(width, height int, backgroundColor color.Color) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), &image.Uniform{backgroundColor}, image.ZP, draw.Src)
	return img
}

func GetContentType(r *minio.Object) (string, error) {
	buffer := make([]byte, 512)

	_, err := r.Read(buffer)
	if err != nil {
		return "", err
	}

	contentType := http.DetectContentType(buffer)

	return contentType, nil
}

func ImageToPaletted(img image.Image, plt color.Palette) *image.Paletted {
	b := img.Bounds()
	pm := image.NewPaletted(b, plt)
	draw.FloydSteinberg.Draw(pm, b, img, b.Min)
	return pm
}

// Request Process
func (api thumbnailHandlers) ProcessRequest(w http.ResponseWriter, r *http.Request) {

	// Get Path
	path := r.URL.Path

	// Set variables
	file := ""
	ext := ""
	width := 0
	height := 0
	id := 0
	sourceFileName := ""
	targetFileName := ""

	// Check request
	switch true {
	case isMatched(regexp.MatchString("(?i)^/[0-9]{1,6}/pictures/thumb/[0-9]{2,3}X-[0-9]{2,3}X-.*\\.(jpg|jfif|jpeg|gif|png|webp|svg|bmp)$", path)):

		// /123/pictures/thumb/000X-000X-file.ext
		parts := regexp.MustCompile("(?i)^/([0-9]{1,6})/pictures/thumb/([0-9]{2,3})X-([0-9]{2,3})X-(.*)\\.(jpg|jfif|jpeg|gif|png|webp|svg|bmp)$").FindStringSubmatch(path)

		// Set variables
		file = parts[4]
		ext = parts[5]
		id, _ = strconv.Atoi(parts[1])
		width, _ = strconv.Atoi(parts[2])
		height, _ = strconv.Atoi(parts[3])

		// Set files
		sourceFileName = fmt.Sprintf("%d/pictures/%s.%s", id, file, ext)
		targetFileName = fmt.Sprintf("%d/pictures/thumb/%dX-%dX-%s.%s", id, width, height, file, ext)

	case isMatched(regexp.MatchString("(?i)^/[0-9]{1,6}/pictures/thumb/[0-9]{2,3}X-.*\\.(jpg|jfif|jpeg|gif|png|webp|svg|bmp)$", path)):

		// /123/pictures/thumb/000X-file.ext
		parts := regexp.MustCompile("(?i)^/([0-9]{1,6})/pictures/thumb/([0-9]{2,3})X-(.*)\\.(jpg|jfif|jpeg|gif|png|webp|svg|bmp)$").FindStringSubmatch(path)

		// Set variables
		file = parts[3]
		ext = parts[4]
		id, _ = strconv.Atoi(parts[1])
		width, _ = strconv.Atoi(parts[2])
		height = 0

		// Set files
		sourceFileName = fmt.Sprintf("%d/pictures/%s.%s", id, file, ext)
		targetFileName = fmt.Sprintf("%d/pictures/thumb/%dX-%s.%s", id, width, file, ext)

	case isMatched(regexp.MatchString("(?i)^/[0-9]{1,6}/dosyalar/_thumbs/.*\\.(jpg|jfif|jpeg|gif|png|webp|svg|bmp)$", path)):

		// /123/dosyalar/_thumbs/file.ext
		parts := regexp.MustCompile("(?i)^/([0-9]{1,6})/dosyalar/_thumbs/(.*)\\.(jpg|jfif|jpeg|gif|png|webp|svg|bmp)$").FindStringSubmatch(path)

		// Set variables
		file = parts[2]
		ext = parts[3]
		id, _ = strconv.Atoi(parts[1])
		width = 100
		height = 100

		// Set files
		sourceFileName = fmt.Sprintf("%d/dosyalar/%s.%s", id, file, ext)
		targetFileName = fmt.Sprintf("%d/dosyalar/_thumbs/%s.%s", id, file, ext)

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
	_, err = api.minioClient.StatObject(*bucketName, targetFileName, minio.StatObjectOptions{})
	if err == nil {
		targetFile, err := api.minioClient.GetObject(*bucketName, targetFileName, minio.GetObjectOptions{})
		if err != nil {
			http.Error(w, err.Error(), 500)
			log.Println("File "+targetFileName+" read error!", err)
			return
		}
		defer targetFile.Close()

		typeName, err := GetContentType(targetFile)
		if err != nil {
			typeName = "application/octet-stream"
		}

		log.Println(targetFileName)
		targetFile.Seek(0, 0)

		w.Header().Add("Content-Type", typeName)
		host, _ := os.Hostname()
		w.Header().Add("X-Serve-From", host)
		w.Header().Add("X-Resized", "false")
		io.Copy(w, targetFile)
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

	// Create pipes
	imageReader, imageWriter := io.Pipe()

	// Check image is small for given width height
	resizeRequire := false; targetWidthHeight := map[string]int{ "width": 0, "height": 0 }
	if imageData.Bounds().Size().X < width && imageData.Bounds().Size().Y < height {
		resizeRequire = true
		if imageData.Bounds().Size().X >= imageData.Bounds().Size().Y {
			targetWidthHeight["width"] = width
			targetWidthHeight["height"] = 0
		} else {
			targetWidthHeight["width"] = 0
			targetWidthHeight["height"] = height
		}
	}

	// Content Type
	targetContentType := sourceFileInfo.ContentType

	// Save image
	targetData := imageData
	switch inputFormat {
	case "jpeg":
		if resizeRequire {
			targetData = imaging.Resize(targetData, targetWidthHeight["width"], targetWidthHeight["height"], imaging.Lanczos)
		}
		targetData = imaging.Fit(targetData, width, height, imaging.Linear)
		targetData = imaging.Sharpen(targetData, .7)
		targetData = imaging.OverlayCenter(CreateBackground(width, height, color.White), targetData, 100)
		go func() {
			defer imageWriter.Close()
			imaging.Encode(imageWriter, targetData, imaging.JPEG, imaging.JPEGQuality(80))
		}()
	case "gif":
		sourceFile.Seek(0, 0)
		gifList, _ := gif.DecodeAll(sourceFile)

		// Create a new RGBA image to hold the incremental frames.
		//firstFrame := gifList.Image[0]
		//b := image.Rect(0, 0, firstFrame.Rect.Dx(), firstFrame.Rect.Dy())
		//transparentImage := image.NewPaletted(b, firstFrame.Palette)
		//draw.FloydSteinberg.Draw(transparentImage, transparentImage.Bounds(), &image.Uniform{C:color.Transparent}, image.Point{X: 0, Y: 0})

		img := make(chan *image.Paletted)
		go resizeFrame(img, gifList.Image, width, height)

		gifList.Config.ColorModel.Convert(color.RGBA{})

		for i := 0; i < len(gifList.Image); i++ {
			gifList.Image[i] = <- img
		}

		// Setup gif config
		gifList.Config.Width = width
		gifList.Config.Height = height

		go func() {
			defer imageWriter.Close()
			gif.EncodeAll(imageWriter, gifList)
		}()
	case "png":
		if resizeRequire {
			targetData = imaging.Resize(targetData, targetWidthHeight["width"], targetWidthHeight["height"], imaging.Lanczos)
		}
		targetData = imaging.Fit(targetData, width, height, imaging.Lanczos)
		targetData = imaging.OverlayCenter(CreateBackground(width, height, color.Transparent), targetData, 100)
		go func() {
			defer imageWriter.Close()
			imaging.Encode(imageWriter, targetData, imaging.PNG, imaging.PNGCompressionLevel(png.BestCompression))
		}()
	default:
		if resizeRequire {
			targetData = imaging.Resize(targetData, targetWidthHeight["width"], targetWidthHeight["height"], imaging.Lanczos)
		}
		targetData = imaging.Fit(targetData, width, height, imaging.Lanczos)
		targetData = imaging.Sharpen(targetData, 3.5)
		targetData = imaging.OverlayCenter(CreateBackground(width, height, color.White), targetData, 100)
		targetContentType = "image/jpeg"
		go func() {
			defer imageWriter.Close()
			imaging.Encode(imageWriter, targetData, imaging.JPEG)
		}()
	}

	// Create reader clone
	var buf bytes.Buffer
	tee := io.TeeReader(imageReader, &buf)

	w.Header().Add("Content-type", targetContentType)
	w.Header().Add("X-Resized", "true")
	host, _ := os.Hostname()
	w.Header().Add("X-Serve-From", host)
	_, _ = io.Copy(w, tee)

	/*
	_, a := api.minioClient.PutObject(*bucketName, targetFileName, io.Reader(&buf), size, minio.PutObjectOptions{ContentType: sourceFileInfo.ContentType})
	if a != nil {
		log.Println(a, targetFileName)
	}*/
}

func resizeFrame(result chan *image.Paletted, frames []*image.Paletted, tw, th int) {
	//or := frames[0].Rect
	//r := image.Rect(0, 0, tw, th)




	beforeBounds := frames[0].Bounds()

	firstFrame := imaging.Fit(frames[0], tw, th, imaging.Box)
	firstFrame = imaging.OverlayCenter(CreateBackground(tw, th, color.Transparent), firstFrame, 100)
	afterBounds := firstFrame.Bounds()

	//log.Println(frames[0].Palette)
	result <- ImageToPaletted(firstFrame, frames[0].Palette)
	for i := 1; i < len(frames); i++ {
		frame := frames[i]
		targetRect := image.Rect((frame.Bounds().Min.X * afterBounds.Size().X) / beforeBounds.Size().X, (frame.Bounds().Min.Y * afterBounds.Size().Y) / beforeBounds.Size().Y, (frame.Bounds().Max.X * afterBounds.Size().X) / beforeBounds.Size().X, (frame.Bounds().Max.Y * afterBounds.Size().Y) / beforeBounds.Size().Y)
		log.Println("Frame processing", i)

		//log.Println(frame.Bounds())
		newImg := imaging.Resize(frame, targetRect.Size().X, targetRect.Size().Y, imaging.Welch)

		img := *image.NewNRGBA(targetRect)
		//draw.Draw(img, targetRect, &image.Uniform{C:color.Transparent}, image.ZP, draw.Src)
		draw.Draw(&img, targetRect, newImg, newImg.Bounds().Min, draw.Over)

		//log.Println(newImg.S)
		result <- ImageToPaletted(&img, frame.Palette)
	}

	close(result)
}