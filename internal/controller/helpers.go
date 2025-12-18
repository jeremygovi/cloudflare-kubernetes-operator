/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import "strings"

// Helper functions shared across controllers

func intPtrToInt(ptr *int, defaultVal int) int {
	if ptr != nil {
		return *ptr
	}
	return defaultVal
}

func boolPtrToBool(ptr *bool, defaultVal bool) *bool {
	if ptr != nil {
		return ptr
	}
	return &defaultVal
}

func uint16PtrToPtr(ptr *uint16) *uint16 {
	return ptr
}

func stringToPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	// Check if error message contains "not found" or similar indicators
	errorMsg := strings.ToLower(err.Error())
	return strings.Contains(errorMsg, "not found") ||
		strings.Contains(errorMsg, "404") ||
		strings.Contains(errorMsg, "could not be found") ||
		strings.Contains(errorMsg, "does not exist") ||
		strings.Contains(errorMsg, "81044") // Cloudflare error code for "Record does not exist"
}
