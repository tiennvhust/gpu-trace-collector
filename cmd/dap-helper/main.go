// dap-helper: the DAP helper aggregator.
//
// » Simpler than the leader by design — see the note at the top of
// » internal/dap/helper.go on why the cheap role has to be cheap. It has no
// » aggregation driver and no public ingress: only the leader calls it.
// »
// » Which means the one thing this binary must get right is refusing the leader
// » when the leader is wrong. That is its entire reason to exist.
package main

import (
	"flag"
	"log"
)

func main() {
	cfgPath := flag.String("config", "configs/dap-helper.yaml", "path to config file")
	flag.Parse()

	// EXERCISE-BEGIN
	// ─── EXERCISE 69: the helper binary ─────────────────────────────────────
	//   1. Load config and HPKE keypairs from the environment.
	//   2. Build the TaskSet — INDEPENDENTLY of the leader's. Do not share a config
	//      file between them, not even in the compose file. If you can accidentally
	//      keep them in sync by sharing a file, you will never discover that the
	//      real deployment cannot, and you will not build the mismatch detection in
	//      EXERCISE 47 that a real deployment needs. Make the local setup have the
	//      same failure modes as production.
	//   3. dap.NewHelper(...), serve helper.Routes().
	//   4. AUTHENTICATE THE LEADER — mutual TLS or a bearer token from the
	//      environment. An unauthenticated helper is a free aggregation oracle:
	//      anyone can submit a job and learn whether reports verify. Then reject
	//      every unauthenticated request BEFORE parsing the body, so an
	//      unauthenticated caller cannot reach the codec at all.
	//   5. Metrics on a separate listener, same reasoning as the leader.
	//
	// Then, to make the local deployment honest: give the helper a DIFFERENT verify
	// key in its config than the leader has, run it, and watch every report fail
	// verification. That is what a real misconfiguration between two organisations
	// looks like, and you want to have seen the symptom once — because from the
	// outside it is indistinguishable from "every client is malicious", and knowing
	// to check the key first will save you an afternoon later.
	// ────────────────────────────────────────────────────────────────────────
	// EXERCISE-END

	log.Fatalf("EXERCISE 69: implement the helper binary (config %s)", *cfgPath)
}
