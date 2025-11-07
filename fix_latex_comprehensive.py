#!/usr/bin/env python3
import re
import sys

def fix_latex_file(filepath):
    with open(filepath, 'r') as f:
        content = f.read()
    
    # First, fix literal \n (backslash-n) to actual newlines
    # But preserve \newline, \newpage, \newtcolorbox, etc.
    content = re.sub(r'\\n([^ew])', r'\n\1', content)
    content = content.replace('}\\n', '}\n')
    content = content.replace('{\\n', '{\n')
    content = content.replace('\\n\n', '\n')
    
    # Fix unclosed \textbf{, \textit{, etc. - comprehensive approach
    # Look for \textbf{...  that reaches end of line without closing }
    for cmd in ['textbf', 'textit', 'texttt', 'emph']:
        # Pattern 1: \textbf{text: continues without } on same line
        # Match until we hit a newline, then add } before the newline
        pattern1 = r'(\\' + cmd + r'\{[^}]+)$'
        lines = content.split('\n')
        fixed_lines = []
        for line in lines:
            match = re.search(pattern1, line)
            if match:
                # Check if this line really needs a closing brace
                # Count opening { after \textbf and closing }
                cmd_pos = line.find('\\' + cmd + '{')
                if cmd_pos != -1:
                    substr = line[cmd_pos:]
                    open_braces = substr.count('{')
                    close_braces = substr.count('}')
                    if open_braces > close_braces:
                        # Missing closing brace - add it at a logical point
                        # Try to add it after punctuation if present
                        if re.search(r'[.:;!?,]', substr):
                            line = re.sub(r'(\\' + cmd + r'\{[^}]*[.:;!?,])([^}])', r'\1} \2', line)
                        else:
                            # Add at end of the \textbf content (before next word)
                            line = re.sub(r'(\\' + cmd + r'\{[^}]+)\s+([a-zA-Z])', r'\1} \2', line)
            fixed_lines.append(line)
        content = '\n'.join(fixed_lines)
    
    # Fix cases like \textbf{96.} 57\%} -> \textbf{96.57\%}
    # This happens when a number is split incorrectly
    content = re.sub(r'\\textbf\{(\d+)\.\}\s+(\d+)(\\%\})', r'\\textbf{\1.\2\3', content)
    content = re.sub(r'\\textbf\{(\d+)\.\}\s+(\d+)\}', r'\\textbf{\1.\2}', content)
    
    # Fix extra closing braces like: \textbf{word} extra:}
    content = re.sub(r'(\\text(?:bf|it|tt)\{[^}]+\}[^}]+)}', r'\1', content)
    
    # Save the fixed content
    with open(filepath, 'w') as f:
        f.write(content)
    
    print(f"Fixed {filepath}")

if __name__ == '__main__':
    if len(sys.argv) < 2:
        print("Usage: fix_latex_comprehensive.py <latex_file>")
        sys.exit(1)
    fix_latex_file(sys.argv[1])
