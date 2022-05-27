package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

var fetchFrom = "https://www.kjvbibles.com/blogs"
var initial []*Book
var enhanced []*Book
var enhancedVerses = make(map[string]map[string]map[string]*Verse)
var enhancements []Enhancement
var initialPath = "./json/initial"
var enhancedPath = "./json/enhanced"
var cachePath = "./cache"

func main() {
	loadInitialBooks()
	fetchEnhancements()
	applyEnhancements()
	// writeEnhancedData()
}

func loadInitialBooks() {
	files, err := ioutil.ReadDir(initialPath)
	if err != nil {
		panic(err)
	}
	fmt.Println("reading initial data...")
	for _, file := range files {
		if filepath.Ext(file.Name()) != ".json" {
			continue
		}
		bytes, err := ioutil.ReadFile(initialPath + "/" + file.Name())
		if err != nil {
			log.Fatal(err)
		}
		book := new(Book)
		if err = json.Unmarshal(bytes, book); err == nil {
			enhancedBook := new(Book)
			*enhancedBook = *book
			enhanced = append(enhanced, enhancedBook)
			for _, c := range book.Chapters {
				c := c
				for _, v := range c.Verses {
					v := v
					if _, ok := enhancedVerses[enhancedBook.Book]; !ok {
						enhancedVerses[enhancedBook.Book] = make(map[string]map[string]*Verse)
					}
					if _, ok := enhancedVerses[enhancedBook.Book][c.Chapter]; !ok {
						enhancedVerses[enhancedBook.Book][c.Chapter] = make(map[string]*Verse)
					}
					enhancedVerses[enhancedBook.Book][c.Chapter][v.Verse] = v
				}
			}
			initial = append(initial, book)
		}
	}
}

func fetchEnhancements() {
	for _, book := range initial {
		for _, chapter := range book.Chapters {
			if cacheBytes, err := ioutil.ReadFile(getCacheFileName(book, chapter)); err == nil {
				fmt.Printf("Restored: %s - %s file from cache!\n", book.Book, chapter.Chapter)
				processChapter(chapter, cacheBytes)
				continue
			}
			fmt.Printf("Fetching: %s - Chapter %s\n", book.Book, chapter.Chapter)
			suffix := fmt.Sprintf("%s/%s-chapter-%s", book.Book, book.Book, chapter.Chapter)
			fullURL := strings.ToLower(path.Join(fetchFrom, suffix))
			fullURL = strings.TrimSpace(strings.ReplaceAll(fullURL, " ", "-"))
			fmt.Println("THE URL OF REQ: ", fullURL)
			fullURL = "https://www.kjvbibles.com"
			resp, err := http.Get(fullURL)
			if err != nil {
				fmt.Printf("fail to fetch url: %s\n", fullURL)
				panic(err)
			}
			bodyBytes, err := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				panic(err)
			}
			body := string(bodyBytes)
			processChapter(chapter, bodyBytes)
			fmt.Println("GOT BODY: ", body)
			cacheData(book, chapter, bodyBytes)
			break
		}
		break
	}
}

func processChapter(chapter *Chapter, cacheBytes []byte) {
	fmt.Println("processing chapter:")
	fmt.Println(string(cacheBytes))
}

func cacheData(book *Book, chapter *Chapter, bytes []byte) {
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		_ = os.MkdirAll(cachePath, 0777)
	}
	fileName := getCacheFileName(book, chapter)
	err := ioutil.WriteFile(fileName, bytes, 0777)
	if err != nil {
		log.Fatalf("fail to cache file: %s\n", fileName)
	}
}

func getCacheFileName(book *Book, chapter *Chapter) string {
	return path.Join(cachePath, strings.ReplaceAll(strings.ToLower(fmt.Sprintf("%s-%s.html", book.Book, chapter.Chapter)), " ", ""))
}

func applyEnhancements() {
	for _, en := range enhancements {
		verse := enhancedVerses[en.book][en.chapter][en.verse]
		verse.Title = en.title
		fmt.Printf("set new verse: %s - %s - %s: %s\n", en.book, en.chapter, en.verse, verse.Title)
	}
	fmt.Printf("applied %v enhancements!\n", len(enhancements))
}

func writeEnhancedBooks() {
	os.RemoveAll(enhancedPath)
	if err := os.MkdirAll(enhancedPath, 0777); err != nil {
		panic(err)
	}

	wroteCount := 0
	for _, book := range enhanced {
		bytes, err := json.Marshal(book)
		if err != nil {
			panic(err)
		}
		wroteCount++
		fileName := fmt.Sprintf("%s/%s.json", enhancedPath, strings.ReplaceAll(book.Book, " ", ""))
		if err := ioutil.WriteFile(fileName, bytes, 0777); err == nil {
			fmt.Println("wrote file: ", fileName)
		}
	}
	fmt.Printf("wrote a total of %v books!\n", wroteCount)

}

type Enhancement struct {
	title   string
	verse   string
	chapter string
	book    string
}

type Book struct {
	Book     string     `json:"book"`
	Chapters []*Chapter `json:"chapters"`
}

type Chapter struct {
	Chapter string   `json:"chapter"`
	Verses  []*Verse `json:"verses"`
}

type Verse struct {
	Title string `json:"title"`
	Verse string `json:"verse"`
	Text  string `json:"text"`
}
