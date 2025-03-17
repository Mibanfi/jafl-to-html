package main

import (
	"archive/zip"
	"bufio"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

// --- CONSTANTS ---

const DEFAULT_DIR = "."
const DEFAULT_OUTPUT = "output.html"
const DESIRED_EXT = ".xml"
const ZIP_EXT = ".zip"

const ANY = -1

const MAP_ATTACHMENT =
`
<img src="%s" id="%s" class="page map"></img>

`

const BOOK_TITLE =
`
<div class="page">
	<h1 class="title">%s</h1>
</div>
`

const HEAD =
`<head>
	<link rel="stylesheet" href="flands.css">
	<link rel="stylesheet" href="personal.css">

	<style>
		@media print {
			@page {
				@top-center {
					content: element(menu);
				}
			}
		}

		#menu {
			position: running(header);
		}
	</style>
</head>`

const COVER_NAME = "Cover.html"
const SHEET_NAME = "Sheet.html"
const MANIFEST_NAME = "Manifest.html"
const CODEWORDS_NAME = "Codewords%s.html"
const WORLDMAP_NAME = "global.jpg"
const RULES_NAME = "Rules.xml"
const QUICKRULES_NAME = "QuickRules.xml"

const BEFORE = true
const AFTER = false

const MENU =
`<div class="menu" id="menu">
	<table>
		<tr>
			<th colspan="4"><a href="#sheet">Adventure Sheet</a></th>
			<th colspan="4"><a href="#manifest">Ship's Manifest</a></th>
		</tr>
		<tr>
			<th colspan="2">Codewords:</th>
			<td><a href="#cd1">A</a></td>
			<td><a href="#cd2">B</a></td>
			<td><a href="#cd3">C</a></td>
			<td><a href="#cd4">D</a></td>
			<td><a href="#cd5">E</a></td>
			<td><a href="#cd6">F</a></td>
		</tr>
		<tr>
			<th colspan="1">Maps:</th>
			<td><a href="#map-world">World</a></td>
			<td><a href="#map-sokara">Sokara</a></td>
			<td><a href="#map-golnir">Golnir</a></td>
			<td><a href="#map-violet-ocean">Violet Ocean</a></td>
			<td><a href="#map-great-steppes">Great Steppes</a></td>
			<td><a href="#map-uttaku">Uttaku</a></td>
			<td><a href="#map-akatsurai">Akatsurai</a></td>
		</tr>
	</table>
</div>
`
const MENU_SINGULAR =	// book number, region name (linkified), region name
`<div class="menu" id="menu">
	<table>
		<tr>
			<th><a href="#sheet">Adventure Sheet</a></th>
			<th><a href="#manifest">Ship's Manifest</a></th>
			<th><a href="#cd%s">Codewords</a></th>
			<th><a href="#map-world">World Map</a></th>
			<th><a href="#map-%s">%s Map</a></th>
		</tr>
	</table>
</div>
`

// --- TYPES ---

type element struct {
	Name string
	Content string
	Attributes map[string]string
}

type stack []element

var root string
var output string
var dir string
var book int
var region = [...]string{"", "Sokara", "Golnir", "Violet Ocean", "Great Steppes", "Uttaku", "Akatsurai"}
var title = [...]string{
	"",
	"The War-Torn Kingdom",
	"Cities of Gold and Glory",
	"Over the Blood-Dark Sea",
	"The Plains of Howling Darkness",
	"The Court of Hidden Faces",
	"Lords of the Rising Sun",
}

// Flags
var b *int

func main() {
	b = flag.Int("b", 0, "Specify a single book number to process")

	flag.Parse()

	// Define the root directory
	root = flag.Arg(0)
	if root == "" {
		root = DEFAULT_DIR
		fmt.Println("Directory not defined. Operating in the current directory...")
	}

	// Define the output file
	output = flag.Arg(1)
	if output == "" {
		output = DEFAULT_OUTPUT
		fmt.Println("Output file not specified. Output will be saved in", DEFAULT_OUTPUT)
	}

	// List all book directories
	var books []string
	readDir, readErr := os.ReadDir(root)
	check(readErr)
	for _, f :=  range readDir {
		if filepath.Ext(f.Name()) == ZIP_EXT {
			books = append(books, f.Name())
		}
	}
	slices.Sort(books)

	// Unzip all book directories
	if existDir("book1", "book2", "book3", "book4", "book5", "book6") {
		fmt.Println("Located book folders.")
	} else {
		fmt.Println("Book folders not found. Extracting from root...")
		for _, d := range books {
			fmt.Println("Extracting", d)
			// Make a directory to store the extracted files
			os.Mkdir(stripExt(d), 0700)

			// Open the archive
			r, err := zip.OpenReader(filepath.Join(root, d))
			check(err)

			// Cycle through all files in archive
			for _, f := range r.File {
				fmt.Print("Extracting ", f.Name, "... ")
				// Open the file
				rc, err := f.Open()
				check(err)

				// Save the file
				rb, err := os.Create(filepath.Join(stripExt(d), f.Name))
				check(err)
				_, err = io.Copy(rb, rc)
				check(err)
				rc.Close()
				rb.Close()
				fmt.Println("done")
			}
			r.Close()
		}
	}

	var content string

	// Cycle through each book
	for book, dir = range books {
		book++
		// If the -b flag was used, only operate on a certain book
		if *b != 0 && book != *b{
			continue
		}
		dir = stripExt(dir)
		fmt.Printf("\n--- CONVERTING BOOK %d ---\n", book)
		fmt.Println("Directory:", dir)
		fmt.Println()

		// Import Adventurers.xml
		fmt.Printf("Processing file %s... ", ADVENTURERS)
		updateStats(filepath.Join(dir, ADVENTURERS))
		fmt.Println("loaded starting classes")

		// Make a sorted slice of all files in the book
		var filenames []string
		readDir, err := os.ReadDir(dir)
		check(err)
		for _, f :=  range readDir {
			filenames = append(filenames, f.Name())
		}
		slices.SortFunc(filenames, betterSort)

		// Add title page
		fmt.Print("Adding Title... ")
			content += fmt.Sprintf(BOOK_TITLE, title[book])
		fmt.Println("Done")

		// Add map
		fmt.Print("Importing Map... ")
			content += fmt.Sprintf(MAP_ATTACHMENT, filepath.Join(dir, region[book] + ".JPG"), "map-"+linkify(region[book]))
		fmt.Println("done")

		// Process all files
		for _, fn := range filenames {
			fmt.Printf("Processing file %s... ", fn)
			if strings.Contains(fn, "temp") || strings.Contains(fn, "old") || fn == ADVENTURERS{
				fmt.Println("ignored")
				continue
			}
			if fn == ADVENTURERS {
				continue
			}
			fn = filepath.Join(dir, fn)
			switch filepath.Ext(fn) {
				case DESIRED_EXT:
					page, err := parse(fn)
					check(err)
					content += page
					fmt.Println("done!")
				default:
					fmt.Println("ignored")
			}
		}

		fmt.Print("--- DONE ---\n\n")
	}

	// Add various materials
	fmt.Print("Importing Adventure Sheet... ")
	load(SHEET_NAME, &content, AFTER)
	fmt.Println("done")

	fmt.Print("Importing Ship's Manifest... ")
	load(MANIFEST_NAME, &content, AFTER)
	fmt.Println("done")

	fmt.Print("Importing World Map... ")
	copyFromRoot(WORLDMAP_NAME)
	content = fmt.Sprintf(MAP_ATTACHMENT, WORLDMAP_NAME, "map-world") + content
	fmt.Println("done")

	fmt.Print("Importing Quick Rules... ")
	copyFromRoot(QUICKRULES_NAME)
	parsedText, err := parse(QUICKRULES_NAME)
	check(err)
	content = parsedText + content
	fmt.Println("done")

	fmt.Print("Importing Rules... ")
	copyFromRoot(RULES_NAME)
	parsedText, err = parse(RULES_NAME)
	check(err)
	content = parsedText + content
	fmt.Println("done")

	fmt.Print("Importing Codewords... ")
	if *b != 0 {
		load(fmt.Sprintf(CODEWORDS_NAME, strconv.Itoa(*b)), &content, AFTER)
	} else {
		for i := 1; i <= 6; i++ {
			load(fmt.Sprintf(CODEWORDS_NAME, strconv.Itoa(i)), &content, AFTER)
		}
	}
	fmt.Println("done")

	fmt.Print("Importing Cover... ")
	load(COVER_NAME, &content, BEFORE)
	fmt.Println("done")

	// Add header
	content = HEAD + content

	// Prepare the output file
	fmt.Print("Creating output file... ")
	outFile, err := os.Create(output)
	check(err)
	defer outFile.Close()
	fmt.Println("done")

	// Save to HTML
	reader := strings.NewReader(content)
	fmt.Println("Saving to HTML...")
	htmlFile, err := os.Create(output)
	io.Copy(htmlFile, reader)
	htmlFile.Close()
	fmt.Println("Done")

	fmt.Println("\nFinished! Output saved in ", output)
}

func copyFromRoot(filename string) {
	_, err := os.Stat(filename)
	if err != nil {
		original, err := os.Open(filepath.Join(root, filename))
		check(err)
		defer original.Close()
		copied, err := os.Create(filename)
		check(err)
		defer copied.Close()
		io.Copy(copied, original)
	}
}

func menu() string {
	// Build the menu
	if *b == 0 {
		return MENU
	} else {
		return fmt.Sprintf(
			MENU_SINGULAR,
		     strconv.Itoa(*b),
				   linkify(region[*b]),
				   region[*b],
		)
	}
}

func stripExt(s string) string {
	return strings.TrimSuffix(s, filepath.Ext(s))
}

func linkify(s string) (out string) {
	for _, w := range strings.Fields(s) {
		out += strings.ToLower(w) + "-"
	}
	out = strings.TrimSuffix(out, "-")
	return
}

func existDir(names ...string) bool {
	for _, s := range names {
		_, err := os.Stat(s)
		if err != nil {
			return false
		}
	}
	return true
}

func load(path string, content *string, before bool) {
	file, err := os.Open(path)
	check(err)
	defer file.Close()
	raw, err := io.ReadAll(file)
	if before {
		*content = string(raw) + *content
	} else {
		*content += string(raw)
	}
}

func check(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(generateCode(err))
	}
}

