package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/chromedp"
)

var fetchFrom = "https://www.kjvbibles.com/blogs"
var initial []*Book
var enhanced []*BookEnhanced
var enhancedVerses = make(map[string]map[int]map[int]*VerseEnhanced)
var urlExceptions = map[string]string{
	"1-samuel-chapter-16":    "1-samuel-chapter-166",
	"deuteronomy-chapter-18": "deuteronomy-chapter-19", // has chapter 18 twice
}
var verseTitlesExceptions = map[string]map[string]map[string]string{
	"1 Corinthians": {
		"13": {
			"1": "Love Is the Greatest Gift",
		},
	},
	"1 Kings": {
		"12": {
			"1": "The Israelites Ask Rehoboam to Lighten Their Burdens",
		},
		"14": {
			"1":  "Jeroboam Sends His Wife, Disguised, to the Prophet Ahijah",
			"21": "Rehoboam’s Evil Reign over the Southern Kingdom of Judah",
		},
	},
	"1 Samuel": {
		"10": {
			"1":  "Samuel Anoints Saul",
			"17": "The Lord Confirms His Choice of Saul",
		},
		"24": {
			"1":  "Elkanah and His Wives Go to Shiloh Every Year to Worship",
			"9":  "Hannah Prays for a Child",
			"19": "Hannah, Having Given Birth to Samuel, Stays at Home till he is Weaned",
		},
	},
	"1 Thessalonians": {
		"2": {
			"1": "Paul Recalls His Visit",
		},
		"3": {
			"1": "Timothy’s Report to Paul",
		},
	},
	"2 Chronicles": {
		"2": {
			"1": "Solomon’s Laborers for the Building of the Temple",
		},
		"3": {
			"1": "The Temple Is Built",
		},
		"4": {
			"1": "The Furnishings for the Temple",
		},
		"5": {
			"1": "The Lord’s Glory Fills the Temple",
		},
		"6": {
			"1": "Solomon Blesses the People and Praises God",
		},
		"7": {
			"1": "God Recognizes Solomon’s Prayer by Fire from Heaven",
		},
	},
	"2 John": {
		"1": {
			"1": "Living in the truth",
			"7": "Reject False Teachers",
		},
	},
	"2 Peter": {
		"1": {
			"1":  "God's Power for Godly Lives",
			"16": "Listen to God's Words",
		},
		"2": {
			"1": "Warnings about False Teachers",
		},
		"3": {
			"1":  "Be Ready for Christ's Return",
			"8":  "God's Patience",
			"11": "Live Holy Lives",
		},
	},
	"Deuteronomy": {
		"1": {
			"1":  "Moses Speaks to the People",
			"19": "Moses Tells of Sending the Spies and the People's Rebellion",
		},
		"2": {
			"1":  "The Story of the Wilderness Wanderings",
			"24": "The Story of the Conquest of Sihon the Amorite, King of Heshbon",
		},
		"3": {
			"1":  "The Story of the Conquest of Og, King of Bashan",
			"12": "The Distribution of Land to the Tribes East of the Jordan",
		},
		"4": {
			"1":  "An Exhortation to Obedience",
			"41": "Moses Appoints the Three Cities of Refuge beyond Jordan",
		},
		"5": {
			"1": "The Covenant Made at Mount Horeb",
			"6": "The Ten Commandments",
		},
		"6": {
			"1": "The Purpose of the Law and an Exhortation to Obey It",
		},
		"7": {
			"1": "God Commands Israel to Destroy the Canaanites and Their Idols",
		},
		"8": {
			"1": "An Exhortation to Obedience and Remembrance",
		},
		"9": {
			"1": "Moses Reminds the People of Their Many Rebellions",
		},
		"10": {
			"1": "God's Mercy in Restoring the Two Tablets of the Ten Commandments",
		},
		"11": {
			"1":  "An Exhortation to Obey the Commandments",
			"8":  "The Promise of God's Great Blessings",
			"18": "A Careful Study of God's Words Is Required",
			"26": "A Blessing and a Curse Are Set before the People",
		},
		"12": {
			"1": "Monuments of Idolatry Are to Be Destroyed",
			"5": "The Proper Place to Worship",
		},
		"13": {
			"1":  "Dealing with False Prophets",
			"6":  "Dealing with a Family Member's Idolatry",
			"12": "Dealing with Idolatrous Cities",
		},
		"14": {
			"1":  "God's Children Are Not to Disfigure Themselves in Mourning",
			"22": "Giving God One-Tenth of Everything",
		},
		"15": {
			"1":  "Canceling Debts Every Seven Years",
			"12": "Laws Regarding Hebrew Slaves",
		},
		"16": {
			"1":  "The Three Major Festivals",
			"18": "Laws about Administering Justice",
		},
		"17": {
			"1":  "Laws about the Sacrifices",
			"8":  "Hard Controversies Ruled upon by Priests and Judges",
			"14": "The Duties of a King",
		},
		"18": {
			"1":  "Laws about the Rightful Dues for the Priests and the Levites",
			"9":  "Laws about the Abominations of the Nations in the Land",
			"15": "The Lord Will Raise up a Prophet",
		},
		"19": {
			"1":  "Laws about the Cities of Refuge",
			"14": "Laws about the Land Boundaries and Witnesses at Trial",
		},
		"20": {
			"1":  "Laws about Warfare and the Soldiers Sent into Battle",
			"10": "What to Do to the Cities That Accept or Refuse the Proclamation of Peace",
		},
		"21": {
			"1":  "The Expiation of an Uncertain Murder",
			"10": "The Treatment of a Captive Taken for a Wife",
			"15": "The Firstborn Is Not to Be Disinherited on Private Preference",
			"18": "A Stubborn Son Shall Be Stoned to Death",
		},
		"22": {
			"1":  "Laws about Personal Property",
			"5":  "Varied Laws",
			"13": "Laws about Sex and Marriage",
		},
		"23": {
			"1":  "Who May or May Not Enter into the Congregation",
			"9":  "Uncleanness to Be Avoided in the Army",
			"15": "Varied Laws",
		},
		"24": {
			"1":  "Laws about Divorce",
			"5":  "Varied Laws",
			"14": "Varied Laws, Including Caring for the Poor, Widows, and Orphans",
		},
		"25": {
			"1":  "A Condemned Man Must Not Be Beaten with More Than Forty Stripes",
			"5":  "Laws about Keeping Family Lines",
			"13": "Varied Laws",
		},
		"26": {
			"1":  "Bringing the Firstfruits of the Land before the Lord in Thankfulness",
			"12": "The Prayer of Him That Gives His Third Year Tithes",
			"16": "The Covenant between God and the People",
		},
		"27": {
			"1":  "The People Are Commanded to Write the Laws upon Stones",
			"11": "Reciting the Curses for Disobedience",
		},
		"28": {
			"1":  "Reciting the Blessings for Obedience",
			"15": "Curses from the Lord",
		},
		"29": {
			"1":  "Israel's Past, Present, and Future",
			"10": "All Are Presented before the Lord to Enter into His Covenant",
		},
		"30": {
			"1":  "Great Mercies Promised to the Repentant",
			"11": "The Commandment Is Not Hard",
			"15": "The Choice between Death and Life",
		},
		"31": {
			"1":  "Joshua Will Lead the People",
			"9":  "Moses Encourages Reading God's Law",
			"14": "God Gives a Charge to Joshua",
		},
		"32": {
			"1": "Moses' Song Which Sets Forth God's Mercy and Vengeance",
		},
		"33": {
			"1":  "The Blessings of the Twelve Tribes",
			"26": "The Everlasting Arms of the Eternal God",
		},
		"34": {
			"1": "Moses View the Land from Mount Nebo",
			"5": "Moses Dies in the Land of Moab",
			"9": "Joshua Succeeds Moses",
		},
	},
}
var enhancements []Enhancement
var logFile = "./logs.txt"
var initialPath = "./json/initial"
var enhancedPath = "./json/enhanced"
var cachePath = "./cache"

