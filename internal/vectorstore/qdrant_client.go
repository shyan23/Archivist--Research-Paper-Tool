package vectorstore

import (
	"context"
	"fmt"
	"log"

	qdrant "github.com/qdrant/go-client/qdrant"
)

// QdrantConfig holds Qdrant connection configuration
type QdrantConfig struct {
	Host           string
	Port           int
	GRPCPort       int
	APIKey         string
	CollectionName string
	UseGRPC        bool
	VectorSize     uint64
	Distance       string
	OnDisk         bool
}

// QdrantClient wraps the Qdrant client with helper methods
type QdrantClient struct {
	client         *qdrant.Client
	config         *QdrantConfig
	collectionName string
}

// NewQdrantClient creates a new Qdrant client
func NewQdrantClient(config *QdrantConfig) (*QdrantClient, error) {
	var client *qdrant.Client
	var err error

	// Create Qdrant client configuration
	qdrantConfig := &qdrant.Config{
		Host: config.Host,
		Port: int(config.Port),
	}

	if config.UseGRPC {
		// Use gRPC for better performance
		qdrantConfig.Port = int(config.GRPCPort)
	}

	client, err = qdrant.NewClient(qdrantConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Qdrant client: %w", err)
	}

	qc := &QdrantClient{
		client:         client,
		config:         config,
		collectionName: config.CollectionName,
	}

	// Initialize collection if it doesn't exist
	if err := qc.initializeCollection(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to initialize collection: %w", err)
	}

	log.Printf("✓ Connected to Qdrant at %s:%d (collection: %s)",
		config.Host, config.Port, config.CollectionName)

	return qc, nil
}

// initializeCollection creates the collection if it doesn't exist
func (qc *QdrantClient) initializeCollection(ctx context.Context) error {
	// Check if collection exists
	exists, err := qc.client.CollectionExists(ctx, qc.collectionName)
	if err != nil {
		return fmt.Errorf("failed to check collection existence: %w", err)
	}

	if exists {
		log.Printf("✓ Collection '%s' already exists", qc.collectionName)
		return nil
	}

	// Create collection
	distance := qdrant.Distance_Cosine
	switch qc.config.Distance {
	case "Euclid":
		distance = qdrant.Distance_Euclid
	case "Dot":
		distance = qdrant.Distance_Dot
	}

	err = qc.client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: qc.collectionName,
		VectorsConfig: &qdrant.VectorsConfig{
			Config: &qdrant.VectorsConfig_Params{
				Params: &qdrant.VectorParams{
					Size:     qc.config.VectorSize,
					Distance: distance,
					OnDisk:   &qc.config.OnDisk,
				},
			},
		},
		// Optimize for small scale (10-100 papers)
		OptimizersConfig: &qdrant.OptimizersConfigDiff{
			IndexingThreshold: qdrant.PtrOf(uint64(20000)),
		},
	})

	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}

	log.Printf("✓ Created collection '%s' with vector size %d", qc.collectionName, qc.config.VectorSize)

	// Create payload indexes for efficient filtering
	if err := qc.createPayloadIndexes(ctx); err != nil {
		return fmt.Errorf("failed to create payload indexes: %w", err)
	}

	return nil
}

// createPayloadIndexes creates indexes on frequently queried fields
func (qc *QdrantClient) createPayloadIndexes(ctx context.Context) error {
	indexes := []struct {
		field      string
		schemaType qdrant.PayloadSchemaType
	}{
		{"paper_title", qdrant.PayloadSchemaType_Keyword},
		{"paper_id", qdrant.PayloadSchemaType_Keyword},
		{"year", qdrant.PayloadSchemaType_Integer},
		{"authors", qdrant.PayloadSchemaType_Keyword},
		{"methodologies", qdrant.PayloadSchemaType_Keyword},
		{"datasets", qdrant.PayloadSchemaType_Keyword},
		{"chunk_type", qdrant.PayloadSchemaType_Keyword},
	}

	for _, idx := range indexes {
		// Convert PayloadSchemaType to FieldType enum
		var fieldType *qdrant.FieldType
		switch idx.schemaType {
		case qdrant.PayloadSchemaType_Keyword:
			fieldType = qdrant.FieldType_FieldTypeKeyword.Enum()
		case qdrant.PayloadSchemaType_Integer:
			fieldType = qdrant.FieldType_FieldTypeInteger.Enum()
		default:
			fieldType = qdrant.FieldType_FieldTypeKeyword.Enum()
		}

		_, err := qc.client.CreateFieldIndex(ctx, &qdrant.CreateFieldIndexCollection{
			CollectionName: qc.collectionName,
			FieldName:      idx.field,
			FieldType:      fieldType,
		})
		if err != nil {
			// Log but don't fail - index might already exist
			log.Printf("Warning: Could not create index on '%s': %v", idx.field, err)
		}
	}

	log.Printf("✓ Created payload indexes for efficient filtering")
	return nil
}

