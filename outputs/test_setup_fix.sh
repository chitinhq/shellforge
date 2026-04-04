#!/usr/bin/env bash

# Test the actual setup.sh logic with simulated kernel versions
echo "Testing setup.sh kernel version check with different simulated kernels:"
echo "======================================================================"

# Create a temporary version of the compare function from setup.sh
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

# Simulate different kernel versions and test
simulate_kernel_check() {
  local kernel_version="$1"
  echo -n "Kernel $kernel_version: "
  
  if compare_kernel_versions "$kernel_version" "5.13"; then
    echo "WARN - Kernel $kernel_version — Landlock needs >= 5.13"
  else
    echo "OK - Kernel $kernel_version >= 5.13"
  fi
}

echo ""
echo "Test cases showing the fix for lexicographic vs numeric comparison:"
echo "-------------------------------------------------------------------"
echo "Old lexicographic comparison would say:"
echo "  '5.10' < '5.13' = true (correct)"
echo "  '5.9' < '5.13' = true (correct)"
echo "  '5.100' < '5.13' = true (WRONG! because '5.100' < '5.13' string-wise)"
echo ""
echo "New numeric comparison says:"
simulate_kernel_check "5.10"
simulate_kernel_check "5.9"
simulate_kernel_check "5.100"
simulate_kernel_check "5.13"
simulate_kernel_check "5.15"
simulate_kernel_check "6.1"
simulate_kernel_check "4.19"

echo ""
echo "Testing with actual uname output format (three parts):"
test_kernel="5.15.0-generic"
trimmed_version=$(echo "$test_kernel" | cut -d. -f1-2)
simulate_kernel_check "$trimmed_version"