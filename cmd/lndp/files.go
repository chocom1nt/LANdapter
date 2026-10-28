package main

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/mattn/go-runewidth"
)

type localEntry struct {
	Name    string
	Path    string
	IsDir   bool
	Size    int64
	ModTime time.Time
}

type localEntriesLoadedMsg struct {
	dir     string
	entries []localEntry
	err     error
}

type remoteFilesLoadedMsg struct {
	files []packageRecord
	err   error
}

type uploadFinishedMsg struct {
	uploaded      []packageRecord
	uploadedPaths []string
	err           error
}

type installFinishedMsg struct {
	jobID string
	err   error
}

type deleteFinishedMsg struct {
	deleted int
	err     error
}

func filepathClean(path string) string { return filepath.Clean(path) }

func loadLocalDirectoryCmd(dir string) tea.Cmd {
	return func() tea.Msg {
		entries, err := readLocalEntries(dir)
		return localEntriesLoadedMsg{dir: dir, entries: entries, err: err}
	}
}

func readLocalEntries(dir string) ([]localEntry, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	abs = filepath.Clean(abs)
	items, err := os.ReadDir(abs)
	if err != nil {
		return nil, fmt.Errorf("read local directory %s: %w", abs, err)
	}
	entries := make([]localEntry, 0, len(items)+1)
	parent := filepath.Dir(abs)
	if parent != abs {
		entries = append(entries, localEntry{Name: "..", Path: parent, IsDir: true})
	}
	// Directories first, then files; os.ReadDir already sorts by name.
	dirs, files := make([]localEntry, 0), make([]localEntry, 0)
	for _, item := range items {
		info, infoErr := item.Info()
		if infoErr != nil {
			continue
		}
		e := localEntry{Name: item.Name(), Path: filepath.Join(abs, item.Name()), IsDir: item.IsDir(), Size: info.Size(), ModTime: info.ModTime()}
		if e.IsDir {
			dirs = append(dirs, e)
		} else {
			files = append(files, e)
		}
	}
	sort.SliceStable(dirs, func(i, j int) bool { return strings.ToLower(dirs[i].Name) < strings.ToLower(dirs[j].Name) })
	sort.SliceStable(files, func(i, j int) bool { return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name) })
	entries = append(entries, dirs...)
	entries = append(entries, files...)
	return entries, nil
}

func loadRemoteFilesCmd() tea.Msg {
	body, err := apiRequest(http.MethodGet, "/files", nil, "", 20*time.Second)
	if err != nil {
		return remoteFilesLoadedMsg{err: fmt.Errorf("list Master packages: %w", err)}
	}
	var files []packageRecord
	if err := json.Unmarshal(body, &files); err != nil {
		return remoteFilesLoadedMsg{err: fmt.Errorf("decode package list: %w", err)}
	}
	if files == nil {
		files = []packageRecord{}
	}
	sort.SliceStable(files, func(i, j int) bool { return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name) })
	return remoteFilesLoadedMsg{files: files}
}

func uploadFilesCmd(paths []string) tea.Cmd {
	paths = append([]string(nil), paths...)
	return func() tea.Msg {
		uploaded := make([]packageRecord, 0, len(paths))
		uploadedPaths := make([]string, 0, len(paths))
		for _, path := range paths {
			pkg, err := uploadOneFile(path)
			if err != nil {
				return uploadFinishedMsg{uploaded: uploaded, uploadedPaths: uploadedPaths, err: fmt.Errorf("%s: %w", filepath.Base(path), err)}
			}
			uploaded = append(uploaded, pkg)
			uploadedPaths = append(uploadedPaths, path)
		}
		return uploadFinishedMsg{uploaded: uploaded, uploadedPaths: uploadedPaths}
	}
}

