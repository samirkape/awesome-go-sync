package parser

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
	"sync"

	"golang.org/x/oauth2"
)

// packageDetails holds the information required for updating package stars.
type packageDetails struct {
	name, rawURL, info string
	tmpLinks           *[]Package
}

// loadMarkdown fetches the markdown file from a given URL and returns it as an io.Reader.
func loadMarkdown() io.Reader {
	response, err := http.Get(URL)
	if err != nil {
		log.Printf("unable to get md file from github: %v", err)
		return nil
	}
	defer response.Body.Close()

	buf, err := io.ReadAll(response.Body)
	if err != nil {
		log.Printf("unable to read md file from github: %v", err)
		return nil
	}

	log.Println("done downloading readme.md from github")
	return bytes.NewReader(buf)
}

// Sync fetches the markdown, parses it, and updates the package information in the database.
func Sync() {
	defer MongoClient.Disconnect(context.Background())
	markdownReader := loadMarkdown()
	packages, _ := ParseMarkdown(markdownReader)
	categorizedPackages := CategorizePackages(packages)
	writePackages(MongoClient, categorizedPackages)
	log.Println("no new packages to sync: updated stars count")
}

// ParseMarkdown parses the markdown file line by line and stores raw links in their respective map keys.
func ParseMarkdown(reader io.Reader) (map[string][]string, int) {
	bufferedReader := bufio.NewReader(reader)
	packageMap := make(map[string][]string)

	var counter int
	var links []string
	var title string

	for {
		line, _, err := bufferedReader.ReadLine()
		if err == io.EOF {
			return packageMap, counter
		}
		lineString := string(line)
		if strings.HasPrefix(lineString, "#") {
			if links != nil {
				if len(links) < 3 {
					continue
				}
				counter += len(links)
				packageMap[title] = links
				links = nil
				title = lineString
			} else {
				title = lineString
			}
		} else if isPackage(lineString) {
			links = append(links, lineString)
		}
	}
}

// isPackage checks if the input string is a valid package link.
func isPackage(inputString string) bool {
	regexPattern := `^\s*-\s\[[a-zA-Z0-9\-_]+\]\(https?:\/\/[^\s)]+\)`
	compiledRegex, err := regexp.Compile(regexPattern)
	if err != nil {
		fmt.Println("Error compiling regex:", err)
		return false
	}
	matched := compiledRegex.MatchString(inputString)
	return matched
}

// getPackageDetailsFromString extracts the package name, URL, and description from a given markdown string.
func getPackageDetailsFromString(markdown string) (name, url, description string) {
	// Define a regular expression pattern to match the markdown string format
	regex := regexp.MustCompile(`- \[([^]]+)\]\(([^)]+)\) - (.+)`)

	// Find the submatches in the markdown string
	matches := regex.FindStringSubmatch(markdown)

	if len(matches) == 4 {
		name = strings.TrimSpace(matches[1])
		url = strings.TrimSpace(matches[2])
		description = strings.TrimSpace(matches[3])
	}

	return name, url, description
}

// getRepoStars fetches the star count for a given repository and updates the package information.
func getRepoStars(details packageDetails, wg *sync.WaitGroup, mu *sync.Mutex) {
	defer wg.Done()

	if details.rawURL == "" {
		return
	}

	ctx := context.Background()
	client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: Config.AccessToken,
		TokenType:   "Bearer",
	}))

	tmpFields := strings.Split(details.rawURL, "/")
	u, _ := url.Parse(STARS)
	u.Path = path.Join(u.Path, tmpFields[len(tmpFields)-2])
	u.Path = path.Join(u.Path, tmpFields[len(tmpFields)-1])
	url := u.String()

	resp, err := client.Get(url)
	if err != nil {
		log.Printf("unable to get star count for %s: %v\n", url, err)
		return
	}
	defer resp.Body.Close()

	var repoMeta RepoDetails
	err = json.NewDecoder(resp.Body).Decode(&repoMeta)
	if err != nil {
		log.Printf("failed to decode star count response for %s: %v", url, err)
		return
	}

	log.Printf("star count for %s: %d\n", url, repoMeta.StargazersCount)

	LD := Package{
		Name:  details.name,
		URL:   details.rawURL,
		Info:  details.info,
		Stars: repoMeta.StargazersCount,
	}

	mu.Lock()
	defer mu.Unlock()
	*details.tmpLinks = append(*details.tmpLinks, LD)
}

// CategorizePackages splits and categorizes the packages based on their titles.
func CategorizePackages(packageMap map[string][]string) Categories {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var i int

	categories := make(Categories, len(packageMap))

	for category, packages := range packageMap {
		var tmpLinks []Package
		token := strings.IndexByte(category, ' ')
		categories[i].Title = category[token+1:]

		for _, e := range packages {
			wg.Add(1)
			name, link, info := getPackageDetailsFromString(e)
			details := packageDetails{
				name:     name,
				rawURL:   link,
				info:     info,
				tmpLinks: &tmpLinks,
			}
			go getRepoStars(details, &wg, &mu)
		}

		wg.Wait()
		categories[i].PackageDetails = append(categories[i].PackageDetails, tmpLinks...)
		i++
	}

	log.Println("package filter successful..")
	return categories
}
