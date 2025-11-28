# 🎉 User Preferences System - Complete Implementation

## Overview

Archivist now has a **complete user preferences system** that:
- ✅ Stores your directory preferences persistently
- ✅ Prompts for setup on first run
- ✅ Allows changes anytime in Settings
- ✅ Auto-saves all changes immediately
- ✅ Works across all sessions

## 📁 Preferences Storage

Your preferences are saved to:
```
~/.config/archivist/preferences.json
```

This file contains:
```json
{
  "input_directory": "/path/to/your/papers",
  "output_directory": "/path/to/your/reports",
  "configured_once": true
}
```

## 🚀 First Time Setup

### When You First Run Archivist

When you launch Archivist for the first time:

```bash
./archivist run
# or
./archivist process
```

You'll see:

```
═══════════════════════════════════════════════════════════════
         🎓 ARCHIVIST - First Time Setup
═══════════════════════════════════════════════════════════════

Welcome to Archivist! Let's set up your directories.

📥 INPUT DIRECTORY (where your PDF papers are stored)
───────────────────────────────────────────────────────

Enter input directory path
Default: ./lib
Path (press Enter for default): ~/research/papers

📤 OUTPUT DIRECTORY (where processed reports will be saved)
───────────────────────────────────────────────────────

Enter output directory path
Default: ./reports
Path (press Enter for default): ~/research/reports

Creating directories...
✅ Created input directory: /home/user/research/papers
✅ Created output directory: /home/user/research/reports

✅ Preferences saved to: /home/user/.config/archivist/preferences.json

═══════════════════════════════════════════════════════════════
Setup complete! You can change these settings anytime from
the Settings menu in the TUI.
═══════════════════════════════════════════════════════════════
```

### Skip Setup (Use Defaults)

If you prefer to use defaults initially:

```
⚠️  No directory preferences found!

You can either:
  1. Run setup now (recommended)
  2. Use defaults (./lib and ./reports)
  3. Configure later in Settings

Run setup now? (y/n): n

Using defaults:
  📥 Input:  ./lib
  📤 Output: ./reports

💡 Tip: You can configure custom directories anytime from Settings!
```

## ⚙️ Changing Directories in Settings

### Two Ways to Change Directories

#### 1. **Visual File Browser** 🎨 (Recommended)

```bash
./archivist run
```

Navigate to:
1. **⚙️ Settings**
2. **📁 Directory Settings**
3. Select either:
   - **📥 Browse Input Directory**
   - **📤 Browse Output Directory**

Then navigate using:
- **↑/↓** or **j/k**: Move
- **Enter**: Open folder
- **S**: **Select THIS directory**
- **H**: Toggle hidden folders
- **G**: Go to home
- **ESC**: Cancel

#### 2. **Type/Paste Path** ✏️

```bash
./archivist run
```

Navigate to:
1. **⚙️ Settings**
2. **📁 Directory Settings**
3. Select either:
   - **✏️  Type Input Directory Path**
   - **✏️  Type Output Directory Path**

Then type or paste your path:
```
📥 Type Input Directory Path

Type or paste the full directory path below:

╭─────────────────────────────────────────────────╮
│ ~/Desktop/research/ai-papers│                   │
╰─────────────────────────────────────────────────╯

💡 Tips:
  • Absolute paths: /home/user/papers
  • Home directory: ~/Documents/papers
  • Relative paths: ./my-papers
  • Spaces are OK: /home/user/my research
  • Press Enter to save, ESC to cancel

Enter: Save & Create Directory • ESC: Cancel
```

**Press Enter** → Directory is created and preference is saved!

## 🔄 Real-Time Updates

**All changes are saved immediately** when you:
- Select a directory in the file browser (press **S**)
- Type a path and press Enter
- Use the "Create Missing Directories" option

Your preferences are automatically:
1. ✅ Saved to `~/.config/archivist/preferences.json`
2. ✅ Applied to the current session
3. ✅ Used in all future sessions

## 📋 Directory Settings Menu

When you navigate to **Settings → Directory Settings**, you'll see:

