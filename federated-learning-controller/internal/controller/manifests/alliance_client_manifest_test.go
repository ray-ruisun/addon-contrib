package manifests

import (
	"encoding/json"
	"strings"
	"testing"

	"github/open-cluster-management/federated-learning/internal/controller/manifests/applier"
)

func TestFLockAllianceClientManifestRender(t *testing.T) {
	params := &FLockAllianceClientParams{
		ManifestName:           "flock-alliance",
		ManifestNamespace:      "cluster-a",
		ClientJobNamespace:     "fl-workload",
		ClientJobName:          "flock-alliance-client",
		ClientJobImage:         "ghcr.io/flock-io/fl-alliance-client:v0.1.0",
		DataVolumeType:         "hostPath",
		DataPath:               "/data",
		RuntimeMode:            "local",
		UseGPU:                 false,
		BlockchainRPC:          "https://sepolia.base.org",
		TokenAddress:           "0x1",
		TaskAddress:            "0x2",
		Stake:                  "0",
		StorageBackend:         "s3",
		LocalSharedDir:         "/data/shared",
		NoIncentive:            false,
		NumParticipants:        1,
		PrivateKeySecretName:   "flock-alliance-secret",
		PrivateKeySecretKey:    "CLIENT_PRIVATE_KEY",
		HFTokenSecretName:      "flock-alliance-secret",
		HFTokenSecretKey:       "HF_TOKEN",
		HasHFTokenSecret:       true,
	}

	renderer := applier.NewRenderer(FLockAllianceClientFiles)
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
	for _, token := range []string{"--env-file", "hostPath", "CLIENT_PRIVATE_KEY", "HF_TOKEN"} {
		if !strings.Contains(rendered, token) {
			t.Fatalf("expected rendered manifest to contain %q", token)
		}
	}
}

func TestFLockAllianceClientManifestRenderWithoutTokenTaskOverrides(t *testing.T) {
	params := &FLockAllianceClientParams{
		ManifestName:         "flock-alliance",
		ManifestNamespace:    "cluster-a",
		ClientJobNamespace:   "fl-workload",
		ClientJobName:        "flock-alliance-client",
		ClientJobImage:       "ghcr.io/flock-io/fl-alliance-client:v0.1.0",
		DataVolumeType:       "hostPath",
		DataPath:             "/data",
		RuntimeMode:          "local",
		UseGPU:               false,
		BlockchainRPC:        "",
		TokenAddress:         "",
		TaskAddress:          "",
		Stake:                "0",
		StorageBackend:       "s3",
		LocalSharedDir:       "/data/shared",
		NoIncentive:          false,
		NumParticipants:      1,
		PrivateKeySecretName: "flock-alliance-secret",
		PrivateKeySecretKey:  "CLIENT_PRIVATE_KEY",
		HasHFTokenSecret:     false,
	}

	renderer := applier.NewRenderer(FLockAllianceClientFiles)
	objects, err := renderer.Render("", "", func(string) (interface{}, error) {
		return params, nil
	})
	if err != nil {
		t.Fatalf("failed to render manifest: %v", err)
	}
	raw, err := json.Marshal(objects[0].Object)
	if err != nil {
		t.Fatalf("failed to marshal object: %v", err)
	}
	rendered := string(raw)
	if strings.Contains(rendered, "blockchain.token_address=") {
		t.Fatalf("did not expect blockchain.token_address override when token address is empty")
	}
	if strings.Contains(rendered, "blockchain.task_address=") {
		t.Fatalf("did not expect blockchain.task_address override when task address is empty")
	}
	if strings.Contains(rendered, "blockchain.rpc=") {
		t.Fatalf("did not expect blockchain.rpc override when rpc is empty")
	}
}

func TestFLockAllianceClientManifestRenderEmptyDirVolume(t *testing.T) {
	params := &FLockAllianceClientParams{
		ManifestName:         "flock-alliance",
		ManifestNamespace:    "cluster-a",
		ClientJobNamespace:   "fl-workload",
		ClientJobName:        "flock-alliance-client",
		ClientJobImage:       "ghcr.io/flock-io/fl-alliance-client:v0.1.0",
		DataVolumeType:       "emptyDir",
		DataPath:             "/ignored-for-emptydir",
		RuntimeMode:          "local",
		UseGPU:               false,
		Stake:                "0",
		StorageBackend:       "s3",
		LocalSharedDir:       "/data/shared",
		NoIncentive:          false,
		NumParticipants:      1,
		PrivateKeySecretName: "flock-alliance-secret",
		PrivateKeySecretKey:  "CLIENT_PRIVATE_KEY",
	}

	renderer := applier.NewRenderer(FLockAllianceClientFiles)
	objects, err := renderer.Render("", "", func(string) (interface{}, error) {
		return params, nil
	})
	if err != nil {
		t.Fatalf("failed to render manifest: %v", err)
	}
	raw, err := json.Marshal(objects[0].Object)
	if err != nil {
		t.Fatalf("failed to marshal object: %v", err)
	}
	rendered := string(raw)
	if !strings.Contains(rendered, "\"emptyDir\"") {
		t.Fatalf("expected emptyDir volume in rendered manifest")
	}
	if strings.Contains(rendered, "\"hostPath\"") {
		t.Fatalf("did not expect hostPath volume when DataVolumeType=emptyDir")
	}
}

func TestFLockAllianceClientManifestRenderPVCVolume(t *testing.T) {
	params := &FLockAllianceClientParams{
		ManifestName:         "flock-alliance",
		ManifestNamespace:    "cluster-a",
		ClientJobNamespace:   "fl-workload",
		ClientJobName:        "flock-alliance-client",
		ClientJobImage:       "ghcr.io/flock-io/fl-alliance-client:v0.1.0",
		DataVolumeType:       "pvc",
		DataVolumeClaimName:  "flock-shared-data",
		DataPath:             "/ignored-for-pvc",
		RuntimeMode:          "local",
		UseGPU:               false,
		Stake:                "0",
		StorageBackend:       "s3",
		LocalSharedDir:       "/data/shared",
		NoIncentive:          false,
		NumParticipants:      1,
		PrivateKeySecretName: "flock-alliance-secret",
		PrivateKeySecretKey:  "CLIENT_PRIVATE_KEY",
	}

	renderer := applier.NewRenderer(FLockAllianceClientFiles)
	objects, err := renderer.Render("", "", func(string) (interface{}, error) {
		return params, nil
	})
	if err != nil {
		t.Fatalf("failed to render manifest: %v", err)
	}
	raw, err := json.Marshal(objects[0].Object)
	if err != nil {
		t.Fatalf("failed to marshal object: %v", err)
	}
	rendered := string(raw)
	if !strings.Contains(rendered, "persistentVolumeClaim") {
		t.Fatalf("expected persistentVolumeClaim in rendered manifest")
	}
	if !strings.Contains(rendered, "flock-shared-data") {
		t.Fatalf("expected pvc claim name in rendered manifest")
	}
}
