#!/usr/bin/env bash

# Test the kernel version comparison logic
compare_kernel_versions() {
  local v1="$1"
  local v2="$2"
  
  # Convert versions to comparable format: 5.13 -> 005013, 5.10 -> 005010
  local v1_padded=$(echo "$v1" | awk -F. '{printf "%03d%03d", $1, $2}')
  local v2_padded=$(echo "$v2" | awk -F. '{printf "%03d%03d", $1, $2}')
  
  # Compare numerically
  if [[ "$v1_padded" -lt "$v2_padded" ]]; then
    return 0  # true: v1 < v2
  else
    return 1  # false: v1 >= v2
  fi
}

echo "Testing kernel version comparisons:"
echo "=================================="

test_comparison() {
  local v1="$1"
  local v2="$2"
  local expected="$3"
  
  if compare_kernel_versions "$v1" "$v2"; then
    result="true"
  else
    result="false"
  fi
  
  if [[ "$result" == "$expected" ]]; then
    echo "✓ $v1 < $v2 = $result (expected: $expected)"
  else
    echo "✗ $v1 < $v2 = $result (expected: $expected) - FAIL"
  fi
}

# Test cases
test_comparison "5.9" "5.13" "true"
test_comparison "5.10" "5.13" "true"
test_comparison "5.13" "5.13" "false"
test_comparison "5.15" "5.13" "false"
test_comparison "6.1" "5.13" "false"
test_comparison "4.19" "5.13" "true"
test_comparison "5.100" "5.13" "false"
test_comparison "5.2" "5.13" "true"

echo ""
echo "Testing edge cases with three-part versions:"
# Test with full kernel version (e.g., 5.15.0)
kernel_version="5.15.0"
trimmed_version=$(echo "$kernel_version" | cut -d. -f1-2)
echo "Full version: $kernel_version, trimmed: $trimmed_version"
test_comparison "$trimmed_version" "5.13" "false"