func generateCode(err error) (code int) {
	for i := 0; i < 10 && i < len(fmt.Sprint(err)); i++ {
		code += int(fmt.Sprint(err)[i])
	}
	return
}

const FIRST_SECTION = "New.xml"
const A = -1
const B = 1
func betterSort(a, b string) int {
	// Check if one of the paragraph is the first (there can only be one first paragraph since it's based on filename)
	switch {
		case a == FIRST_SECTION:
			return A
		case b == FIRST_SECTION:
			return B
	}

	// Pure numbers go last
	n1, e1 := strconv.Atoi(strings.TrimSuffix(a, filepath.Ext(a)))
	n2, e2 := strconv.Atoi(strings.TrimSuffix(b, filepath.Ext(b)))
	switch {
		case (e1 == nil && e2 == nil):
			return n1 - n2
		case (e1 == nil && e2 != nil):
			return B
		case (e1 != nil && e2 == nil):
			return A
		default:
			return strings.Compare(a, b)
	}
}

func capitalize(in string) (out string) {
	for _, w := range strings.Fields(in) {
		runes := []rune(w)
		runes[0] = unicode.ToUpper(runes[0])
		out += string(runes) + " "
	}
	out = strings.TrimSpace(out)
	return
}

const ADVENTURERS = "Adventurers.xml"
const ADVENTURER_FORMAT =
`
`

