// Package firestore provides access to the Cloud Firestore API.
//
// This package is DEPRECATED. Use package cloud.google.com/go/firestore instead.
//
// See https://cloud.google.com/firestore
//
// Usage example:
//
//   import "google.golang.org/api/firestore/v1"
//   ...
//   firestoreService, err := firestore.New(oauthHttpClient)
package firestore // import "google.golang.org/api/firestore/v1"

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	context "golang.org/x/net/context"
	ctxhttp "golang.org/x/net/context/ctxhttp"
	gensupport "google.golang.org/api/gensupport"
	googleapi "google.golang.org/api/googleapi"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// Always reference these packages, just in case the auto-generated code
// below doesn't.
var _ = bytes.NewBuffer
var _ = strconv.Itoa
var _ = fmt.Sprintf
var _ = json.NewDecoder
var _ = io.Copy
var _ = url.Parse
var _ = gensupport.MarshalJSON
var _ = googleapi.Version
var _ = errors.New
var _ = strings.Replace
var _ = context.Canceled
var _ = ctxhttp.Do

const apiId = "firestore:v1"
const apiName = "firestore"
const apiVersion = "v1"
const basePath = "https://firestore.googleapis.com/"

// OAuth2 scopes used by this API.
const (
	// View and manage your data across Google Cloud Platform services
	CloudPlatformScope = "https://www.googleapis.com/auth/cloud-platform"

	// View and manage your Google Cloud Datastore data
	DatastoreScope = "https://www.googleapis.com/auth/datastore"
)

func New(client *http.Client) (*Service, error) {
	if client == nil {
		return nil, errors.New("client is nil")
	}
	s := &Service{client: client, BasePath: basePath}
	s.Projects = NewProjectsService(s)
	return s, nil
}

type Service struct {
	client    *http.Client
	BasePath  string // API endpoint base URL
	UserAgent string // optional additional User-Agent fragment

	Projects *ProjectsService
}

func (s *Service) userAgent() string {
	if s.UserAgent == "" {
		return googleapi.UserAgent
	}
	return googleapi.UserAgent + " " + s.UserAgent
}

func NewProjectsService(s *Service) *ProjectsService {
	rs := &ProjectsService{s: s}
	rs.Locations = NewProjectsLocationsService(s)
	return rs
}

type ProjectsService struct {
	s *Service

	Locations *ProjectsLocationsService
}

func NewProjectsLocationsService(s *Service) *ProjectsLocationsService {
	rs := &ProjectsLocationsService{s: s}
	return rs
}

type ProjectsLocationsService struct {
	s *Service
}

