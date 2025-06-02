package main

import (
	"fmt"
	"io/ioutil"
    "html/template"
	"log"    
	"sort"
	"os"
	"regexp"    
	"path/filepath"
	"strings"
)

type ImageData struct {
	Images []string
}


func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <folder>")
		return
	}

	rootFolder := os.Args[1]

	err := filepath.Walk(rootFolder, func(path string, info os.FileInfo, err error) error {

		if err != nil {
			return err
		}

		if strings.Contains(path, "__MACOSX") {
			return nil
		}
		if info.IsDir() && path == rootFolder {
			return nil // Skip processing the root folder itself
		}
		if info.IsDir() {
			dirName := filepath.Base(path)
			fmt.Printf("processing %v - %v\n", path, dirName)

			return processSubFolder(path)
		}

		return nil
	})

	if err != nil {
		log.Fatalf("Error processing folder: %v", err)
	}
}

func getImagesInOrder(subFolder string) ([]string, error) {
	files, err := ioutil.ReadDir(subFolder)
	if err != nil {
		return nil, err
	}

	images := make(map[int][]string)
	for _, file := range files {
		if !file.IsDir() && isImageFile(file.Name()) {
			num := extractNumber(file.Name())
			// fmt.Printf("found file %v with number %v\n", file.Name(), num)
			images[num] = append(images[num], file.Name())
		}
		if file.IsDir() {
			subDirPath := filepath.Join(subFolder, file.Name())
			subDirImages, _ := getImagesInOrder(subDirPath)
			subDirNum := extractNumber(file.Name())
			images[subDirNum] = append(images[subDirNum], subDirImages...)
		}
	}

	if len(images) > 0 {
		return orderImages(images), nil
	}

	return nil, nil
}


func processSubFolder(subFolder string) error {
	images, err := getImagesInOrder(subFolder)
	if err != nil {
		return err
	}

	return createHTML(subFolder, images)
}

func orderImages(imageMap map[int][]string) []string {
	var orderedImages []string
	var keys []int
	for k := range imageMap {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	for _, key := range keys {
		orderedImages = append(orderedImages, imageMap[key]...)
	}
	return orderedImages
}


func isImageFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".gif"
}

// extractNumber extracts the numeric part from a filename.
func extractNumber(filename string) int {
	re := regexp.MustCompile(`(\d+)`)
	match := re.FindStringSubmatch(filename)

	if len(match) > 1 {
		var num int
		fmt.Sscan(match[1], &num)
		return num
	}
	return 0
}

func createHTML(subFolder string, images []string) error {
	data := ImageData{Images: images}
	tmpl := template.Must(template.New("image").Parse(`{{range .Images}}<div style="max-width:900px; max-height:630px;"><img src="{{.}}" style="max-width: 100%; height: auto;" title="{{.}}" alt="{{.}}"></div>{{end}}`))

	for i, img := range data.Images {
		fullImagePath := filepath.Join(subFolder, img)

		relativePath, err := filepath.Rel(filepath.Dir(filepath.Join(filepath.Dir(subFolder), filepath.Base(subFolder)+".html")), fullImagePath)


		if err != nil {
			fmt.Printf("Error calculating relative path: %v\n", err)
		}
		fmt.Printf("createHTML: relativePath = %v\n", relativePath)
		data.Images[i] = filepath.ToSlash(relativePath) // Ensure path uses forward slashes for web compatibility
	}

	htmlFileName := filepath.Join(filepath.Dir(subFolder), filepath.Base(subFolder)+".html")
	f, err := os.Create(htmlFileName)
	if err != nil {
		return err
	}
	// fmt.Printf("creating %v\n", htmlFileName)
	defer f.Close()
	return tmpl.Execute(f, data)
}

