package controller

import (
	"testing"

	flv1alpha1 "github/open-cluster-management/federated-learning/api/v1alpha1"
)

func validAllianceSpec() flv1alpha1.AllianceSpec {
	return flv1alpha1.AllianceSpec{
		RuntimeMode:     flv1alpha1.AllianceRuntimeRedHatOCM,
		ModelAPIURL:     "http://127.0.0.1:5000",
		BlockchainRPC:   "http://127.0.0.1:8545",
		TokenAddress:    "0x0000000000000000000000000000000000000001",
		TaskAddress:     "0x0000000000000000000000000000000000000002",
		StorageBackend:  flv1alpha1.AllianceStorageS3,
		NumParticipants: 1,
		FLocKitImage:    "ghcr.io/flock-io/flockit:v0.1.0",
		PrivateKeySecret: flv1alpha1.SecretRef{
			Name: "fl-alliance-secret",
			Key:  "CLIENT_PRIVATE_KEY",
		},
	}
}

func TestNormalizeAllianceSpec(t *testing.T) {
	spec := normalizeAllianceSpec(flv1alpha1.AllianceSpec{})

	if spec.RuntimeMode != flv1alpha1.AllianceRuntimeRedHatOCM {
		t.Fatalf("expected runtime mode %q, got %q", flv1alpha1.AllianceRuntimeRedHatOCM, spec.RuntimeMode)
	}
	if spec.ModelAPIURL != "http://127.0.0.1:5000" {
		t.Fatalf("unexpected modelApiUrl default: %q", spec.ModelAPIURL)
	}
	if spec.StorageBackend != flv1alpha1.AllianceStorageS3 {
		t.Fatalf("expected storage backend %q, got %q", flv1alpha1.AllianceStorageS3, spec.StorageBackend)
	}
	if spec.PrivateKeySecret.Name == "" || spec.PrivateKeySecret.Key == "" {
		t.Fatalf("expected non-empty private key secret defaults")
	}
	if spec.FLocKitPort != 5000 {
		t.Fatalf("expected flockitPort=5000, got %d", spec.FLocKitPort)
	}
}

func TestValidateAllianceSpec(t *testing.T) {
	tests := []struct {
		name    string
		spec    flv1alpha1.AllianceSpec
		wantErr bool
	}{
		{
			name:    "valid redhat_ocm",
			spec:    validAllianceSpec(),
			wantErr: false,
		},
		{
			name: "valid local runtime",
			spec: func() flv1alpha1.AllianceSpec {
				spec := validAllianceSpec()
				spec.RuntimeMode = flv1alpha1.AllianceRuntimeLocal
				spec.ModelAPIURL = ""
				return spec
			}(),
			wantErr: false,
		},
		{
			name: "invalid redhat_ocm model url",
			spec: func() flv1alpha1.AllianceSpec {
				spec := validAllianceSpec()
				spec.ModelAPIURL = "127.0.0.1:5000"
				return spec
			}(),
			wantErr: true,
		},
		{
			name: "invalid storage backend",
			spec: func() flv1alpha1.AllianceSpec {
				spec := validAllianceSpec()
				spec.StorageBackend = "nfs"
				return spec
			}(),
			wantErr: true,
		},
		{
			name: "partial hf token secret",
			spec: func() flv1alpha1.AllianceSpec {
				spec := validAllianceSpec()
				spec.HFTokenSecret = &flv1alpha1.SecretRef{Name: "fl-alliance-secret"}
				return spec
			}(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAllianceSpec(tt.spec)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}
