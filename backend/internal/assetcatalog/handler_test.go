package assetcatalog

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

type fakeCatalogService struct {
	catalog Catalog
	err     error
}

func (f *fakeCatalogService) GetCatalog(context.Context) (Catalog, error) { return f.catalog, f.err }

func authenticatedRequest() events.APIGatewayV2HTTPRequest {
	request := events.APIGatewayV2HTTPRequest{}
	request.RequestContext.Authorizer = &events.APIGatewayV2HTTPRequestContextAuthorizerDescription{
		JWT: &events.APIGatewayV2HTTPRequestContextAuthorizerJWTDescription{Claims: map[string]string{"sub": "user-123"}},
	}
	return request
}

func TestHandlerReturnsCatalogEnvelope(t *testing.T) {
	service := &fakeCatalogService{catalog: Catalog{CatalogVersion: "v1", ExpiresAt: "2026-08-14T18:15:00Z", Assets: []Asset{{AssetID: "body-00"}}}}
	got, err := NewHandler(service).GetCatalog(context.Background(), authenticatedRequest())
	if err != nil {
		t.Fatal(err)
	}
	if got.StatusCode != 200 {
		t.Fatalf("response = %#v", got)
	}
	var body struct {
		Data Catalog `json:"data"`
	}
	if err := json.Unmarshal([]byte(got.Body), &body); err != nil {
		t.Fatal(err)
	}
	if body.Data.CatalogVersion != "v1" || len(body.Data.Assets) != 1 {
		t.Fatalf("body = %#v", body)
	}
}

func TestHandlerRequiresVerifiedCognitoClaims(t *testing.T) {
	got, err := NewHandler(&fakeCatalogService{}).GetCatalog(context.Background(), events.APIGatewayV2HTTPRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if got.StatusCode != 401 || !jsonErrorCode(got.Body, "unauthenticated") {
		t.Fatalf("response = %#v", got)
	}
}

func TestHandlerDoesNotExposeInternalErrors(t *testing.T) {
	got, err := NewHandler(&fakeCatalogService{err: errors.New("secret S3 detail")}).GetCatalog(context.Background(), authenticatedRequest())
	if err != nil {
		t.Fatal(err)
	}
	if got.StatusCode != 500 || !jsonErrorCode(got.Body, "internal_error") || stringsContains(got.Body, "secret S3 detail") {
		t.Fatalf("response = %#v", got)
	}
}

func stringsContains(value, part string) bool {
	for index := 0; index+len(part) <= len(value); index++ {
		if value[index:index+len(part)] == part {
			return true
		}
	}
	return false
}

func jsonErrorCode(body, want string) bool {
	var decoded struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	return json.Unmarshal([]byte(body), &decoded) == nil && decoded.Error.Code == want
}
