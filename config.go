package ephi

import (
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/cloudkms/v1"
	"google.golang.org/appengine"
)

var (
	token              = ""
	DefaultDelay       = 15
	AccessToken  = ""
)

func BytesToString(data []byte) string {
	return strings.TrimSpace(string(data[:]))
}

func getPorjectId(ctx context.Context) string {
	if appengine.IsDevAppServer() {
		// For local development return your project id
		return "api-project-613886847980"

	}
	return appengine.AppID(ctx)
}

func SetupConfig(ctx context.Context, appconfig AppConfig) error {
	data, _ := base64.StdEncoding.DecodeString(appconfig.Token)
	res, _ := decrypt(ctx, data)
	token = BytesToString(res)
	data, _ = base64.StdEncoding.DecodeString(appconfig.AccessToken)
	res, _ = decrypt(ctx, data)
	AccessToken = BytesToString(res)
	DefaultDelay = appconfig.DefaultDelay

	return nil
}

func decrypt(ctx context.Context, ciphertext []byte) ([]byte, error) {
	client, err := google.DefaultClient(ctx, cloudkms.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	cloudKmsService, err := cloudkms.New(client)
	parentName := fmt.Sprintf("projects/%s/locations/%s/keyRings/%s/cryptoKeys/%s",
		getPorjectId(ctx), "us-central1", "Ephi", "ephikey")
	req := &cloudkms.DecryptRequest{
		Ciphertext: base64.StdEncoding.EncodeToString(ciphertext),
	}
	resp, err := cloudKmsService.Projects.Locations.KeyRings.CryptoKeys.Decrypt(parentName, req).Do()
	if err != nil {
		return nil, err
	}
	return base64.StdEncoding.DecodeString(resp.Plaintext)
}