func main() {
	loadInitialBooks()
	fetchBibleData()
	// applyEnhancements()
	// writeEnhancedBooks()
}

func getEnhancedBook(title string) *BookEnhanced {
	for _, b := range enhanced {
		if b.Title == title {
			return b
		}
	}
	return nil
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

func countTotalToFetch() int {
	nb := 0
	for _, b := range initial {
		for _, _ = range b.Chapters {
			nb++
		}
	}
	return nb
}

func fetchBibleData() {
	fmt.Println("fetching/processing data...")
	for _, book := range initial {
		for _, chapter := range book.Chapters {
			fullURL := getFullUrl(book, chapter)
			if _, err := os.Stat(getCacheFileName(book, chapter)); err == nil {
				if cacheBytes, err := ioutil.ReadFile(getCacheFileName(book, chapter)); err == nil {
					tryWriteEnhancements(book, chapter, string(cacheBytes))
				}
				continue
			}
			fmt.Printf("Fetching: %s - Chapter %s\n", book.Book, chapter.Chapter)

			bodyStr, err := scrapeURL(fullURL)
			if err != nil {
				logError(fmt.Errorf("failed to scrape: %s, error: %w", fullURL, err))
			}
			tryWriteEnhancements(book, chapter, bodyStr)
			cacheData(book, chapter, []byte(bodyStr))
			time.Sleep(time.Second * 2)
		}
	}
}

func scrapeURL(url string) (res string, err error) {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	err = chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.WaitReady("body"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			node, err := dom.GetDocument().Do(ctx)
			if err != nil {
				return err
			}
			res, err = dom.GetOuterHTML().WithNodeID(node.NodeID).Do(ctx)
			return err
		}),
	)
	if err != nil {
		logError(fmt.Errorf("error scraping %s: %w", url, err))
	}

	return res, nil
}