// uploadOneFile streams the file through multipart instead of buffering the
// whole installer into memory. Master limits uploads to 100 MiB.
func uploadOneFile(path string) (packageRecord, error) {
	base, token, err := readAPIConfig()
	if err != nil {
		return packageRecord{}, err
	}
	file, err := os.Open(path)
	if err != nil {
		return packageRecord{}, fmt.Errorf("open local file: %w", err)
	}
	defer file.Close()
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)
	request, err := http.NewRequest(http.MethodPost, base+"/upload", pr)
	if err != nil {
		_ = pr.Close()
		_ = pw.Close()
		return packageRecord{}, err
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", writer.FormDataContentType())
	go func() {
		part, createErr := writer.CreateFormFile("file", filepath.Base(path))
		if createErr != nil {
			_ = pw.CloseWithError(createErr)
			return
		}
		if _, copyErr := io.Copy(part, file); copyErr != nil {
			_ = pw.CloseWithError(copyErr)
			return
		}
		if closeErr := writer.Close(); closeErr != nil {
			_ = pw.CloseWithError(closeErr)
			return
		}
		_ = pw.Close()
	}()
	client := &http.Client{Timeout: 3 * time.Minute, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Do(request)
	if err != nil {
		_ = pr.Close()
		return packageRecord{}, fmt.Errorf("upload request failed: %w", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return packageRecord{}, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return packageRecord{}, fmt.Errorf("Master returned HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}
	var result struct {
		FileID string `json:"file_id"`
		Name   string `json:"name"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return packageRecord{}, fmt.Errorf("decode upload response: %w", err)
	}
	if result.FileID == "" {
		return packageRecord{}, fmt.Errorf("Master did not return a package ID")
	}
	if result.Name == "" {
		result.Name = filepath.Base(path)
	}
	info, _ := os.Stat(path)
	pkg := packageRecord{ID: result.FileID, Name: result.Name, Type: filepath.Ext(result.Name), UploadedAt: time.Now().Format(time.RFC3339)}
	if info != nil {
		pkg.Size = info.Size()
	}
	return pkg, nil
}

func installPackagesCmd(fileIDs, clientIDs []string, mode string) tea.Cmd {
	fileIDs = append([]string(nil), fileIDs...)
	clientIDs = append([]string(nil), clientIDs...)
	return func() tea.Msg {
		payload, err := json.Marshal(map[string]interface{}{"file_ids": fileIDs, "client_ids": clientIDs, "mode": mode})
		if err != nil {
			return installFinishedMsg{err: err}
		}
		body, err := apiRequest(http.MethodPost, "/install", payload, "application/json", 30*time.Second)
		if err != nil {
			return installFinishedMsg{err: err}
		}
		var result struct {
			JobID string `json:"job_id"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			return installFinishedMsg{err: fmt.Errorf("decode install job response: %w", err)}
		}
		return installFinishedMsg{jobID: result.JobID}
	}
}

func deletePackagesCmd(ids []string) tea.Cmd {
	ids = append([]string(nil), ids...)
	return func() tea.Msg {
		deleted := 0
		for _, id := range ids {
			_, err := apiRequest(http.MethodDelete, "/files/"+id, nil, "", 20*time.Second)
			if err != nil {
				return deleteFinishedMsg{deleted: deleted, err: err}
			}
			deleted++
		}
		return deleteFinishedMsg{deleted: deleted}
	}
}

func (m *model) handleFilesKey(key string) tea.Cmd {
	switch key {
	case "up", "k":
		if m.filePane == "local" {
			if m.localCursor > 0 {
				m.localCursor--
			}
		} else if m.remoteCursor > 0 {
			m.remoteCursor--
		}
	case "down", "j":
		if m.filePane == "local" {
			if m.localCursor < len(m.localEntries)-1 {
				m.localCursor++
			}
		} else if m.remoteCursor < len(m.remoteFiles)-1 {
			m.remoteCursor++
		}
	case "left", "h":
		m.filePane = "local"
	case "right", "l":
		m.filePane = "remote"
	case "backspace":
		m.localDir = filepath.Dir(m.localDir)
		m.localCursor = 0
		m.filesLoading = true
		return loadLocalDirectoryCmd(m.localDir)
	case "enter":
		if m.filePane == "local" {
			if e, ok := m.selectedLocalEntry(); ok {
				if e.IsDir {
					m.localDir = e.Path
					m.localCursor = 0
					m.filesLoading = true
					return loadLocalDirectoryCmd(m.localDir)
				}
				toggleSelection(m.selectedLocal, e.Path)
			}
		} else if f, ok := m.selectedRemoteEntry(); ok {
			toggleSelection(m.selectedRemote, f.ID)
		}
	case " ", "space":
		if m.filePane == "local" {
			if e, ok := m.selectedLocalEntry(); ok && !e.IsDir {
				toggleSelection(m.selectedLocal, e.Path)
			}
		} else if f, ok := m.selectedRemoteEntry(); ok {
			toggleSelection(m.selectedRemote, f.ID)
		}
	case "u":
		if m.uploading {
			m.setNotice("Upload already in progress", true)
			return nil
		}
		paths := m.targetLocalPaths()
		if len(paths) == 0 {
			m.setNotice("Mark local files with Space before uploading", true)
			return nil
		}
		m.uploading = true
		m.setNotice(fmt.Sprintf("Uploading %d file(s)…", len(paths)), false)
		return uploadFilesCmd(paths)
	case "i":
		if m.installing {
			m.setNotice("Install request already in progress", true)
			return nil
		}
		fileIDs := m.targetRemoteFileIDs()
		if len(fileIDs) == 0 {
			m.setNotice("Select package(s) in the Master pane first", true)
			return nil
		}
		clientIDs := m.selectedOnlineClientIDs()
		if len(clientIDs) == 0 {
			m.setNotice("No online installation targets. Mark online clients on the Clients tab or select an online client.", true)
			return nil
		}
		m.installing = true
		m.setNotice(fmt.Sprintf("Dispatching %d package(s) to %d client(s)…", len(fileIDs), len(clientIDs)), false)
		return installPackagesCmd(fileIDs, clientIDs, m.installMode)
	case "m":
		switch m.installMode {
		case "quiet":
			m.installMode = "normal"
		case "normal":
			m.installMode = "interactive"
		default:
			m.installMode = "quiet"
		}
		m.setNotice("Install mode: "+m.installMode, false)
	case "x", "delete":
		if m.filePane != "remote" {
			m.setNotice("Switch to the Master pane to delete uploaded packages", true)
			return nil
		}
		if len(m.targetRemoteFileIDs()) == 0 {
			m.setNotice("Select package(s) to delete first", true)
			return nil
		}
		m.confirmAction = "delete"
		m.setNotice("This removes uploaded packages from Master. Press y to confirm.", true)
	case "n":
		m.setNotice("", false)
	}
	return nil
}

func (m model) selectedLocalEntry() (localEntry, bool) {
	if m.localCursor < 0 || m.localCursor >= len(m.localEntries) {
		return localEntry{}, false
	}
	return m.localEntries[m.localCursor], true
}
func (m model) selectedRemoteEntry() (packageRecord, bool) {
	if m.remoteCursor < 0 || m.remoteCursor >= len(m.remoteFiles) {
		return packageRecord{}, false
	}
	return m.remoteFiles[m.remoteCursor], true
}
func (m model) targetLocalPaths() []string {
	paths := make([]string, 0, len(m.selectedLocal))
	for path, selected := range m.selectedLocal {
		if selected {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)
	return paths
}
func (m model) targetRemoteFileIDs() []string {
	ids := make([]string, 0)
	for _, f := range m.remoteFiles {
		if m.selectedRemote[f.ID] {
			ids = append(ids, f.ID)
		}
	}
	if len(ids) == 0 {
		if f, ok := m.selectedRemoteEntry(); ok && f.ID != "" {
			ids = append(ids, f.ID)
		}
	}
	return ids
}

func (m model) renderFileManager(b *strings.Builder, width int) {
	if width < 72 {
		fmt.Fprintf(b, " %s  %s\n", style("FILES", "1;36"), style("LOCAL ⇄ MASTER · "+m.installMode, "1;33"))
	} else {
		fmt.Fprintf(b, " %s  %s  %s\n", style("FAR-STYLE FILE MANAGER", "1;36"), style("LOCAL → MASTER → INSTALL", "90"), style("mode: "+m.installMode, "1;33"))
	}
	leftWidth := max(20, (width-3)/2)
	rightWidth := max(20, width-leftWidth-3)
	leftTitle := clipText(" LOCAL: "+m.localDir, leftWidth)
	rightTitle := clipText(fmt.Sprintf(" MASTER PACKAGES (%d) ", len(m.remoteFiles)), rightWidth)
	if m.filePane == "local" {
		leftTitle = style(leftTitle, "1;37;44")
	} else {
		leftTitle = style(leftTitle, "1;36")
	}
	if m.filePane == "remote" {
		rightTitle = style(rightTitle, "1;37;44")
	} else {
		rightTitle = style(rightTitle, "1;36")
	}
	fmt.Fprintf(b, "%s │ %s\n", padStyled(leftTitle, leftWidth), padStyled(rightTitle, rightWidth))
	leftHead := fmt.Sprintf("%-2s %-*s %8s", "", max(5, leftWidth-13), "NAME", "SIZE")
	rightHead := fmt.Sprintf("%-2s %-*s %8s", "", max(5, rightWidth-13), "PACKAGE", "SIZE")
	fmt.Fprintf(b, "%s │ %s\n", style(cell(leftHead, leftWidth), "90"), style(cell(rightHead, rightWidth), "90"))
	b.WriteString(style(strings.Repeat("─", leftWidth)+"─┼─"+strings.Repeat("─", rightWidth), "90") + "\n")
	rows := max(3, m.height-15)
	localStart := clamp(m.localCursor-rows+1, 0, max(0, len(m.localEntries)-rows))
	remoteStart := clamp(m.remoteCursor-rows+1, 0, max(0, len(m.remoteFiles)-rows))
	for i := 0; i < rows; i++ {
		left, right := "", ""
		localIndex := localStart + i
		remoteIndex := remoteStart + i
		if localIndex < len(m.localEntries) {
			e := m.localEntries[localIndex]
			marker := " "
			if m.selectedLocal[e.Path] {
				marker = "✓"
			}
			cursor := " "
			if localIndex == m.localCursor {
				cursor = "▸"
			}
			name := e.Name
			size := humanSize(e.Size)
			if e.IsDir {
				name = "▸ " + name
				size = "<DIR>"
			}
			name = clipText(name, max(4, leftWidth-14))
			left = fmt.Sprintf("%s%s %-*s %8s", cursor, marker, max(4, leftWidth-14), name, size)
			if localIndex == m.localCursor {
				left = style(cell(left, leftWidth), "1;30;47")
			} else if e.IsDir {
				left = style(cell(left, leftWidth), "1;34")
			} else {
				left = style(cell(left, leftWidth), fileColor(e.Name))
			}
		}
		if remoteIndex < len(m.remoteFiles) {
			f := m.remoteFiles[remoteIndex]
			marker := " "
			if m.selectedRemote[f.ID] {
				marker = "✓"
			}
			cursor := " "
			if remoteIndex == m.remoteCursor {
				cursor = "▸"
			}
			name := clipText(f.Name, max(4, rightWidth-14))
			right = fmt.Sprintf("%s%s %-*s %8s", cursor, marker, max(4, rightWidth-14), name, humanSize(f.Size))
			if remoteIndex == m.remoteCursor {
				right = style(cell(right, rightWidth), "1;30;47")
			} else {
				right = style(cell(right, rightWidth), fileColor(f.Name))
			}
		}
		fmt.Fprintf(b, "%s │ %s\n", padStyled(left, leftWidth), padStyled(right, rightWidth))
	}
	if m.filesLoading {
		b.WriteString(style("◌ Refreshing file lists…", "33") + "\n")
	}
	if m.filesErr != nil {
		b.WriteString(style("! "+clipText(m.filesErr.Error(), max(8, width-2)), "1;31") + "\n")
	}
	localCount := countSelectedClients(m.selectedLocal)
	remoteCount := countSelectedClients(m.selectedRemote)
	if width < 80 {
		fmt.Fprintf(b, "\n Local %d · Master %d · clients %d\n", localCount, remoteCount, countSelectedClients(m.selectedClients))
	} else {
		fmt.Fprintf(b, "\n Local: %d selected · Master: %d selected · upload %s · target clients %d\n", localCount, remoteCount, m.installMode, countSelectedClients(m.selectedClients))
	}
	if m.filePane == "local" {
		if e, ok := m.selectedLocalEntry(); ok {
			fmt.Fprintf(b, " Path: %s\n", clipText(e.Path, max(8, width-7)))
		}
	} else if f, ok := m.selectedRemoteEntry(); ok {
		packageLine := fmt.Sprintf(" Package: %s · %s · ID %s", f.Name, humanSize(f.Size), f.ID)
		fmt.Fprintln(b, clipText(packageLine, width))
	}
	if m.filesErr == nil && len(m.localEntries) == 0 {
		b.WriteString("The local directory is empty or unavailable.\n")
	}
	if width < 64 {
		b.WriteString("\n" + style("←/→ panes · Enter open/mark · Space select", "90") + "\n")
		b.WriteString(style("U upload · I install · M mode · X delete", "90") + "\n")
		b.WriteString(style("Backspace parent", "90") + "\n")
	} else {
		b.WriteString("\n" + style("←/→ switch pane · Enter open/toggle · Space mark · U upload · I install · M mode · X delete package · Backspace parent", "90") + "\n")
	}
}

func countSelectedClients(selected map[string]bool) int {
	n := 0
	for _, v := range selected {
		if v {
			n++
		}
	}
	return n
}
func fileColor(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".exe", ".msi", ".msix", ".appx":
		return "1;32"
	case ".zip", ".7z", ".rar", ".tar", ".gz":
		return "1;33"
	case ".inf", ".sys", ".cat", ".cab":
		return "1;35"
	case ".ps1", ".sh", ".bat", ".cmd":
		return "1;36"
	default:
		return "37"
	}
}
func humanSize(size int64) string {
	if size < 0 {
		return "—"
	}
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}
	units := []string{"KiB", "MiB", "GiB", "TiB"}
	value := float64(size)
	index := -1
	for value >= 1024 && index < len(units)-1 {
		value /= 1024
		index++
	}
	if index < 0 {
		return fmt.Sprintf("%d B", size)
	}
	return fmt.Sprintf("%.1f %s", value, units[index])
}
func clipText(value string, width int) string {
	if width <= 0 {
		return ""
	}
	value = strings.ReplaceAll(value, "\n", " ")
	if runewidth.StringWidth(value) > width {
		return runewidth.Truncate(value, width, "…")
	}
	return value
}
