package main

import (
	"archivist/internal/app"
	"archivist/internal/graph"
	"context"
	"fmt"
	"log"
	"os"
	"time"
)

func main() {
	fmt.Println("🔧 Initializing Archivist Knowledge Graph...")

	// Load configuration
	configPath := "config/config.yaml"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	config, err := app.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("❌ Failed to load config: %v", err)
	}

	// Check if graph is enabled
	if !config.Graph.Enabled {
		fmt.Println("⚠️  Graph feature is disabled in config")
		fmt.Println("   Set graph.enabled: true in config/config.yaml")
		os.Exit(1)
	}

	// Create graph builder
	graphConfig := &graph.GraphConfig{
		URI:      config.Graph.Neo4j.URI,
		Username: config.Graph.Neo4j.Username,
		Password: config.Graph.Neo4j.Password,
		Database: config.Graph.Neo4j.Database,
	}

	fmt.Printf("📡 Connecting to Neo4j at %s...\n", graphConfig.URI)

	builder, err := graph.NewGraphBuilder(graphConfig)
	if err != nil {
		log.Fatalf("❌ Failed to create graph builder: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		builder.Close(ctx)
	}()

	fmt.Println("✓ Connected to Neo4j")

	// Initialize schema
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Println("\n🔨 Creating constraints and indexes...")
	if err := builder.InitializeSchema(ctx); err != nil {
		log.Fatalf("❌ Failed to initialize schema: %v", err)
	}

	// Get current stats
	fmt.Println("\n📊 Current graph statistics:")
	stats, err := builder.GetStats(ctx)
	if err != nil {
		log.Printf("⚠️  Failed to get stats: %v", err)
	} else {
		fmt.Printf("   Papers: %d\n", stats.PaperCount)
		fmt.Printf("   Concepts: %d\n", stats.ConceptCount)
		fmt.Printf("   Citations: %d\n", stats.CitationCount)
		fmt.Printf("   Similarities: %d\n", stats.SimilarityCount)
	}

	fmt.Println("\n✅ Knowledge graph initialized successfully!")
	fmt.Println("\n🚀 Ready to use:")
	fmt.Println("   archivist process lib/ --enable-graph")
	fmt.Println("   archivist explore \"your query\"")
	fmt.Println("   archivist graph --show-citations")
}
