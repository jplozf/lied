// ****************************************************************************
//
//	 _ _          _
//	| (_) ___  __| |
//	| | |/ _ \/ _` |
//	| | |  __/ (_| |
//	|_|_|\___|\__,_|
//
// ****************************************************************************
// L I E D   -   Copyright © JPL 2024
// ****************************************************************************
// Package rss provides an RSS/Atom Feed Reader plugin for lied.
// It fetches the configured feed URLs, lists their items and shows the
// selected article's content — all from within the editor.
// ****************************************************************************
package rss

// ****************************************************************************
// IMPORTS
// ****************************************************************************
import (
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"lied/conf"
	"lied/edit"
	"lied/menu"
	"lied/ui"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// ****************************************************************************
// CONSTANTS
// ****************************************************************************
const (
	PluginID        = "rssreader"
	ContentPageName = "rssReader"
	// UniqueID is used as the synthetic FName when this plugin is open in the
	// views list so that duplicate opens can be detected.
	UniqueID     = PluginID + "://" + PluginID
	fetchTimeout = 10 * time.Second
)

// defaultFeeds seeds the feed list file the first time it is created.
var defaultFeeds = []string{
	"https://news.ycombinator.com/rss",
	"http://feeds.bbci.co.uk/news/world/rss.xml",
}

var tagRe = regexp.MustCompile(`<[^>]*>`)

// ****************************************************************************
// Feed parsing types
// ****************************************************************************

// FeedItem holds the fields displayed for a single article.
type FeedItem struct {
	Title       string
	Link        string
	Description string
	PubDate     string
	PubTime     time.Time
	Source      string
}

type rssXML struct {
	XMLName xml.Name `xml:"rss"`
	Channel struct {
		Title string `xml:"title"`
		Items []struct {
			Title       string `xml:"title"`
			Link        string `xml:"link"`
			Description string `xml:"description"`
			PubDate     string `xml:"pubDate"`
		} `xml:"item"`
	} `xml:"channel"`
}

type atomXML struct {
	XMLName xml.Name `xml:"feed"`
	Title   string   `xml:"title"`
	Entries []struct {
		Title string `xml:"title"`
		Link  struct {
			Href string `xml:"href,attr"`
		} `xml:"link"`
		Summary   string `xml:"summary"`
		Content   string `xml:"content"`
		Published string `xml:"published"`
		Updated   string `xml:"updated"`
	} `xml:"entry"`
}

// ****************************************************************************
// RSSPlugin
// ****************************************************************************

// RSSPlugin implements ui.ViewPlugin and shows an aggregated list of RSS/Atom
// feed items together with an article preview panel.
type RSSPlugin struct {
	// TblFeeds is the selectable table that lists all fetched articles.
	TblFeeds *tview.Table
	// TxtArticle shows the content of the selected article.
	TxtArticle *tview.TextView
	layout     *tview.Flex
	items      []FeedItem
	feedURLs   []string
}

// NewRSSPlugin creates and wires up the RSS Reader plugin.  It registers its
// content page directly with ui.PgsEditorContent so that it is rendered
// within the standard editor frame (header / key-hints / status bar) without
// needing its own full-screen layout.
func NewRSSPlugin() *RSSPlugin {
	p := &RSSPlugin{}

	p.TblFeeds = tview.NewTable()
	p.TblFeeds.SetBorder(true)
	p.TblFeeds.SetTitle("RSS Feeds")
	p.TblFeeds.SetSelectable(true, false)

	p.TxtArticle = tview.NewTextView()
	p.TxtArticle.SetBorder(true)
	p.TxtArticle.SetTitle("Article")
	p.TxtArticle.SetDynamicColors(true)
	p.TxtArticle.SetScrollable(true)
	p.TxtArticle.SetWrap(true)

	p.layout = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(p.TblFeeds, 0, 2, true).
		AddItem(p.TxtArticle, 0, 3, false)

	ui.PgsEditorContent.AddPage(ContentPageName, p.layout, true, false)

	// Keep the shared Outline panel in sync with the highlighted article.
	p.TblFeeds.SetSelectionChangedFunc(func(row, column int) {
		p.RefreshOutline()
	})

	return p
}

// ****************************************************************************
// ViewPlugin interface implementation
// ****************************************************************************

func (p *RSSPlugin) ID() string    { return PluginID }
func (p *RSSPlugin) Title() string { return "RSS Reader" }
func (p *RSSPlugin) Icon() string  { return "📰" }

// Activate switches the application to the RSS reader content page and sets
// focus to the feed table.  It also repurposes the shared Search and Outline
// panels: Search filters the article list by title, and Outline shows the
// selected article's properties.
func (p *RSSPlugin) Activate() {
	ui.PgsApp.SwitchToPage("edit")
	ui.PgsEditorContent.SwitchToPage(ContentPageName)
	p.configureSearchPanel()
	p.RefreshOutline()
	ui.App.SetFocus(p.TblFeeds)
	ui.LblKeys.SetText(p.KeyHints())
}

// configureSearchPanel adapts the shared Find form to search articles by
// title instead of searching text/hex buffer content.
func (p *RSSPlugin) configureSearchPanel() {
	ui.SetFindPanelVisible(true)
	ui.FrmFind.SetTitle("Find Article")
	ui.TxtReplace.SetDisabled(true)
	ui.ChkToggleReplace.SetDisabled(true)
	ui.FrmFind.GetButton(2).SetDisabled(true) // Replace One
	ui.FrmFind.GetButton(3).SetDisabled(true) // Replace All
	ui.DpdSearchType.SetDisabled(true)
	ui.DpdSearchType.SetCurrentOption(0)
	ui.FrmFind.GetButton(0).SetSelectedFunc(func() { p.FindNext() })
	ui.FrmFind.GetButton(1).SetSelectedFunc(func() { p.FindPrevious() })
}

// FocusWidget returns the feed table as the primary focus target.
func (p *RSSPlugin) FocusWidget() tview.Primitive { return p.TblFeeds }

// Open fetches the feeds if they have not been loaded yet.  param is unused.
func (p *RSSPlugin) Open(_ any) error {
	if len(p.items) == 0 {
		p.Refresh()
	}
	return nil
}

// Close is a no-op; the RSS reader holds no resources that need cleanup.
func (p *RSSPlugin) Close() error { return nil }

// IsDirty always returns false because the RSS reader is read-only.
func (p *RSSPlugin) IsDirty() bool { return false }

// StatusFields returns values for the bottom status bar widgets.
func (p *RSSPlugin) StatusFields() ui.ViewStatus {
	return ui.ViewStatus{
		ReadWrite: "--",
		Cursor:    fmt.Sprintf("%d article(s)", len(p.items)),
		Dirty:     "",
		Percent:   "",
		Size:      "",
		Encoding:  "rss",
	}
}

// KeyHints returns the two-line key-hint string for the LblKeys bar.
func (p *RSSPlugin) KeyHints() string {
	return "F1=Help F2=Panel F6=Previous F7=Next F8=Settings F9=Context F10=Menu F12=Exit\n" +
		"[Enter] Read  [o] Open link  [F5] Refresh  [Ctrl+F] Find  [Ctrl+T] Close"
}

func (p *RSSPlugin) InternalCommand() string { return "!rss" }

func (p *RSSPlugin) CommandOpensPluginView() bool { return true }

func (p *RSSPlugin) ExecuteInternalCommand() error {
	// Command is handled by opening plugin view in dispatcher.
	return nil
}

func (p *RSSPlugin) ShowContextMenu(defaultMenu func()) bool {
	m := (&menu.Menu{}).New(" RSS Reader ", ui.PopupParentPage(), p.FocusWidget())
	_, hasItem := p.SelectedItem()

	m.AddItem("mnuRSSRefresh", "Refresh feeds", func(any) {
		p.Refresh()
	}, nil, true, false)
	m.AddSeparator()
	m.AddItem("mnuRSSRead", "Read article", func(any) {
		if item, ok := p.SelectedItem(); ok {
			p.ShowArticle(item)
		}
	}, nil, hasItem, false)
	m.AddItem("mnuRSSOpen", "Open link in browser", func(any) {
		p.OpenLink()
	}, nil, hasItem, false)
	m.AddSeparator()
	m.AddItem("mnuRSSEditFeeds", "Edit feed list...", func(any) {
		edit.OpenView(p.feedsPath(), true)
	}, nil, true, false)

	ui.PgsApp.AddPage("dlgRSSReaderMenu", m.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgRSSReaderMenu")
	return true
}

// ****************************************************************************
// Public helpers used by lied.go keyboard handlers
// ****************************************************************************

// feedsPath returns the path of the user-editable feed URL list.
func (p *RSSPlugin) feedsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return conf.FILE_RSS_FEEDS
	}
	return filepath.Join(home, conf.APP_FOLDER, conf.FILE_RSS_FEEDS)
}

