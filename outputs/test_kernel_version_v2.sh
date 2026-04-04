#!/usr/bin/env bash

# Test the kernel version comparison logic (updated version)
compare_kernel_versions() {
  local v1="$1"
  local v2="$2"
  
  # Extract major and minor versions
  local v1_major=$(echo "$v1" | cut -d. -f1)
  local v1_minor=$(echo "$v1" | cut -d. -f2)
  local v2_major=$(echo "$v2" | cut -d. -f1)
  local v2_minor=$(echo "$v2" | cut -d. -f2)
  
  # Compare major version first, then minor
  if [[ "$v1_major" -lt "$v2_major" ]]; then
    return 0  # true: v1 < v2
  elif [[ "$v1_major" -gt "$v2_major" ]]; then
    return 1  # false: v1 > v2
  else
    # Same major version, compare minor
    if [[ "$v1_minor" -lt "$v2_minor" ]]; then
      return 0  # true: v1 < v2
    else
      return 1  # false: v1 >= v2
    fi
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
test_comparison "3.10" "5.13" "true"
test_comparison "5.13" "5.9" "false"
test_comparison "5.9" "5.9" "false"

echo ""
echo "Testing edge cases with three-part versions:"
# Test with full kernel version (e.g., 5.15.0)
kernel_version="5.15.0"
trimmed_version=$(echo "$kernel_version" | cut -d. -f1-2)
echo "Full version: $kernel_version, trimmed: $trimmed_version"
test_comparison "$trimmed_version" "5.13" "false"

echo ""
echo "Testing the actual current kernel:"
current_kernel=$(uname -r | cut -d. -f1-2)
echo "Current kernel version: $current_kernel"
if compare_kernel_versions "$current_kernel" "5.13"; then
  echo "Current kernel ($current_kernel) < 5.13"
else
  echo "Current kernel ($current_kernel) >= 5.13"
fi