const (
	READING_CONTENT = 0
	EXPECTING_ELEMENT = 1
	OPENING_ELEMENT = 2
	CLOSING_ELEMENT = 3
	EXPECTING_ATTRIBUTE = 4
	READING_ATTRIBUTE = 5
	EXPECTING_VALUE = 6
	READING_VALUE = 7
	SKIPPING_ELEMENT = 8
)
const (
	EOL = "\n"
	HISTORY_LENGTH = 50
	POSITION =
`[ Byte %d | Line %d | Mode %d | ]
Buffer contents: [name: %s, attr: %s, value: %s]
Stack contents: %s
Last few characters: %s`
)
func parse(filename string) (output string, err error) {
	var stack stack
	var mode byte
	var name, attr, value string
	var byteCount, lineCount int
	var history []rune
	var depth int
	file, errOpen := os.Open(filename)
	check(errOpen)
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanRunes)
	for scanner.Scan() {
		// Get byte
		c := scanner.Text()
		// Update position (for debugging)
		byteCount++
		if c == EOL {
			lineCount++
		}
		// Update history
		history = append(history, []rune(c)[0])
		if len(history) > HISTORY_LENGTH {
			history = history[1:]
		}
		position := fmt.Sprintf(POSITION, byteCount, lineCount, mode, name, attr, value, stack, string(history))

		// Operate this iteration
		switch mode {
			case EXPECTING_ELEMENT:
				switch c {
					case " ":
						continue
					case "?", "!":
						depth = 1
						mode = SKIPPING_ELEMENT
					case ">":
						err = errors.New(fmt.Sprintf("Element was opened and closed immediately %s", position))
						return
					case "/":
						name = ""
						mode = CLOSING_ELEMENT
					default:
						mode = OPENING_ELEMENT
						name = c
				}
			case OPENING_ELEMENT:
				switch c {
					case " ":
						stack.addElement(name)
						mode = EXPECTING_ATTRIBUTE
					case ">":
						stack.addElement(name)
						mode = READING_CONTENT
					case "/":
						stack.addElement(name)
						mode = CLOSING_ELEMENT
					default:
						name += c
				}
			case CLOSING_ELEMENT:
				switch c {
					case " ":
						continue
					case ">":
						if name != stack.Name() {
							err = errors.New(fmt.Sprintf("Tried to close an element that couldn't be closed %s", position))
							return
						} else {
							stack.popElement(&output)
							name = ""
							mode = READING_CONTENT
						}
					default:
						name += c
				}
			case EXPECTING_ATTRIBUTE:
				switch c {
					case ">":
						mode = READING_CONTENT
					case " ":
						continue
					case "/":
						mode = CLOSING_ELEMENT
					default:
						attr = c
						mode = READING_ATTRIBUTE
				}
			case READING_ATTRIBUTE:
				switch c {
					case "=":
						mode = EXPECTING_VALUE
					default:
						attr += c
				}
			case EXPECTING_VALUE:
				switch c {
					case "\"":
						value = ""
						mode = READING_VALUE
				}
			case READING_VALUE:
				switch c {
					case "\"":
						stack.addAttribute(attr, value)
						mode = EXPECTING_ATTRIBUTE
					default:
						value += c
				}
			case READING_CONTENT:
				switch c {
					case "<":
						mode = EXPECTING_ELEMENT
					default:
						stack.extendContent(c, &output)
				}
			case SKIPPING_ELEMENT:
				switch c {
					case "<":
						depth++
					case ">":
						depth--
						if depth == 0 {
							mode = READING_CONTENT
						}
				}
		}
	}
	return
}