// loadFeedURLs reads the feed URL list, seeding it with defaults on first run.
func (p *RSSPlugin) loadFeedURLs() {
	path := p.feedsPath()
	data, err := os.ReadFile(path)
	if err != nil {
		p.feedURLs = append([]string(nil), defaultFeeds...)
		_ = os.WriteFile(path, []byte(strings.Join(p.feedURLs, "\n")+"\n"), 0600)
		return
	}

	var urls []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		urls = append(urls, line)
	}
	if len(urls) == 0 {
		urls = append([]string(nil), defaultFeeds...)
	}
	p.feedURLs = urls
}

// Refresh reloads the feed URL list and fetches every configured feed in the
// background, then repopulates the feed table.
func (p *RSSPlugin) Refresh() {
	p.loadFeedURLs()
	feedURLs := append([]string(nil), p.feedURLs...)

	ui.PleaseWait()
	ui.SetStatus("Fetching RSS feeds...")

	go func() {
		var mu sync.Mutex
		var all []FeedItem
		var errCount int
		var wg sync.WaitGroup

		for _, feedURL := range feedURLs {
			wg.Add(1)
			go func(u string) {
				defer wg.Done()
				items, err := fetchFeed(u)
				mu.Lock()
				defer mu.Unlock()
				if err != nil {
					errCount++
					return
				}
				all = append(all, items...)
			}(feedURL)
		}
		wg.Wait()

		sort.Slice(all, func(i, j int) bool {
			return all[i].PubTime.After(all[j].PubTime)
		})

		ui.App.QueueUpdateDraw(func() {
			p.items = all
			p.populateTable()
			ui.JobsDone()
			if errCount > 0 {
				ui.SetStatus(fmt.Sprintf("RSS feeds refreshed (%d article(s), %d feed error(s))", len(all), errCount))
			} else {
				ui.SetStatus(fmt.Sprintf("RSS feeds refreshed (%d article(s))", len(all)))
			}
		})
	}()
}

