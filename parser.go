package parser

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"

	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/oauth2"
)

var PackageCounter int

func loadMarkdown() io.Reader {
	b, err := http.Get(URL)
	if err != nil {
		log.Printf("unable to get md file from github: %v", err)
		return nil
	}
	defer b.Body.Close()

	buf, err := ioutil.ReadAll(b.Body)

	if err != nil {
		log.Printf("unable to get md file from github %v", err)
		return nil
	}

	log.Println("done downloading readme.md from github")
	return bytes.NewReader(buf)
}

func SyncReq(newCount int) (bool, int) {
	var result olderCount
	collection := MongoClient.Database(Config.UserDBName).Collection(Config.UserDBOldCtr)
	collection.FindOne(context.TODO(), bson.D{}).Decode(&result)
	if Config.SudoWrite == "1" || (newCount > result.Old) {
		update := bson.D{{Key: "old", Value: newCount}}
		_, err := collection.ReplaceOne(context.TODO(), bson.D{}, update)
		if err != nil {
			log.Printf("enable to update old pkg count: %v", err)
			return false, newCount - result.Old
		}
		return true, newCount - result.Old
	}
	return false, newCount - result.Old
}

func Sync() {
	defer MongoClient.Disconnect(context.TODO())
	buf := loadMarkdown()
	m, count := GetSlice(buf)
	final := SplitLinks(m)
	check, diff := SyncReq(count)
	// DBWrite(MongoClient, final)
	if check {
		log.Printf("adding %d new packages\n", diff)
	}
	DBUpdate(MongoClient, final)
	log.Println("no new packages to sync: updated stars count")
}

// Open file specified in  filename and return its handle
func FileHandle(filename string) *os.File {
	awsm, err := os.Open(filename)
	if err != nil {
		fmt.Println("cannot read file")
		os.Exit(-1)
	}
	return awsm
}

// GetSlice is a driver function that gets filehandler as an input,
// reads file line-by-line and store slice of raw links into their particular map key
func GetSlice(f io.Reader) (map[string][]string, int) {
	rd := bufio.NewReader(f)
	m := make(map[string][]string)
	var links []string
	var title string
	counter := 0
	for {
		line, err := rd.ReadString('\n')
		if strings.HasPrefix(line, "#") || err == io.EOF {
			if links != nil {
				if len(links) < 3 {
					continue
				}
				counter += len(links)
				m[title] = links
				links = nil
				title = line
				if err == io.EOF {
					break
				}
			} else {
				title = line
			}
		} else if strings.HasPrefix(line, "-") {
			links = append(links, line)
		}
	}
	log.Println("parsing successful..")
	return m, counter
}

// TrimString is a post-processing function that divides an input strings into,
// name -- package name
// url  -- package url
// info  -- a short info about the package
func trimString(raw string) (name, url, description string) {
	sre := regexp.MustCompile(`\[(.*?)\]`)
	rre := regexp.MustCompile(`\((.*?)\)`)
	_name := sre.FindAllString(raw, -1)
	_url := rre.FindAllString(raw, -1)
	for _, u := range _url {
		if strings.Contains(u, ".com") {
			url = u
		}
	}
	if _name == nil || _url == nil {
		return "", "", ""
	}
	name = strings.Trim(_name[0], "[")
	name = strings.Trim(name, "]")
	url = strings.Trim(url, "(")
	url = strings.Trim(url, ")")
	info := strings.Split(raw, "- ")
	if len(info) <= 1 {
		return name, url, ""
	}
	PackageCounter++
	return name, url, info[1]
}

type MyRoundTripper struct {
	r http.RoundTripper
}

func (mrt MyRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Add("Authorization", "Bearer: "+Config.AccessToken)
	return mrt.r.RoundTrip(r)
}

func getRepoStars(name, rawUrl, info string, tmpLinks *[]Package, mu *sync.Mutex, wg *sync.WaitGroup) {
	defer wg.Done()
	var repoMeta RepoDetails
	if rawUrl == "" {
		return
	}

	ctx := context.Background()
	client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: Config.AccessToken,
		TokenType:   "Bearer",
	}))

	tmpFields := strings.Split(rawUrl, "/")
	u, _ := url.Parse(STARS)
	u.Path = path.Join(u.Path, tmpFields[len(tmpFields)-2])
	u.Path = path.Join(u.Path, tmpFields[len(tmpFields)-1])
	url := u.String()
	resp, err := client.Get(url)
	if err != nil {
		log.Printf("unable to get star count for %s: %v\n", url, err)
		return
	} else {
		json.NewDecoder(resp.Body).Decode(&repoMeta)
		log.Printf("star count for %s: %d\n", url, repoMeta.StargazersCount)
	}

	LD := Package{
		Name:  name,
		URL:   rawUrl,
		Info:  info,
		Stars: repoMeta.StargazersCount,
	}

	mu.Lock()
	defer mu.Unlock()
	*tmpLinks = append(*tmpLinks, LD)
}

// Split is a driver function for splitting the Line from []Package
// it calls TrimString for splitting and handles a creation and appending of
// a result into a LinkDetails struct.
func SplitLinks(m map[string][]string) Categories {
	var mu sync.Mutex
	var wg sync.WaitGroup

	categories := make(Categories, len(m))
	i := 0

	for key, value := range m {
		var tmpLinks []Package
		token := strings.IndexByte(key, ' ')
		categories[i].Title = key[token+1:]
		wg.Add(len(value))
		for _, e := range value {
			name, url, info := trimString(e)
			go getRepoStars(name, url, info, &tmpLinks, &mu, &wg)
		}
		wg.Wait()
		categories[i].PackageDetails = append(categories[i].PackageDetails, tmpLinks...)
		i++
	}
	log.Println("package filter successful..")
	return categories
}
