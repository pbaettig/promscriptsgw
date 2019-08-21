#!/bin/bash

echo "memory_used1: $(free | awk '/^Mem:/ {print $3}')"
echo "memory_used2: $(free | awk '/^Mem:/ {print $3}')"
echo "memory_used3: $(free | awk '/^Mem:/ {print $3}')"
echo "memory_used4: $(free | awk '/^Mem:/ {print $3}')"
echo "memory_used5: $(free | awk '/^Mem:/ {print $3}')"