// populateTable rebuilds the feed table from p.items.
func (p *RSSPlugin) populateTable() {
	p.TblFeeds.Clear()

	headers := []string{"Published", "Source", "Title"}
	for col, h := range headers {
		p.TblFeeds.SetCell(0, col,
			tview.NewTableCell(h).
				SetTextColor(tcell.ColorYellow).
				SetAttributes(tcell.AttrBold).
				SetSelectable(false))
	}
	p.TblFeeds.SetFixed(1, 0)

	for i, item := range p.items {
		row := i + 1
		published := item.PubDate
		if !item.PubTime.IsZero() {
			published = item.PubTime.Format("2006-01-02 15:04")
		}
		p.TblFeeds.SetCell(row, 0, tview.NewTableCell(published).SetTextColor(tcell.ColorWhite))
		p.TblFeeds.SetCell(row, 1, tview.NewTableCell(item.Source).SetTextColor(tcell.ColorLightCyan))
		p.TblFeeds.SetCell(row, 2, tview.NewTableCell(item.Title).SetTextColor(tcell.ColorWhite))
	}

	p.TblFeeds.SetTitle(fmt.Sprintf("RSS Feeds (%d)", len(p.items)))
	if len(p.items) > 0 {
		p.TblFeeds.Select(1, 0)
	}
	p.RefreshOutline()
}

