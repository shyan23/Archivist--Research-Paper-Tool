#!/usr/bin/env python3
import re
import sys

def fix_latex_file(filepath):
    with open(filepath, 'r') as f:
        content = f.read()
    
    # Fix unclosed \textbf{, \textit{, \texttt{ followed by ANY punctuation or newline
    # Pattern: \textbf{Word. more text -> \textbf{Word.} more text
    # Pattern: \textbf{Word: more text -> \textbf{Word:} more text
    for cmd in ['textbf', 'textit', 'texttt', 'emph']:
        # Match: \cmd{[content without }][punctuation or newline][any char that's not }]
        # This catches cases where the closing brace is missing after punctuation
        pattern = r'(\\' + cmd + r'\{[^}]*[.:;!?,])([^}])'
        content = re.sub(pattern, r'\1} \2', content)
        
        # Also match cases where there's text after without punctuation
        # \textbf{word text -> \textbf{word} text (when followed by space and lowercase)
        pattern2 = r'(\\' + cmd + r'\{[^}]+)\s+([a-z])'
        content = re.sub(pattern2, r'\1} \2', content)
    
    # Fix literal \n (backslash-n) to actual newlines
    # But preserve \newline, \newpage, \newtcolorbox, etc.
    # Pattern: \n not followed by 'e' or 'ew' (which would make it \newline, etc.)
    content = re.sub(r'\\n([^ew])', r'\n\1', content)
    # Also fix \n at end of braces
    content = content.replace('}\\n', '}\n')
    content = content.replace('{\\n', '{\n')
    content = content.replace('\\n\n', '\n')  # Fix double newlines from \n\n
    
    # Save the fixed content
    with open(filepath, 'w') as f:
        f.write(content)
    
    print(f"Fixed {filepath}")

if __name__ == '__main__':
    if len(sys.argv) < 2:
        print("Usage: fix_latex.py <latex_file>")
        sys.exit(1)
    fix_latex_file(sys.argv[1])
