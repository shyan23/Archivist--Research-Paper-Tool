# TUI Similar Search Integration - Implementation Guide

## What's Been Implemented

### ✅ Completed Files

1. **internal/tui/types.go** - Added new screen types and model fields
   - `screenSearchMode` - Choose between manual/similar search
   - `screenSimilarPaperSelect` - Select paper for similarity search
   - `screenSimilarFactorsEdit` - Edit extracted factors
   - Added model fields for similar search state

2. **internal/tui/similar_search.go** - Complete implementation
   - `renderSearchModeScreen()` - UI for choosing search mode
   - `renderSimilarPaperSelectScreen()` - UI for selecting a paper
   - `renderSimilarFactorsEditScreen()` - Interactive factor editor
   - `handleSearchModeSelection()` - Mode selection logic
   - `handleSimilarPaperSelection()` - Paper selection + essence extraction
   - `handleSimilarFactorsEdit()` - Factor editing with keyboard controls
   - `executeSimilarSearch()` - Execute search with edited factors

3. **internal/tui/styles.go** - Added highlight style
   - `highlightStyle` for selected factors in editor

4. **internal/tui/handlers.go** - Partial integration
   - Updated "search_papers" action to show search mode menu
   - Added handlers for new screens in `handleEnter()`

## 🚧 Remaining Integration Steps

### Step 1: Update model.go - Update() function

Add handling for new messages and screens:

```go
// In the Update() function, after tea.WindowSizeMsg handling:

case essenceExtractedMsg:
	return m.handleEssenceExtracted(msg.(essenceExtractedMsg))

// In the tea.KeyMsg section, add:
if m.screen == screenSimilarFactorsEdit {
	return m.handleSimilarFactorsEdit(msg)
}

// In the list update switch at the end:
case screenSearchMode:
	m.searchModeMenu, cmd = m.searchModeMenu.Update(msg)
case screenSimilarPaperSelect:
	m.similarPaperList, cmd = m.similarPaperList.Update(msg)
case screenSimilarFactorsEdit:
	// Handled separately above
	cmd = nil
```

### Step 2: Update views.go - View() function

Add rendering for new screens:

```go
// In the View() function's switch statement:

case screenSearchMode:
	return m.renderSearchModeScreen()
case screenSimilarPaperSelect:
	return m.renderSimilarPaperSelectScreen()
case screenSimilarFactorsEdit:
	return m.renderSimilarFactorsEditScreen()
```

### Step 3: Test the Integration

1. **Build the application**:
   ```bash
   go build -o rph ./cmd/main
   ```

2. **Start the search service**:
   ```bash
   cd services/search-engine
   source venv/bin/activate
   python run.py
   ```

3. **Run the TUI**:
   ```bash
   ./rph run
   ```

4. **Test the flow**:
   - Select "Search Papers"
   - Choose "Find Similar Papers"
   - Select a paper from your library
   - Wait for essence extraction
   - Edit/add/remove factors
   - Press Tab to search
   - View results

## User Flow

```
Main Menu
  └─> Search Papers
       └─> Search Mode Menu
            ├─> Manual Search (existing)
            │    └─> Enter query → Results
            │
            └─> Find Similar Papers (NEW!)
                 └─> Select Paper from Library
                      └─> [AI Extracts Essence]
                           └─> Edit Factors Screen
                                ├─> View extracted factors
                                ├─> Delete factors (press 'd')
                                ├─> Add new factors (type + Enter)
                                └─> Search (press Tab)
                                     └─> View Results
```

## Factor Editor Controls

| Key | Action |
|-----|--------|
| ↑/↓ | Navigate factors |
| d | Delete selected factor |
| Type + Enter | Add new factor |
| Tab | Start search with current factors |
| Esc | Go back |

## Example Factor Editing Session

**Initial Extracted Factors:**
```
1. Transformer architecture using attention mechanisms
2. self-attention
3. multi-head attention
4. positional encoding
5. layer normalization
6. residual connections
```

**User Can:**
- **Delete** irrelevant factors (e.g., "residual connections")
- **Add** specific terms (e.g., "BERT", "pre-training")
- **Refine** search focus before querying

**Final Search Query:**
```
"Transformer architecture using attention mechanisms self-attention
multi-head attention BERT pre-training"
```

## Features

### ✨ Interactive Factor Editing
- Real-time add/remove of search factors
- Visual highlighting of selected factor
- No rigid search query - user has full control

### 🎯 Smart Essence Extraction
- Automatically extracts:
  - Main methodology
  - Key technical concepts
  - Specific techniques
  - Related research areas
- Powered by Gemini AI

### 🔍 Flexible Search
- Edit AI suggestions before searching
- Add domain-specific terms
- Remove overly broad concepts
- Combine automated extraction with human expertise

## Error Handling

The implementation includes error handling for:
- Search service not running
- Paper selection with no papers in library
- Essence extraction failures
- Empty factor lists
- Search API errors

## Performance Notes

- Essence extraction takes 5-15 seconds (depends on paper length)
- Shows loading indicator during extraction
- Search is synchronous (could be made async)
- Results are cached by search service

## Future Enhancements

- [ ] Async essence extraction with progress bar
- [ ] Save/load factor profiles
- [ ] Factor importance weighting (slider for each factor)
- [ ] Visual similarity heatmap
- [ ] Batch similar search (multiple papers)
- [ ] Export factor lists for reuse

## Troubleshooting

### "Search service not running"
- Start the service: `cd services/search-engine && python run.py`
- Check port 8000 is not in use

### "No papers found in library"
- Add PDFs to the `lib/` directory (configured in config.yaml)

### "Essence extraction failed"
- Check Gemini API key is configured
- Verify PDF is readable (not corrupted)
- Check API quota/limits

### Factors not displaying correctly
- Ensure `highlightStyle` is defined in `styles.go`
- Check list initialization in `handleEssenceExtracted()`

## Code Locations

| Component | File | Lines |
|-----------|------|-------|
| Screen types | `internal/tui/types.go` | 26-29 |
| Model fields | `internal/tui/types.go` | 69-77 |
| UI rendering | `internal/tui/similar_search.go` | 33-98 |
| Factor editing | `internal/tui/similar_search.go` | 242-330 |
| Search execution | `internal/tui/similar_search.go` | 332-391 |
| Mode selection | `internal/tui/handlers.go` | 54-81 |
| Handlers | `internal/tui/handlers.go` | 174-183 |

## Testing Checklist

- [ ] Search mode menu displays correctly
- [ ] Paper selection lists library PDFs
- [ ] Essence extraction shows loading indicator
- [ ] Factors display in editable list
- [ ] Up/down navigation works
- [ ] Delete factor (d key) works
- [ ] Add factor (type + enter) works
- [ ] Tab key triggers search
- [ ] Search results display correctly
- [ ] Back navigation works at each step
- [ ] Error messages display appropriately

## Related Documentation

- [Paper Analysis Modules](./PAPER_ANALYSIS_MODULES.md) - CLI commands for similar papers
- [Similar Paper Finder](../internal/analyzer/similar_papers.go) - Core extraction logic
- [Search Client](../internal/search/client.go) - Search API integration