func (s stack)String() (output string) {
	if len(s) > 0 {
		output += "["
		for _, e := range s {
			output += e.Name
			output += ", "
		}
		output = output[:len(output)-2]
		output += "]"
	} else {
		return "[]"
	}
	return
}

func (s *stack)addElement(name string) {
	var e element
	e.Name = name
	e.Attributes = make(map[string]string)
	*s = append(*s, e)
}

func (s *stack)addAttribute(key, value string) {
	(*s)[len(*s)-1].Attributes[key] = value
}

const SECTION = "section"
func (s *stack)popElement(output *string) {
	processedElement := replace((*s)[len(*s)-1])
	name := (*s)[len(*s)-1].Name
	*s = (*s)[0:len(*s)-1]
	switch {
		case name == SECTION:
			*output += processedElement
		case len(*s) > 0:
			(*s)[len(*s)-1].Content += processedElement
	}
}

func (s *stack)extendContent(c string, output *string) {
	if len(*s) > 0 {
		(*s)[len(*s)-1].Content += c
	} else {
		*output += c
	}
}

func (s *stack)Name() string {
	if len(*s) > 0 {
		return (*s)[len(*s)-1].Name
	} else {
		return ""
	}
}

func (e element) String() (output string) {
	output += "\n<"
	output += e.Name
	for attribute, value := range e.Attributes {
		output += " "
		output += attribute
		output += "=\""
		output += value
		output += "\""
	}
	output += ">\n\t"
	output += e.Content
	output += "\n</"
	output += e.Name
	output += ">"
	return
}

