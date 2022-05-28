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
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

var fetchFrom = "https://www.kjvbibles.com/blogs"
var initial []*Book
var enhanced []*BookEnhanced
var enhancedVerses = make(map[string]map[int]map[int]*VerseEnhanced)
var enhancements []Enhancement
var logFile = "./logs.txt"
var initialPath = "./json/initial"
var enhancedPath = "./json/enhanced"
var cachePath = "./cache"

func main() {
	loadInitialBooks()
	fetchBibleData()
	applyEnhancements()
	writeEnhancedBooks()
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
			enhancedBook := convertBookToEnhanced(book)
			enhanced = append(enhanced, enhancedBook)
			for _, c := range enhancedBook.Chapters {
				c := c
				for _, v := range c.Verses {
					v := v
					if _, ok := enhancedVerses[enhancedBook.Title]; !ok {
						enhancedVerses[enhancedBook.Title] = make(map[int]map[int]*VerseEnhanced)
					}
					if _, ok := enhancedVerses[enhancedBook.Title][c.Nb]; !ok {
						enhancedVerses[enhancedBook.Title][c.Nb] = make(map[int]*VerseEnhanced)
					}
					enhancedVerses[enhancedBook.Title][c.Nb][v.Nb] = v
				}
			}
			initial = append(initial, book)
		}
	}
}

func convertBookToEnhanced(book *Book) *BookEnhanced {
	var chaptersEn []*ChapterEnhanced
	for _, chap := range book.Chapters {
		chapEn := new(ChapterEnhanced)
		nb, err := strconv.Atoi(chap.Chapter)
		if err != nil {
			panic(fmt.Errorf("faield to convert chapter.chapter to int: %s", chap.Chapter))
		}
		var versesEn []*VerseEnhanced
		for _, verse := range chap.Verses {
			verseEn := new(VerseEnhanced)
			nb, err := strconv.Atoi(verse.Verse)
			if err != nil {
				panic(fmt.Errorf("failed to convert verse to int: %s", verse.Verse))
			}
			verseEn.Nb = nb
			verseEn.Text = verse.Text
			if verse.Title != "" {
				verseEn.Title = verse.Title
			}
			versesEn = append(versesEn, verseEn)
		}
		chapEn.Verses = versesEn
		chapEn.Nb = nb
		chaptersEn = append(chaptersEn, chapEn)
	}
	return &BookEnhanced{
		Title:    book.Book,
		Chapters: chaptersEn,
	}
}

func fetchBibleData() {
	fmt.Println("fetching/processing data...")
	for _, book := range initial {
		for _, chapter := range book.Chapters {
			if _, err := os.Stat(getCacheFileName(book, chapter)); err == nil {
				if cacheBytes, err := ioutil.ReadFile(getCacheFileName(book, chapter)); err == nil {
					fmt.Println("reading: ", getCacheFileName(book, chapter))
					tryWriteEnhancements(book, chapter, cacheBytes)
				}
				continue
			}
			fmt.Printf("Fetching: %s - Chapter %s\n", book.Book, chapter.Chapter)
			suffix := fmt.Sprintf("%s/%s-chapter-%s", book.Book, book.Book, chapter.Chapter)
			fullURL := strings.ToLower(fetchFrom + "/" + suffix)
			fullURL = strings.TrimSpace(strings.ReplaceAll(fullURL, " ", "-"))
			resp, err := http.Get(fullURL)
			if err != nil {
				logError(fmt.Sprintf("fail to fetch url: %s\n", fullURL), err)
				continue
			}
			bodyBytes, err := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				logError(fmt.Sprintf("error reading body for: %s - %s\n", book.Book, chapter.Chapter), err)
				continue
			}
			tryWriteEnhancements(book, chapter, bodyBytes)
			cacheData(book, chapter, bodyBytes)
			time.Sleep(2 * time.Second)
		}
	}
}

func logError(s string, err error) {
	loggedErr := fmt.Errorf("%s. Error: %w\n", s, err)
	fmt.Println(loggedErr)
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		_ = ioutil.WriteFile(logFile, []byte(""), 0644)
	}
	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println("error opening log file:", err)
	}
	defer file.Close()
	if _, err := file.WriteString(loggedErr.Error()); err != nil {
		log.Println("error writing to log file:", err)
	}

}

