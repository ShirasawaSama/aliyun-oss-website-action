package operation

import (
	"fmt"
	"path"
	"strings"
	"sync"

	"aliyun-oss-website-action/config"
	"aliyun-oss-website-action/utils"

	"github.com/fangbinwei/aliyun-oss-go-sdk/oss"
)

// UploadObjects upload files to OSS
func UploadObjects(root string, bucket *oss.Bucket, records <-chan utils.FileInfoType) []error {
	if !strings.HasSuffix(root, "/") {
		root += "/"
	}
	var sw sync.WaitGroup
	var mutex sync.Mutex
	var errs []error
	for item := range records {
		sw.Add(1)
		var tokens = make(chan struct{}, 10)
		go func(item utils.FileInfoType) {
			defer sw.Done()
			fPath := item.Path
			objectKey := strings.TrimPrefix(item.PathOSS, root)
			tokens <- struct{}{}
			options := getHTTPHeader(&item)
			err := bucket.PutObjectFromFile(objectKey, fPath, options...)
			<-tokens
			if err != nil {
				mutex.Lock()
				errs = append(errs, fmt.Errorf("[FAILED] objectKey: %s\nfilePath: %s\nDetail: %v", objectKey, fPath, err))
				mutex.Unlock()
				return
			}
			fmt.Printf("objectKey: %s\nfilePath: %s\n", objectKey, fPath)
			fmt.Println()
		}(item)
	}
	sw.Wait()
	if len(errs) > 0 {
		return errs
	}
	return nil
}

func getHTTPHeader(item *utils.FileInfoType) []oss.Option {
	return []oss.Option{
		getCacheControlOption(item.Info.Name()),
	}
}

func getCacheControlOption(filename string) oss.Option {
	var value string
	if isHTML(filename) {
		value = config.HTMLCacheControl
	} else if isImage(filename) {
		// pic name may not contains hash, so use different strategy
		// 10 days
		value = config.ImageCacheControl
	} else {
		// static assets like .js .css, use contentHash in file name, so html can update these files.
		// 30 days
		value = config.OtherCacheControl
	}
	return oss.CacheControl(value)
}

func isHTML(filename string) bool {
	return strings.HasSuffix(strings.ToLower(filename), ".html")
}

func isImage(filename string) bool {
	imageExts := []string{
		".png",
		".jpg",
		".jpeg",
		".webp",
		".gif",
		".bmp",
		".tiff",
		".ico",
		".svg",
	}
	return func() bool {
		ext := path.Ext(filename)
		for _, e := range imageExts {
			if e == ext {
				return true
			}
		}
		return false
	}()
}
