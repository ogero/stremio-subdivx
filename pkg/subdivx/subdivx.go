package subdivx

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/nwaples/rardecode"
)

const userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/107.0.0.0 Safari/537.36"

var versionREMatcher = regexp.MustCompile(`>v([0-9.a-z]+)<`)

type customTransport struct {
	http.RoundTripper
}

func (e *customTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Set("Accept-Language", "en")
	r.Header.Set("User-Agent", userAgent)
	return e.RoundTripper.RoundTrip(r)
}

var httpClient = &http.Client{
	Timeout:   time.Second * 30,
	Transport: &customTransport{http.DefaultTransport},
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

// SearchTitleResponse represents a minimal version of the search title response
type SearchTitleResponse struct {
	ITotalRecords int `json:"iTotalRecords"`
	AaData        []struct {
		ID int `json:"id"`
	} `json:"aaData"`
}

// Token retrieves a token and cookies
func Token() (string, string, error) {

	res, err := httpClient.Get(`https://www.subdivx.com/inc/gt.php?gt=1`)
	if err != nil {
		return "", "", fmt.Errorf("failed to http.Client.Get: %w", err)
	}
	defer res.Body.Close()

	tokenResponse := struct {
		Token string `json:"token"`
	}{}

	err = json.NewDecoder(res.Body).Decode(&tokenResponse)
	if err != nil {
		return "", "", fmt.Errorf("failed to json.NewDecoder.Decode: %w", err)
	}

	return tokenResponse.Token, res.Header.Get("Set-Cookie"), nil
}

// SearchTitle searches the subtitles for the specified title. A token and cookies are required.
func SearchTitle(token string, setCookie string, title string) (*SearchTitleResponse, error) {

	webVersion, err := func() (string, error) {
		res, err := httpClient.Get(`https://www.subdivx.com/`)
		if err != nil {
			return "", fmt.Errorf("failed to http.Client.Get: %w", err)
		}
		defer res.Body.Close()

		html, err := io.ReadAll(res.Body)
		if err != nil {
			return "", fmt.Errorf("failed to io.ReadAll: %w", err)
		}

		matches := versionREMatcher.FindSubmatch(html)
		if matches == nil || len(matches) != 2 {
			return "", fmt.Errorf("failed to regexp.Regexp.FindSubmatch")
		}

		return fmt.Sprintf("%s", bytes.ReplaceAll(matches[1], []byte("."), []byte(""))), nil
	}()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch web version: failed to io.ReadAll: %w", err)
	}

	formData := url.Values{}
	formData.Set("tabla", "resultados")
	formData.Set("filtros", "")
	formData.Set(fmt.Sprintf("buscar%s", webVersion), title)
	formData.Set("token", token)

	encodedForm := formData.Encode()

	cookie, err := http.ParseSetCookie(setCookie)
	if err != nil {
		return nil, fmt.Errorf("failed to http.ParseSetCookie: %w", err)
	}

	req, _ := http.NewRequest(http.MethodPost, `https://www.subdivx.com/inc/ajax.php`, strings.NewReader(encodedForm))
	req.Header.Set("Cookie", cookie.String())
	req.Header.Set("Content-Type", `application/x-www-form-urlencoded; charset=UTF-8`)

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to http.Client.Do: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("invalid status code: %d", res.StatusCode)
	}

	searchTitleResponse := &SearchTitleResponse{}
	err = json.NewDecoder(res.Body).Decode(searchTitleResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to json.NewDecoder.Decode: %w", err)
	}

	return searchTitleResponse, nil
}

// DownloadSRTByID fetches the subtitles archive for the specified ID, extracts it and returns the content of the first SRT.
func DownloadSRTByID(id string) ([]byte, error) {

	locationHeader, err := func() (string, error) {
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("https://www.subdivx.com/descargar.php?id=%s", id), nil)

		res, err := httpClient.Do(req)
		if err != nil {
			return "", fmt.Errorf("failed to http.Client.Do: %w", err)
		}
		defer res.Body.Close()

		if res.StatusCode != 302 {
			drain, _ := io.ReadAll(res.Body)
			fmt.Println(string(drain))
			return "", fmt.Errorf("invalid status code: %d, expected 302", res.StatusCode)
		}

		locationHeader := res.Header.Get("Location")
		if locationHeader == "" {
			return "", fmt.Errorf("invalid redirect location: %s", locationHeader)
		}

		return locationHeader, nil
	}()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch subtitle direct download link: %w", err)
	}

	subCompressedFileContents, err := func() ([]byte, error) {
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("https://www.subdivx.com/%s", locationHeader), nil)

		res, err := httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to http.Client.Do: %w", err)
		}
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("invalid status code: %d", res.StatusCode)
		}

		buf := new(bytes.Buffer)
		lr := io.LimitReader(res.Body, 200*1024)
		if _, err := io.Copy(buf, lr); err != nil {
			return nil, fmt.Errorf("failed to io.Copy with io.LimitReader: %w", err)
		}

		return buf.Bytes(), nil
	}()
	if err != nil {
		return nil, fmt.Errorf("failed to download subtitle archive: %w", err)
	}

	if len(subCompressedFileContents) < 4 {
		return nil, fmt.Errorf("archive file too short")
	}

	// Check if it's ZIP or RAR by magic bytes
	isZip := bytes.HasPrefix(subCompressedFileContents, []byte("PK\x03\x04"))          // standard ZIP signature
	isRar := bytes.HasPrefix(subCompressedFileContents, []byte("Rar!\x1A\x07\x00")) || // RAR 1.5-4.0
		bytes.HasPrefix(subCompressedFileContents, []byte("Rar!\x1A\x07\x01\x00")) // RAR 5.0

	var subtitleContent []byte
	switch {
	case isZip:
		subtitleContent, err = func() ([]byte, error) {
			zr, err := zip.NewReader(bytes.NewReader(subCompressedFileContents), int64(len(subCompressedFileContents)))
			if err != nil {
				return nil, fmt.Errorf("invalid ZIP: %w", err)
			}

			var srtFile *zip.File
			for _, file := range zr.File {
				if strings.HasSuffix(strings.ToLower(file.Name), ".srt") {
					srtFile = file
					break
				}
			}
			if srtFile == nil {
				return nil, fmt.Errorf("no SRT file found in ZIP")
			}

			rc, err := srtFile.Open()
			if err != nil {
				return nil, fmt.Errorf("failed to open SRT in ZIP: %v", err)
			}
			defer rc.Close()

			return io.ReadAll(rc)
		}()
	case isRar:
		subtitleContent, err = func() ([]byte, error) {
			rr, err := rardecode.NewReader(bytes.NewReader(subCompressedFileContents), "")
			if err != nil {
				return nil, fmt.Errorf("invalid RAR: %w", err)
			}

			for {
				header, err := rr.Next()
				if err == io.EOF {
					break
				}
				if err != nil {
					return nil, fmt.Errorf("failed to read RAR: %v", err)
				}
				if strings.HasSuffix(strings.ToLower(header.Name), ".srt") {
					return io.ReadAll(rr)
				}
			}
			return nil, fmt.Errorf("no SRT file found in ZIP")
		}()
	default:
		return nil, fmt.Errorf("unknown archive format (not ZIP or RAR)")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to decompress archive (not ZIP or RAR)")
	}

	return subtitleContent, nil
}