func getFullUrl(book *Book, chapter *Chapter) string {
	bookPath := strings.ReplaceAll(strings.ToLower(book.Book+"/"), " ", "-")
	chapterPath := strings.ToLower(fmt.Sprintf("%s-chapter-%s", strings.ReplaceAll(book.Book, " ", "-"), chapter.Chapter))
	if book.Book == "1-samuel" && chapter.Chapter == "16" {
		fmt.Println("salut: ")
	}
	// some URL don't have the one that they should so replace with exception
	if val, ok := urlExceptions[chapterPath]; ok {
		chapterPath = val
	}
	fullURL := strings.ToLower(fetchFrom + "/" + bookPath + chapterPath)
	fullURL = strings.TrimSpace(strings.ReplaceAll(fullURL, " ", "-"))
	return fullURL
}

func logError(err error) {
	fmt.Println(err)
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		_ = ioutil.WriteFile(logFile, []byte(""), 0644)
	}
	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if file != nil {
		defer file.Close()
	}
	if err != nil {
		log.Println("error opening log file:", err)
		return
	}
	if _, err := file.WriteString(err.Error()); err != nil {
		log.Println("error writing to log file:", err)
	}
}

func tryWriteEnhancements(book *Book, chapter *Chapter, htmlData string) {
	if book.Book == "2 Chronicles" && chapter.Chapter == "9" {
		fmt.Println("test")
	}
	display := fmt.Sprintf("%s - %s", book.Book, chapter.Chapter)
	// todo here
	// if !(book.Book == "Genesis" && chapter.Chapter == "1") {
	// 	return
	// }
	fmt.Printf("Writing enchancements for %s - %s \n\n", book.Book, chapter.Chapter)
	document, err := goquery.NewDocumentFromReader(strings.NewReader(htmlData))

	if err != nil {
		fmt.Println("failed to create document")
		panic(err)
	}

	exception := getException(book, chapter)
	if err := titleIsExpectedD(document, book, chapter); exception == nil && err != nil {
		logError(err)
	}

	if book.Book == "1 Chronicles" && chapter.Chapter == "13" {
		fmt.Println("debug")
	}

	if exception != nil {

		c, _ := strconv.Atoi(chapter.Chapter)
		for verse, title := range exception {
			v, _ := strconv.Atoi(verse)
			en := Enhancement{
				book:    book.Book,
				chapter: c,
				verse:   v,
				title:   title,
			}
			enhancements = append(enhancements, en)

			fmt.Printf("Created new exception enhancements for %s - %s\n%+v\n", book.Book, chapter.Chapter, en)
		}
		return
	}

	parSelector := ""
	verseTexts := document.Find(".verse-text")
	pcount := len(document.Find(".post p").Nodes)
	switch len(chapter.Verses) {
	case pcount:
		parSelector = ".post p"
	case len(verseTexts.Nodes):
		parSelector = ".post .verse-text"
	default:
		parseSingleChapter(book, chapter, document)
		return
	}
	containers := document.Find(parSelector)
	if containers == nil {
		panic(fmt.Errorf("fail to find text container for %s - %v", book.Book, chapter.Chapter))
	}

	currT := ""
	curVerse := 0
	startVerse := -1
	containers.Each(func(i int, par *goquery.Selection) {
		title := ""
		text := ""
		if currT == "" {
			strong := par.Find("strong").First()
			if strong == nil {
				panic("unable to find strong")
			}
			// title
			title = strings.TrimSpace(strong.Text())
			title = strings.TrimSuffix(title, ".")
		} else {
			title = currT
		}

		// text
		text = par.Text()
		text = strings.ReplaceAll(text, "\u2009", " ") // replace thin spaces by spaces
		text = strings.TrimSpace(text)
		currT = ""
		if text == title {
			currT = title
			return
		}

		if strings.HasPrefix(text, title) {
			text = strings.TrimSpace(strings.Replace(text, title, "", 1))
		}

		// get chapter text
		for text != "" {
			startText := verseStartText(curVerse + 1)
			curVerse++
			if startVerse == -1 {
				startVerse = curVerse
			}
			if curVerse == 17 {
				fmt.Println("look")
			}

			if strings.HasPrefix(text, startText) {
				if i := strings.Index(text, verseStartText(curVerse+1)); i != -1 {
					text = strings.Replace(text, startText, "", 1)
					i = strings.Index(text, verseStartText(curVerse+1))
					actualText := strings.TrimSpace(text[:i])
					setEnhancedVerseText(book.Book, chapter.Chapter, fmt.Sprintf("%d", curVerse), actualText)
					text = strings.TrimSpace(text[i:])

				} else {
					// at end of chapter
					text = strings.Replace(text, startText, "", 1)
					actual := strings.TrimSpace(text)
					setEnhancedVerseText(book.Book, chapter.Chapter, fmt.Sprintf("%d", curVerse), actual)

					text = strings.Replace(text, actual, "", 1)
				}
			} else {
				// at end of paragraph
				text = strings.Replace(text, startText, "", 1)
				actual := strings.TrimSpace(text)
				setEnhancedVerseText(book.Book, chapter.Chapter, fmt.Sprintf("%d", curVerse), actual)
				text = strings.Replace(text, actual, "", 1)
			}
		}

		// title per verse
		chapNb, _ := strconv.Atoi(chapter.Chapter)
		en := Enhancement{
			book:    book.Book,
			chapter: chapNb,
			verse:   startVerse,
			title:   title,
		}
		enhancements = append(enhancements, en)
		startVerse = -1

		fmt.Printf("Created new enhancements for %s - %s - %d\n%+v\n", book.Book, chapter.Chapter, startVerse, en)
	})

	// verify if verse count is the same
	if len(chapter.Verses) != curVerse {
		panic(fmt.Errorf("chapter %s doesnt have the same amount of verse, original:%v, actual:%v", display, len(chapter.Verses), curVerse))
	}

}

