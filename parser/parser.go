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
	"reflect"
	"regexp"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
)

var PackageCounter int

func downloadMd() io.Reader {
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
	buf := downloadMd()
	m, count := GetSlice(buf)
	final := SplitLinks(m)
	check, diff := SyncReq(count)
	if check {
		DBWrite(MongoClient, final)
		log.Printf("sync successful.. added %d new packages\n", diff)
	} else {
		log.Println("no new packages to sync")
	}
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
		} else if strings.HasPrefix(line, "*") {
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

func getRepoStars(rawUrl string) (int, error) {
	if rawUrl == "" {
		return 0, nil
	}
	var repoMeta RepoDetails
	tmpFields := strings.Split(rawUrl, "/")
	u, _ := url.Parse(STARS)
	u.Path = path.Join(u.Path, tmpFields[len(tmpFields)-2])
	u.Path = path.Join(u.Path, tmpFields[len(tmpFields)-1])
	url := u.String()
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("unable to get star count: %v", err)
		return -1, err
	}
	json.NewDecoder(resp.Body).Decode(&repoMeta)
	return repoMeta.StargazersCount, nil
}

// Split is a driver function for splitting the Line from []Package
// it calls TrimString for splitting and handles a creation and appending of
// a result into a LinkDetails struct.
func SplitLinks(m map[string][]string) Categories {
	categories := make(Categories, len(m))
	i := 0
	for key, value := range m {
		var TmpLinks []Package
		token := strings.IndexByte(key, ' ')
		categories[i].Title = key[token+1:]
		for _, e := range value {
			name, url, info := trimString(e)
			stars, _ := getRepoStars(url)
			LD := Package{Name: name, URL: url, Info: info, Stars: stars}
			if reflect.ValueOf(LD).IsZero() {
				continue
			}
			TmpLinks = append(TmpLinks, LD)
		}
		categories[i].PackageDetails = append(categories[i].PackageDetails, TmpLinks...)
		i++
	}
	log.Println("package filter successful..")
	return categories
}
