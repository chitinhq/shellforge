#!/bin/bash
# Trigger RunPod GPU session for ShellForge dogfood run
runpod run --image shellforge-v1 --gpus a10x --command "cd /home/jared/agentguard-workspace/shellforge && go run ./cmd/shellforge/main.go"