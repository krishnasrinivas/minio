/*
 * MinIO Cloud Storage, (C) 2016, 2017, 2018, 2019 MinIO, Inc.
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

import (
	"net/http"

	"github.com/gorilla/mux"
)

const (
	adminAPIPathPrefix = "/minio/admin"
)

// adminAPIHandlers provides HTTP handlers for MinIO admin API.
type adminAPIHandlers struct {
}

// registerAdminRouter - Add handler functions for each service REST API routes.
func registerAdminRouter(router *mux.Router, enableConfigOps, enableIAMOps bool) {

	adminAPI := adminAPIHandlers{}
	// Admin router
	adminRouter := router.PathPrefix(adminAPIPathPrefix).Subrouter()

	// Version handler
	adminRouter.Methods(http.MethodGet).Path("/version").HandlerFunc(httpRecordTraffic(httpTraceAll(adminAPI.VersionHandler)))

	adminV1Router := adminRouter.PathPrefix("/v1").Subrouter()

	/// Service operations

	// Service status
	adminV1Router.Methods(http.MethodGet).Path("/service").HandlerFunc(httpRecordTraffic(httpTraceAll(adminAPI.ServiceStatusHandler)))

	// Service restart and stop - TODO
	adminV1Router.Methods(http.MethodPost).Path("/service").HandlerFunc(httpRecordTraffic(httpTraceAll(adminAPI.ServiceStopNRestartHandler)))

	// Info operations
	adminV1Router.Methods(http.MethodGet).Path("/info").HandlerFunc(httpRecordTraffic(httpTraceAll(adminAPI.ServerInfoHandler)))

	if globalIsDistXL || globalIsXL {
		/// Heal operations

		// Heal processing endpoint.
		adminV1Router.Methods(http.MethodPost).Path("/heal/").HandlerFunc(httpRecordTraffic(httpTraceAll(adminAPI.HealHandler)))
		adminV1Router.Methods(http.MethodPost).Path("/heal/{bucket}").HandlerFunc(httpRecordTraffic(httpTraceAll(adminAPI.HealHandler)))
		adminV1Router.Methods(http.MethodPost).Path("/heal/{bucket}/{prefix:.*}").HandlerFunc(httpRecordTraffic(httpTraceAll(adminAPI.HealHandler)))

		adminV1Router.Methods(http.MethodPost).Path("/background-heal/status").HandlerFunc(httpRecordTraffic(httpTraceAll(adminAPI.BackgroundHealStatusHandler)))

		/// Health operations

	}
	// Performance command - return performance details based on input type
	adminV1Router.Methods(http.MethodGet).Path("/performance").HandlerFunc(httpRecordTraffic(httpTraceAll(adminAPI.PerfInfoHandler))).Queries("perfType", "{perfType:.*}")

	// Profiling operations
	adminV1Router.Methods(http.MethodPost).Path("/profiling/start").HandlerFunc(httpRecordTraffic(httpTraceAll(adminAPI.StartProfilingHandler))).
		Queries("profilerType", "{profilerType:.*}")
	adminV1Router.Methods(http.MethodGet).Path("/profiling/download").HandlerFunc(httpRecordTraffic(httpTraceAll(adminAPI.DownloadProfilingHandler)))

	/// Config operations
	if enableConfigOps {
		// Get config
		adminV1Router.Methods(http.MethodGet).Path("/config").HandlerFunc(httpRecordTraffic(httpTraceHdrs(adminAPI.GetConfigHandler)))
		// Set config
		adminV1Router.Methods(http.MethodPut).Path("/config").HandlerFunc(httpRecordTraffic(httpTraceHdrs(adminAPI.SetConfigHandler)))

		// Get config keys/values
		adminV1Router.Methods(http.MethodGet).Path("/config-keys").HandlerFunc(httpRecordTraffic(httpTraceHdrs(adminAPI.GetConfigKeysHandler)))
		// Set config keys/values
		adminV1Router.Methods(http.MethodPut).Path("/config-keys").HandlerFunc(httpRecordTraffic(httpTraceHdrs(adminAPI.SetConfigKeysHandler)))
	}

	if enableIAMOps {
		// -- IAM APIs --

		// Add policy IAM
		adminV1Router.Methods(http.MethodPut).Path("/add-canned-policy").HandlerFunc(httpRecordTraffic(httpTraceHdrs(adminAPI.AddCannedPolicy))).Queries("name",
			"{name:.*}")

		// Add user IAM
		adminV1Router.Methods(http.MethodPut).Path("/add-user").HandlerFunc(httpRecordTraffic(httpTraceHdrs(adminAPI.AddUser))).Queries("accessKey", "{accessKey:.*}")
		adminV1Router.Methods(http.MethodPut).Path("/set-user-policy").HandlerFunc(httpRecordTraffic(httpTraceHdrs(adminAPI.SetUserPolicy))).
			Queries("accessKey", "{accessKey:.*}").Queries("name", "{name:.*}")
		adminV1Router.Methods(http.MethodPut).Path("/set-user-status").HandlerFunc(httpRecordTraffic(httpTraceHdrs(adminAPI.SetUserStatus))).
			Queries("accessKey", "{accessKey:.*}").Queries("status", "{status:.*}")

		// Remove policy IAM
		adminV1Router.Methods(http.MethodDelete).Path("/remove-canned-policy").HandlerFunc(httpRecordTraffic(httpTraceHdrs(adminAPI.RemoveCannedPolicy))).Queries("name", "{name:.*}")

		// Remove user IAM
		adminV1Router.Methods(http.MethodDelete).Path("/remove-user").HandlerFunc(httpRecordTraffic(httpTraceHdrs(adminAPI.RemoveUser))).Queries("accessKey", "{accessKey:.*}")

		// List users
		adminV1Router.Methods(http.MethodGet).Path("/list-users").HandlerFunc(httpRecordTraffic(httpTraceHdrs(adminAPI.ListUsers)))

		// List policies
		adminV1Router.Methods(http.MethodGet).Path("/list-canned-policies").HandlerFunc(httpRecordTraffic(httpTraceHdrs(adminAPI.ListCannedPolicies)))
	}

	// -- Top APIs --
	// Top locks
	adminV1Router.Methods(http.MethodGet).Path("/top/locks").HandlerFunc(httpRecordTraffic(httpTraceHdrs(adminAPI.TopLocksHandler)))

	// HTTP Trace
	adminV1Router.Methods(http.MethodGet).Path("/trace").HandlerFunc(httpRecordTraffic(adminAPI.TraceHandler))
	// If none of the routes match, return error.
	adminV1Router.NotFoundHandler = http.HandlerFunc(httpRecordTraffic(httpTraceHdrs(notFoundHandlerJSON)))
}