// GoogleFirestoreAdminV1beta1ExportDocumentsMetadata: Metadata for
// ExportDocuments operations.
type GoogleFirestoreAdminV1beta1ExportDocumentsMetadata struct {
	// CollectionIds: Which collection ids are being exported.
	CollectionIds []string `json:"collectionIds,omitempty"`

	// EndTime: The time the operation ended, either successfully or
	// otherwise. Unset if
	// the operation is still active.
	EndTime string `json:"endTime,omitempty"`

	// OperationState: The state of the export operation.
	//
	// Possible values:
	//   "STATE_UNSPECIFIED" - Unspecified.
	//   "INITIALIZING" - Request is being prepared for processing.
	//   "PROCESSING" - Request is actively being processed.
	//   "CANCELLING" - Request is in the process of being cancelled after
	// user called
	// google.longrunning.Operations.CancelOperation on the operation.
	//   "FINALIZING" - Request has been processed and is in its
	// finalization stage.
	//   "SUCCESSFUL" - Request has completed successfully.
	//   "FAILED" - Request has finished being processed, but encountered an
	// error.
	//   "CANCELLED" - Request has finished being cancelled after user
	// called
	// google.longrunning.Operations.CancelOperation.
	OperationState string `json:"operationState,omitempty"`

	// OutputUriPrefix: Where the entities are being exported to.
	OutputUriPrefix string `json:"outputUriPrefix,omitempty"`

	// ProgressBytes: An estimate of the number of bytes processed.
	ProgressBytes *GoogleFirestoreAdminV1beta1Progress `json:"progressBytes,omitempty"`

	// ProgressDocuments: An estimate of the number of documents processed.
	ProgressDocuments *GoogleFirestoreAdminV1beta1Progress `json:"progressDocuments,omitempty"`

	// StartTime: The time that work began on the operation.
	StartTime string `json:"startTime,omitempty"`

	// ForceSendFields is a list of field names (e.g. "CollectionIds") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "CollectionIds") to include
	// in API requests with the JSON null value. By default, fields with
	// empty values are omitted from API requests. However, any field with
	// an empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *GoogleFirestoreAdminV1beta1ExportDocumentsMetadata) MarshalJSON() ([]byte, error) {
	type NoMethod GoogleFirestoreAdminV1beta1ExportDocumentsMetadata
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// GoogleFirestoreAdminV1beta1ExportDocumentsResponse: Returned in the
// google.longrunning.Operation response field.
type GoogleFirestoreAdminV1beta1ExportDocumentsResponse struct {
	// OutputUriPrefix: Location of the output files. This can be used to
	// begin an import
	// into Cloud Firestore (this project or another project) after the
	// operation
	// completes successfully.
	OutputUriPrefix string `json:"outputUriPrefix,omitempty"`

	// ForceSendFields is a list of field names (e.g. "OutputUriPrefix") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "OutputUriPrefix") to
	// include in API requests with the JSON null value. By default, fields
	// with empty values are omitted from API requests. However, any field
	// with an empty value appearing in NullFields will be sent to the
	// server as null. It is an error if a field in this list has a
	// non-empty value. This may be used to include null fields in Patch
	// requests.
	NullFields []string `json:"-"`
}

func (s *GoogleFirestoreAdminV1beta1ExportDocumentsResponse) MarshalJSON() ([]byte, error) {
	type NoMethod GoogleFirestoreAdminV1beta1ExportDocumentsResponse
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// GoogleFirestoreAdminV1beta1ImportDocumentsMetadata: Metadata for
// ImportDocuments operations.
type GoogleFirestoreAdminV1beta1ImportDocumentsMetadata struct {
	// CollectionIds: Which collection ids are being imported.
	CollectionIds []string `json:"collectionIds,omitempty"`

	// EndTime: The time the operation ended, either successfully or
	// otherwise. Unset if
	// the operation is still active.
	EndTime string `json:"endTime,omitempty"`

	// InputUriPrefix: The location of the documents being imported.
	InputUriPrefix string `json:"inputUriPrefix,omitempty"`

	// OperationState: The state of the import operation.
	//
	// Possible values:
	//   "STATE_UNSPECIFIED" - Unspecified.
	//   "INITIALIZING" - Request is being prepared for processing.
	//   "PROCESSING" - Request is actively being processed.
	//   "CANCELLING" - Request is in the process of being cancelled after
	// user called
	// google.longrunning.Operations.CancelOperation on the operation.
	//   "FINALIZING" - Request has been processed and is in its
	// finalization stage.
	//   "SUCCESSFUL" - Request has completed successfully.
	//   "FAILED" - Request has finished being processed, but encountered an
	// error.
	//   "CANCELLED" - Request has finished being cancelled after user
	// called
	// google.longrunning.Operations.CancelOperation.
	OperationState string `json:"operationState,omitempty"`

	// ProgressBytes: An estimate of the number of bytes processed.
	ProgressBytes *GoogleFirestoreAdminV1beta1Progress `json:"progressBytes,omitempty"`

	// ProgressDocuments: An estimate of the number of documents processed.
	ProgressDocuments *GoogleFirestoreAdminV1beta1Progress `json:"progressDocuments,omitempty"`

	// StartTime: The time that work began on the operation.
	StartTime string `json:"startTime,omitempty"`

	// ForceSendFields is a list of field names (e.g. "CollectionIds") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "CollectionIds") to include
	// in API requests with the JSON null value. By default, fields with
	// empty values are omitted from API requests. However, any field with
	// an empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *GoogleFirestoreAdminV1beta1ImportDocumentsMetadata) MarshalJSON() ([]byte, error) {
	type NoMethod GoogleFirestoreAdminV1beta1ImportDocumentsMetadata
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// GoogleFirestoreAdminV1beta1IndexOperationMetadata: Metadata for index
// operations. This metadata populates
// the metadata field of google.longrunning.Operation.
type GoogleFirestoreAdminV1beta1IndexOperationMetadata struct {
	// Cancelled: True if the [google.longrunning.Operation] was cancelled.
	// If the
	// cancellation is in progress, cancelled will be true
	// but
	// google.longrunning.Operation.done will be false.
	Cancelled bool `json:"cancelled,omitempty"`

	// DocumentProgress: Progress of the existing operation, measured in
	// number of documents.
	DocumentProgress *GoogleFirestoreAdminV1beta1Progress `json:"documentProgress,omitempty"`

	// EndTime: The time the operation ended, either successfully or
	// otherwise. Unset if
	// the operation is still active.
	EndTime string `json:"endTime,omitempty"`

	// Index: The index resource that this operation is acting on. For
	// example:
	// `projects/{project_id}/databases/{database_id}/indexes/{index
	// _id}`
	Index string `json:"index,omitempty"`

	// OperationType: The type of index operation.
	//
	// Possible values:
	//   "OPERATION_TYPE_UNSPECIFIED" - Unspecified. Never set by server.
	//   "CREATING_INDEX" - The operation is creating the index. Initiated
	// by a `CreateIndex` call.
	OperationType string `json:"operationType,omitempty"`

	// StartTime: The time that work began on the operation.
	StartTime string `json:"startTime,omitempty"`

	// ForceSendFields is a list of field names (e.g. "Cancelled") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "Cancelled") to include in
	// API requests with the JSON null value. By default, fields with empty
	// values are omitted from API requests. However, any field with an
	// empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *GoogleFirestoreAdminV1beta1IndexOperationMetadata) MarshalJSON() ([]byte, error) {
	type NoMethod GoogleFirestoreAdminV1beta1IndexOperationMetadata
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// GoogleFirestoreAdminV1beta1LocationMetadata: The metadata message for
// google.cloud.location.Location.metadata.
type GoogleFirestoreAdminV1beta1LocationMetadata struct {
}

// GoogleFirestoreAdminV1beta1Progress: Measures the progress of a
// particular metric.
type GoogleFirestoreAdminV1beta1Progress struct {
	// WorkCompleted: An estimate of how much work has been completed. Note
	// that this may be
	// greater than `work_estimated`.
	WorkCompleted int64 `json:"workCompleted,omitempty,string"`

	// WorkEstimated: An estimate of how much work needs to be performed.
	// Zero if the
	// work estimate is unavailable. May change as work progresses.
	WorkEstimated int64 `json:"workEstimated,omitempty,string"`

	// ForceSendFields is a list of field names (e.g. "WorkCompleted") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "WorkCompleted") to include
	// in API requests with the JSON null value. By default, fields with
	// empty values are omitted from API requests. However, any field with
	// an empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *GoogleFirestoreAdminV1beta1Progress) MarshalJSON() ([]byte, error) {
	type NoMethod GoogleFirestoreAdminV1beta1Progress
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// GoogleFirestoreAdminV1beta2FieldOperationMetadata: Metadata for
// google.longrunning.Operation results from
// FirestoreAdmin.UpdateField.
type GoogleFirestoreAdminV1beta2FieldOperationMetadata struct {
	// BytesProgress: The progress, in bytes, of this operation.
	BytesProgress *GoogleFirestoreAdminV1beta2Progress `json:"bytesProgress,omitempty"`

	// DocumentProgress: The progress, in documents, of this operation.
	DocumentProgress *GoogleFirestoreAdminV1beta2Progress `json:"documentProgress,omitempty"`

	// EndTime: The time this operation completed. Will be unset if
	// operation still in
	// progress.
	EndTime string `json:"endTime,omitempty"`

	// Field: The field resource that this operation is acting on. For
	// example:
	// `projects/{project_id}/databases/{database_id}/collectionGrou
	// ps/{collection_id}/fields/{field_path}`
	Field string `json:"field,omitempty"`

	// IndexConfigDeltas: A list of IndexConfigDelta, which describe the
	// intent of this
	// operation.
	IndexConfigDeltas []*GoogleFirestoreAdminV1beta2IndexConfigDelta `json:"indexConfigDeltas,omitempty"`

	// StartTime: The time this operation started.
	StartTime string `json:"startTime,omitempty"`

	// State: The state of the operation.
	//
	// Possible values:
	//   "OPERATION_STATE_UNSPECIFIED" - Unspecified.
	//   "INITIALIZING" - Request is being prepared for processing.
	//   "PROCESSING" - Request is actively being processed.
	//   "CANCELLING" - Request is in the process of being cancelled after
	// user called
	// google.longrunning.Operations.CancelOperation on the operation.
	//   "FINALIZING" - Request has been processed and is in its
	// finalization stage.
	//   "SUCCESSFUL" - Request has completed successfully.
	//   "FAILED" - Request has finished being processed, but encountered an
	// error.
	//   "CANCELLED" - Request has finished being cancelled after user
	// called
	// google.longrunning.Operations.CancelOperation.
	State string `json:"state,omitempty"`

	// ForceSendFields is a list of field names (e.g. "BytesProgress") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "BytesProgress") to include
	// in API requests with the JSON null value. By default, fields with
	// empty values are omitted from API requests. However, any field with
	// an empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *GoogleFirestoreAdminV1beta2FieldOperationMetadata) MarshalJSON() ([]byte, error) {
	type NoMethod GoogleFirestoreAdminV1beta2FieldOperationMetadata
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// GoogleFirestoreAdminV1beta2Index: Cloud Firestore indexes enable
// simple and complex queries against
// documents in a database.
type GoogleFirestoreAdminV1beta2Index struct {
	// Fields: The fields supported by this index.
	//
	// For composite indexes, this is always 2 or more fields.
	// The last field entry is always for the field path `__name__`. If,
	// on
	// creation, `__name__` was not specified as the last field, it will be
	// added
	// automatically with the same direction as that of the last field
	// defined. If
	// the final field in a composite index is not directional, the
	// `__name__`
	// will be ordered ASCENDING (unless explicitly specified).
	//
	// For single field indexes, this will always be exactly one entry with
	// a
	// field path equal to the field path of the associated field.
	Fields []*GoogleFirestoreAdminV1beta2IndexField `json:"fields,omitempty"`

	// Name: Output only.
	// A server defined name for this index.
	// The form of this name for composite indexes will
	// be:
	// `projects/{project_id}/databases/{database_id}/collectionGroups/{c
	// ollection_id}/indexes/{composite_index_id}`
	// For single field indexes, this field will be empty.
	Name string `json:"name,omitempty"`

	// QueryScope: Indexes with a collection query scope specified allow
	// queries
	// against a collection that is the child of a specific document,
	// specified at
	// query time, and that has the same collection id.
	//
	// Indexes with a collection group query scope specified allow queries
	// against
	// all collections descended from a specific document, specified at
	// query
	// time, and that have the same collection id as this index.
	//
	// Possible values:
	//   "QUERY_SCOPE_UNSPECIFIED" - The query scope is unspecified. Not a
	// valid option.
	//   "COLLECTION" - Indexes with a collection query scope specified
	// allow queries
	// against a collection that is the child of a specific document,
	// specified
	// at query time, and that has the collection id specified by the index.
	QueryScope string `json:"queryScope,omitempty"`

	// State: Output only.
	// The serving state of the index.
	//
	// Possible values:
	//   "STATE_UNSPECIFIED" - The state is unspecified.
	//   "CREATING" - The index is being created.
	// There is an active long-running operation for the index.
	// The index is updated when writing a document.
	// Some index data may exist.
	//   "READY" - The index is ready to be used.
	// The index is updated when writing a document.
	// The index is fully populated from all stored documents it applies to.
	//   "NEEDS_REPAIR" - The index was being created, but something went
	// wrong.
	// There is no active long-running operation for the index,
	// and the most recently finished long-running operation failed.
	// The index is not updated when writing a document.
	// Some index data may exist.
	// Use the google.longrunning.Operations API to determine why the
	// operation
	// that last attempted to create this index failed, then re-create
	// the
	// index.
	State string `json:"state,omitempty"`

	// ForceSendFields is a list of field names (e.g. "Fields") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "Fields") to include in API
	// requests with the JSON null value. By default, fields with empty
	// values are omitted from API requests. However, any field with an
	// empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *GoogleFirestoreAdminV1beta2Index) MarshalJSON() ([]byte, error) {
	type NoMethod GoogleFirestoreAdminV1beta2Index
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// GoogleFirestoreAdminV1beta2IndexConfigDelta: Information about an
// index configuration change.
type GoogleFirestoreAdminV1beta2IndexConfigDelta struct {
	// ChangeType: Specifies how the index is changing.
	//
	// Possible values:
	//   "CHANGE_TYPE_UNSPECIFIED" - The type of change is not specified or
	// known.
	//   "ADD" - The single field index is being added.
	//   "REMOVE" - The single field index is being removed.
	ChangeType string `json:"changeType,omitempty"`

	// Index: The index being changed.
	Index *GoogleFirestoreAdminV1beta2Index `json:"index,omitempty"`

	// ForceSendFields is a list of field names (e.g. "ChangeType") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "ChangeType") to include in
	// API requests with the JSON null value. By default, fields with empty
	// values are omitted from API requests. However, any field with an
	// empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *GoogleFirestoreAdminV1beta2IndexConfigDelta) MarshalJSON() ([]byte, error) {
	type NoMethod GoogleFirestoreAdminV1beta2IndexConfigDelta
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// GoogleFirestoreAdminV1beta2IndexField: A field in an index.
// The field_path describes which field is indexed, the value_mode
// describes
// how the field value is indexed.
type GoogleFirestoreAdminV1beta2IndexField struct {
	// ArrayConfig: Indicates that this field supports operations on
	// `array_value`s.
	//
	// Possible values:
	//   "ARRAY_CONFIG_UNSPECIFIED" - The index does not support additional
	// array queries.
	//   "CONTAINS" - The index supports array containment queries.
	ArrayConfig string `json:"arrayConfig,omitempty"`

	// FieldPath: Can be __name__.
	// For single field indexes, this must match the name of the field or
	// may
	// be omitted.
	FieldPath string `json:"fieldPath,omitempty"`

	// Order: Indicates that this field supports ordering by the specified
	// order or
	// comparing using =, <, <=, >, >=.
	//
	// Possible values:
	//   "ORDER_UNSPECIFIED" - The ordering is unspecified. Not a valid
	// option.
	//   "ASCENDING" - The field is ordered by ascending field value.
	//   "DESCENDING" - The field is ordered by descending field value.
	Order string `json:"order,omitempty"`

	// ForceSendFields is a list of field names (e.g. "ArrayConfig") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "ArrayConfig") to include
	// in API requests with the JSON null value. By default, fields with
	// empty values are omitted from API requests. However, any field with
	// an empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *GoogleFirestoreAdminV1beta2IndexField) MarshalJSON() ([]byte, error) {
	type NoMethod GoogleFirestoreAdminV1beta2IndexField
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// GoogleFirestoreAdminV1beta2Progress: Describes the progress of the
// operation.
// Unit of work is generic and must be interpreted based on where
// Progress
// is used.
type GoogleFirestoreAdminV1beta2Progress struct {
	// CompletedWork: The amount of work completed.
	CompletedWork int64 `json:"completedWork,omitempty,string"`

	// EstimatedWork: The amount of work estimated.
	EstimatedWork int64 `json:"estimatedWork,omitempty,string"`

	// ForceSendFields is a list of field names (e.g. "CompletedWork") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "CompletedWork") to include
	// in API requests with the JSON null value. By default, fields with
	// empty values are omitted from API requests. However, any field with
	// an empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *GoogleFirestoreAdminV1beta2Progress) MarshalJSON() ([]byte, error) {
	type NoMethod GoogleFirestoreAdminV1beta2Progress
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// ListLocationsResponse: The response message for
// Locations.ListLocations.
type ListLocationsResponse struct {
	// Locations: A list of locations that matches the specified filter in
	// the request.
	Locations []*Location `json:"locations,omitempty"`

	// NextPageToken: The standard List next-page token.
	NextPageToken string `json:"nextPageToken,omitempty"`

	// ServerResponse contains the HTTP response code and headers from the
	// server.
	googleapi.ServerResponse `json:"-"`

	// ForceSendFields is a list of field names (e.g. "Locations") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "Locations") to include in
	// API requests with the JSON null value. By default, fields with empty
	// values are omitted from API requests. However, any field with an
	// empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *ListLocationsResponse) MarshalJSON() ([]byte, error) {
	type NoMethod ListLocationsResponse
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// Location: A resource that represents Google Cloud Platform location.
type Location struct {
	// DisplayName: The friendly name for this location, typically a nearby
	// city name.
	// For example, "Tokyo".
	DisplayName string `json:"displayName,omitempty"`

	// Labels: Cross-service attributes for the location. For example
	//
	//     {"cloud.googleapis.com/region": "us-east1"}
	Labels map[string]string `json:"labels,omitempty"`

	// LocationId: The canonical id for this location. For example:
	// "us-east1".
	LocationId string `json:"locationId,omitempty"`

	// Metadata: Service-specific metadata. For example the available
	// capacity at the given
	// location.
	Metadata googleapi.RawMessage `json:"metadata,omitempty"`

	// Name: Resource name for the location, which may vary between
	// implementations.
	// For example: "projects/example-project/locations/us-east1"
	Name string `json:"name,omitempty"`

	// ServerResponse contains the HTTP response code and headers from the
	// server.
	googleapi.ServerResponse `json:"-"`

	// ForceSendFields is a list of field names (e.g. "DisplayName") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "DisplayName") to include
	// in API requests with the JSON null value. By default, fields with
	// empty values are omitted from API requests. However, any field with
	// an empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *Location) MarshalJSON() ([]byte, error) {
	type NoMethod Location
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// method id "firestore.projects.locations.get":

type ProjectsLocationsGetCall struct {
	s            *Service
	name         string
	urlParams_   gensupport.URLParams
	ifNoneMatch_ string
	ctx_         context.Context
	header_      http.Header
}

// Get: Gets information about a location.
func (r *ProjectsLocationsService) Get(name string) *ProjectsLocationsGetCall {
	c := &ProjectsLocationsGetCall{s: r.s, urlParams_: make(gensupport.URLParams)}
	c.name = name
	return c
}

// Fields allows partial responses to be retrieved. See
// https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *ProjectsLocationsGetCall) Fields(s ...googleapi.Field) *ProjectsLocationsGetCall {
	c.urlParams_.Set("fields", googleapi.CombineFields(s))
	return c
}

// IfNoneMatch sets the optional parameter which makes the operation
// fail if the object's ETag matches the given value. This is useful for
// getting updates only after the object has changed since the last
// request. Use googleapi.IsNotModified to check whether the response
// error from Do is the result of In-None-Match.
func (c *ProjectsLocationsGetCall) IfNoneMatch(entityTag string) *ProjectsLocationsGetCall {
	c.ifNoneMatch_ = entityTag
	return c
}

// Context sets the context to be used in this call's Do method. Any
// pending HTTP request will be aborted if the provided context is
// canceled.
func (c *ProjectsLocationsGetCall) Context(ctx context.Context) *ProjectsLocationsGetCall {
	c.ctx_ = ctx
	return c
}

// Header returns an http.Header that can be modified by the caller to
// add HTTP headers to the request.
func (c *ProjectsLocationsGetCall) Header() http.Header {
	if c.header_ == nil {
		c.header_ = make(http.Header)
	}
	return c.header_
}

func (c *ProjectsLocationsGetCall) doRequest(alt string) (*http.Response, error) {
	reqHeaders := make(http.Header)
	for k, v := range c.header_ {
		reqHeaders[k] = v
	}
	reqHeaders.Set("User-Agent", c.s.userAgent())
	if c.ifNoneMatch_ != "" {
		reqHeaders.Set("If-None-Match", c.ifNoneMatch_)
	}
	var body io.Reader = nil
	c.urlParams_.Set("alt", alt)
	urls := googleapi.ResolveRelative(c.s.BasePath, "v1/{+name}")
	urls += "?" + c.urlParams_.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	req.Header = reqHeaders
	googleapi.Expand(req.URL, map[string]string{
		"name": c.name,
	})
	return gensupport.SendRequest(c.ctx_, c.s.client, req)
}

// Do executes the "firestore.projects.locations.get" call.
// Exactly one of *Location or error will be non-nil. Any non-2xx status
// code is an error. Response headers are in either
// *Location.ServerResponse.Header or (if a response was returned at
// all) in error.(*googleapi.Error).Header. Use googleapi.IsNotModified
// to check whether the returned error was because
// http.StatusNotModified was returned.
func (c *ProjectsLocationsGetCall) Do(opts ...googleapi.CallOption) (*Location, error) {
	gensupport.SetOptions(c.urlParams_, opts...)
	res, err := c.doRequest("json")
	if res != nil && res.StatusCode == http.StatusNotModified {
		if res.Body != nil {
			res.Body.Close()
		}
		return nil, &googleapi.Error{
			Code:   res.StatusCode,
			Header: res.Header,
		}
	}
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	ret := &Location{
		ServerResponse: googleapi.ServerResponse{
			Header:         res.Header,
			HTTPStatusCode: res.StatusCode,
		},
	}
	target := &ret
	if err := gensupport.DecodeResponse(target, res); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Gets information about a location.",
	//   "flatPath": "v1/projects/{projectsId}/locations/{locationsId}",
	//   "httpMethod": "GET",
	//   "id": "firestore.projects.locations.get",
	//   "parameterOrder": [
	//     "name"
	//   ],
	//   "parameters": {
	//     "name": {
	//       "description": "Resource name for the location.",
	//       "location": "path",
	//       "pattern": "^projects/[^/]+/locations/[^/]+$",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "v1/{+name}",
	//   "response": {
	//     "$ref": "Location"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/cloud-platform",
	//     "https://www.googleapis.com/auth/datastore"
	//   ]
	// }

}

// method id "firestore.projects.locations.list":

type ProjectsLocationsListCall struct {
	s            *Service
	name         string
	urlParams_   gensupport.URLParams
	ifNoneMatch_ string
	ctx_         context.Context
	header_      http.Header
}

// List: Lists information about the supported locations for this
// service.
func (r *ProjectsLocationsService) List(name string) *ProjectsLocationsListCall {
	c := &ProjectsLocationsListCall{s: r.s, urlParams_: make(gensupport.URLParams)}
	c.name = name
	return c
}

// Filter sets the optional parameter "filter": The standard list
// filter.
func (c *ProjectsLocationsListCall) Filter(filter string) *ProjectsLocationsListCall {
	c.urlParams_.Set("filter", filter)
	return c
}

// PageSize sets the optional parameter "pageSize": The standard list
// page size.
func (c *ProjectsLocationsListCall) PageSize(pageSize int64) *ProjectsLocationsListCall {
	c.urlParams_.Set("pageSize", fmt.Sprint(pageSize))
	return c
}

// PageToken sets the optional parameter "pageToken": The standard list
// page token.
func (c *ProjectsLocationsListCall) PageToken(pageToken string) *ProjectsLocationsListCall {
	c.urlParams_.Set("pageToken", pageToken)
	return c
}

// Fields allows partial responses to be retrieved. See
// https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *ProjectsLocationsListCall) Fields(s ...googleapi.Field) *ProjectsLocationsListCall {
	c.urlParams_.Set("fields", googleapi.CombineFields(s))
	return c
}

// IfNoneMatch sets the optional parameter which makes the operation
// fail if the object's ETag matches the given value. This is useful for
// getting updates only after the object has changed since the last
// request. Use googleapi.IsNotModified to check whether the response
// error from Do is the result of In-None-Match.
func (c *ProjectsLocationsListCall) IfNoneMatch(entityTag string) *ProjectsLocationsListCall {
	c.ifNoneMatch_ = entityTag
	return c
}

// Context sets the context to be used in this call's Do method. Any
// pending HTTP request will be aborted if the provided context is
// canceled.
func (c *ProjectsLocationsListCall) Context(ctx context.Context) *ProjectsLocationsListCall {
	c.ctx_ = ctx
	return c
}

// Header returns an http.Header that can be modified by the caller to
// add HTTP headers to the request.
func (c *ProjectsLocationsListCall) Header() http.Header {
	if c.header_ == nil {
		c.header_ = make(http.Header)
	}
	return c.header_
}

func (c *ProjectsLocationsListCall) doRequest(alt string) (*http.Response, error) {
	reqHeaders := make(http.Header)
	for k, v := range c.header_ {
		reqHeaders[k] = v
	}
	reqHeaders.Set("User-Agent", c.s.userAgent())
	if c.ifNoneMatch_ != "" {
		reqHeaders.Set("If-None-Match", c.ifNoneMatch_)
	}
	var body io.Reader = nil
	c.urlParams_.Set("alt", alt)
	urls := googleapi.ResolveRelative(c.s.BasePath, "v1/{+name}/locations")
	urls += "?" + c.urlParams_.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	req.Header = reqHeaders
	googleapi.Expand(req.URL, map[string]string{
		"name": c.name,
	})
	return gensupport.SendRequest(c.ctx_, c.s.client, req)
}

// Do executes the "firestore.projects.locations.list" call.
// Exactly one of *ListLocationsResponse or error will be non-nil. Any
// non-2xx status code is an error. Response headers are in either
// *ListLocationsResponse.ServerResponse.Header or (if a response was
// returned at all) in error.(*googleapi.Error).Header. Use
// googleapi.IsNotModified to check whether the returned error was
// because http.StatusNotModified was returned.
func (c *ProjectsLocationsListCall) Do(opts ...googleapi.CallOption) (*ListLocationsResponse, error) {
	gensupport.SetOptions(c.urlParams_, opts...)
	res, err := c.doRequest("json")
	if res != nil && res.StatusCode == http.StatusNotModified {
		if res.Body != nil {
			res.Body.Close()
		}
		return nil, &googleapi.Error{
			Code:   res.StatusCode,
			Header: res.Header,
		}
	}
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	ret := &ListLocationsResponse{
		ServerResponse: googleapi.ServerResponse{
			Header:         res.Header,
			HTTPStatusCode: res.StatusCode,
		},
	}
	target := &ret
	if err := gensupport.DecodeResponse(target, res); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Lists information about the supported locations for this service.",
	//   "flatPath": "v1/projects/{projectsId}/locations",
	//   "httpMethod": "GET",
	//   "id": "firestore.projects.locations.list",
	//   "parameterOrder": [
	//     "name"
	//   ],
	//   "parameters": {
	//     "filter": {
	//       "description": "The standard list filter.",
	//       "location": "query",
	//       "type": "string"
	//     },
	//     "name": {
	//       "description": "The resource that owns the locations collection, if applicable.",
	//       "location": "path",
	//       "pattern": "^projects/[^/]+$",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "pageSize": {
	//       "description": "The standard list page size.",
	//       "format": "int32",
	//       "location": "query",
	//       "type": "integer"
	//     },
	//     "pageToken": {
	//       "description": "The standard list page token.",
	//       "location": "query",
	//       "type": "string"
	//     }
	//   },
	//   "path": "v1/{+name}/locations",
	//   "response": {
	//     "$ref": "ListLocationsResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/cloud-platform",
	//     "https://www.googleapis.com/auth/datastore"
	//   ]
	// }

}

// Pages invokes f for each page of results.
// A non-nil error returned from f will halt the iteration.
// The provided context supersedes any context provided to the Context method.
func (c *ProjectsLocationsListCall) Pages(ctx context.Context, f func(*ListLocationsResponse) error) error {
	c.ctx_ = ctx
	defer c.PageToken(c.urlParams_.Get("pageToken")) // reset paging to original point
	for {
		x, err := c.Do()
		if err != nil {
			return err
		}
		if err := f(x); err != nil {
			return err
		}
		if x.NextPageToken == "" {
			return nil
		}
		c.PageToken(x.NextPageToken)
	}
}
