package manifests

import (
	"encoding/json"
	"strings"
	"testing"

	"github/open-cluster-management/federated-learning/internal/controller/manifests/applier"
)

func TestAllianceClientManifestRender(t *testing.T) {
	params := &AllianceClientParams{
		ManifestName:           "fl-alliance",
		ManifestNamespace:      "cluster-a",
		ClientJobNamespace:     "fl-workload",
		ClientJobName:          "fl-alliance-client",
		ClientJobImage:         "ghcr.io/flock-io/fl-alliance-client:v0.1.0",
		FLocKitImage:           "ghcr.io/flock-io/flockit:v0.1.0",
		DataPath:               "/data",
		RuntimeMode:            "redhat_ocm",
		ModelAPIURL:            "http://127.0.0.1:5000",
		UseGPU:                 false,
		BlockchainRPC:          "https://sepolia.base.org",
		TokenAddress:           "0x1",
		TaskAddress:            "0x2",
		Stake:                  "0",
		StorageBackend:         "s3",
		LocalSharedDir:         "/data/shared",
		NoIncentive:            false,
		NumParticipants:        1,
		PrivateKeySecretName:   "fl-alliance-secret",
		PrivateKeySecretKey:    "CLIENT_PRIVATE_KEY",
		HFTokenSecretName:      "fl-alliance-secret",
		HFTokenSecretKey:       "HF_TOKEN",
		HasHFTokenSecret:       true,
		FLocKitConfigPath:      "templates/llm_finetuning/configs/addon_default.yaml",
		FLocKitPort:            5000,
		FLocKitOverrides:       "",
		FLocKitDataSource:      "",
		FLocKitDataIndicesPath: "",
	}

	renderer := applier.NewRenderer(AllianceClientFiles)
	objects, err := renderer.Render("", "", func(string) (interface{}, error) {
		return params, nil
	})
	if err != nil {
		t.Fatalf("failed to render manifest: %v", err)
	}
	if len(objects) != 1 {
		t.Fatalf("expected 1 object, got %d", len(objects))
	}
	if objects[0].GetKind() != "ManifestWork" {
		t.Fatalf("expected kind ManifestWork, got %s", objects[0].GetKind())
	}

	raw, err := json.Marshal(objects[0].Object)
	if err != nil {
		t.Fatalf("failed to marshal object: %v", err)
	}
	rendered := string(raw)
	for _, token := range []string{"flockit-http", "tcpSocket", "CLIENT_PRIVATE_KEY", "HF_TOKEN"} {
		if !strings.Contains(rendered, token) {
			t.Fatalf("expected rendered manifest to contain %q", token)
		}
	}
}