// SelectedItem returns the currently selected article, or false when no
// valid row is selected.
func (p *RSSPlugin) SelectedItem() (FeedItem, bool) {
	row, _ := p.TblFeeds.GetSelection()
	idx := row - 1
	if idx < 0 || idx >= len(p.items) {
		return FeedItem{}, false
	}
	return p.items[idx], true
}

// ShowArticle fills TxtArticle with the content of the given article.
func (p *RSSPlugin) ShowArticle(item FeedItem) {
	p.TxtArticle.Clear()
	fmt.Fprintf(p.TxtArticle, "[yellow]%s[-]\n[gray]%s — %s[-]\n\n%s\n\n[blue]%s[-]",
		item.Title, item.Source, item.PubDate, item.Description, item.Link)
	p.TxtArticle.ScrollToBeginning()
	p.TxtArticle.SetTitle(fmt.Sprintf("Article — %s", item.Source))
}

// OpenLink opens the selected article's link in the system's default browser.
func (p *RSSPlugin) OpenLink() {
	item, ok := p.SelectedItem()
	if !ok || item.Link == "" {
		ui.SetStatus("No link to open")
		return
	}
	if err := exec.Command("xdg-open", item.Link).Start(); err != nil {
		ui.SetStatus(fmt.Sprintf("Failed to open link: %v", err))
		return
	}
	ui.SetStatus("Opening " + item.Link)
}

// ****************************************************************************
// Outline panel integration
// ****************************************************************************

// RefreshOutline populates the shared Outline panel with the properties of
// the currently selected article.
func (p *RSSPlugin) RefreshOutline() {
	ui.TblOutline.Clear()
	item, ok := p.SelectedItem()
	if !ok {
		ui.TblOutline.SetTitle("Outline")
		return
	}

	fields := [][2]string{
		{"Title", item.Title},
		{"Source", item.Source},
		{"Published", item.PubDate},
		{"Link", item.Link},
	}
	for row, f := range fields {
		ui.TblOutline.SetCell(row, 0,
			tview.NewTableCell(f[0]).SetTextColor(tcell.ColorLightCyan).SetAlign(tview.AlignLeft))
		ui.TblOutline.SetCell(row, 1,
			tview.NewTableCell(f[1]).SetTextColor(tcell.ColorWhite).SetAlign(tview.AlignLeft))
	}
	ui.TblOutline.ScrollToBeginning()
	ui.TblOutline.SetTitle(fmt.Sprintf("Outline — %s", item.Source))
}

// ****************************************************************************
// Search panel integration
// ****************************************************************************

// matchesItem reports whether row's title contains the given case-insensitive
// substring.
func (p *RSSPlugin) matchesItem(row int, needle string) bool {
	cell := p.TblFeeds.GetCell(row, 2)
	if cell == nil {
		return false
	}
	return strings.Contains(strings.ToLower(cell.Text), needle)
}

