package cmd

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/minio/minio/cmd/logger"
	"github.com/minio/minio/pkg/madmin"
)

var (
	// error returned when transition storage-class already exists
	errTransitionStorageClassAlreadyExists = AdminError{
		Code:       "XMinioAdminStorageClassAlreadyExists",
		Message:    "Specified transition storage-class already exists",
		StatusCode: http.StatusConflict,
	}
	// error returned when transition storage-class is not found
	errTransitionStorageClassNotFound = AdminError{
		Code:       "XMinioAdminStorageClassNotFound",
		Message:    "Specified transition storage-class was not found",
		StatusCode: http.StatusNotFound,
	}
)

func (api adminAPIHandlers) AddStorageClassHandler(w http.ResponseWriter, r *http.Request) {
	ctx := newContext(r, w, "AddStorageClass")

	defer logger.AuditLog(w, r, "AddStorageClass", mustGetClaimsFromToken(r))

	objectAPI := newObjectLayerFn()
	if objectAPI == nil || globalNotificationSys == nil || globalTransitionStorageClassConfigMgr == nil {
		writeErrorResponseJSON(ctx, w, errorCodes.ToAPIErr(ErrServerNotInitialized), r.URL)
		return
	}

	var cfg madmin.TransitionStorageClassConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeErrorResponseJSON(ctx, w, toAdminAPIErr(ctx, err), r.URL)
		return
	}

	err := globalTransitionStorageClassConfigMgr.Add(cfg)
	if err != nil {
		writeErrorResponseJSON(ctx, w, toAdminAPIErr(ctx, err), r.URL)
		return
	}

	err = saveGlobalTransitionStorageClassConfig()
	if err != nil {
		writeErrorResponseJSON(ctx, w, toAdminAPIErr(ctx, err), r.URL)
		return
	}
	globalNotificationSys.LoadTransitionStorageClassConfig(ctx)

	writeSuccessNoContent(w)
}

func (api adminAPIHandlers) RemoveStorageClassHandler(w http.ResponseWriter, r *http.Request) {
	ctx := newContext(r, w, "RemoveStorageClass")

	defer logger.AuditLog(w, r, "RemoveStorageClass", mustGetClaimsFromToken(r))

	objectAPI := newObjectLayerFn()
	if objectAPI == nil || globalNotificationSys == nil || globalTransitionStorageClassConfigMgr == nil {
		writeErrorResponseJSON(ctx, w, errorCodes.ToAPIErr(ErrServerNotInitialized), r.URL)
		return
	}

	var vars = mux.Vars(r)
	scName := vars["name"]

	globalTransitionStorageClassConfigMgr.RemoveStorageClass(scName)
	err := saveGlobalTransitionStorageClassConfig()
	if err != nil {
		writeErrorResponseJSON(ctx, w, toAdminAPIErr(ctx, err), r.URL)
		return
	}
	globalNotificationSys.LoadTransitionStorageClassConfig(ctx)

	writeSuccessNoContent(w)
}

func (api adminAPIHandlers) ListStorageClassHandler(w http.ResponseWriter, r *http.Request) {
	ctx := newContext(r, w, "ListStorageClass")

	defer logger.AuditLog(w, r, "ListStorageClass", mustGetClaimsFromToken(r))

	objectAPI := newObjectLayerFn()
	if objectAPI == nil || globalNotificationSys == nil || globalTransitionStorageClassConfigMgr == nil {
		writeErrorResponseJSON(ctx, w, errorCodes.ToAPIErr(ErrServerNotInitialized), r.URL)
		return
	}

	storageClasses := globalTransitionStorageClassConfigMgr.ListStorageClasses()
	data, err := json.Marshal(storageClasses)
	if err != nil {
		writeErrorResponseJSON(ctx, w, toAdminAPIErr(ctx, err), r.URL)
		return
	}

	writeSuccessResponseJSON(w, data)
}

func (api adminAPIHandlers) EditStorageClassHandler(w http.ResponseWriter, r *http.Request) {
	ctx := newContext(r, w, "EditStorageClass")

	defer logger.AuditLog(w, r, "EditStorageClass", mustGetClaimsFromToken(r))

	objectAPI := newObjectLayerFn()
	if objectAPI == nil || globalNotificationSys == nil || globalTransitionStorageClassConfigMgr == nil {
		writeErrorResponseJSON(ctx, w, errorCodes.ToAPIErr(ErrServerNotInitialized), r.URL)
		return
	}

	// FIXME: edit should allow only creds to be updated
	var sc madmin.TransitionStorageClassConfig
	if err := json.NewDecoder(r.Body).Decode(&sc); err != nil {
		writeErrorResponseJSON(ctx, w, toAdminAPIErr(ctx, err), r.URL)
		return
	}
	if err := globalTransitionStorageClassConfigMgr.Edit(sc); err != nil {
		writeErrorResponseJSON(ctx, w, toAdminAPIErr(ctx, err), r.URL)
		return
	}

	if err := saveGlobalTransitionStorageClassConfig(); err != nil {
		writeErrorResponseJSON(ctx, w, toAdminAPIErr(ctx, err), r.URL)
		return
	}
	globalNotificationSys.LoadTransitionStorageClassConfig(ctx)

	writeSuccessNoContent(w)
}
