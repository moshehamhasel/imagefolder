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
		// Skip __MACOSX folders and files
		if strings.Contains(file.Name(), "__MACOSX") {
			continue
		}
		
		if !file.IsDir() && isImageFile(file.Name()) {
			num := extractNumber(file.Name())
			// Store just the filename for images in current directory
			images[num] = append(images[num], file.Name())
		}
		if file.IsDir() {
			subDirPath := filepath.Join(subFolder, file.Name())
			
			// Skip __MACOSX directories
			if strings.Contains(subDirPath, "__MACOSX") {
				continue
			}
			
			subDirImages, err := getImagesInOrder(subDirPath)
			if err != nil {
				continue // Skip directories with errors
			}
			
			subDirNum := extractNumber(file.Name())
			
			// Prepend subdirectory name to each image path
			for _, img := range subDirImages {
				fullPath := filepath.Join(file.Name(), img)
				images[subDirNum] = append(images[subDirNum], fullPath)
			}
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

// ImageInfo holds both the path and filename for template rendering
type ImageInfo struct {
	Path     string
	Filename string
}

func createHTML(subFolder string, images []string) error {
	// Convert images to ImageInfo structs for better template control
	var imageInfos []ImageInfo
	
	// Create the HTML file path
	htmlFileName := filepath.Join(filepath.Dir(subFolder), filepath.Base(subFolder)+".html")
	
	for _, img := range images {
		// The image path is already relative to subFolder from getImagesInOrder
		fullImagePath := filepath.Join(subFolder, img)
		
		// Calculate relative path from HTML file location to image
		relativePath, err := filepath.Rel(filepath.Dir(htmlFileName), fullImagePath)
		if err != nil {
			continue
		}
		
		// Extract just the filename for title/alt attributes
		filename := filepath.Base(img)
		
		imageInfos = append(imageInfos, ImageInfo{
			Path:     filepath.ToSlash(relativePath), // Ensure path uses forward slashes for web compatibility
			Filename: filename,
		})
	}

	// Updated template with CSS for PDF printing - one image per page
	tmpl := template.Must(template.New("image").Parse(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Image Gallery</title>
    <style>
        @page {
            margin: 0.5in;
            size: A4;
        }
        
        body {
            margin: 0;
            padding: 0;
            font-family: Arial, sans-serif;
        }
        
        .image-container {
            page-break-after: always;
            page-break-inside: avoid;
            width: 100%;
            height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            flex-direction: column;
            box-sizing: border-box;
        }
        
        .image-container:last-child {
            page-break-after: avoid;
        }
        
        .image-container img {
            max-width: 100%;
            max-height: 90vh;
            object-fit: contain;
            box-shadow: 0 2px 8px rgba(0,0,0,0.1);
        }
        
        .image-title {
            margin-top: 10px;
            text-align: center;
            font-size: 12px;
            color: #666;
        }
        
        @media print {
            .image-container {
                height: 100vh;
                page-break-after: always;
                page-break-inside: avoid;
            }
            
            .image-container img {
                max-height: 95vh;
            }
        }
    </style>
</head>
<body>
    {{range .}}<div class="image-container">
        <img src="{{.Path}}" title="{{.Filename}}" alt="{{.Filename}}">
        <div class="image-title">{{.Filename}}</div>
    </div>{{end}}
</body>
</html>`))

	f, err := os.Create(htmlFileName)
	if err != nil {
		return err
	}
	defer f.Close()
	
	return tmpl.Execute(f, imageInfos)
}