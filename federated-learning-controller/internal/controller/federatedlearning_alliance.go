package controller

import (
	"fmt"
	"net/url"
	"strings"

	flv1alpha1 "github/open-cluster-management/federated-learning/api/v1alpha1"
)

const (
	defaultAllianceRuntimeMode       = flv1alpha1.AllianceRuntimeRedHatOCM
	defaultAllianceModelAPIURL       = "http://127.0.0.1:5000"
	defaultAllianceStake             = "0"
	defaultAllianceStorageBackend    = flv1alpha1.AllianceStorageS3
	defaultAllianceLocalSharedDir    = "/data/shared"
	defaultAllianceNumParticipants   = 1
	defaultAlliancePrivateSecretName = "fl-alliance-secret"
	defaultAlliancePrivateSecretKey  = "CLIENT_PRIVATE_KEY"
	defaultAllianceFLocKitConfigPath = "templates/llm_finetuning/configs/addon_default.yaml"
	defaultAllianceFLocKitPort       = 5000
	defaultAllianceDataPath          = "/data"
)

func normalizeAllianceSpec(spec flv1alpha1.AllianceSpec) flv1alpha1.AllianceSpec {
	if strings.TrimSpace(spec.RuntimeMode) == "" {
		spec.RuntimeMode = defaultAllianceRuntimeMode
	}
	if strings.TrimSpace(spec.ModelAPIURL) == "" {
		spec.ModelAPIURL = defaultAllianceModelAPIURL
	}
	if strings.TrimSpace(spec.Stake) == "" {
		spec.Stake = defaultAllianceStake
	}
	if strings.TrimSpace(spec.StorageBackend) == "" {
		spec.StorageBackend = defaultAllianceStorageBackend
	}
	if strings.TrimSpace(spec.LocalSharedDir) == "" {
		spec.LocalSharedDir = defaultAllianceLocalSharedDir
	}
	if spec.NumParticipants < 1 {
		spec.NumParticipants = defaultAllianceNumParticipants
	}
	if strings.TrimSpace(spec.PrivateKeySecret.Name) == "" {
		spec.PrivateKeySecret.Name = defaultAlliancePrivateSecretName
	}
	if strings.TrimSpace(spec.PrivateKeySecret.Key) == "" {
		spec.PrivateKeySecret.Key = defaultAlliancePrivateSecretKey
	}
	if strings.TrimSpace(spec.FLocKitConfigPath) == "" {
		spec.FLocKitConfigPath = defaultAllianceFLocKitConfigPath
	}
	if spec.FLocKitPort <= 0 {
		spec.FLocKitPort = defaultAllianceFLocKitPort
	}
	if strings.TrimSpace(spec.DataPath) == "" {
		spec.DataPath = defaultAllianceDataPath
	}
	return spec
}

func validateAllianceSpec(spec flv1alpha1.AllianceSpec) error {
	switch spec.RuntimeMode {
	case flv1alpha1.AllianceRuntimeDocker, flv1alpha1.AllianceRuntimeLocal, flv1alpha1.AllianceRuntimeRedHatOCM:
	default:
		return fmt.Errorf("unsupported alliance.runtimeMode: %s", spec.RuntimeMode)
	}

	if spec.RuntimeMode == flv1alpha1.AllianceRuntimeRedHatOCM {
		parsed, err := url.Parse(spec.ModelAPIURL)
		if err != nil || parsed == nil || parsed.Scheme == "" || parsed.Host == "" {
			return fmt.Errorf("alliance.modelApiUrl must be a valid http(s) URL when runtimeMode=redhat_ocm")
		}
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return fmt.Errorf("alliance.modelApiUrl must use http or https when runtimeMode=redhat_ocm")
		}
	}

	if spec.StorageBackend != flv1alpha1.AllianceStorageS3 && spec.StorageBackend != flv1alpha1.AllianceStorageLocal {
		return fmt.Errorf("unsupported alliance.storageBackend: %s (expected s3 or local)", spec.StorageBackend)
	}

	if spec.NumParticipants < 1 {
		return fmt.Errorf("alliance.numParticipants must be >= 1")
	}

	if strings.TrimSpace(spec.BlockchainRPC) == "" {
		return fmt.Errorf("alliance.blockchainRpc is required")
	}
	if strings.TrimSpace(spec.TokenAddress) == "" {
		return fmt.Errorf("alliance.tokenAddress is required")
	}
	if strings.TrimSpace(spec.TaskAddress) == "" {
		return fmt.Errorf("alliance.taskAddress is required")
	}
	if strings.TrimSpace(spec.FLocKitImage) == "" {
		return fmt.Errorf("alliance.flockitImage is required")
	}
	if strings.TrimSpace(spec.PrivateKeySecret.Name) == "" || strings.TrimSpace(spec.PrivateKeySecret.Key) == "" {
		return fmt.Errorf("alliance.privateKeySecret.name and alliance.privateKeySecret.key are required")
	}
	if spec.HFTokenSecret != nil {
		if strings.TrimSpace(spec.HFTokenSecret.Name) == "" || strings.TrimSpace(spec.HFTokenSecret.Key) == "" {
			return fmt.Errorf("alliance.hfTokenSecret must include both name and key when set")
		}
	}
	return nil
}
