package controller

import (
	"fmt"
	"net/url"
	"strings"

	flv1alpha1 "github/open-cluster-management/federated-learning/api/v1alpha1"
)

const (
	defaultFLockAllianceRuntimeMode       = flv1alpha1.FLockAllianceRuntimeRedHatOCM
	defaultFLockAllianceModelAPIURL       = "http://127.0.0.1:5000"
	defaultFLockAllianceStake             = "0"
	defaultFLockAllianceStorageBackend    = flv1alpha1.FLockAllianceStorageS3
	defaultFLockAllianceLocalSharedDir    = "/data/shared"
	defaultFLockAllianceNumParticipants   = 1
	defaultFLockAlliancePrivateSecretName = "flock-alliance-secret"
	defaultFLockAlliancePrivateSecretKey  = "CLIENT_PRIVATE_KEY"
	defaultFLockAllianceFLocKitConfigPath = "templates/llm_finetuning/configs/addon_default.yaml"
	defaultFLockAllianceFLocKitPort       = 5000
	defaultFLockAllianceDataPath          = "/data"
)

func normalizeFLockAllianceSpec(spec flv1alpha1.FLockAllianceSpec) flv1alpha1.FLockAllianceSpec {
	if strings.TrimSpace(spec.RuntimeMode) == "" {
		spec.RuntimeMode = defaultFLockAllianceRuntimeMode
	}
	if strings.TrimSpace(spec.ModelAPIURL) == "" {
		spec.ModelAPIURL = defaultFLockAllianceModelAPIURL
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
	if strings.TrimSpace(spec.PrivateKeySecret.Name) == "" {
		spec.PrivateKeySecret.Name = defaultFLockAlliancePrivateSecretName
	}
	if strings.TrimSpace(spec.PrivateKeySecret.Key) == "" {
		spec.PrivateKeySecret.Key = defaultFLockAlliancePrivateSecretKey
	}
	if strings.TrimSpace(spec.FLocKitConfigPath) == "" {
		spec.FLocKitConfigPath = defaultFLockAllianceFLocKitConfigPath
	}
	if spec.FLocKitPort <= 0 {
		spec.FLocKitPort = defaultFLockAllianceFLocKitPort
	}
	if strings.TrimSpace(spec.DataPath) == "" {
		spec.DataPath = defaultFLockAllianceDataPath
	}
	return spec
}

func validateFLockAllianceSpec(spec flv1alpha1.FLockAllianceSpec) error {
	switch spec.RuntimeMode {
	case flv1alpha1.FLockAllianceRuntimeDocker, flv1alpha1.FLockAllianceRuntimeLocal, flv1alpha1.FLockAllianceRuntimeRedHatOCM:
	default:
		return fmt.Errorf("unsupported FLockAlliance.runtimeMode: %s", spec.RuntimeMode)
	}

	if spec.RuntimeMode == flv1alpha1.FLockAllianceRuntimeRedHatOCM {
		parsed, err := url.Parse(spec.ModelAPIURL)
		if err != nil || parsed == nil || parsed.Scheme == "" || parsed.Host == "" {
			return fmt.Errorf("FLockAlliance.modelApiUrl must be a valid http(s) URL when runtimeMode=redhat_ocm")
		}
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return fmt.Errorf("FLockAlliance.modelApiUrl must use http or https when runtimeMode=redhat_ocm")
		}
	}

	if spec.StorageBackend != flv1alpha1.FLockAllianceStorageS3 && spec.StorageBackend != flv1alpha1.FLockAllianceStorageLocal {
		return fmt.Errorf("unsupported FLockAlliance.storageBackend: %s (expected s3 or local)", spec.StorageBackend)
	}

	if spec.NumParticipants < 1 {
		return fmt.Errorf("FLockAlliance.numParticipants must be >= 1")
	}

	if strings.TrimSpace(spec.BlockchainRPC) == "" {
		return fmt.Errorf("FLockAlliance.blockchainRpc is required")
	}
	if strings.TrimSpace(spec.TokenAddress) == "" {
		return fmt.Errorf("FLockAlliance.tokenAddress is required")
	}
	if strings.TrimSpace(spec.TaskAddress) == "" {
		return fmt.Errorf("FLockAlliance.taskAddress is required")
	}
	if strings.TrimSpace(spec.FLocKitImage) == "" {
		return fmt.Errorf("FLockAlliance.flockitImage is required")
	}
	if strings.TrimSpace(spec.PrivateKeySecret.Name) == "" || strings.TrimSpace(spec.PrivateKeySecret.Key) == "" {
		return fmt.Errorf("FLockAlliance.privateKeySecret.name and FLockAlliance.privateKeySecret.key are required")
	}
	if spec.HFTokenSecret != nil {
		if strings.TrimSpace(spec.HFTokenSecret.Name) == "" || strings.TrimSpace(spec.HFTokenSecret.Key) == "" {
			return fmt.Errorf("FLockAlliance.hfTokenSecret must include both name and key when set")
		}
	}
	return nil
}
