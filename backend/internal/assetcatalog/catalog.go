// Package assetcatalog loads the approved asset manifest and returns short-lived
// retrieval URLs for authenticated Dollhouse clients.
package assetcatalog

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const maxCatalogBytes = 1024 * 1024

type s3API interface {
	GetObject(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}

type presignAPI interface {
	PresignGetObject(context.Context, *s3.GetObjectInput, ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error)
}

// Manifest is the private, versioned catalog stored in S3.
type Manifest struct {
	CatalogVersion string          `json:"catalogVersion"`
	Assets         []ManifestAsset `json:"assets"`
}

// ManifestAsset maps an opaque public asset ID to one immutable S3 version.
type ManifestAsset struct {
	AssetID     string `json:"assetId"`
	Category    string `json:"category"`
	Key         string `json:"key"`
	VersionID   string `json:"versionId"`
	ContentType string `json:"contentType"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
}

// Asset is the frontend-facing catalog entry. S3 keys and version IDs remain
// private implementation details carried by the signed URL.
type Asset struct {
	AssetID     string `json:"assetId"`
	Category    string `json:"category"`
	URL         string `json:"url"`
	ContentType string `json:"contentType"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
}

// Catalog is returned by GET /assets/catalog.
type Catalog struct {
	CatalogVersion string  `json:"catalogVersion"`
	ExpiresAt      string  `json:"expiresAt"`
	Assets         []Asset `json:"assets"`
}

// Service reads the approved manifest and signs retrieval URLs with the
// Lambda's least-privilege S3 read credentials.
type Service struct {
	client     s3API
	presigner  presignAPI
	bucketName string
	catalogKey string
	urlTTL     time.Duration
	now        func() time.Time
}

// NewService creates an asset catalog service.
func NewService(client s3API, presigner presignAPI, bucketName, catalogKey string, urlTTL time.Duration) *Service {
	return &Service{
		client: client, presigner: presigner, bucketName: bucketName,
		catalogKey: catalogKey, urlTTL: urlTTL, now: time.Now,
	}
}

// GetCatalog returns the current approved catalog with version-specific URLs.
func (s *Service) GetCatalog(ctx context.Context) (Catalog, error) {
	if strings.TrimSpace(s.bucketName) == "" {
		return Catalog{}, errors.New("ASSETS_BUCKET_NAME is not configured")
	}
	if strings.TrimSpace(s.catalogKey) == "" {
		return Catalog{}, errors.New("ASSET_CATALOG_KEY is not configured")
	}
	if s.urlTTL <= 0 {
		return Catalog{}, errors.New("asset URL TTL must be positive")
	}

	manifest, err := s.loadManifest(ctx)
	if err != nil {
		return Catalog{}, err
	}

	expiresAt := s.now().UTC().Add(s.urlTTL).Truncate(time.Second)
	assets := make([]Asset, 0, len(manifest.Assets))
	for _, item := range manifest.Assets {
		request, err := s.presigner.PresignGetObject(ctx, &s3.GetObjectInput{
			Bucket:    &s.bucketName,
			Key:       &item.Key,
			VersionId: &item.VersionID,
		}, func(options *s3.PresignOptions) {
			options.Expires = s.urlTTL
		})
		if err != nil {
			return Catalog{}, fmt.Errorf("sign asset %q: %w", item.AssetID, err)
		}
		assets = append(assets, Asset{
			AssetID: item.AssetID, Category: item.Category, URL: request.URL,
			ContentType: item.ContentType, Width: item.Width, Height: item.Height,
		})
	}

	return Catalog{
		CatalogVersion: manifest.CatalogVersion,
		ExpiresAt:      expiresAt.Format(time.RFC3339),
		Assets:         assets,
	}, nil
}

func (s *Service) loadManifest(ctx context.Context) (Manifest, error) {
	output, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &s.bucketName,
		Key:    &s.catalogKey,
	})
	if err != nil {
		return Manifest{}, fmt.Errorf("read asset catalog: %w", err)
	}
	defer output.Body.Close()

	encoded, err := io.ReadAll(io.LimitReader(output.Body, maxCatalogBytes+1))
	if err != nil {
		return Manifest{}, fmt.Errorf("read asset catalog body: %w", err)
	}
	if len(encoded) > maxCatalogBytes {
		return Manifest{}, errors.New("asset catalog exceeds size limit")
	}

	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	var manifest Manifest
	if err := decoder.Decode(&manifest); err != nil {
		return Manifest{}, fmt.Errorf("decode asset catalog: %w", err)
	}
	if err := ensureEOF(decoder); err != nil {
		return Manifest{}, err
	}
	if err := validateManifest(manifest); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func validateManifest(manifest Manifest) error {
	if strings.TrimSpace(manifest.CatalogVersion) == "" {
		return errors.New("asset catalog version is required")
	}
	if len(manifest.Assets) == 0 {
		return errors.New("asset catalog must contain at least one asset")
	}

	allowedCategories := map[string]struct{}{
		"body": {}, "hair": {}, "eyes": {}, "nose": {}, "mouth": {}, "clothing": {},
	}
	seenIDs := make(map[string]struct{}, len(manifest.Assets))
	for index, item := range manifest.Assets {
		if strings.TrimSpace(item.AssetID) == "" {
			return fmt.Errorf("asset %d has an empty assetId", index)
		}
		if _, exists := seenIDs[item.AssetID]; exists {
			return fmt.Errorf("asset catalog contains duplicate assetId %q", item.AssetID)
		}
		seenIDs[item.AssetID] = struct{}{}
		if _, allowed := allowedCategories[item.Category]; !allowed {
			return fmt.Errorf("asset %q has unsupported category %q", item.AssetID, item.Category)
		}
		cleanKey := path.Clean(item.Key)
		if item.Key == "" || strings.HasPrefix(item.Key, "/") || cleanKey != item.Key || cleanKey == ".." || strings.HasPrefix(cleanKey, "../") {
			return fmt.Errorf("asset %q has an invalid S3 key", item.AssetID)
		}
		if strings.TrimSpace(item.VersionID) == "" {
			return fmt.Errorf("asset %q has no S3 version", item.AssetID)
		}
		if item.ContentType != "image/png" {
			return fmt.Errorf("asset %q has unsupported content type %q", item.AssetID, item.ContentType)
		}
		if item.Width <= 0 || item.Height <= 0 {
			return fmt.Errorf("asset %q has invalid dimensions", item.AssetID)
		}
	}
	return nil
}

func ensureEOF(decoder *json.Decoder) error {
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return errors.New("asset catalog must contain one JSON object")
	}
	return nil
}