```
📁 Directory Configuration

> 📥 Browse Input Directory
  ✅ /home/user/research/papers (Visual folder browser)

  ✏️  Type Input Directory Path
  Manually type or paste the folder path

  📤 Browse Output Directory
  ✅ /home/user/research/reports (Visual folder browser)

  ✏️  Type Output Directory Path
  Manually type or paste the folder path

  ✨ Create Missing Directories
  Create directories if they don't exist

  🔙 Back
  Return to settings menu
```

### Status Indicators

- ✅ = Directory exists and is ready
- ⚠️  = Directory doesn't exist (will be created when you select it)

## 💡 Usage Examples

### Example 1: Organize by Project

```bash
# First project
./archivist run
# Set directories to ~/ml-project/papers and ~/ml-project/reports
# Process papers...

# Second project
./archivist run
# Change directories to ~/cv-project/papers and ~/cv-project/reports
# Process papers...
```

Each time you change directories, preferences are saved!

### Example 2: Command-Line Override

Even with saved preferences, you can override for a single run:

```bash
# Use saved preferences
./archivist process

# Override for this run only
./archivist process --input-dir /tmp/new-papers --output-dir /tmp/new-reports
```

The override doesn't change your saved preferences!

### Example 3: Network Drive

Set up network drive once:

```bash
./archivist run
# Settings → Directory Settings → Browse Input Directory
# Navigate to /mnt/nas/research-papers
# Press S to select
```

Now works forever with that network drive!

## 🎯 Priority Order

Directories are selected in this order:

1. **Command-line flags** (`--input-dir`, `--output-dir`) - Temporary override
2. **User preferences** (`~/.config/archivist/preferences.json`) - Persistent
3. **Config file** (`config/config.yaml`) - Default fallback

## 📝 Files Created

### Preferences File
```
~/.config/archivist/preferences.json
```
- Stores your directory choices
- Automatically created on first setup
- Updated every time you change directories

### Config File
```
config/config.yaml
```
- Contains default settings
- Used as fallback if no preferences exist
- Can be edited manually

## 🛠️ Technical Details

### File Locations

- **Preferences**: `~/.config/archivist/preferences.json`
- **Created automatically** when you:
  - Complete first-time setup
  - Change directories in Settings
  - Use the file browser or text input

### Auto-Creation

When you select or type a directory:
1. Path is validated and expanded (`~/` becomes `/home/user/`)
2. Converted to absolute path
3. Directory is created if it doesn't exist (`mkdir -p`)
4. Preference is saved immediately
5. Changes apply to current session

### Preference Updates

Every time you:
- Browse and select a directory (press **S**)
- Type a path and press Enter
- Use "Create Missing Directories"

The system:
1. Updates the in-memory config
2. Saves to `preferences.json`
3. Shows updated paths in Settings menu

## 🎨 Why Files Aren't Selectable in Browser

**You're selecting a FOLDER, not a file!**

The file browser is designed to select **directories** where you want to:
- **Input**: Read all your PDF papers FROM
- **Output**: Save all processed reports TO

Individual files (PDFs) are shown but grayed out because you need to pick the **containing folder**, not specific files.

### How to Select a Directory

1. Navigate to the folder you want
2. Press **S** (for Select)
3. The **current folder** you're viewing becomes your choice!

For example, if you're viewing:
```
/home/user/Desktop/
  📁 research/
  📁 documents/
  📄 paper.pdf
```

And you want to use `/home/user/Desktop/` as your input folder:
- Just press **S** right there!

If you want to use `/home/user/Desktop/research/`:
- Press Enter on "research/" folder
- Then press **S**

## 🎉 Summary

✅ **First Run**: Interactive setup or use defaults
✅ **Settings Menu**: Change directories anytime
✅ **Two Methods**: Visual browser OR type/paste
✅ **Auto-Save**: All changes saved immediately
✅ **Persistent**: Works across all sessions
✅ **Flexible**: Override with command-line flags
✅ **Smart**: Auto-creates directories
✅ **Clear**: Status indicators show what exists

**No more hardcoded ./lib and ./reports!** 🎊

Your preferences, your way! 🚀
