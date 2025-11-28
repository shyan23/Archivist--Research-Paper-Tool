# Archivist - New Features Summary

## 🎉 What's New

### 1. **Interactive API Key Management**
- **Automatic Prompt**: If API key is not found, you'll get a beautiful dialog
- **Link Provided**: Direct link to get your key: https://aistudio.google.com/app/apikey
- **Auto-Save**: Option to save to `.env` file automatically
- **No More Errors**: No more cryptic "API key not found" errors!

### 2. **Flexible Directory Configuration**

#### Method 1: Command-Line Flags ⚡
```bash
# Quick one-time override
./archivist process --input-dir ~/my-papers --output-dir ~/my-reports

# Process specific folder
./archivist process --input-dir /data/research
```

#### Method 2: Beautiful TUI File Browser 🎨
```bash
./archivist run
```
Then:
1. Navigate to **⚙️  Settings**
2. Select **📁 Directory Settings**
3. Choose **📥 Browse Input Directory** or **📤 Browse Output Directory**
4. Use the visual file browser:
   - **↑/↓** or **j/k**: Navigate
   - **Enter**: Open folders
   - **S**: Select current directory
   - **H**: Toggle hidden files
   - **G**: Go to home directory
   - **R**: Refresh
   - **ESC**: Cancel

#### Method 3: Configuration Wizard 🧙
```bash
./archivist configure
```
Step-by-step setup including API key and directories.

## 🎨 Visual File Browser Features

### Beautiful Interface
```
╭─────────────────────────────────────────────────────────────╮
│       📥 Select Input Directory (Papers Source)              │
╰─────────────────────────────────────────────────────────────╯

Current Path: /home/user/Desktop/Code/Archivist

╭─────────────────────────────────────────────────────────────╮
│ ▶ 📁 cmd                                                     │
│   📁 config                                                  │
│   📁 docs                                                    │
│   📁 internal                                                │
│   📁 lib                                                     │
│   📁 reports                                                 │
│   📄 go.mod (file - not selectable)                        │
│   📄 go.sum (file - not selectable)                        │
╰─────────────────────────────────────────────────────────────╯

⌨️  Keyboard Shortcuts:
  ↑/k: Up  ↓/j: Down  Enter: Open folder  S: Select
  H: Toggle hidden  G: Go home  R: Refresh  ESC: Cancel
```

### Smart Features
- **Directory Icons**: 📁 for folders, 📄 for files, ⬆️  for parent
- **Visual Selection**: Highlighted current selection with pink background
- **Status Indicators**: ✅ for existing directories, ⚠️  for missing ones
- **Auto-Creation**: "Create Missing Directories" option
- **Smart Navigation**: Automatically scrolls to keep selection visible
- **Hidden Files Toggle**: Show/hide dot files with **H** key
- **Quick Home**: Jump to home directory with **G** key

## 📋 Settings Menu

### Current Display
The settings menu now shows:
```
📁 Directory Settings
📥 Input: ...Desktop/Code/Archivist/lib | 📤 Output: ...ivist/reports
```
(Long paths are smartly truncated for readability)

### Directory Settings Screen
```
📥 Browse Input Directory
✅ /home/user/Desktop/Code/Archivist/lib (Where papers are read from)

📤 Browse Output Directory
✅ /home/user/Desktop/Code/Archivist/reports (Where reports are saved)

✨ Create Missing Directories
Create directories if they don't exist

🔙 Back
Return to settings menu
```

## 🚀 Quick Start Guide

### First Time Setup
```bash
# Step 1: Configure everything
./archivist configure

# Step 2: Launch TUI
./archivist run

# Step 3: Adjust directories if needed
# Navigate to Settings → Directory Settings
```

### Daily Usage
```bash
# Option A: Use TUI (recommended)
./archivist run

# Option B: Command line with custom directories
./archivist process --input-dir ~/research/papers --output-dir ~/research/reports

# Option C: Use configured defaults
./archivist process
```

## 💡 Pro Tips

1. **Organize by Project**: Use different directories for different research projects
   ```bash
   ./archivist process --input-dir ~/ml-project/papers --output-dir ~/ml-project/reports
   ```

2. **Batch Processing**: Point to a folder with many papers
   ```bash
   ./archivist process --input-dir /data/downloaded-papers
   ```

3. **Network Drives**: Works with mounted network drives too!
   ```bash
   ./archivist process --input-dir /mnt/nas/research-papers
   ```

4. **Keyboard Navigation**: Master these shortcuts in file browser:
   - **j/k** or **↑/↓**: Navigate (vim-style or arrow keys)
   - **Enter**: Go into folder
   - **S**: Select this directory (**S** for Select!)
   - **ESC**: Cancel and go back
   - **H**: Show hidden folders (like .config, .local)
   - **G**: Jump to home directory instantly

5. **Visual Feedback**: Look for status indicators:
   - ✅ = Directory exists and ready
   - ⚠️  = Directory missing (use "Create Missing Directories")

## 🛠️ Technical Details

### Files Modified/Created
1. **internal/app/config.go** - API key prompt logic
2. **internal/wizard/config_wizard.go** - Wizard API key step
3. **cmd/main/commands/process.go** - Directory flags
4. **internal/tui/types.go** - File browser types
5. **internal/tui/settings.go** - Settings menu
6. **internal/tui/filebrowser.go** - Visual file browser (NEW!)
7. **internal/tui/handlers.go** - Settings handlers
8. **internal/tui/views.go** - Settings views
9. **internal/tui/model.go** - File browser integration

### Features Implemented
- ✅ Interactive API key prompt with auto-save
- ✅ Configuration wizard API key step
- ✅ Command-line directory flags
- ✅ Beautiful visual file browser
- ✅ Directory navigation with keyboard shortcuts
- ✅ Smart path truncation in displays
- ✅ Status indicators (existing/missing)
- ✅ Auto-create missing directories option
- ✅ Hidden files toggle
- ✅ Quick home directory jump
- ✅ Real-time directory browsing

## 🎯 User Experience Improvements

### Before
- Had to manually edit config.yaml
- Type absolute paths (error-prone)
- No visual feedback
- Unclear if directories exist

### After
- Beautiful visual file browser
- Navigate with keyboard like a pro
- See exactly what you're selecting
- Clear status indicators
- Multiple ways to configure (CLI, TUI, Wizard)
- No more typing long paths!

## 🎨 Color Scheme
- **Purple** (#7D56F4): Primary color (headers, borders)
- **Pink** (#FF06B7): Selection highlight
- **Green** (#04B575): Success/confirmation
- **Orange** (#FFA500): Warnings
- **White** (#FAFAFA): Primary text
- **Gray** (#626262): Secondary text/hints

## 📝 Summary

You now have **three flexible ways** to configure directories:
1. **Quick CLI flags**: For one-off changes
2. **Beautiful TUI browser**: For visual selection
3. **Configuration wizard**: For permanent setup

Plus, the API key management is now completely automated with helpful prompts!

Enjoy your enhanced Archivist experience! 🎉
