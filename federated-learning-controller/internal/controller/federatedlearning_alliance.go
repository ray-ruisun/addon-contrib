package controller

import (
	"fmt"
	"strings"

	flv1alpha1 "github/open-cluster-management/federated-learning/api/v1alpha1"
)

const (
	defaultFLockAllianceRuntimeMode       = flv1alpha1.FLockAllianceRuntimeLocal
	defaultFLockAllianceStake             = "0"
	defaultFLockAllianceStorageBackend    = flv1alpha1.FLockAllianceStorageS3
	defaultFLockAllianceLocalSharedDir    = "/data/shared"
	defaultFLockAllianceNumParticipants   = 1
	defaultFLockAllianceDataVolumeType    = flv1alpha1.FLockAllianceDataVolumeHostPath
)

func normalizeFLockAllianceSpec(spec flv1alpha1.FLockAllianceSpec) flv1alpha1.FLockAllianceSpec {
	if strings.TrimSpace(spec.RuntimeMode) == "" {
		spec.RuntimeMode = defaultFLockAllianceRuntimeMode
	}
	if strings.TrimSpace(spec.Stake) == "" {
		spec.Stake = defaultFLockAllianceStake
	}
	if strings.TrimSpace(spec.StorageBackend) == "" {
		spec.StorageBackend = defaultFLockAllianceStorageBackend
	}
	if strings.TrimSpace(spec.LocalSharedDir) == "" {
		spec.LocalSharedDir = defaultFLockAllianceLocalSharedDir
	}
	if spec.NumParticipants < 1 {
		spec.NumParticipants = defaultFLockAllianceNumParticipants
	}
	if strings.TrimSpace(spec.DataVolumeType) == "" {
		spec.DataVolumeType = defaultFLockAllianceDataVolumeType
	}
	return spec
}

func validateFLockAllianceSpec(spec flv1alpha1.FLockAllianceSpec) error {
	switch spec.RuntimeMode {
	case flv1alpha1.FLockAllianceRuntimeDocker, flv1alpha1.FLockAllianceRuntimeLocal:
	default:
		return fmt.Errorf("unsupported FLockAlliance.runtimeMode: %s", spec.RuntimeMode)
	}

	if spec.StorageBackend != flv1alpha1.FLockAllianceStorageS3 && spec.StorageBackend != flv1alpha1.FLockAllianceStorageLocal {
		return fmt.Errorf("unsupported FLockAlliance.storageBackend: %s (expected s3 or local)", spec.StorageBackend)
	}

	if spec.NumParticipants < 1 {
		return fmt.Errorf("FLockAlliance.numParticipants must be >= 1")
	}
	switch spec.DataVolumeType {
	case flv1alpha1.FLockAllianceDataVolumeHostPath, flv1alpha1.FLockAllianceDataVolumeEmptyDir, flv1alpha1.FLockAllianceDataVolumePVC:
	default:
		return fmt.Errorf(
			"unsupported FLockAlliance.dataVolumeType: %s (expected hostPath, emptyDir, or pvc)",
			spec.DataVolumeType,
		)
	}
	if spec.DataVolumeType == flv1alpha1.FLockAllianceDataVolumePVC &&
		strings.TrimSpace(spec.DataVolumeClaimName) == "" {
		return fmt.Errorf("FLockAlliance.dataVolumeClaimName is required when dataVolumeType=pvc")
	}
	return nil
}
