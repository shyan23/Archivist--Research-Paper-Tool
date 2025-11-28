# Directory Settings Feature

## Overview

Archivist now includes a beautiful, interactive TUI for configuring input and output directories! You can change these directories both at startup and during runtime.

## Features

### 1. **Startup Directory Selection**
When you launch the application, you can navigate to Settings to configure directories.

### 2. **Runtime Directory Management**
Change directories any time without restarting the application!

### 3. **Beautiful TUI Interface**
- Colorful, modern interface using Bubble Tea and Lip Gloss
- Real-time path validation
- Visual feedback for changes
- Helpful tips and examples

## How to Use

### Option 1: Command-Line Flags (One-Time Override)

```bash
# Use custom input directory
./rph process --input-dir /path/to/papers

# Use custom output directory
./rph process --output-dir /path/to/reports

# Use both
./rph process --input-dir /path/to/papers --output-dir /path/to/reports
```

### Option 2: Interactive TUI (Permanent Change)

1. **Launch the TUI:**
   ```bash
   ./rph run
   ```

2. **Navigate to Settings:**
   - Use arrow keys to navigate
   - Select "⚙️  Settings" from the main menu
   - Press Enter

3. **Configure Directories:**
   - Select "📁 Directory Settings"
   - Choose what to change:
     - **📥 Change Input Directory**: Where papers are read from
     - **📤 Change Output Directory**: Where processed papers go

4. **Enter New Path:**
   - Type the new directory path
   - Supports multiple path formats:
     - Absolute: `/home/user/papers`
     - Relative: `./my-papers` or `../papers`
     - Home directory: `~/Documents/papers`
   - Press Enter to apply
   - Press ESC to cancel

5. **Save Changes:**
   - Select "💾 Save & Apply Changes"
   - Directories are created if they don't exist
   - Changes take effect immediately!

### Option 3: Configuration Wizard

```bash
./rph configure
```

Follow the prompts to set up directories permanently in config.yaml.

## Visual Preview

```
╭─────────────────────────────────────────────────────────────────────────╮
│            📥 Set Input Directory (Papers Source)                       │
╰─────────────────────────────────────────────────────────────────────────╯

╭─────────────────────────────────────────────────────────────────────────╮
│ Current: /home/user/Archivist/lib                                      │
╰─────────────────────────────────────────────────────────────────────────╯

New path:
╭─────────────────────────────────────────────────────────────────────────╮
│ ~/research/papers│                                                      │
╰─────────────────────────────────────────────────────────────────────────╯

💡 Tips:
  • Use absolute paths: /home/user/papers
  • Use relative paths: ./my-papers or ../papers
  • Use home directory: ~/Documents/papers
  • Press Enter to confirm, ESC to cancel

Enter: Apply • ESC: Cancel • Ctrl+C: Quit
```

## Keyboard Shortcuts

### Main Menu
- **Arrow Keys / j/k**: Navigate
- **Enter**: Select
- **q / Ctrl+C**: Quit

### Settings Menu
- **Arrow Keys**: Navigate options
- **Enter**: Select option
- **ESC**: Go back

### Directory Input
- **Type**: Enter path
- **Enter**: Apply changes
- **ESC**: Cancel
- **Backspace**: Delete character

## Features

### ✨ Beautiful Interface
- 🎨 Colorful, modern design
- 📦 Boxed layouts with rounded borders
- ✅ Visual feedback for successful changes
- 🎯 Clear, intuitive navigation

### 🔒 Safety Features
- ✓ Automatic directory creation
- ✓ Path validation and expansion
- ✓ Relative and absolute path support
- ✓ Home directory expansion (~/)

### ⚡ Real-time Updates
- Changes apply immediately
- No need to restart the application
- Visual confirmation of changes
- Current settings always displayed

## Examples

### Example 1: Change Input Directory
```
1. ./rph run
2. Select "⚙️  Settings"
3. Select "📁 Directory Settings"
4. Select "📥 Change Input Directory"
5. Type: ~/research/papers
6. Press Enter
7. Select "💾 Save & Apply Changes"
```

### Example 2: Change Output Directory
```
1. ./rph run
2. Navigate to Settings → Directory Settings
3. Select "📤 Change Output Directory"
4. Type: /home/user/processed-papers
5. Press Enter
6. Save changes
```

### Example 3: Quick Override (No TUI)
```bash
# Process papers from a different location
./rph process --input-dir /tmp/papers --output-dir /tmp/reports /tmp/papers
```

## Tips

1. **Use Tab Completion**: If your shell supports it, use tab to complete paths
2. **Relative Paths**: Great for project-specific folders (e.g., `./papers`)
3. **Absolute Paths**: Best for system-wide folders (e.g., `/data/research`)
4. **Home Directory**: Use `~/` for user-specific folders
5. **Spaces in Paths**: Supported! Just type normally, no quotes needed in TUI

## Troubleshooting

### Directory Not Found
- The application will create the directory automatically
- Make sure you have write permissions for the parent directory

### Path Not Accepted
- Check for typos
- Ensure the parent directory exists
- Use absolute paths if relative paths cause issues

### Changes Not Persisting
- Make sure to select "💾 Save & Apply Changes" after modifying directories
- Check config.yaml to verify changes were saved

## API Key Management

The application also includes automatic API key management:

- **First Run**: Prompted to enter your GEMINI_API_KEY
- **Interactive Prompt**: If key not found in .env
- **Auto-Save**: Option to save to .env file
- **Configuration Wizard**: Set up API key during initial configuration

Get your API key from: https://aistudio.google.com/app/apikey

## Summary

The new directory settings feature provides:
- 🎨 Beautiful, colorful TUI
- 📁 Easy directory configuration
- ⚡ Runtime changes without restart
- 🔧 Multiple configuration methods
- ✅ Safe path handling
- 🚀 Instant application

Enjoy your enhanced Archivist experience!