// findItem selects the next (or, if backward, previous) article whose title
// contains the Find field's text, wrapping around the table.
func (p *RSSPlugin) findItem(backward bool) {
	needle := strings.ToLower(strings.TrimSpace(ui.TxtFind.GetText()))
	rowCount := p.TblFeeds.GetRowCount()
	if needle == "" || rowCount <= 1 {
		ui.FrmFind.SetTitle("Find Article")
		ui.SetStatus("Nothing to search")
		return
	}

	total := 0
	for r := 1; r < rowCount; r++ {
		if p.matchesItem(r, needle) {
			total++
		}
	}
	if total == 0 {
		ui.FrmFind.SetTitle("Find Article (0/0)")
		ui.SetStatus(fmt.Sprintf("No article matching '%s'", needle))
		return
	}

	current, _ := p.TblFeeds.GetSelection()
	if current < 1 {
		current = 1
	}
	dataRows := rowCount - 1
	step := 1
	if backward {
		step = -1
	}
	start := current - 1 // 0-based index within data rows
	for i := 1; i <= dataRows; i++ {
		idx := ((start+i*step)%dataRows + dataRows) % dataRows
		r := idx + 1 // back into the 1..rowCount-1 range
		if p.matchesItem(r, needle) {
			p.TblFeeds.Select(r, 0)
			p.RefreshOutline()
			ui.FrmFind.SetTitle(fmt.Sprintf("Find Article (%d/%d)", r, total))
			ui.SetStatus(fmt.Sprintf("Found '%s' at %s", needle, p.TblFeeds.GetCell(r, 2).Text))
			return
		}
	}
}

// FindNext selects the next article matching the shared Find field.
func (p *RSSPlugin) FindNext() { p.findItem(false) }

// FindPrevious selects the previous article matching the shared Find field.
func (p *RSSPlugin) FindPrevious() { p.findItem(true) }

// ****************************************************************************
// Feed fetching & parsing
// ****************************************************************************

// fetchFeed downloads and parses a single RSS or Atom feed URL.
func fetchFeed(feedURL string) ([]FeedItem, error) {
	client := &http.Client{Timeout: fetchTimeout}
	resp, err := client.Get(feedURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return parseFeed(data, feedURL)
}

// parseFeed recognises RSS 2.0 and Atom formats and returns their items in a
// common FeedItem shape.
func parseFeed(data []byte, source string) ([]FeedItem, error) {
	var rf rssXML
	if err := xml.Unmarshal(data, &rf); err == nil && len(rf.Channel.Items) > 0 {
		name := rf.Channel.Title
		if name == "" {
			name = source
		}
		items := make([]FeedItem, 0, len(rf.Channel.Items))
		for _, it := range rf.Channel.Items {
			items = append(items, FeedItem{
				Title:       strings.TrimSpace(it.Title),
				Link:        strings.TrimSpace(it.Link),
				Description: cleanHTML(it.Description),
				PubDate:     strings.TrimSpace(it.PubDate),
				PubTime:     parseFeedDate(it.PubDate),
				Source:      name,
			})
		}
		return items, nil
	}

	var af atomXML
	if err := xml.Unmarshal(data, &af); err == nil && len(af.Entries) > 0 {
		name := af.Title
		if name == "" {
			name = source
		}
		items := make([]FeedItem, 0, len(af.Entries))
		for _, e := range af.Entries {
			date := e.Published
			if date == "" {
				date = e.Updated
			}
			desc := e.Summary
			if desc == "" {
				desc = e.Content
			}
			items = append(items, FeedItem{
				Title:       strings.TrimSpace(e.Title),
				Link:        strings.TrimSpace(e.Link.Href),
				Description: cleanHTML(desc),
				PubDate:     strings.TrimSpace(date),
				PubTime:     parseFeedDate(date),
				Source:      name,
			})
		}
		return items, nil
	}

	return nil, fmt.Errorf("unrecognized feed format at %s", source)
}

// parseFeedDate tries the date layouts commonly found in RSS and Atom feeds.
func parseFeedDate(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}
	layouts := []string{
		time.RFC1123Z,
		time.RFC1123,
		time.RFC3339,
		"Mon, 2 Jan 2006 15:04:05 -0700",
		"2006-01-02T15:04:05Z",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

// cleanHTML strips HTML tags and unescapes entities for plain-text display.
func cleanHTML(s string) string {
	s = tagRe.ReplaceAllString(s, "")
	return strings.TrimSpace(html.UnescapeString(s))
}
