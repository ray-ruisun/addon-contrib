#!/usr/bin/env bats
# Argv shape of the python invocation. FLockAlliance reads its config from
# config/conf.yaml first and then layers `--override key=value` flags on
# top, so the precise argv is the contract between the entrypoint and the
# downstream YAML. A regression here typically silently fails to apply a
# runtime knob (storage backend, GPU on/off, num_participants) without any
# log line that points at the cause.

load test_helper

@test "always emits the static --override block (mode/gpu/stake/backend/incentive/data)" {
  STORAGE_BACKEND=s3 USE_GPU=false STAKE=0 NO_INCENTIVE=false \
  TASK_ADDRESS=0xT \
  run run_entrypoint
  [ "$status" -eq 0 ]
  [[ "$output" == *"--config config/conf.yaml"* ]]
  [[ "$output" == *"--override runtime.mode=local"* ]]
  [[ "$output" == *"--override runtime.gpu=false"* ]]
  [[ "$output" == *"--override blockchain.stake=0"* ]]
  [[ "$output" == *"--override storage.backend=s3"* ]]
  [[ "$output" == *"--override training.no_incentive=false"* ]]
  [[ "$output" == *"--override data.inputs=/data"* ]]
}

@test "appends storage.local.shared_dir only when LOCAL_STORAGE_DIR is set" {
  STORAGE_BACKEND=local LOCAL_STORAGE_DIR=/data/shared \
  TASK_ADDRESS=0xT \
  run run_entrypoint
  [ "$status" -eq 0 ]
  [[ "$output" == *"--override storage.local.shared_dir=/data/shared"* ]]
}

@test "omits storage.local.shared_dir when LOCAL_STORAGE_DIR is empty" {
  STORAGE_BACKEND=s3 LOCAL_STORAGE_DIR="" \
  TASK_ADDRESS=0xT \
  run run_entrypoint
  [ "$status" -eq 0 ]
  [[ "$output" != *"--override storage.local.shared_dir"* ]]
}

@test "appends training.num_participants only when NUM_PARTICIPANTS is set" {
  NUM_PARTICIPANTS=4 TASK_ADDRESS=0xT \
  run run_entrypoint
  [ "$status" -eq 0 ]
  [[ "$output" == *"--override training.num_participants=4"* ]]
}

@test "GPU variant runtime.gpu=true is forwarded into the argv" {
  USE_GPU=true TASK_ADDRESS=0xT \
  run run_entrypoint
  [ "$status" -eq 0 ]
  [[ "$output" == *"--override runtime.gpu=true"* ]]
}

@test "S3-compatible nami fields do not leak into the python argv" {
  # nami fields are consumed by FLockAlliance's storage layer via env vars,
  # not via --override; emitting them as overrides would bypass the YAML
  # validation that the storage layer performs at startup.
  STORAGE_BACKEND=nami \
  S3_COMPAT_ENDPOINT_URL=http://minio:9000 \
  S3_COMPAT_BUCKET=flock-dev \
  S3_COMPAT_ACCESS_KEY=AK \
  S3_COMPAT_SECRET_KEY=SK \
  TASK_ADDRESS=0xT \
  run run_entrypoint
  [ "$status" -eq 0 ]
  [[ "$output" != *"--override storage.s3"* ]]
  [[ "$output" != *"--override S3_COMPAT"* ]]
}

@test "emits effective summary lines that downstream debugging relies on" {
  STORAGE_BACKEND=s3 USE_GPU=false TASK_ADDRESS=0xT \
  run run_entrypoint
  [ "$status" -eq 0 ]
  [[ "$output" == *"effective: STORAGE_BACKEND=s3 USE_GPU=false"* ]]
  [[ "$output" == *"effective: BLOCKCHAIN_RPC="* ]]
  [[ "$output" == *"TASK_ADDRESS=<set>0xT"* ]]
}

@test "nami backend emits its S3 summary line for troubleshooting" {
  STORAGE_BACKEND=nami \
  S3_COMPAT_ENDPOINT_URL=http://minio:9000 \
  S3_COMPAT_BUCKET=flock-dev \
  S3_COMPAT_REGION=us-east-1 \
  TASK_ADDRESS=0xT \
  run run_entrypoint
  [ "$status" -eq 0 ]
  [[ "$output" == *"effective: S3_COMPAT_ENDPOINT_URL=http://minio:9000 S3_COMPAT_BUCKET=flock-dev S3_COMPAT_REGION=us-east-1"* ]]
}

@test "no .env file path skips the env load step entirely" {
  FLOCK_ALLIANCE_ENV_FILE="" \
  TASK_ADDRESS=0xT \
  run run_entrypoint
  [ "$status" -eq 0 ]
  [[ "$output" == *"FLOCK_ALLIANCE_ENV_FILE is empty; skipping node env load"* ]]
}