// THE GREAT REPLACING GALORE

const TICKBOX = "◻"

const FMT_SECTION =
`
<div class="page">
%s
<h2 id="%s"><span class="section-title">%s</span><span class="tickboxes">%s</span></h2>
%s
</div>
`

const FMT_LINK =
`<a href="#%s">%s</a>`

const FMT_TURNTO =
`<span class="turn-to">► Turn to %s</span>`

const FMT_SHOPITEM =
`<tr class="shop-item">
	<td colspan="4" class="shop-item-name">%s</td>
	<td colspan="1" class="shop-item-buy-price">%s</td>
	<td colspan="1" class="shop-item-sell-price">%s</td>
</tr>`

const FMT_SHOPHEADER =
`<tr class="shop-header">
<th colspan="4">Item</th>
<th colspan="1">Buy Price</th>
<th colspan="1">Sell Price</th>
</tr>`

const FMT_HEADER =
`<tr>
<th colspan="6">%s</th>
</tr>`

const FMT_BRANCHOPTION =
`<tr class="branch-option">
	<th>%s</th>
	<td>%s</td>
</tr>`

const FMT_TABLE =
`<table class="%s">
%s
</table>`

const FMT_RESURRECTION =
`<span class="resurrection">Resurrection of %s: Book %s, Section %s (%s)</span>`

const FMT_ROLL =
`roll %s dice`

const FMT_CHECK =
`make a %s check against a difficulty of %s`

const FMT_ITEM =
`<span class="item">%s</span>`

const FMT_IMAGE =
`<img src="%s"></img>`

const FMT_FIGHT =
`<table class="fight">
<tr>
<th colspan="3">%s</th>
</tr>
<tr>
<td>Combat: %s</td>
<td>Defence: %s</td>
<td>Stamina: %s</td>
</tr>
</table>`

type Group struct {
	Text string `xml:",innerxml"`
	Goto Goto `xml:"goto"`
}

type Goto struct {
	Section string `xml:",section"`
	Book string `xml:",book"`
}

