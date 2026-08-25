package assetcatalog

import (
	"context"
	"errors"
	"fmt"
)

// Reference identifies one catalog asset and the category required by its
// position in a doll configuration.
type Reference struct {
	Field    string
	AssetID  string
	Category string
}

// SelectionError reports a client-supplied asset reference that is not valid
// for the current approved catalog.
type SelectionError struct {
	Message string
}

func (e *SelectionError) Error() string { return e.Message }

// IsSelectionError reports whether err represents invalid client input rather
// than a catalog storage or decoding failure.
func IsSelectionError(err error) bool {
	var target *SelectionError
	return errors.As(err, &target)
}

// Validator checks saved doll references against the private approved catalog.
type Validator struct {
	client     s3API
	bucketName string
	catalogKey string
}

// NewValidator creates an approved asset validator.
func NewValidator(client s3API, bucketName, catalogKey string) *Validator {
	return &Validator{client: client, bucketName: bucketName, catalogKey: catalogKey}
}

// Validate ensures every reference exists and belongs to the category required
// by the doll configuration field.
func (v *Validator) Validate(ctx context.Context, references []Reference) error {
	loader := &Service{client: v.client, bucketName: v.bucketName, catalogKey: v.catalogKey}
	manifest, err := loader.loadManifest(ctx)
	if err != nil {
		return err
	}

	categories := make(map[string]string, len(manifest.Assets))
	for _, asset := range manifest.Assets {
		categories[asset.AssetID] = asset.Category
	}
	for _, reference := range references {
		category, approved := categories[reference.AssetID]
		if !approved {
			return &SelectionError{Message: fmt.Sprintf("%s references unapproved asset ID %q", reference.Field, reference.AssetID)}
		}
		if category != reference.Category {
			return &SelectionError{Message: fmt.Sprintf("%s requires a %s asset, but %q is %s", reference.Field, reference.Category, reference.AssetID, category)}
		}
	}
	return nil
}
