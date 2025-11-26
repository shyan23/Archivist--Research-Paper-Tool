# TUI Similar Search - Final Integration Steps

## Quick Integration Guide

You need to add 3 small code snippets to complete the integration:

### 1. Update `model.go` - Add message handler

Find the `Update()` function and add this after the `tea.WindowSizeMsg` case:

```go
case essenceExtractedMsg:
	return m.handleEssenceExtracted(msg.(essenceExtractedMsg))
```

### 2. Update `model.go` - Add screen-specific key handling

In the `tea.KeyMsg` case, add this BEFORE the search and chat input handlers:

```go
// Handle similar factors editing separately
if m.screen == screenSimilarFactorsEdit {
	return m.handleSimilarFactorsEdit(msg)
}
```

### 3. Update `model.go` - Add list updates

Find the switch statement that updates lists (around line 186-200), add these cases:

```go
case screenSearchMode:
	m.searchModeMenu, cmd = m.searchModeMenu.Update(msg)
case screenSimilarPaperSelect:
	m.similarPaperList, cmd = m.similarPaperList.Update(msg)
case screenSimilarFactorsEdit:
	// Handled separately in key handler
	cmd = nil
```

### 4. Update `views.go` - Add rendering

Find the `View()` function's switch statement, add these cases:

```go
case screenSearchMode:
	return m.renderSearchModeScreen()
case screenSimilarPaperSelect:
	return m.renderSimilarPaperSelectScreen()
case screenSimilarFactorsEdit:
	return m.renderSimilarFactorsEditScreen()
```

## That's it!

Now build and test:

```bash
go build -o rph ./cmd/main
./rph run
```

Select "Search Papers" → "Find Similar Papers" and enjoy!