func tryWriteEnhancements(book *Book, chapter *Chapter, htmlData []byte) {
	fmt.Printf("Writing enchancements for %s - %s \n\n", book.Book, chapter.Chapter)
	document, err := goquery.NewDocumentFromReader(strings.NewReader(string(htmlData)))
	if err != nil {
		fmt.Println("failed to find document")
		panic(err)
	}
	strongs := document.Find("strong")
	if strongs == nil {
		panic("could not find strongs")
	}
	paragraphs := document.Find(".post p")
	if paragraphs == nil {
		panic(fmt.Errorf("fail to find paragraphs for %s - %v", book.Book, chapter.Chapter))
	}
	var titles []string
	strongs.Each(func(i int, strong *goquery.Selection) {
		if !strong.Parent().Is("p") {
			return
		}
		title := strings.TrimSpace(strong.Text())
		if title != "" {
			titles = append(titles, title)
		}
	})
	fmt.Println("titles: ")
	for _, t := range titles {
		fmt.Println(t)
	}
	fmt.Println("found count  : ", len(titles))

	paragraphs.Each(func(i int, par *goquery.Selection) {
		text := strings.TrimSpace(par.Text())
		title := ""
		for i, t := range titles {
			if strings.HasPrefix(text, title) {
				title = t
				titles = append(titles[:i], titles[i+1:]...)
				break
			}
		}
		if title == "" {
			panic(fmt.Errorf("failed to find title for %s - %v", book.Book, chapter.Chapter))
		}
		text = strings.TrimSpace(strings.Replace(text, title, "", 1))
		fmt.Println("text now : ", text)
		nbStr := ""
		if strings.HasPrefix(text, "Creation") {
			fmt.Println("this is it")
		}
		for _, r := range text {
			s := string(r)
			if s != " " {
				nbStr += s
				continue
			}
			break
		}
		nb, err := strconv.Atoi(nbStr)
		if err != nil {
			panic(fmt.Errorf("failed to convert verse %s to int with text: %s \n and title: %s : %w", nbStr, text, title, err))
		}
		chapNb, err := strconv.Atoi(chapter.Chapter)
		if err != nil {
			panic(fmt.Errorf("failed to convert chapter %s to int: %w \n\n with text: %s and title: %s", chapter.Chapter, err, text, title))
		}
		en := Enhancement{
			book:    book.Book,
			chapter: chapNb,
			verse:   nb,
			title:   title,
		}
		enhancements = append(enhancements, en)
		fmt.Printf("Created new enhancements for %s - %s\n%+v\n", book.Book, chapter.Chapter, en)
	})
}

func isGenesis1(book *Book, chapter *Chapter) bool {
	return book.Book == "Genesis" && chapter.Chapter == "1"
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
	wd, _ := os.Getwd()
	return path.Join(wd, cachePath, strings.ReplaceAll(strings.ToLower(fmt.Sprintf("%s-%s.html", book.Book, chapter.Chapter)), " ", ""))
}

func applyEnhancements() {
	for _, en := range enhancements {
		verse := enhancedVerses[en.book][en.chapter][en.verse]
		verse.Title = en.title
		fmt.Printf("set new verse: %s - %d - %d: %s\n", en.book, en.chapter, en.verse, verse.Title)
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
		fileName := fmt.Sprintf("%s/%s.json", enhancedPath, strings.ReplaceAll(book.Title, " ", ""))
		if err := ioutil.WriteFile(fileName, bytes, 0777); err == nil {
			fmt.Println("wrote file: ", fileName)
		}
	}
	fmt.Printf("wrote a total of %v books!\n", wroteCount)

}

type Enhancement struct {
	title   string
	verse   int
	chapter int
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
	Title string `json:"title,omitempty"`
	Verse string `json:"verse"`
	Text  string `json:"text"`
}

type BookEnhanced struct {
	Title    string             `json:"title,omitempty"`
	Chapters []*ChapterEnhanced `json:"chapters"`
}

type ChapterEnhanced struct {
	Nb     int              `json:"nb"`
	Verses []*VerseEnhanced `json:"verses"`
}

type VerseEnhanced struct {
	Title string `json:"title,omitempty"`
	Nb    int    `json:"nb"`
	Text  string `json:"text"`
}
