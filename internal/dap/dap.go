// Package dap implements the Distributed Aggregation Protocol,
// draft-ietf-ppm-dap: the HTTP protocol that carries Prio3 reports from clients
// to two aggregators and aggregate shares from the aggregators to a collector.
//
// » WHAT DAP ADDS TO VDAF, and the distinction is worth being precise about
// » because it comes up immediately in conversation. VDAF
// » (internal/vdaf/prio3) is pure maths: given a measurement, produce shares;
// » given shares, produce an aggregate. It says nothing about who talks to whom,
// » what happens when the helper is down, how you stop a report being counted
// » twice, or when a batch is allowed to be collected.
// »
// » DAP is all of that: four roles, the HTTP endpoints between them, the batch
// » rules that make the aggregate safe to publish, and the failure handling. It
// » is a DISTRIBUTED SYSTEMS specification with cryptography inside it, which is
// » why it is the part of this project that plays to your existing strengths.
// » Do not treat it as the boring wrapper around the interesting maths — the
// » anti-replay set, the batch-overlap rules and the aggregation-job state
// » machine are where a real deployment gets its privacy properties, and they are
// » ordinary systems engineering.
//
// » THE FOUR ROLES:
// »   CLIENT      the device. Shards a measurement, encrypts one input share to
// »               each aggregator's HPKE public key, uploads both to the leader.
// »   LEADER      an aggregator. Receives uploads, drives the aggregation
// »               protocol with the helper, holds one aggregate share, serves
// »               collection requests.
// »   HELPER      an aggregator. Responds to the leader; holds the other
// »               aggregate share. Run by a DIFFERENT ORGANISATION — this is the
// »               whole trust model, not a deployment detail.
// »   COLLECTOR   asks for an aggregate over a batch, gets one share from each
// »               aggregator, combines them. Only this role sees a result.
// »
// » Note the client uploads BOTH encrypted shares to the leader, which forwards
// » the helper's. The client makes one request, to one endpoint — because a
// » device on a metered radio should not have to reach two hosts, and because it
// » means the helper needs no public ingress. Bandwidth and operational
// » simplicity, driving a protocol decision. That reasoning recurs constantly in
// » device-to-cloud work.
//
// » Specs:
// »   draft-ietf-ppm-dap (currently -19) — the protocol
// »   draft-ietf-ppm-dap-taskprov — task provisioning; co-authored by S. Wang of
// »     Apple and C. Patton of Cloudflare. Read this one closely: it exists
// »     because two organisations agreeing on a task's parameters by email does
// »     not scale, and it is the most directly Apple-relevant document in the set.
// »   https://datatracker.ietf.org/doc/draft-ietf-ppm-dap/
//
// » Reference implementation to read for shape (not to copy): ISRG's janus.
// »   https://github.com/divviup/janus
package dap

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
)

// ErrTODO marks an unimplemented scaffold function.
var ErrTODO = errors.New("dap: not implemented«, see the EXERCISE block above this function»")

// Role identifies a DAP participant.
type Role uint8

// The four DAP roles.
const (
	RoleClient Role = iota
	RoleLeader
	RoleHelper
	RoleCollector
)

// String implements fmt.Stringer.
func (r Role) String() string {
	switch r {
	case RoleClient:
		return "client"
	case RoleLeader:
		return "leader"
	case RoleHelper:
		return "helper"
	case RoleCollector:
		return "collector"
	}
	return "unknown"
}

// IDLen is the length of a TaskID, ReportID or CollectionJobID in bytes.
const IDLen = 16

// TaskID identifies an aggregation task: one VDAF, one set of parameters, one
// pair of aggregators, one privacy budget.
//
// » THE TASK IS THE UNIT OF PRIVACY, and this is the sentence to remember from
// » this file. Everything else is scoped to it: which VDAF and parameters, which
// » two aggregators, the minimum batch size, the ε budget
// » (internal/dp.Ledger keys on this), the time precision, the HPKE keys. If you
// » find yourself wanting per-tenant or per-metric variation in any of those, the
// » answer is a new task, not a special case in the code.
type TaskID [IDLen]byte