func parseSingleChapter(book *Book, chapter *Chapter, doc *goquery.Document) {
	if book.Book == "2 Chronicles" && chapter.Chapter == "9" {
		fmt.Println("test")
	}

	var titles []string
	document := doc.Clone()
	document = document.Find(".post strong")
	document.Each(func(i int, s *goquery.Selection) {
		titles = append(titles, strings.TrimSpace(s.Text()))
	})

	verseTitle := ""
	fmt.Println("what is up")
	text := doc.Find(".post").Text()
	text = strings.ReplaceAll(text, "\u2009", " ") // replace thin spaces by spaces
	startInd := strings.Index(text, titles[0])

	// title at top
	if startInd != -1 {
		verseTitle = strings.TrimSpace(titles[0])
		createEnhancement(book, chapter, 1, verseTitle)
	}
	text = text[startInd+len(titles[0]):]
	if i := strings.Index(text, "< Previous Chapter"); i != -1 {
		text = text[:i-1]
		text = strings.TrimSpace(text)
	}

	curVerse := 0
	for text != "" {
		startText := verseStartText(curVerse + 1)
		curVerse++
		if verseTitle = endsWithTitle(text, titles, curVerse); len(verseTitle) > 0 {
			verseTitle = strings.TrimSpace(verseTitle)
			createEnhancement(book, chapter, curVerse+1, verseTitle)
			ind := strings.Index(text, verseTitle)
			// remove title from text
			text = text[:ind] + text[ind+len(verseTitle):]
		}

		if strings.HasPrefix(text, startText) {
			if i := strings.Index(text, verseStartText(curVerse+1)); i != -1 {
				text = strings.Replace(text, startText, "", 1)
				i = strings.Index(text, verseStartText(curVerse+1))
				actual := strings.TrimSpace(text[:i])
				setEnhancedVerseText(book.Book, chapter.Chapter, fmt.Sprintf("%d", curVerse), actual)
				text = strings.TrimSpace(text[i:])
			} else {
				// at end of verse
				text = strings.Replace(text, startText, "", 1)
				actual := strings.TrimSpace(text)
				setEnhancedVerseText(book.Book, chapter.Chapter, fmt.Sprintf("%d", curVerse), actual)
				text = strings.Replace(text, actual, "", 1)
			}
		} else {
			// at end of verse
			text = strings.Replace(text, startText, "", 1)
			actual := strings.TrimSpace(text)
			setEnhancedVerseText(book.Book, chapter.Chapter, fmt.Sprintf("%d", curVerse), actual)
			text = strings.Replace(text, actual, "", 1)
		}
	}
}

