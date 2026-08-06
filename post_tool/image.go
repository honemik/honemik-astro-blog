package main

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type ImageWriteOptions struct {
	Root        string
	Date        string
	Slug        string
	Format      string
	Quality     int
	MaxWidth    int
	DryRun      bool
	ImageURLDir string
}

type ImageWriteResult struct {
	Token       string `json:"token"`
	Source      string `json:"source"`
	URL         string `json:"url,omitempty"`
	Path        string `json:"path,omitempty"`
	Hash        string `json:"hash,omitempty"`
	Converted   bool   `json:"converted"`
	Reused      bool   `json:"reused"`
	Warning     string `json:"warning,omitempty"`
	Missing     bool   `json:"missing,omitempty"`
	OriginalExt string `json:"originalExt,omitempty"`
}

func writeImportedImages(odt *ODTFile, refs []ODTImageRef, opts ImageWriteOptions) (map[string]string, []ImageWriteResult, error) {
	if opts.ImageURLDir == "" {
		opts.ImageURLDir = "/img/post"
	}
	if opts.Quality <= 0 {
		opts.Quality = 82
	}
	if opts.MaxWidth <= 0 {
		opts.MaxWidth = 1600
	}
	opts.Format = strings.ToLower(strings.TrimSpace(opts.Format))
	if opts.Format == "" {
		opts.Format = "webp"
	}

	imageDir := postImagesDir(opts.Root)
	if !opts.DryRun {
		if err := os.MkdirAll(imageDir, 0755); err != nil {
			return nil, nil, err
		}
	}

	tokenURLs := map[string]string{}
	hashURLs := map[string]ImageWriteResult{}
	var results []ImageWriteResult
	uniqueIndex := 0

	for _, ref := range refs {
		data, ok := odt.Images[normalizeODTPath(ref.Src)]
		if !ok {
			result := ImageWriteResult{Token: ref.Token, Source: ref.Src, Missing: true, Warning: "image referenced in content.xml was not found in the ODT archive"}
			results = append(results, result)
			continue
		}

		hash := hashBytes(data)
		if prev, seen := hashURLs[hash]; seen {
			tokenURLs[ref.Token] = prev.URL
			reused := prev
			reused.Token = ref.Token
			reused.Source = ref.Src
			reused.Reused = true
			results = append(results, reused)
			continue
		}

		uniqueIndex++
		ext := strings.ToLower(filepath.Ext(ref.Src))
		if ext == "" {
			ext = ".jpg"
		}
		base := fmt.Sprintf("%s-%s-%02d", opts.Date, opts.Slug, uniqueIndex)
		result := ImageWriteResult{
			Token:       ref.Token,
			Source:      ref.Src,
			Hash:        hash,
			OriginalExt: ext,
		}

		var finalName string
		var warning string
		var converted bool

		if opts.Format == "webp" && canConvertToWebP(ext) {
			finalName = base + ".webp"
			dest := filepath.Join(imageDir, finalName)
			if !opts.DryRun {
				ok, convertWarning := convertToWebP(data, ext, dest, opts.Quality, opts.MaxWidth)
				if ok {
					converted = true
				} else {
					warning = convertWarning
					finalName = base + ext
					dest = filepath.Join(imageDir, finalName)
					if err := os.WriteFile(dest, data, 0644); err != nil {
						return nil, nil, err
					}
				}
			} else {
				converted = webPConverterAvailable()
				if !converted {
					warning = "no WebP converter found; import would copy the original image"
					finalName = base + ext
				}
			}
		} else {
			finalName = base + ext
			if opts.Format == "webp" && !canConvertToWebP(ext) {
				warning = "image type is not converted to WebP"
			}
			if !opts.DryRun {
				if err := os.WriteFile(filepath.Join(imageDir, finalName), data, 0644); err != nil {
					return nil, nil, err
				}
			}
		}

		url := opts.ImageURLDir + "/" + finalName
		path := filepath.Join(imageDir, finalName)
		result.URL = url
		result.Path = path
		result.Converted = converted
		result.Warning = warning
		tokenURLs[ref.Token] = url
		hashURLs[hash] = result
		results = append(results, result)
	}

	return tokenURLs, results, nil
}

func replaceImageTokens(body string, tokenURLs map[string]string) string {
	for token, url := range tokenURLs {
		body = strings.ReplaceAll(body, token, url)
	}
	return body
}

func canConvertToWebP(ext string) bool {
	switch strings.ToLower(ext) {
	case ".jpg", ".jpeg", ".png", ".bmp", ".tif", ".tiff", ".webp":
		return true
	default:
		return false
	}
}

func webPConverterAvailable() bool {
	if _, err := exec.LookPath("cwebp"); err == nil {
		return true
	}
	if _, err := exec.LookPath("magick"); err == nil {
		return true
	}
	return false
}

func convertToWebP(data []byte, ext, dest string, quality, maxWidth int) (bool, string) {
	if strings.ToLower(ext) == ".webp" {
		if err := os.WriteFile(dest, data, 0644); err != nil {
			return false, err.Error()
		}
		return false, "source image is already WebP"
	}

	tmpDir, err := os.MkdirTemp("", "odt2post-image-*")
	if err != nil {
		return false, err.Error()
	}
	defer os.RemoveAll(tmpDir)

	src := filepath.Join(tmpDir, "source"+ext)
	if err := os.WriteFile(src, data, 0644); err != nil {
		return false, err.Error()
	}

	if cwebp, err := exec.LookPath("cwebp"); err == nil {
		args := []string{"-quiet", "-q", fmt.Sprint(quality), src, "-o", dest}
		if resize := webPResizeArgs(data, maxWidth); len(resize) > 0 {
			args = append([]string{"-quiet", "-q", fmt.Sprint(quality)}, resize...)
			args = append(args, src, "-o", dest)
		}
		cmd := exec.Command(cwebp, args...)
		if output, runErr := cmd.CombinedOutput(); runErr == nil {
			return true, ""
		} else if strings.TrimSpace(string(output)) != "" {
			return false, strings.TrimSpace(string(output))
		}
		return false, "cwebp failed"
	}

	if magick, err := exec.LookPath("magick"); err == nil {
		args := []string{src, "-auto-orient"}
		if maxWidth > 0 {
			args = append(args, "-resize", fmt.Sprintf("%dx%d>", maxWidth, maxWidth))
		}
		args = append(args, "-quality", fmt.Sprint(quality), dest)
		cmd := exec.Command(magick, args...)
		if output, runErr := cmd.CombinedOutput(); runErr == nil {
			return true, ""
		} else if strings.TrimSpace(string(output)) != "" {
			return false, strings.TrimSpace(string(output))
		}
		return false, "ImageMagick failed"
	}

	return false, "no WebP converter found; install cwebp or ImageMagick magick"
}

func webPResizeArgs(data []byte, maxWidth int) []string {
	if maxWidth <= 0 {
		return nil
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil
	}
	if cfg.Width <= maxWidth && cfg.Height <= maxWidth {
		return nil
	}
	if cfg.Width >= cfg.Height {
		return []string{"-resize", fmt.Sprint(maxWidth), "0"}
	}
	return []string{"-resize", "0", fmt.Sprint(maxWidth)}
}
