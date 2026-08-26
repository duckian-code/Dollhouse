package assetcatalog

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type fakeS3 struct {
	body string
	err  error
}

func (f *fakeS3) GetObject(_ context.Context, _ *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &s3.GetObjectOutput{Body: io.NopCloser(strings.NewReader(f.body))}, nil
}

type presignCall struct {
	key, version string
	expires      time.Duration
}

type fakePresigner struct {
	calls []presignCall
	err   error
}

func (f *fakePresigner) PresignGetObject(_ context.Context, input *s3.GetObjectInput, optionFns ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error) {
	if f.err != nil {
		return nil, f.err
	}
	options := s3.PresignOptions{}
	for _, option := range optionFns {
		option(&options)
	}
	f.calls = append(f.calls, presignCall{key: *input.Key, version: *input.VersionId, expires: options.Expires})
	return &v4.PresignedHTTPRequest{URL: "https://assets.example.test/" + *input.Key}, nil
}

const validManifest = `{
  "catalogVersion":"v1",
  "assets":[
    {"assetId":"body-00","category":"body","key":"body-00.png","versionId":"version-1","contentType":"image/png","width":64,"height":64},
    {"assetId":"hair-00","category":"hair","key":"hair-00.png","versionId":"version-2","contentType":"image/png","width":64,"height":64}
  ]
}`

func TestGetCatalogSignsEveryImmutableAssetVersion(t *testing.T) {
	presigner := &fakePresigner{}
	service := NewService(&fakeS3{body: validManifest}, presigner, "assets", "catalog/v1.json", 15*time.Minute)
	service.now = func() time.Time { return time.Date(2026, 8, 14, 12, 0, 0, 0, time.FixedZone("test", -6*60*60)) }

	catalog, err := service.GetCatalog(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if catalog.CatalogVersion != "v1" || catalog.ExpiresAt != "2026-08-14T18:15:00Z" || len(catalog.Assets) != 2 {
		t.Fatalf("catalog = %#v", catalog)
	}
	if catalog.Assets[0].AssetID != "body-00" || catalog.Assets[0].URL != "https://assets.example.test/body-00.png" {
		t.Fatalf("first asset = %#v", catalog.Assets[0])
	}
	if len(presigner.calls) != 2 || presigner.calls[0] != (presignCall{key: "body-00.png", version: "version-1", expires: 15 * time.Minute}) {
		t.Fatalf("presign calls = %#v", presigner.calls)
	}
}

func TestGetCatalogRejectsInvalidManifest(t *testing.T) {
	tests := []string{
		`{"catalogVersion":"","assets":[]}`,
		`{"catalogVersion":"v1","assets":[{"assetId":"body-00","category":"body","key":"../body.png","versionId":"v1","contentType":"image/png","width":64,"height":64}]}`,
		`{"catalogVersion":"v1","assets":[{"assetId":"body-00","category":"body","key":"body.png","versionId":"","contentType":"image/png","width":64,"height":64}]}`,
		`{"catalogVersion":"v1","assets":[{"assetId":"body-00","category":"room","key":"body.png","versionId":"v1","contentType":"image/png","width":64,"height":64}]}`,
	}
	for _, body := range tests {
		service := NewService(&fakeS3{body: body}, &fakePresigner{}, "assets", "catalog/v1.json", time.Minute)
		if _, err := service.GetCatalog(context.Background()); err == nil {
			t.Errorf("manifest should be rejected: %s", body)
		}
	}
}

func TestGetCatalogReturnsStorageAndSigningErrors(t *testing.T) {
	service := NewService(&fakeS3{err: errors.New("S3 unavailable")}, &fakePresigner{}, "assets", "catalog/v1.json", time.Minute)
	if _, err := service.GetCatalog(context.Background()); err == nil {
		t.Fatal("expected storage error")
	}
	service = NewService(&fakeS3{body: validManifest}, &fakePresigner{err: errors.New("signing failed")}, "assets", "catalog/v1.json", time.Minute)
	if _, err := service.GetCatalog(context.Background()); err == nil {
		t.Fatal("expected signing error")
	}
}
