/*
 * MinIO Cloud Storage, (C) 2019 MinIO, Inc.
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
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"
	miniogo "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/tags"
	xhttp "github.com/minio/minio/cmd/http"
	"github.com/minio/minio/cmd/logger"
	sse "github.com/minio/minio/pkg/bucket/encryption"
	"github.com/minio/minio/pkg/bucket/lifecycle"
	"github.com/minio/minio/pkg/event"
	"github.com/minio/minio/pkg/hash"
	"github.com/minio/minio/pkg/madmin"
	"github.com/minio/minio/pkg/s3select"
)

const (
	// Disabled means the lifecycle rule is inactive
	Disabled = "Disabled"

	TransitionStatus       = "transition-status"
	TransitionedObjectName = "transitioned-object"
	TransitionStorageClass = "transition-sc"
)

// LifecycleSys - Bucket lifecycle subsystem.
type LifecycleSys struct{}

// Get - gets lifecycle config associated to a given bucket name.
func (sys *LifecycleSys) Get(bucketName string) (lc *lifecycle.Lifecycle, err error) {
	if globalIsGateway {
		objAPI := newObjectLayerFn()
		if objAPI == nil {
			return nil, errServerNotInitialized
		}

		return nil, BucketLifecycleNotFound{Bucket: bucketName}
	}

	return globalBucketMetadataSys.GetLifecycleConfig(bucketName)
}

// NewLifecycleSys - creates new lifecycle system.
func NewLifecycleSys() *LifecycleSys {
	return &LifecycleSys{}
}

type transitionState struct {
	// add future metrics here
	transitionCh chan ObjectInfo
}

func (t *transitionState) queueTransitionTask(oi ObjectInfo) {
	select {
	case t.transitionCh <- oi:
	default:
	}
}

var (
	globalTransitionState      *transitionState
	globalTransitionConcurrent = runtime.GOMAXPROCS(0) / 2
)

func newTransitionState() *transitionState {

	// fix minimum concurrent transition to 1 for single CPU setup
	if globalTransitionConcurrent == 0 {
		globalTransitionConcurrent = 1
	}
	ts := &transitionState{
		transitionCh: make(chan ObjectInfo, 10000),
	}
	go func() {
		<-GlobalContext.Done()
		close(ts.transitionCh)
	}()
	return ts
}

// addWorker creates a new worker to process tasks
func (t *transitionState) addWorker(ctx context.Context, objectAPI ObjectLayer) {
	// Add a new worker.
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case oi, ok := <-t.transitionCh:
				if !ok {
					return
				}
				if err := transitionObject(ctx, objectAPI, oi); err != nil {
					logger.LogIf(ctx, err)
				}
			}
		}
	}()
}

func initBackgroundTransition(ctx context.Context, objectAPI ObjectLayer) {
	if globalTransitionState == nil {
		return
	}

	// Start with globalTransitionConcurrent.
	for i := 0; i < globalTransitionConcurrent; i++ {
		globalTransitionState.addWorker(ctx, objectAPI)
	}
}

func validateLifecycleTransition(ctx context.Context, bucket string, lfc *lifecycle.Lifecycle) error {
	for _, rule := range lfc.Rules {
		if rule.Transition.StorageClass != "" {
			sameTarget, destbucket, err := validateTransitionDestination(ctx, bucket, rule.Transition.StorageClass)
			if err != nil {
				return err
			}
			if sameTarget && destbucket == bucket {
				return fmt.Errorf("Transition destination cannot be the same as the source bucket")
			}
		}
	}
	return nil
}

// validateTransitionDestination returns error if transition destination bucket missing or not configured
// It also returns true if transition destination is same as this server.
func validateTransitionDestination(ctx context.Context, bucket string, targetLabel string) (bool, string, error) {
	tgt := globalBucketTargetSys.GetRemoteTargetWithLabel(ctx, bucket, targetLabel)
	if tgt == nil {
		return false, "", BucketRemoteTargetNotFound{Bucket: bucket}
	}
	arn, err := madmin.ParseARN(tgt.Arn)
	if err != nil {
		return false, "", BucketRemoteTargetNotFound{Bucket: bucket}
	}
	if arn.Type != madmin.ILMService {
		return false, "", BucketRemoteArnTypeInvalid{}
	}
	clnt := globalBucketTargetSys.GetRemoteTargetClient(ctx, tgt.Arn)
	if clnt == nil {
		return false, "", BucketRemoteTargetNotFound{Bucket: bucket}
	}
	if found, _ := clnt.BucketExists(ctx, arn.Bucket); !found {
		return false, "", BucketRemoteDestinationNotFound{Bucket: arn.Bucket}
	}
	sameTarget, _ := isLocalHost(clnt.EndpointURL().Hostname(), clnt.EndpointURL().Port(), globalMinioPort)
	return sameTarget, arn.Bucket, nil
}

// return true if ARN representing transition storage class is present in a active rule
// for the lifecycle configured on this bucket
func transitionSCInUse(ctx context.Context, lfc *lifecycle.Lifecycle, bucket, arnStr string) bool {
	tgtLabel := globalBucketTargetSys.GetRemoteLabelWithArn(ctx, bucket, arnStr)
	if tgtLabel == "" {
		return false
	}
	for _, rule := range lfc.Rules {
		if rule.Status == Disabled {
			continue
		}
		if rule.Transition.StorageClass != "" && rule.Transition.StorageClass == tgtLabel {
			return true
		}
	}
	return false
}

// set PutObjectOptions for PUT operation to transition data to target cluster
func putTransitionOpts(objInfo ObjectInfo) (putOpts miniogo.PutObjectOptions) {
	meta := make(map[string]string)

	tag, err := tags.ParseObjectTags(objInfo.UserTags)
	if err != nil {
		return
	}
	putOpts = miniogo.PutObjectOptions{
		UserMetadata:    meta,
		UserTags:        tag.ToMap(),
		ContentType:     objInfo.ContentType,
		ContentEncoding: objInfo.ContentEncoding,
		StorageClass:    objInfo.StorageClass,
		Internal: miniogo.AdvancedPutOptions{
			SourceVersionID: objInfo.VersionID,
			SourceMTime:     objInfo.ModTime,
			SourceETag:      objInfo.ETag,
		},
	}
	if mode, ok := objInfo.UserDefined[xhttp.AmzObjectLockMode]; ok {
		rmode := miniogo.RetentionMode(mode)
		putOpts.Mode = rmode
	}
	if retainDateStr, ok := objInfo.UserDefined[xhttp.AmzObjectLockRetainUntilDate]; ok {
		rdate, err := time.Parse(time.RFC3339, retainDateStr)
		if err != nil {
			return
		}
		putOpts.RetainUntilDate = rdate
	}
	if lhold, ok := objInfo.UserDefined[xhttp.AmzObjectLockLegalHold]; ok {
		putOpts.LegalHold = miniogo.LegalHoldStatus(lhold)
	}

	return
}

// handle deletes of transitioned objects or object versions when one of the following is true:
// 1. temporarily restored copies of objects (restored with the PostRestoreObject API) expired.
// 2. life cycle expiry date is met on the object.
// 3. Object is removed through DELETE api call
func deleteTransitionedObject(ctx context.Context, objectAPI ObjectLayer, bucket, object string, lcOpts lifecycle.ObjectOpts, action lifecycle.Action, tgtObjName, transitionSC string, expiryEvent bool) error {
	// TODO: remove this
	lc, err := globalLifecycleSys.Get(bucket)
	if err != nil {
		return err
	}
	sc := getLifeCycleTransitionStorageClass(ctx, lc, bucket, lcOpts)
	// TODO: ^^ not needed

	//TODO: replace with transitionSC here
	tgt := globalBucketTargetSys.GetRemoteTargetWithLabel(ctx, bucket, sc)
	if tgt == nil {
		return fmt.Errorf("remote target not configured")
	}
	tgtClient := globalBucketTargetSys.GetRemoteTargetClient(ctx, tgt.Arn)
	if tgtClient == nil {
		return fmt.Errorf("remote target not configured")
	}

	var opts ObjectOptions
	opts.Versioned = globalBucketVersioningSys.Enabled(bucket)
	opts.VersionID = lcOpts.VersionID
	switch action {
	case lifecycle.DeleteRestoredAction, lifecycle.DeleteRestoredVersionAction:
		// delete locally restored copy of object or object version
		// from the source, while leaving metadata behind. The data on
		// transitioned tier lies untouched and still accessible
		opts.TransitionStatus = lcOpts.TransitionStatus
		_, err = objectAPI.DeleteObject(ctx, bucket, object, opts)
		return err
	case lifecycle.DeleteAction, lifecycle.DeleteVersionAction:
		// When an object is past expiry, delete the data from transitioned tier and
		// metadata from source
		if err := tgtClient.RemoveObject(context.Background(), tgt.TargetBucket, tgtObjName, miniogo.RemoveObjectOptions{}); err != nil {
			logger.LogIf(ctx, err)
		}
		// Delete metadata on source, now that transition tier has been cleaned up.
		_, err = objectAPI.DeleteObject(ctx, bucket, object, opts)
		if err != nil {
			return err
		}
		if expiryEvent {
			eventName := event.ObjectRemovedDelete
			if lcOpts.DeleteMarker {
				eventName = event.ObjectRemovedDeleteMarkerCreated
			}
			objInfo := ObjectInfo{
				Name:         object,
				VersionID:    lcOpts.VersionID,
				DeleteMarker: lcOpts.DeleteMarker,
			}
			// Notify object deleted event.
			sendEvent(eventArgs{
				EventName:  eventName,
				BucketName: bucket,
				Object:     objInfo,
				Host:       "Internal: [ILM-EXPIRY]",
			})
		}
	}

	// should never reach here
	return nil
}
func genTransitionObjName() (string, error) {
	u, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	us := u.String()
	obj := fmt.Sprintf("%s/%s/%s", us[0:2], us[2:4], us)
	return obj, nil
}

// transition object to target specified by the transition ARN. When an object is transitioned to another
// storage specified by the transition ARN, the metadata is left behind on source cluster and original content
// is moved to the transition tier. Note that in the case of encrypted objects, entire encrypted stream is moved
// to the transition tier without decrypting or re-encrypting.
func transitionObject(ctx context.Context, objectAPI ObjectLayer, oi ObjectInfo) error {
	lc, err := globalLifecycleSys.Get(oi.Bucket)
	if err != nil {
		return err
	}
	lcOpts := lifecycle.ObjectOpts{
		Name:     oi.Name,
		UserTags: oi.UserTags,
	}
	sc := getLifeCycleTransitionStorageClass(ctx, lc, oi.Bucket, lcOpts)
	tgt := globalBucketTargetSys.GetRemoteTargetWithLabel(ctx, oi.Bucket, sc)
	if tgt == nil {
		return fmt.Errorf("remote target not configured")
	}
	opts := ObjectOptions{Transition: TransitionOptions{
		Status:       lifecycle.TransitionPending,
		StorageClass: sc,
	},
		VersionID: oi.VersionID,
		Versioned: globalBucketVersioningSys.Enabled(oi.Bucket),
	}
	return objectAPI.TransitionObject(ctx, oi.Bucket, oi.Name, opts)
}

// getLifeCycleTransitionStorageClass returns storage class for transition target
func getLifeCycleTransitionStorageClass(ctx context.Context, lc *lifecycle.Lifecycle, bucket string, obj lifecycle.ObjectOpts) string {
	for _, rule := range lc.FilterActionableRules(obj) {
		if rule.Transition.StorageClass != "" {
			return rule.Transition.StorageClass
		}
	}
	return ""
}

// getTransitionedObjectReader returns a reader from the transitioned tier.
func getTransitionedObjectReader(ctx context.Context, bucket, object string, rs *HTTPRangeSpec, h http.Header, oi ObjectInfo, opts ObjectOptions) (gr *GetObjectReader, err error) {
	//TODO: get target client given transition-sc
	tgt := globalBucketTargetSys.GetRemoteTargetWithLabel(ctx, bucket, oi.TransitionStorageClass)
	if tgt == nil {
		return nil, fmt.Errorf("remote target not configured")
	}

	tgtClient := globalBucketTargetSys.GetRemoteTargetClient(ctx, tgt.Arn)
	if tgtClient == nil {
		return nil, fmt.Errorf("remote target not configured")
	}
	fn, off, length, err := NewGetObjectReader(rs, oi, opts)
	if err != nil {
		return nil, ErrorRespToObjectError(err, bucket, object)
	}
	gopts := miniogo.GetObjectOptions{}

	// get correct offsets for encrypted object
	if off >= 0 && length >= 0 {
		if err := gopts.SetRange(off, off+length-1); err != nil {
			return nil, ErrorRespToObjectError(err, bucket, object)
		}
	}
	reader, err := tgtClient.GetObject(ctx, tgtClient.Bucket, oi.transitionedObjName, gopts)
	if err != nil {
		return nil, err
	}
	closeReader := func() { reader.Close() }

	return fn(reader, h, opts.CheckPrecondFn, closeReader)
}

// RestoreRequestType represents type of restore.
type RestoreRequestType string

const (
	// SelectRestoreRequest specifies select request. This is the only valid value
	SelectRestoreRequest RestoreRequestType = "SELECT"
)

// Encryption specifies encryption setting on restored bucket
type Encryption struct {
	EncryptionType sse.SSEAlgorithm `xml:"EncryptionType"`
	KMSContext     string           `xml:"KMSContext,omitempty"`
	KMSKeyID       string           `xml:"KMSKeyId,omitempty"`
}

// MetadataEntry denotes name and value.
type MetadataEntry struct {
	Name  string `xml:"Name"`
	Value string `xml:"Value"`
}

// S3Location specifies s3 location that receives result of a restore object request
type S3Location struct {
	BucketName   string          `xml:"BucketName,omitempty"`
	Encryption   Encryption      `xml:"Encryption,omitempty"`
	Prefix       string          `xml:"Prefix,omitempty"`
	StorageClass string          `xml:"StorageClass,omitempty"`
	Tagging      *tags.Tags      `xml:"Tagging,omitempty"`
	UserMetadata []MetadataEntry `xml:"UserMetadata"`
}

// OutputLocation specifies bucket where object needs to be restored
type OutputLocation struct {
	S3 S3Location `xml:"S3,omitempty"`
}

// IsEmpty returns true if output location not specified.
func (o *OutputLocation) IsEmpty() bool {
	return o.S3.BucketName == ""
}

// SelectParameters specifies sql select parameters
type SelectParameters struct {
	s3select.S3Select
}

// IsEmpty returns true if no select parameters set
func (sp *SelectParameters) IsEmpty() bool {
	return sp == nil || sp.S3Select == s3select.S3Select{}
}

var (
	selectParamsXMLName = "SelectParameters"
)

// UnmarshalXML - decodes XML data.
func (sp *SelectParameters) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	// Essentially the same as S3Select barring the xml name.
	if start.Name.Local == selectParamsXMLName {
		start.Name = xml.Name{Space: "", Local: "SelectRequest"}
	}
	return sp.S3Select.UnmarshalXML(d, start)
}

// RestoreObjectRequest - xml to restore a transitioned object
type RestoreObjectRequest struct {
	XMLName          xml.Name           `xml:"http://s3.amazonaws.com/doc/2006-03-01/ RestoreRequest" json:"-"`
	Days             int                `xml:"Days,omitempty"`
	Type             RestoreRequestType `xml:"Type,omitempty"`
	Tier             string             `xml:"Tier,-"`
	Description      string             `xml:"Description,omitempty"`
	SelectParameters *SelectParameters  `xml:"SelectParameters,omitempty"`
	OutputLocation   OutputLocation     `xml:"OutputLocation,omitempty"`
}

// Maximum 2MiB size per restore object request.
const maxRestoreObjectRequestSize = 2 << 20

// parseRestoreRequest parses RestoreObjectRequest from xml
func parseRestoreRequest(reader io.Reader) (*RestoreObjectRequest, error) {
	req := RestoreObjectRequest{}
	if err := xml.NewDecoder(io.LimitReader(reader, maxRestoreObjectRequestSize)).Decode(&req); err != nil {
		return nil, err
	}
	return &req, nil
}

// validate a RestoreObjectRequest as per AWS S3 spec https://docs.aws.amazon.com/AmazonS3/latest/API/API_RestoreObject.html
func (r *RestoreObjectRequest) validate(ctx context.Context, objAPI ObjectLayer) error {
	if r.Type != SelectRestoreRequest && !r.SelectParameters.IsEmpty() {
		return fmt.Errorf("Select parameters can only be specified with SELECT request type")
	}
	if r.Type == SelectRestoreRequest && r.SelectParameters.IsEmpty() {
		return fmt.Errorf("SELECT restore request requires select parameters to be specified")
	}

	if r.Type != SelectRestoreRequest && !r.OutputLocation.IsEmpty() {
		return fmt.Errorf("OutputLocation required only for SELECT request type")
	}
	if r.Type == SelectRestoreRequest && r.OutputLocation.IsEmpty() {
		return fmt.Errorf("OutputLocation required for SELECT requests")
	}

	if r.Days != 0 && r.Type == SelectRestoreRequest {
		return fmt.Errorf("Days cannot be specified with SELECT restore request")
	}
	if r.Days == 0 && r.Type != SelectRestoreRequest {
		return fmt.Errorf("restoration days should be at least 1")
	}
	// Check if bucket exists.
	if !r.OutputLocation.IsEmpty() {
		if _, err := objAPI.GetBucketInfo(ctx, r.OutputLocation.S3.BucketName); err != nil {
			return err
		}
		if r.OutputLocation.S3.Prefix == "" {
			return fmt.Errorf("Prefix is a required parameter in OutputLocation")
		}
		if r.OutputLocation.S3.Encryption.EncryptionType != xhttp.AmzEncryptionAES {
			return NotImplemented{}
		}
	}
	return nil
}

// set ObjectOptions for PUT call to restore temporary copy of transitioned data
func putRestoreOpts(bucket, object string, rreq *RestoreObjectRequest, objInfo ObjectInfo) (putOpts ObjectOptions) {
	meta := make(map[string]string)
	sc := rreq.OutputLocation.S3.StorageClass
	if sc == "" {
		sc = objInfo.StorageClass
	}
	meta[strings.ToLower(xhttp.AmzStorageClass)] = sc

	if rreq.Type == SelectRestoreRequest {
		for _, v := range rreq.OutputLocation.S3.UserMetadata {
			if !strings.HasPrefix("x-amz-meta", strings.ToLower(v.Name)) {
				meta["x-amz-meta-"+v.Name] = v.Value
				continue
			}
			meta[v.Name] = v.Value
		}
		meta[xhttp.AmzObjectTagging] = rreq.OutputLocation.S3.Tagging.String()
		if rreq.OutputLocation.S3.Encryption.EncryptionType != "" {
			meta[xhttp.AmzServerSideEncryption] = xhttp.AmzEncryptionAES
		}
		return ObjectOptions{
			Versioned:        globalBucketVersioningSys.Enabled(bucket),
			VersionSuspended: globalBucketVersioningSys.Suspended(bucket),
			UserDefined:      meta,
		}
	}
	for k, v := range objInfo.UserDefined {
		meta[k] = v
	}
	meta[xhttp.AmzObjectTagging] = objInfo.UserTags

	return ObjectOptions{
		Versioned:        globalBucketVersioningSys.Enabled(bucket),
		VersionSuspended: globalBucketVersioningSys.Suspended(bucket),
		UserDefined:      meta,
		VersionID:        objInfo.VersionID,
		MTime:            objInfo.ModTime,
		Expires:          objInfo.Expires,
	}
}

var (
	errRestoreHDRMissing   = fmt.Errorf("x-amz-restore header not found")
	errRestoreHDRMalformed = fmt.Errorf("x-amz-restore header malformed")
)

// parse x-amz-restore header from user metadata to get the status of ongoing request and expiry of restoration
// if any. This header value is of format: ongoing-request=true|false, expires=time
func parseRestoreHeaderFromMeta(meta map[string]string) (ongoing bool, expiry time.Time, err error) {
	restoreHdr, ok := meta[xhttp.AmzRestore]
	if !ok {
		return ongoing, expiry, errRestoreHDRMissing
	}
	rslc := strings.SplitN(restoreHdr, ",", 2)
	if len(rslc) != 2 {
		return ongoing, expiry, errRestoreHDRMalformed
	}
	rstatusSlc := strings.SplitN(rslc[0], "=", 2)
	if len(rstatusSlc) != 2 {
		return ongoing, expiry, errRestoreHDRMalformed
	}
	rExpSlc := strings.SplitN(rslc[1], "=", 2)
	if len(rExpSlc) != 2 {
		return ongoing, expiry, errRestoreHDRMalformed
	}

	expiry, err = time.Parse(http.TimeFormat, rExpSlc[1])
	if err != nil {
		return
	}
	return rstatusSlc[1] == "true", expiry, nil
}

// restoreTransitionedObject is similar to PostObjectRestore from AWS GLACIER
// storage class. When PostObjectRestore API is called, a temporary copy of the object
// is restored locally to the bucket on source cluster until the restore expiry date.
// The copy that was transitioned continues to reside in the transitioned tier.
func restoreTransitionedObject(ctx context.Context, bucket, object string, objAPI ObjectLayer, objInfo ObjectInfo, rreq *RestoreObjectRequest, restoreExpiry time.Time) error {
	var rs *HTTPRangeSpec
	gr, err := getTransitionedObjectReader(ctx, bucket, object, rs, http.Header{}, objInfo, ObjectOptions{
		VersionID: objInfo.VersionID})
	if err != nil {
		return err
	}
	defer gr.Close()
	hashReader, err := hash.NewReader(gr, objInfo.Size, "", "", objInfo.Size, globalCLIContext.StrictS3Compat)
	if err != nil {
		return err
	}
	pReader := NewPutObjReader(hashReader, nil, nil)
	opts := putRestoreOpts(bucket, object, rreq, objInfo)
	opts.UserDefined[xhttp.AmzRestore] = fmt.Sprintf("ongoing-request=%t, expiry-date=%s", false, restoreExpiry.Format(http.TimeFormat))
	if _, err := objAPI.PutObject(ctx, bucket, object, pReader, opts); err != nil {
		return err
	}

	return nil
}
