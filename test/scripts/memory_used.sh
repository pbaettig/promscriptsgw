#!/bin/bash

echo "memory_used: $(free | awk '/^Mem:/ {print $3}')"