/*
 * Minio Cloud Storage, (C) 2016, 2017 Minio, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cmd

import "testing"

// Test printing Gateway common message.
func TestPrintGatewayCommonMessage(t *testing.T) {
	apiEndpoints := []string{"127.0.0.1:9000"}
	printGatewayCommonMsg(apiEndpoints, "abcd1", "abcd123")
}

// Test print gateway startup message.
func TestPrintGatewayStartupMessage(t *testing.T) {
	apiEndpoints := []string{"127.0.0.1:9000"}
	printGatewayStartupMessage(apiEndpoints, "abcd1", "abcd123", "azure")
}
