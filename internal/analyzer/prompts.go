package analyzer

// AnalysisPrompt is the comprehensive prompt for paper analysis
const AnalysisPrompt = `You are analyzing an AI/ML research paper for CS students. Create a comprehensive, student-friendly LaTeX document.

═══════════════════════════════════════════════════════════
CRITICAL OUTPUT REQUIREMENTS
═══════════════════════════════════════════════════════════
1. Output ONLY valid LaTeX code - absolutely nothing else
2. Start with \documentclass[11pt,a4paper]{article}
3. End with \end{document}
4. NO markdown code blocks (no ` + "```" + `latex blocks)
5. NO explanatory text before or after the LaTeX code
6. NO comments about the document quality

═══════════════════════════════════════════════════════════
REQUIRED PACKAGES (use these)
═══════════════════════════════════════════════════════════
\usepackage[utf8]{inputenc}
\usepackage{amsmath,amssymb,amsfonts}
\usepackage{graphicx}
\usepackage{hyperref}
\usepackage{xcolor}
\usepackage{geometry}
\usepackage[breakable]{tcolorbox}
\usepackage{enumitem}

\geometry{margin=1in}

═══════════════════════════════════════════════════════════
CUSTOM ENVIRONMENTS (define these after packages)
═══════════════════════════════════════════════════════════
Create two tcolorbox environments:
1. "keyinsight" - for highlighting breakthroughs (blue theme, breakable)
2. "prerequisite" - for listing prerequisites (green theme, breakable)

═══════════════════════════════════════════════════════════
REQUIRED SECTIONS (in this order)
═══════════════════════════════════════════════════════════

SECTION 1: Executive Summary
- Write 3-5 sentences as a cohesive paragraph
- What is the paper about? Why does it matter?
- NO bullet points in this section

SECTION 2: Problem Statement
- What specific problem does this paper solve?
- Why is it important?
- What are limitations of prior approaches?
- Write as flowing paragraphs

SECTION 3: Methods Overview
- List primary techniques/architectures used
- You can use itemize, enumerate, or description environments
- Keep it concise - just names and 1-sentence descriptions

SECTION 4: Detailed Methodology
Split into subsections:

4.1: Prerequisites
- Use the "prerequisite" environment (colored box)
- List fundamental concepts students need (be specific)
- List prior papers/work to understand first
- Use description lists for clarity

4.2: Architecture and Approach
- Step-by-step breakdown of the methodology
- Write in teaching mode - clear explanations
- Use paragraphs that flow naturally
- You MAY create tables to describe architectures
- DO NOT reference external images - describe visually with text/tables instead

4.3: Mathematical Formulations
- Explain key equations with full context
- Define all variables
- Use proper LaTeX math environments (equation, align, etc.)
- Explain WHY each formula matters

4.4: Implementation Details
- Key algorithmic steps
- Design choices and rationale
- Use itemize or enumerate as appropriate

SECTION 5: The Breakthrough
- Use the "keyinsight" environment (colored box)
- Explain the novel contribution
- What's the "WOW moment"?
- Write as a cohesive paragraph (text will wrap naturally)
- NO manual line breaks or forced formatting

SECTION 6: Experimental Setup
Split into subsections:

6.1: Datasets
- List benchmark datasets used
- Brief description of each

6.2: Evaluation Metrics
- What metrics were used?
- Why do they matter?

6.3: Baselines
- What methods were compared against?

SECTION 7: Results and Improvements
Split into subsections:

7.1: Quantitative Results
- Present specific numbers
- Write in complete sentences
- Example: "The model achieved 95.2% accuracy on CIFAR-10, a 3.1% improvement over the baseline."

7.2: Qualitative Improvements
- Non-numerical benefits
- Model efficiency, applicability, etc.








═══════════════════════════════════════════════════════════
FORMATTING GUIDELINES
═══════════════════════════════════════════════════════════
✓ Write for CS students, not experts
✓ Explain technical terms when first introduced
✓ Use full paragraphs that wrap naturally
✓ Properly escape LaTeX special characters: % $ & # _ { } ~ ^
✓ Use \textbf{} for emphasis, not manual bolding
✓ Use proper LaTeX environments (itemize, enumerate, description)
✓ Ensure all environments are properly closed
✓ Add \newpage after \tableofcontents if you want
✓ You MAY create tables using tabular environment if it helps
✓ Keep text inside tcolorbox environments as flowing paragraphs

✗ Do NOT use markdown syntax
✗ Do NOT hardcode line breaks with \\ except in tables/equations
✗ Do NOT use manual spacing with \vspace unless necessary
✗ Do NOT include any non-LaTeX content
✗ DO NOT use \includegraphics or reference external image files (they don't exist)
✗ DO NOT create \label{} and \ref{} references to figures that don't exist
✗ If you want to describe an architecture/diagram, use text, tables, or ASCII art in verbatim environment

═══════════════════════════════════════════════════════════
TITLE FORMAT
═══════════════════════════════════════════════════════════
\title{[Extract Paper Title Here]: Student Guide}
\author{Generated by Research Paper Helper}
\date{\today}

═══════════════════════════════════════════════════════════
NOW GENERATE THE DOCUMENT
═══════════════════════════════════════════════════════════
Output the complete LaTeX document starting with \documentclass and ending with \end{document}.
Remember: ONLY LaTeX code, nothing else!`

// MetadataExtractionPrompt is focused on just extracting basic info
const MetadataExtractionPrompt = `Extract the title from this research paper and provide it in this exact format:

TITLE: [exact paper title]

Be precise and extract the exact title as it appears in the paper.`

// ValidationPrompt checks the generated LaTeX
const ValidationPrompt = `Review this LaTeX code for syntax errors. Check:
1. All environments are properly opened and closed (\begin{} and \end{} match)
2. All special characters are properly escaped
3. All equations are properly formatted
4. No markdown syntax mixed in

LaTeX code to validate:
%s

If there are errors, output ONLY the corrected LaTeX code. If it's valid, output: VALID`