func replace(e element) (out string) {
	// Remove hidden tags
	if e.Attributes["hidden"] == "t" {
		return
	}

	switch e.Name {

		// SECTION REPLACEMENT ----------------------------------------------------
		case "section":
			var tickboxes string
			var id string
			var boxCount int
			var ok bool
			if _, ok = e.Attributes["boxes"]; ok {
				boxCount, _ = strconv.Atoi(e.Attributes["boxes"])
				for i := 0; i < boxCount; i++ {
					tickboxes += " " + TICKBOX
				}
			}
			if profession, ok := e.Attributes["profession"]; ok {
				e.Content = (printStats(profession) + e.Content)
				id = strconv.Itoa(book) + "-" + strings.Fields(e.Attributes["name"])[0]
			} else {
				id = strconv.Itoa(book) + "-" + e.Attributes["name"]
			}

			out = fmt.Sprintf(FMT_SECTION, menu(), id, e.Attributes["name"], tickboxes, e.Content)

		// ------------------------------------------------------------------------

		// ITEM REPLACEMENT -------------------------------------------------------

		// If the tag is an item, we need to assemble the item's name.
		// Then we'll decide whether to display it as a shop item or a pickup
		case "weapon", "armor", "item", "tool", "ship", "cargo", "buy", "sell", "trade", "gain", "lose":
			var name string
			classItem := []string{"weapon", "armor", "item", "tool", "ship", "cargo", "stamina", "rank", "ability", "title"}
			// If the tag has content, that content will always override anything else.
			if strings.TrimSpace(e.Content) != "" {
				name = e.Content
			} else {
				// Otherwise, we must first find the base name (before any modifiers).
				// Let's check if there is a name attribute (the most straightforward way.)
				name, _ = e.Attributes["name"]
				// Some tags, especially 'trade' tags, instead have the name inside an attribute called like its type
				// Also, the 'crew' attribute exists but, when displaying the item, it is always bypassed in favor of its price in 'shards'.
				// So in the 'buy' and 'sell' tags, every item displays its name, EXCEPT for crews, which display the price
				// There is no logic in this
				if name == "" {
					for k, v := range e.Attributes {
						if slices.Contains(classItem, k) {
							name = v
						}
					}
					// Crews display the shard value so maybe
					if name == "" {
						name = e.Attributes["shards"] + " shards"
						// Some rare cases do not have a name at all, and instead inherit it from their tag name.
						// It is weird, I know, but some items in this game are generic so that you can flavour them as you like, especially weapons.
						if name == "" {
							name = e.Name
						}
					}
				}
				name = capitalize(name)

				// Now that we have found the base name, we must attach any properties it may have
				var properties string
				// Ships have a 'capacity' value that is not specified in tags because the game's internal logic keeps track of it
				switch name {
					case "barque":
						properties += "capacity: 1, "
					case "brigantine":
						properties += "capacity: 2, "
					case "galleon":
						properties += "capacity: 3, "
				}
				if _, ok := e.Attributes["initialCrew"]; ok {
					properties += "initial crew: " + e.Attributes["initialCrew"] + ", "
				}
				if _, ok := e.Attributes["bonus"]; ok {
					properties += "+" + e.Attributes["bonus"]
				}
				if _, ok := e.Attributes["ability"]; ok {
					properties += " to " + e.Attributes["ability"]
				}
				if properties != "" {
					properties = strings.TrimSuffix(properties, ", ")
					name += " (" + properties + ")"
				}
				// Let us put the freshly baked item into a span with class 'item'
				// This is important for formatting, as the books display items in a different font
				name = fmt.Sprintf(FMT_ITEM, name)
			}


			// Done! Now we must decide to format it either as a shop item or a pickup.
			// Fortunately for us, shop items are easily recognizable as they have a 'buy' or 'sell' attribute containing their price.
			// In fact, 'buy' and 'sell' tags use an attribute called 'shards' to record their price.
			// Sounds confusing? It is
			buy, ok1 := e.Attributes["buy"]
			sell, ok2 := e.Attributes["sell"]
			if ok1 || ok2 {
				if buy == "" {
					buy = "-"
				}
				if sell == "" {
					sell = "-"
				}
				out = fmt.Sprintf(FMT_SHOPITEM, name, buy, sell)
			} else {
				out = name
			}

		// ------------------------------------------------------------------------

		// BRANCH REPLACEMENT -----------------------------------------------------

		case "choice", "outcome", "success", "failure":
			// Branch options behave as table rows if they have a 'section' attribute, or as regular text otherwise.
			// Except for outcomes which are always table rows
			if sc, ok := e.Attributes["section"]; ok {
				// If there is no description, we autofill it.
				var description string
				if strings.TrimSpace(e.Content) == "" {
					description = e.Attributes["range"]
					if description == "" {
						description = capitalize(e.Name)
					}
				} else {
					description = e.Content
				}
				if bk, ok := e.Attributes["book"]; ok {
					sc = bk + "-" + sc
				} else {
					sc = strconv.Itoa(book) + "-" + sc
				}
				out = fmt.Sprintf(FMT_BRANCHOPTION, description, fmt.Sprintf(FMT_LINK, sc, fmt.Sprintf(FMT_TURNTO, e.Attributes["section"])))
			} else if e.Name == "outcome" {
				out = fmt.Sprintf(FMT_BRANCHOPTION, e.Attributes["range"], e.Content)
			} else {
				out = e.Content
			}

		// ------------------------------------------------------------------------

		// TABLE REPLACEMENT ------------------------------------------------------

		case "market", "choices", "outcomes":
			// These are easy because they never contain any Attributes.
			// But! They do always contain more tags in them that act as table rows
			// So I just make them a table to store the actual contents in
			var content string
			if e.Name == "market" {
				content = FMT_SHOPHEADER + e.Content
			} else {
				content = e.Content
			}
			out = fmt.Sprintf(FMT_TABLE, e.Name, content)

		// ------------------------------------------------------------------------


		// STANDALONE REPLACEMENTS ------------------------------------------------
		// These tags do not contain other tags within them
		// So it is just a matter of checking if there is content
		// And if there is none, replace it with a stock autofill string

		case "fight":
			out = fmt.Sprintf(FMT_FIGHT, e.Attributes["name"], e.Attributes["combat"], e.Attributes["defence"], e.Attributes["stamina"])

		case "resurrection":
			if strings.TrimSpace(e.Content) == "" {
				out = fmt.Sprintf(FMT_RESURRECTION, e.Attributes["god"], e.Attributes["book"], e.Attributes["section"], e.Attributes["text"])
			} else {
				out = e.Content
			}

		case "header":
			out = fmt.Sprintf(FMT_HEADER, capitalize(e.Attributes["type"]))

		case "goto":
			if strings.TrimSpace(e.Content) == "" {
				out = fmt.Sprintf(FMT_TURNTO, e.Attributes["section"])
			} else {
				out = e.Content
			}

		case "random":
			if strings.TrimSpace(e.Content) == "" {
				out = fmt.Sprintf(FMT_ROLL, e.Attributes["dice"])
			} else {
				out = e.Content
			}

		case "rankcheck":
			if strings.TrimSpace(e.Content) == "" {
				out = fmt.Sprintf(FMT_ROLL, e.Attributes["dice"]) + "and try to do lower than your Rank"
			} else {
				out = e.Content
			}

		case "difficulty":
			if strings.TrimSpace(e.Content) == "" {
				out = fmt.Sprintf(FMT_CHECK, e.Attributes["ability"], e.Attributes["level"])
			} else {
				out = e.Content
			}

		case "tick":
			if strings.TrimSpace(e.Content) == "" {
				out = "tick the box"
			} else {
				out = e.Content
			}

		case "if":
			out = strings.TrimSpace(e.Content)

		case "disease":
			if strings.TrimSpace(e.Content) == "" {
				out = e.Attributes["name"]
			} else {
				out = e.Content
			}
		case "image":
			out = fmt.Sprintf(FMT_IMAGE, filepath.Join(dir, e.Attributes["file"]))


		// ------------------------------------------------------------------------

		// THE DEVIOUS GROUP TAG --------------------------------------------------

		case "group":
			// 'group' tags only render the content of their inner 'text' tag
			// So I need to unmarshal that
			var group Group
			xml.Unmarshal([]byte(e.Content), &group)
			out = group.Text
			e.Attributes["section"], e.Attributes["book"] = group.Goto.Section, group.Goto.Book

		// ------------------------------------------------------------------------

		// DELETED TAGS -----------------------------------------------------------

		case "desc", "adjust", "effect":
			// These tags are straight up deleted because they are not rendered in the game
			return

		// ------------------------------------------------------------------------

		// IGNORED TAGS -----------------------------------------------------------

		default:
			// Unspecified tags are left as they are
			out = e.String()

		// ------------------------------------------------------------------------
	}

	// If there's a section attribute, add a link to that section
	sc, _ := e.Attributes["section"]
	bk, _ := e.Attributes["book"]

	switch {
		case sc != "" && bk == "":
			out = fmt.Sprintf(FMT_LINK, strconv.Itoa(book) + "-" + sc, out)
		case sc != "" && bk != "":
			out = fmt.Sprintf(FMT_LINK, bk + "-" + sc, out)
	}
	return
}

