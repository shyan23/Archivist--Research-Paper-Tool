package citation

import (
	"archivist/internal/graph"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ManualCitationData holds manually specified citation information
type ManualCitationData struct {
	Paper     string             `yaml:"paper"`
	Citations []ManualCitation  `yaml:"citations"`
}

// ManualCitation represents a manually specified citation
type ManualCitation struct {
	TargetTitle string `yaml:"target_title"`
	TargetFile  string `yaml:"target_file,omitempty"`
	Relation    string `yaml:"relation"`          // "cites", "extends", "compares_with"
	Importance  string `yaml:"importance"`        // "high", "medium", "low"
	Context     string `yaml:"context,omitempty"` // Why it's cited
}

// LoadManualCitations reads user-provided citation metadata from YAML
func LoadManualCitations(paperTitle string, papersDir string) (*ManualCitationData, error) {
	// Sanitize paper title for filename
	filename := sanitizeForFilename(paperTitle) + "_citations.yaml"
	filepath := filepath.Join(papersDir, filename)

	// Check if file exists
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		return nil, nil // No manual citations file, not an error
	}

	// Read YAML file
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read manual citations file: %w", err)
	}

	// Parse YAML
	var manualData ManualCitationData
	if err := yaml.Unmarshal(data, &manualData); err != nil {
		return nil, fmt.Errorf("failed to parse manual citations YAML: %w", err)
	}

	return &manualData, nil
}

// MergeCitations merges automatic and manual citation data
// Manual citations override automatic ones for the same papers
func MergeCitations(auto *graph.CitationData, manual *ManualCitationData) *graph.CitationData {
	if manual == nil {
		return auto
	}

	// Create a map of manual target titles for quick lookup
	manualTargets := make(map[string]bool)
	for _, mc := range manual.Citations {
		manualTargets[strings.ToLower(mc.TargetTitle)] = true
	}

	// Filter out auto citations that have manual overrides
	filteredAuto := make([]graph.InTextCitation, 0)
	for _, citation := range auto.InTextCitations {
		// Check if this citation has a manual override by comparing reference title
		hasOverride := false
		if citation.ReferenceIndex > 0 && citation.ReferenceIndex <= len(auto.References) {
			refTitle := auto.References[citation.ReferenceIndex-1].Title
			if manualTargets[strings.ToLower(refTitle)] {
				hasOverride = true
			}
		}

		if !hasOverride {
			filteredAuto = append(filteredAuto, citation)
		}
	}

	auto.InTextCitations = filteredAuto

	// Add manual citations to the override map
	for _, mc := range manual.Citations {
		auto.ManualOverrides[mc.TargetTitle] = mc.Context
	}

	return auto
}

// CreateManualCitationsTemplate generates a template YAML file for a paper
func CreateManualCitationsTemplate(paperTitle string, papersDir string, references []graph.Reference) error {
	filename := sanitizeForFilename(paperTitle) + "_citations.yaml"
	filepath := filepath.Join(papersDir, filename)

	// Check if file already exists
	if _, err := os.Stat(filepath); err == nil {
		return fmt.Errorf("manual citations file already exists: %s", filepath)
	}

	// Create template
	template := ManualCitationData{
		Paper:     paperTitle,
		Citations: make([]ManualCitation, 0),
	}

	// Add placeholders for each reference
	for i, ref := range references {
		if i >= 5 {
			break // Limit to first 5 as examples
		}

		template.Citations = append(template.Citations, ManualCitation{
			TargetTitle: ref.Title,
			Relation:    "cites",
			Importance:  "medium",
			Context:     "TODO: Describe why this paper is cited",
		})
	}

	// Marshal to YAML
	data, err := yaml.Marshal(template)
	if err != nil {
		return fmt.Errorf("failed to marshal template: %w", err)
	}

	// Add header comment
	header := `# Manual Citation Overrides for: ` + paperTitle + `
# This file allows you to manually specify citation relationships
# that may have been missed or incorrectly extracted.
#
# Fields:
#   target_title: Title of the cited paper
#   target_file:  (Optional) Filename if you want direct mapping
#   relation:     Type of citation (cites, extends, compares_with)
#   importance:   high, medium, or low
#   context:      Brief description of why it's cited
#
# Example:
# citations:
#   - target_title: "Attention Is All You Need"
#     relation: "extends"
#     importance: "high"
#     context: "Core inspiration for our attention mechanism"
#

`
	fullContent := header + string(data)

	// Write to file
	if err := os.WriteFile(filepath, []byte(fullContent), 0644); err != nil {
		return fmt.Errorf("failed to write template file: %w", err)
	}

	fmt.Printf("✓ Created manual citations template: %s\n", filepath)
	return nil
}

// ConvertManualToGraph converts manual citations to graph relationships
func ConvertManualToGraph(sourcePaper string, manual *ManualCitationData) []graph.CitationRelationship {
	if manual == nil {
		return []graph.CitationRelationship{}
	}

	relationships := make([]graph.CitationRelationship, 0)
	for _, mc := range manual.Citations {
		relationships = append(relationships, graph.CitationRelationship{
			SourcePaper: sourcePaper,
			TargetPaper: mc.TargetTitle,
			Importance:  mc.Importance,
			Context:     mc.Context,
		})
	}

	return relationships
}

// Helper functions

func sanitizeForFilename(s string) string {
	// Remove invalid filename characters
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
		" ", "_",
	)

	sanitized := replacer.Replace(s)

	// Limit length
	if len(sanitized) > 200 {
		sanitized = sanitized[:200]
	}

	return sanitized
}

// ValidateManualCitations checks if a manual citations file is valid
func ValidateManualCitations(filepath string) error {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var manualData ManualCitationData
	if err := yaml.Unmarshal(data, &manualData); err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}

	if manualData.Paper == "" {
		return fmt.Errorf("missing 'paper' field")
	}

	for i, citation := range manualData.Citations {
		if citation.TargetTitle == "" {
			return fmt.Errorf("citation %d: missing target_title", i)
		}
		if citation.Importance != "high" && citation.Importance != "medium" && citation.Importance != "low" {
			return fmt.Errorf("citation %d: invalid importance '%s' (must be high, medium, or low)", i, citation.Importance)
		}
	}

	return nil
}