// ReportID uniquely identifies one report within a task, and is the report's
// anti-replay key.
//
// » Client-chosen and random. That looks like a weakness — a malicious client
// » could reuse one — but reuse only lets a client suppress its OWN report, since
// » an aggregator that has seen an ID rejects the second copy. It cannot suppress
// » anyone else's, because it cannot guess their IDs. Worth working through:
// » "client-chosen identifier" is normally a red flag, and understanding exactly
// » why it is safe here is a good example of reasoning about a threat model rather
// » than applying a rule.
type ReportID [IDLen]byte

// CollectionJobID identifies one collection request.
type CollectionJobID [IDLen]byte

// String renders an ID as hex, for logs and URLs.
func (t TaskID) String() string { return hex.EncodeToString(t[:]) }

// String renders the report ID as hex.
func (r ReportID) String() string { return hex.EncodeToString(r[:]) }

// String renders the collection job ID as hex.
func (c CollectionJobID) String() string { return hex.EncodeToString(c[:]) }

// NewReportID returns a fresh random report ID.
func NewReportID() (ReportID, error) {
	var id ReportID
	if _, err := rand.Read(id[:]); err != nil {
		return id, fmt.Errorf("dap: generating report id: %w", err)
	}
	return id, nil
}

// ParseTaskID decodes a hex-encoded task ID.
func ParseTaskID(s string) (TaskID, error) {
	var id TaskID
	b, err := hex.DecodeString(s)
	if err != nil {
		return id, fmt.Errorf("dap: task id not hex: %w", err)
	}
	if len(b) != IDLen {
		return id, fmt.Errorf("dap: task id must be %d bytes, got %d", IDLen, len(b))
	}
	copy(id[:], b)
	return id, nil
}

// Media types for DAP messages, from the draft's IANA considerations section.
//
// » DAP uses distinct media types per message rather than one JSON envelope, so
// » a misrouted request fails at the HTTP layer instead of deserialising into
// » something plausible. Check these strings against the current draft; they
// » carry the version and change with it.
const (
	MediaTypeReport                = "application/dap-report"
	MediaTypeAggregationJobInitReq = "application/dap-aggregation-job-init-req"
	MediaTypeAggregationJobResp    = "application/dap-aggregation-job-resp"
	MediaTypeAggregateShareReq     = "application/dap-aggregate-share-req"
	MediaTypeAggregateShare        = "application/dap-aggregate-share"
	MediaTypeCollectReq            = "application/dap-collect-req"
	MediaTypeCollection            = "application/dap-collection"
	MediaTypeHPKEConfigList        = "application/dap-hpke-config-list"
)

// ProblemType values for DAP error responses (RFC 9457 problem details).
//
// » RFC 9457 rather than ad-hoc JSON, because the leader and helper are run by
// » DIFFERENT ORGANISATIONS: when the helper rejects a batch, the leader's
// » on-call engineer needs to know why without opening a support ticket. That is
// » the same reason internal/obs labels rejections with a reason — a rejection
// » without a machine-readable cause is an outage you debug by guessing.
const (
	ProblemInvalidMessage      = "urn:ietf:params:ppm:dap:error:invalidMessage"
	ProblemUnrecognizedTask    = "urn:ietf:params:ppm:dap:error:unrecognizedTask"
	ProblemReportRejected      = "urn:ietf:params:ppm:dap:error:reportRejected"
	ProblemReportTooEarly      = "urn:ietf:params:ppm:dap:error:reportTooEarly"
	ProblemBatchInvalid        = "urn:ietf:params:ppm:dap:error:batchInvalid"
	ProblemInvalidBatchSize    = "urn:ietf:params:ppm:dap:error:invalidBatchSize"
	ProblemBatchOverlap        = "urn:ietf:params:ppm:dap:error:batchOverlap"
	ProblemBatchQueriedTooMany = "urn:ietf:params:ppm:dap:error:batchQueriedMultipleTimes"
	ProblemStepMismatch        = "urn:ietf:params:ppm:dap:error:stepMismatch"
)

// Problem is an RFC 9457 problem-details response body.
type Problem struct {
	Type   string `json:"type"`
	Title  string `json:"title,omitempty"`
	Status int    `json:"status,omitempty"`
	Detail string `json:"detail,omitempty"`
	TaskID string `json:"taskid,omitempty"`
}

// Error implements error so handlers can return a Problem directly.
func (p *Problem) Error() string {
	if p.Detail != "" {
		return p.Type + ": " + p.Detail
	}
	return p.Type
}
