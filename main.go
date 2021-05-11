package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg"
	"image/png"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	_ "github.com/Soreil/svg"
	"github.com/disintegration/imaging"
	"github.com/minio/minio-go"
	"github.com/pkg/errors"
	_ "golang.org/x/image/webp"
)

// Cli Flags
var (
	address     = flag.String("a", "0.0.0.0:2222", "Server address")
	bucketName  = flag.String("b", "", "Bucket name")
	endPoint    = flag.String("e", "http://minio1.servers.platinbox.org:9000", "Server Endpoint")
	region      = flag.String("r", "fr-par", "Region")
	gifsicleCmd string
)

// Thumbnail Handler
type thumbnailHandlers struct {
	minioClient *minio.Client
}

// Init
func init() {
	var err error
	gifsicleCmd, err = exec.LookPath("gifsicle")
	if err != nil {
		log.Fatalln("gifsicle not found")
	}
}

// Run gifsicle command
func runGifsicle(r io.Reader, args []string, w io.Writer) error {
	cmd := exec.Command(gifsicleCmd, args...)
	var stderr bytes.Buffer
	cmd.Stdin = r
	cmd.Stdout = w
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return errors.New("Girsicle error! - " + err.Error() +  " - " + strings.Trim(string(stderr.Bytes()), " "))
	}
	return nil
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
	minioClient, err := minio.NewWithRegion(findHost(*endPoint), accessKey, secretKey, isSecure(*endPoint), *region)
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

func calculateAspectRatioFit(srcWidth, srcHeight, maxWidth, maxHeight float64) (int, int) {
	if srcWidth == 0 || srcHeight == 0 {
		return 0, 0
	}
	ratio := []float64{ maxWidth / srcWidth, maxHeight / srcHeight }
	minRatio := math.Min(ratio[0], ratio[1])
	return int(math.Round(srcWidth * minRatio)), int(math.Round(srcHeight * minRatio))
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

		actualWidth := float64(targetData.Bounds().Size().X)
		actualHeight := float64(targetData.Bounds().Size().Y)

		lastWidth, lastHeight := calculateAspectRatioFit(actualWidth, actualHeight, float64(width), float64(height))

		offsetX := (lastWidth - width) / 2
		offsetY := (lastHeight - height) / 2

		tR, tW := io.Pipe()
		defer tR.Close()

		go func() {
			defer tW.Close()
			var err error = nil
			if resizeRequire {
				targetWidth := strconv.Itoa(targetWidthHeight["width"])
				if targetWidth == "0" {
					targetWidth = "_"
				}
				targetHeight := strconv.Itoa(targetWidthHeight["height"])
				if targetHeight == "0" {
					targetHeight = "_"
				}
				err = runGifsicle(sourceFile, []string{ "--resize", fmt.Sprintf("%sx%s", targetWidth, targetHeight) }, tW)
			} else {
				err = runGifsicle(sourceFile, []string{ "--resize-fit", fmt.Sprintf("%dx%d", width, height) }, tW)
			}
			if err != nil {
				log.Println(err)
			}
		}()

		go func() {
			defer imageWriter.Close()
			runGifsicle(tR, []string{ "--crop", fmt.Sprintf("%d,%d+%dx%d", offsetX, offsetY, width, height), "--no-warnings", "--no-ignore-errors" }, imageWriter)
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
	size, _ := io.Copy(w, tee)

	_, a := api.minioClient.PutObject(*bucketName, targetFileName, io.Reader(&buf), size, minio.PutObjectOptions{ContentType: sourceFileInfo.ContentType})
	if a != nil {
		log.Println(a, targetFileName)
	}
}
