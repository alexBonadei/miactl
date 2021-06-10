package auth

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/davidebianchi/go-jsonclient"
	"github.com/mia-platform/miactl/internal/mocks"
	"github.com/stretchr/testify/require"
)

func TestLogin(t *testing.T) {
	const (
		username            = "random"
		password            = "secret"
		appID               = "film"
		providerID          = "provia"
		baseURL             = "http://auth-provider/"
		expectedAccessToken = "YWNjZXNzVG9rZW4="
	)

	t.Run("successful login", func(t *testing.T) {
		mockConfigs := mocks.ServerConfigs{
			{
				Endpoint:    "/api/oauth/token",
				Method:      http.MethodPost,
				RequestBody: nil,
				Reply: map[string]interface{}{
					"accessToken":  expectedAccessToken,
					"refreshToken": "cmVmcmVzaFRva2Vu",
					"expiresAt":    1619799800,
				},
				ReplyStatus: http.StatusOK,
			},
		}

		s, err := mocks.HTTPServer(t, mockConfigs, nil)
		require.NoError(t, err, "mock must start correctly")
		defer s.Close()

		secureClient := getTestClient(t, s.URL, false)
		accessToken, err := secureClient.Login(username, password, providerID)

		require.Nil(t, err)
		require.Equal(t, expectedAccessToken, accessToken, "Access token differs from expected")
	})

	t.Run("successful login - insecure connection enabled", func(t *testing.T) {
		serverCfg := mocks.CertificatesConfig{
			CertPath: "../testdata/server-cert.pem",
			KeyPath:  "../testdata/server-key.pem",
		}

		mockConfigs := mocks.ServerConfigs{
			{
				Endpoint:    "/api/oauth/token",
				Method:      http.MethodPost,
				RequestBody: nil,
				Reply: map[string]interface{}{
					"accessToken":  expectedAccessToken,
					"refreshToken": "cmVmcmVzaFRva2Vu",
					"expiresAt":    1619799800,
				},
				ReplyStatus: http.StatusOK,
			},
		}

		s, err := mocks.HTTPServer(t, mockConfigs, &serverCfg)
		require.NoError(t, err, "mock must start correctly")
		defer s.Close()

		insecureClient := getTestClient(t, s.URL, true)
		accessToken, err := insecureClient.Login(username, password, providerID)

		require.Nil(t, err)
		require.Equal(t, expectedAccessToken, accessToken, "Access token differs from expected")
	})

	t.Run("failed login", func(t *testing.T) {
		mockConfigs := mocks.ServerConfigs{
			{
				Endpoint:    "/api/oauth/token",
				Method:      http.MethodPost,
				RequestBody: nil,
				Reply:       map[string]interface{}{},
				ReplyStatus: http.StatusUnauthorized,
			},
		}

		s, err := mocks.HTTPServer(t, mockConfigs, nil)
		require.NoError(t, err, "mock must start correctly")
		defer s.Close()

		secureClient := getTestClient(t, s.URL, false)
		accessToken, err := secureClient.Login(username, password, providerID)

		require.Error(t, err)
		require.Contains(t, err.Error(), "auth error:")
		require.Empty(t, accessToken, "Access token must be empty string")
	})
}

func getTestClient(t *testing.T, url string, skipCertificate bool) IAuth {
	t.Helper()

	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	customTransport.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: skipCertificate,
	}
	if !strings.HasSuffix(url, "/") {
		url = fmt.Sprintf("%s/", url)
	}
	clientOptions := jsonclient.Options{
		BaseURL: url,
		HTTPClient: &http.Client{
			Timeout:   time.Second * 10,
			Transport: customTransport,
		},
	}

	client, err := jsonclient.New(clientOptions)
	require.NoError(t, err, "error creating client")

	return AuthClient{
		JSONClient: client,
	}
}
