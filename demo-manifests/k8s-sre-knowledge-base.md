# KubeSynapse SRE - Internal Troubleshooting Guide

## Scenario: OOMKilled (Out of Memory)
- **Symptom**: Pod status shows `OOMKilled` or `Reason: Error`.
- **Root Cause**: The application requested more memory than its `limit` allows.
- **Remediation**: 
    1. Identify the peak memory usage using `kubectl top pod`.
    2. Increase the memory `limits` in the deployment manifest.
    3. Check for memory leaks in the heap dumps if the usage keeps climbing.

## Scenario: CrashLoopBackOff (Startup Failure)
- **Symptom**: Pod restarts repeatedly.
- **Root Cause**: Often caused by missing Environment Variables (config error) or a failing Readiness Probe.
- **Remediation**:
    1. Check logs with `kubectl logs <pod-name> --previous`.
    2. Verify all `ConfigMaps` and `Secrets` required by the pod are correctly mapped.
    3. If the app is healthy but K8s kills it, increase `initialDelaySeconds` in the readiness probe.

## Scenario: TLS Handshake Timeout / Service Refused
- **Symptom**: `TLS handshake timeout` or `Connection Refused` when accessing via NodePort.
- **Root Cause**: The cluster node is under extreme CPU/RAM pressure, or the `kube-proxy` is failing.
- **Remediation**:
    1. Check Docker Desktop resource settings (needs 8GB+ RAM).
    2. Restart Docker Desktop to clear system-level resource locks.
    3. Ensure no other heavy processes are running on the host Mac.

## Scenario: n8n 404 Webhook Error
- **Symptom**: Agent logs show `status: 404` for the n8n URL.
- **Root Cause**: The n8n workflow is either not IMPORTED or not ACTIVE.
- **Remediation**:
    1. Open the n8n UI at `http://localhost:30678`.
    2. Ensure the "KubeSynapse Pipeline" is set to "Active" (top-right toggle).
