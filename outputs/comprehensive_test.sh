#!/usr/bin/env bash

echo "Comprehensive test of kernel version comparison fix for issue #50"
echo "================================================================="
echo "Issue: kernel version comparison in setup.sh is lexicographic, not numeric"
echo ""

# The fixed function from setup.sh
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

echo "Test 1: Basic comparisons (should all pass)"
echo "-------------------------------------------"

test_passes=0
test_fails=0

run_test() {
  local v1="$1"
  local v2="$2"
  local expected="$3"  # "lt" for less than, "ge" for greater or equal
  local description="$4"
  
  if compare_kernel_versions "$v1" "$v2"; then
    result="lt"
  else
    result="ge"
  fi
  
  if [[ "$result" == "$expected" ]]; then
    echo "✓ $description: $v1 < $v2 = $result"
    ((test_passes++))
  else
    echo "✗ $description: $v1 < $v2 = $result (expected: $expected)"
    ((test_fails++))
  fi
}

# Critical test cases that would fail with lexicographic comparison
run_test "5.100" "5.13" "ge" "5.100 >= 5.13 (lexicographic would say 5.100 < 5.13!)"
run_test "5.10" "5.9" "ge" "5.10 >= 5.9 (lexicographic would say 5.10 < 5.9!)"
run_test "5.2" "5.10" "lt" "5.2 < 5.10"
run_test "5.9" "5.10" "lt" "5.9 < 5.10"

# Normal test cases
run_test "5.9" "5.13" "lt" "5.9 < 5.13"
run_test "5.10" "5.13" "lt" "5.10 < 5.13"
run_test "5.13" "5.13" "ge" "5.13 >= 5.13"
run_test "5.15" "5.13" "ge" "5.15 >= 5.13"
run_test "6.1" "5.13" "ge" "6.1 >= 5.13"
run_test "4.19" "5.13" "lt" "4.19 < 5.13"

echo ""
echo "Test 2: Real-world kernel version formats"
echo "-----------------------------------------"

# Simulate what uname -r returns
simulate_uname() {
  local full_version="$1"
  local trimmed=$(echo "$full_version" | cut -d. -f1-2)
  echo "$trimmed"
}

echo "Testing with actual uname -r output formats:"
run_test "$(simulate_uname '5.15.0-generic')" "5.13" "ge" "5.15.0-generic → 5.15 >= 5.13"
run_test "$(simulate_uname '5.10.147-generic')" "5.13" "lt" "5.10.147-generic → 5.10 < 5.13"
run_test "$(simulate_uname '6.1.0-20-generic')" "5.13" "ge" "6.1.0-20-generic → 6.1 >= 5.13"

echo ""
echo "Test 3: Edge cases"
echo "------------------"
run_test "3.10" "5.13" "lt" "3.10 < 5.13 (different major version)"
run_test "10.1" "5.13" "ge" "10.1 >= 5.13 (major version jump)"
run_test "5.0" "5.13" "lt" "5.0 < 5.13"
run_test "5.999" "5.13" "ge" "5.999 >= 5.13"

echo ""
echo "Summary:"
echo "--------"
echo "Total tests: $((test_passes + test_fails))"
echo "Passed: $test_passes"
echo "Failed: $test_fails"

if [[ $test_fails -eq 0 ]]; then
  echo "✅ All tests passed! The fix correctly handles numeric kernel version comparison."
else
  echo "❌ Some tests failed. The fix needs improvement."
  exit 1
fi