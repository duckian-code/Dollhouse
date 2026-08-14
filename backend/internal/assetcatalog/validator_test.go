package assetcatalog

import (
	"context"
	"errors"
	"testing"
)

func TestValidatorAcceptsApprovedAssetsInRequiredCategories(t *testing.T) {
	validator := NewValidator(&fakeS3{body: validManifest}, "assets", "catalog/v1.json")
	err := validator.Validate(context.Background(), []Reference{
		{Field: "bodyAssetId", AssetID: "body-00", Category: "body"},
		{Field: "hairAssetId", AssetID: "hair-00", Category: "hair"},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestValidatorRejectsUnknownAndCategoryMismatchedAssets(t *testing.T) {
	validator := NewValidator(&fakeS3{body: validManifest}, "assets", "catalog/v1.json")
	tests := [][]Reference{
		{{Field: "bodyAssetId", AssetID: "body-unknown", Category: "body"}},
		{{Field: "bodyAssetId", AssetID: "hair-00", Category: "body"}},
	}
	for _, references := range tests {
		err := validator.Validate(context.Background(), references)
		if err == nil || !IsSelectionError(err) {
			t.Fatalf("Validate(%#v) error = %v, want selection error", references, err)
		}
	}
}

func TestValidatorKeepsCatalogFailuresInternal(t *testing.T) {
	validator := NewValidator(&fakeS3{err: errors.New("S3 unavailable")}, "assets", "catalog/v1.json")
	err := validator.Validate(context.Background(), []Reference{{Field: "bodyAssetId", AssetID: "body-00", Category: "body"}})
	if err == nil || IsSelectionError(err) {
		t.Fatalf("error = %v, want internal catalog error", err)
	}
}