// Close closes the Qdrant client connection
func (qc *QdrantClient) Close() error {
	// The new Qdrant client handles connection cleanup automatically
	return qc.client.Close()
}

// UpsertPoint inserts or updates a point in the collection
func (qc *QdrantClient) UpsertPoint(ctx context.Context, point *Point) error {
	qdrantPoint := &qdrant.PointStruct{
		Id:      &qdrant.PointId{PointIdOptions: &qdrant.PointId_Uuid{Uuid: point.ID}},
		Vectors: &qdrant.Vectors{VectorsOptions: &qdrant.Vectors_Vector{Vector: &qdrant.Vector{Data: point.Vector}}},
		Payload: point.Payload,
	}

	_, err := qc.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: qc.collectionName,
		Points:         []*qdrant.PointStruct{qdrantPoint},
	})

	return err
}

// UpsertBatch inserts or updates multiple points
func (qc *QdrantClient) UpsertBatch(ctx context.Context, points []*Point) error {
	qdrantPoints := make([]*qdrant.PointStruct, len(points))

	for i, point := range points {
		qdrantPoints[i] = &qdrant.PointStruct{
			Id:      &qdrant.PointId{PointIdOptions: &qdrant.PointId_Uuid{Uuid: point.ID}},
			Vectors: &qdrant.Vectors{VectorsOptions: &qdrant.Vectors_Vector{Vector: &qdrant.Vector{Data: point.Vector}}},
			Payload: point.Payload,
		}
	}

	_, err := qc.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: qc.collectionName,
		Points:         qdrantPoints,
	})

	return err
}

// Search performs a vector similarity search
func (qc *QdrantClient) Search(ctx context.Context, query *SearchQuery) ([]*SearchResult, error) {
	// Create query parameters
	queryParams := &qdrant.QueryPoints{
		CollectionName: qc.collectionName,
		Query:          qdrant.NewQuery(query.Vector...),
		Limit:          qdrant.PtrOf(uint64(query.Limit)),
		WithPayload:    qdrant.NewWithPayload(true),
	}

	// Add score threshold if provided
	if query.ScoreThreshold > 0 {
		queryParams.ScoreThreshold = &query.ScoreThreshold
	}

	// Add filters if provided
	if query.Filter != nil {
		queryParams.Filter = query.Filter
	}

	result, err := qc.client.Query(ctx, queryParams)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	results := make([]*SearchResult, len(result))
	for i, hit := range result {
		results[i] = &SearchResult{
			ID:      hit.Id.GetUuid(),
			Score:   float64(hit.Score),
			Payload: hit.Payload,
		}
	}

	return results, nil
}

// DeleteByPaperTitle deletes all points (chunks) for a given paper
func (qc *QdrantClient) DeleteByPaperTitle(ctx context.Context, paperTitle string) error {
	filter := &qdrant.Filter{
		Must: []*qdrant.Condition{
			{
				ConditionOneOf: &qdrant.Condition_Field{
					Field: &qdrant.FieldCondition{
						Key: "paper_title",
						Match: &qdrant.Match{
							MatchValue: &qdrant.Match_Keyword{Keyword: paperTitle},
						},
					},
				},
			},
		},
	}

	_, err := qc.client.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: qc.collectionName,
		Points: &qdrant.PointsSelector{
			PointsSelectorOneOf: &qdrant.PointsSelector_Filter{Filter: filter},
		},
	})

	return err
}

// GetCollectionInfo returns information about the collection
func (qc *QdrantClient) GetCollectionInfo(ctx context.Context) (*qdrant.CollectionInfo, error) {
	return qc.client.GetCollectionInfo(ctx, qc.collectionName)
}

// ScrollPoints retrieves points with pagination
func (qc *QdrantClient) ScrollPoints(ctx context.Context, limit uint32, offset *qdrant.PointId) ([]*qdrant.RetrievedPoint, error) {
	result, err := qc.client.Scroll(ctx, &qdrant.ScrollPoints{
		CollectionName: qc.collectionName,
		Limit:          &limit,
		Offset:         offset,
		WithPayload:    qdrant.NewWithPayload(true),
	})

	if err != nil {
		return nil, err
	}

	// The Scroll method returns the points directly as a slice
	return result, nil
}
