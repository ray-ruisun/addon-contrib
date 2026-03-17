package controller

import (
	"testing"

	flv1alpha1 "github/open-cluster-management/federated-learning/api/v1alpha1"
)

func validFLockAllianceSpec() flv1alpha1.FLockAllianceSpec {
	return flv1alpha1.FLockAllianceSpec{
		RuntimeMode:     flv1alpha1.FLockAllianceRuntimeLocal,
		BlockchainRPC:   "http://127.0.0.1:8545",
		TokenAddress:    "0x0000000000000000000000000000000000000001",
		TaskAddress:     "0x0000000000000000000000000000000000000002",
		StorageBackend:  flv1alpha1.FLockAllianceStorageS3,
		NumParticipants: 1,
		PrivateKeySecret: flv1alpha1.SecretRef{
			Name: "flock-alliance-secret",
			Key:  "CLIENT_PRIVATE_KEY",
		},
	}
}

func TestNormalizeFLockAllianceSpec(t *testing.T) {
	spec := normalizeFLockAllianceSpec(flv1alpha1.FLockAllianceSpec{})

	if spec.RuntimeMode != flv1alpha1.FLockAllianceRuntimeLocal {
		t.Fatalf("expected runtime mode %q, got %q", flv1alpha1.FLockAllianceRuntimeLocal, spec.RuntimeMode)
	}
	if spec.StorageBackend != flv1alpha1.FLockAllianceStorageS3 {
		t.Fatalf("expected storage backend %q, got %q", flv1alpha1.FLockAllianceStorageS3, spec.StorageBackend)
	}
	if spec.DataVolumeType != flv1alpha1.FLockAllianceDataVolumeHostPath {
		t.Fatalf(
			"expected data volume type %q, got %q",
			flv1alpha1.FLockAllianceDataVolumeHostPath,
			spec.DataVolumeType,
		)
	}
	if spec.PrivateKeySecret.Name == "" || spec.PrivateKeySecret.Key == "" {
		t.Fatalf("expected non-empty private key secret defaults")
	}
}

func TestValidateFLockAllianceSpec(t *testing.T) {
	tests := []struct {
		name    string
		spec    flv1alpha1.FLockAllianceSpec
		wantErr bool
	}{
		{
			name:    "valid local runtime",
			spec:    validFLockAllianceSpec(),
			wantErr: false,
		},
		{
			name: "valid docker runtime",
			spec: func() flv1alpha1.FLockAllianceSpec {
				spec := validFLockAllianceSpec()
				spec.RuntimeMode = flv1alpha1.FLockAllianceRuntimeDocker
				return spec
			}(),
			wantErr: false,
		},
		{
			name: "invalid runtime",
			spec: func() flv1alpha1.FLockAllianceSpec {
				spec := validFLockAllianceSpec()
				spec.RuntimeMode = "invalid"
				return spec
			}(),
			wantErr: true,
		},
		{
			name: "invalid storage backend",
			spec: func() flv1alpha1.FLockAllianceSpec {
				spec := validFLockAllianceSpec()
				spec.StorageBackend = "nfs"
				return spec
			}(),
			wantErr: true,
		},
		{
			name: "token and task can come from env file",
			spec: func() flv1alpha1.FLockAllianceSpec {
				spec := validFLockAllianceSpec()
				spec.TokenAddress = ""
				spec.TaskAddress = ""
				return spec
			}(),
			wantErr: false,
		},
		{
			name: "valid emptyDir data volume",
			spec: func() flv1alpha1.FLockAllianceSpec {
				spec := validFLockAllianceSpec()
				spec.DataVolumeType = flv1alpha1.FLockAllianceDataVolumeEmptyDir
				return spec
			}(),
			wantErr: false,
		},
		{
			name: "valid pvc data volume",
			spec: func() flv1alpha1.FLockAllianceSpec {
				spec := validFLockAllianceSpec()
				spec.DataVolumeType = flv1alpha1.FLockAllianceDataVolumePVC
				spec.DataVolumeClaimName = "flock-shared-data"
				return spec
			}(),
			wantErr: false,
		},
		{
			name: "invalid data volume type",
			spec: func() flv1alpha1.FLockAllianceSpec {
				spec := validFLockAllianceSpec()
				spec.DataVolumeType = "nfs"
				return spec
			}(),
			wantErr: true,
		},
		{
			name: "invalid pvc data volume missing claim",
			spec: func() flv1alpha1.FLockAllianceSpec {
				spec := validFLockAllianceSpec()
				spec.DataVolumeType = flv1alpha1.FLockAllianceDataVolumePVC
				spec.DataVolumeClaimName = ""
				return spec
			}(),
			wantErr: true,
		},
		{
			name: "partial hf token secret",
			spec: func() flv1alpha1.FLockAllianceSpec {
				spec := validFLockAllianceSpec()
				spec.HFTokenSecret = &flv1alpha1.SecretRef{Name: "flock-alliance-secret"}
				return spec
			}(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFLockAllianceSpec(tt.spec)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}