// --- ADVENTURERS MANAGEMENT ---

type AdventurersRaw struct {
	XMLName xml.Name `xml:"adventurers"`
	Stamina ParameterRaw `xml:"stamina"`
	Rank ParameterRaw `xml:"rank"`
	Gold ParameterRaw `xml:"gold"`
	Abilities []AbilityRaw `xml:"abilities>profession"`
	Equipment EquipmentRaw `xml:"items"`
	Professions []ProfessionRaw `xml:"starting>adventurer"`
}

type EquipmentRaw struct {
	Items []ItemRaw `xml:",any"`
}

type ParameterRaw struct {
	Value string `xml:"amount,attr"`
}

type AbilityRaw struct {
	Profession string `xml:"name,attr"`
	Content string `xml:",innerxml"`
}

type ItemRaw struct {
	XMLName xml.Name
	Profession string `xml:"profession,attr"`
	Name string `xml:"name,attr"`
	Bonus string `xml:"bonus,attr"`
}

type ProfessionRaw struct {
	PersonName string `xml:"name,attr"`
	Profession string `xml:"profession,attr"`
	Description string `xml:",innerxml"`
}

type Profession struct {
	Name string
	PersonName string
	Description string
	Rank string
	Stamina string
	Gold string
	Abilities []string
	Equipment []Item
}

