package authorization

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

func requestWithClaims(claims map[string]string) events.APIGatewayV2HTTPRequest {
	request := events.APIGatewayV2HTTPRequest{}
	request.RequestContext.Authorizer = &events.APIGatewayV2HTTPRequestContextAuthorizerDescription{
		JWT: &events.APIGatewayV2HTTPRequestContextAuthorizerJWTDescription{Claims: claims},
	}
	return request
}

func TestAuthenticateReadsIdentityAndCognitoGroups(t *testing.T) {
	principal, failure := Authenticate(requestWithClaims(map[string]string{
		"sub": " user-123 ", "cognito:username": "alice", "name": "Alice", "cognito:groups": `["User","Admin"]`,
	}))
	if failure != nil {
		t.Fatalf("failure = %#v", failure)
	}
	if principal.Subject != "user-123" || !principal.InGroup(UserGroup) || !principal.InGroup(AdminGroup) {
		t.Fatalf("principal = %#v", principal)
	}
}

func TestAuthenticateAcceptsAPIGatewayGroupString(t *testing.T) {
	principal, failure := Authenticate(requestWithClaims(map[string]string{"sub": "user-123", "cognito:groups": "[User Admin]"}))
	if failure != nil || !principal.InGroup(UserGroup) || !principal.InGroup(AdminGroup) {
		t.Fatalf("principal = %#v, failure = %#v", principal, failure)
	}
}

func TestAuthenticateRejectsMissingVerifiedSubject(t *testing.T) {
	for _, request := range []events.APIGatewayV2HTTPRequest{{}, requestWithClaims(map[string]string{"cognito:groups": `["Admin"]`})} {
		_, failure := Authenticate(request)
		if failure == nil || failure.StatusCode != http.StatusUnauthorized || errorCode(failure.Body) != "unauthenticated" {
			t.Fatalf("failure = %#v", failure)
		}
	}
}

func TestRequireAdminUsesOnlyVerifiedGroupClaims(t *testing.T) {
	standard := requestWithClaims(map[string]string{"sub": "user-123", "cognito:groups": `["User"]`})
	if _, failure := RequireAdmin(standard); failure == nil || failure.StatusCode != http.StatusForbidden || errorCode(failure.Body) != "forbidden" {
		t.Fatalf("standard-user failure = %#v", failure)
	}
	admin := requestWithClaims(map[string]string{"sub": "admin-123", "cognito:groups": `["User","Admin"]`})
	if principal, failure := RequireAdmin(admin); failure != nil || principal.Subject != "admin-123" {
		t.Fatalf("principal = %#v, failure = %#v", principal, failure)
	}
}

func errorCode(body string) string {
	var decoded struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal([]byte(body), &decoded)
	return decoded.Error.Code
}