func createEnhancement(book *Book, chapter *Chapter, verse int, title string) {
	if title == "" {
		return
	}

	chapNb, _ := strconv.Atoi(chapter.Chapter)
	en := Enhancement{
		book:    book.Book,
		chapter: chapNb,
		verse:   verse,
		title:   title,
	}
	enhancements = append(enhancements, en)
	fmt.Printf("ENHANCEMENT: %s - %s - %s, title: %s\n", book.Book, chapter.Chapter, fmt.Sprintf("%d", verse), title)
}

func startsWith(text string, titles []string) int {
	for i, title := range titles {
		if strings.HasPrefix(text, title) {
			return i
		}
	}
	return -1
}

func endsWithTitle(text string, titles []string, curVerse int) string {
	start := verseStartText(curVerse + 1)
	if i := strings.Index(text, start); i != -1 {
		text = text[:i]
	}
	for _, title := range titles {
		if strings.HasSuffix(text, title) {
			return title
		}
	}
	return ""
}

func getException(book *Book, chapter *Chapter) map[string]string {
	_, ok := verseTitlesExceptions[book.Book]
	if ok {
		v, ok := verseTitlesExceptions[book.Book][chapter.Chapter]
		if ok {
			return v
		}
	}
	return nil
}

func verseStartText(verse int) string {
	return fmt.Sprintf("%d ", verse)
}

func setEnhancedVerseText(book, chapter, verse, text string) {
	var v *VerseEnhanced
	eb := getEnhancedBook(book)
	cNb, _ := strconv.Atoi(chapter)
	vNb, _ := strconv.Atoi(verse)
	for _, c := range eb.Chapters {
		if c.Nb == cNb {
			for _, ver := range c.Verses {
				if ver.Nb == vNb {
					v = ver
					break
				}
			}
		}
	}
	if v == nil {
		panic(fmt.Errorf("failed to find en: %s", book+"-"+chapter+"-"+verse))
	}
	v.Text = text
	fmt.Printf("set %s - %s - %s with text: %s\n", book, chapter, verse, text)
}

func isNewVerse(currentChapter int, text string) bool {
	return strings.HasPrefix(text, fmt.Sprintf("%d ", (currentChapter+1)))
}

func titleIsExpected(document *goquery.Selection, book *Book, chapter *Chapter) error {
	pageTitleElement := document.Find("title")
	if pageTitleElement == nil {
		return fmt.Errorf("could not find page title")
	}
	pageTitle := pageTitleElement.Text()
	pageTitle = strings.ReplaceAll(pageTitle, "– KJV Bibles", "")
	pageTitle = strings.TrimSpace(pageTitle)
	expectedTitle := strings.TrimSpace(fmt.Sprintf("%s Chapter %v", book.Book, chapter.Chapter))
	if !strings.EqualFold(pageTitle, expectedTitle) {
		fmt.Println("page title: ", pageTitle)
		fmt.Println("expected title: ", expectedTitle)
		errorMessage := "check Website, page title doesn't match current book/chapter: " + pageTitle + ", expecting: " + expectedTitle
		return fmt.Errorf(errorMessage)
	}
	return nil
}
func titleIsExpectedD(document *goquery.Document, book *Book, chapter *Chapter) error {
	pageTitleElement := document.Find("title")
	if pageTitleElement == nil {
		return fmt.Errorf("could not find page title")
	}
	pageTitle := pageTitleElement.Text()
	pageTitle = strings.ReplaceAll(pageTitle, "– KJV Bibles", "")
	pageTitle = strings.TrimSpace(pageTitle)
	func(strs []string) {
		for _, str := range strs {
			pageTitle = strings.ReplaceAll(pageTitle, str, "")
		}
	}([]string{"<", ">"})
	expectedTitle := strings.TrimSpace(fmt.Sprintf("%s Chapter %v", book.Book, chapter.Chapter))
	if !strings.EqualFold(pageTitle, expectedTitle) {
		fmt.Println("page title: ", pageTitle)
		fmt.Println("expected title: ", expectedTitle)
		errorMessage := "check Website, page title doesn't match current book/chapter: " + pageTitle + ", expecting: " + expectedTitle
		return fmt.Errorf(errorMessage)
	}
	return nil
}

func verifyTitle() bool {
	return false
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

func deepClone(books []*Book) []*Book {
	bytes, err := json.Marshal(books)
	if err != nil {
		panic(fmt.Errorf("error marshalign for deep copy: %w", err))
	}
	clone := make([]*Book, 0)
	if err := json.Unmarshal(bytes, &clone); err != nil {
		panic(fmt.Errorf("error unmarshanlig for ddep copy: %w", err))
	}
	return clone
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