type Item struct {
	Name string
	Type string
	Bonus string
}

var Starting map[string]Profession

func updateStats(fn string) () {
	// Get data from file
	var data []byte
	file, _ := os.Open(fn)
	defer file.Close()
	data, _ = io.ReadAll(file)

	// Process data
	var startingRaw AdventurersRaw
	errUnmarshal := xml.Unmarshal(data, &startingRaw)
	check(errUnmarshal)

	// Insert data into slice
	Starting = make(map[string]Profession)
	for _, p := range startingRaw.Professions {
		var profession Profession
		profession.Name = p.Profession
		profession.PersonName = p.PersonName
		profession.Description = p.Description
		profession.Rank = startingRaw.Rank.Value
		profession.Stamina = startingRaw.Stamina.Value
		profession.Gold = startingRaw.Gold.Value
		for _, a := range startingRaw.Abilities {
			if a.Profession == profession.Name {
				profession.Abilities = strings.Fields(a.Content)
				break
			}
		}
		for _, e := range startingRaw.Equipment.Items {
			if e.Profession == "" || e.Profession == profession.Name {
				var item Item
				item.Name = e.Name
				item.Bonus = e.Bonus
				item.Type = e.XMLName.Local
				profession.Equipment = append(profession.Equipment, item)
			}
		}
		// Insert the freshly baked profession into the slice
		Starting[profession.Name] = profession
	}
	return
}

/* CLASSES:
 * stats-sheet
 * 	stats-abilities-header
 * 		stats-ability-label
 * 	stats-abilities-values
 * 		stats-ability-value
 * 	stats-common-header
 * 		stats-stamina-label
 * 		stats-rank-label
 * 		stats-gold-label
 * 	stats-common-values
 * 		stats-stamina-value
 * 		stats-rank-value
 * 		stats-gold-value
 * 	equipment-header
 * 		equipment-label
 * 	equipment-value
 * 		equipment-item-type
 * 		equipment-item-name
 */
const STATS_FORMAT =	// p.Name, p.Abilities..., p.Stamina, p.Rank, p.Gold, startingEquip
`<h3 class="profession">
%s
</h3>
<table class="stats-sheet">
<tr>
<th>Charisma</th>
<th>Combat</th>
<th>Magic</th>
<th>Sanctity</th>
<th>Scouting</th>
<th>Thievery</th>
</tr>
<tr>
<td>%s</td>
<td>%s</td>
<td>%s</td>
<td>%s</td>
<td>%s</td>
<td>%s</td>
</tr>
<tr>
<th colspan="2">Stamina</th>
<th colspan="2">Rank</th>
<th colspan="2">Gold</th>
</tr>
<tr>
<td colspan="2">%s</td>
<td colspan="2">%s</td>
<td colspan="2">%s</td>
</tr>
<tr>
<th colspan="6">Starting equipment</th>
</tr>
%s
</table>`
const STARTING_EQUIP_FORMAT =
`<tr>
<th colspan="2">%s</th>
<td class="item" colspan="4">%s</td>
</tr>`
func printStats(name string) string {
	var p Profession
	var startingEquip string
	var ok bool
	p, ok = Starting[name]
	if !ok {
		fmt.Printf("Found no match for %s!\nThese are the starting professions registered from Adventurers.xml:\n", name)
		for k := range Starting {
			fmt.Println(k)
		}
		fmt.Println("Aborting.")
		os.Exit(666)
	}
	cha, com, mag, san, sco, thi := p.Abilities[0], p.Abilities[1], p.Abilities[2], p.Abilities[3], p.Abilities[4], p.Abilities[5]
	for _, e := range p.Equipment {
		var nameFull string
		e.Name, e.Type = capitalize(e.Name), capitalize(e.Type)
		if e.Bonus != "" {
			nameFull = e.Name + " (+" + e.Bonus + ")"
		} else {
			nameFull = e.Name
		}
		startingEquip += fmt.Sprintf(STARTING_EQUIP_FORMAT, e.Type, nameFull)
	}
	return fmt.Sprintf(STATS_FORMAT, p.Name, cha, com, mag, san, sco, thi, p.Stamina, p.Rank, p.Gold, startingEquip)
}
