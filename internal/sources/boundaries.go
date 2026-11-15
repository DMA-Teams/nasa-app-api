package sources

import (
	"context"
	"fmt"

	"github.com/dma-teams/nasa-app-api/internal/geo"
)

const geoBoundariesMetadataURL = "https://www.geoboundaries.org/api/current/gbOpen/HND/"

type BoundariesClient struct {
	client      *Client
	metadataURL string
}

func NewBoundariesClient(c *Client) *BoundariesClient {
	return &BoundariesClient{client: c, metadataURL: geoBoundariesMetadataURL}
}

func (b *BoundariesClient) Fetch(ctx context.Context, level string) (geo.FeatureCollection, error) {
	var meta struct {
		Simplified string `json:"simplifiedGeometryGeoJSON"`
	}

	if err := b.client.GetJSON(ctx, b.metadataURL+level+"/", &meta); err != nil {
		return geo.FeatureCollection{}, err
	}

	if meta.Simplified == "" {
		return geo.FeatureCollection{}, fmt.Errorf("geoBoundaries %s without simplified GeoJSON: %w", level, ErrUnexpectedPayload)
	}

	body, err := b.client.Get(ctx, meta.Simplified)
	if err != nil {
		return geo.FeatureCollection{}, err
	}

	return geo.ParseFeatureCollection(body)
}
