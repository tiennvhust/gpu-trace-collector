package dap

import "errors"

// » HPKE (RFC 9180) is what makes the two-aggregator split real. Without it the
// » client would have to send the helper's share to the helper directly; with it,
// » the client encrypts share 1 to the leader's public key and share 2 to the
// » helper's, uploads both to the leader, and the leader physically cannot open
// » the second one. One request from the device, two sealed envelopes.
// »
// » WHAT HPKE IS, in one line: standardised, misuse-resistant public-key
// » encryption of a single message — a KEM to establish a shared secret, a KDF to
// » derive keys, an AEAD to encrypt. It exists because "encrypt this to a public
// » key" was, astonishingly, not a solved interoperable problem before 2022, and
// » everyone rolled their own with subtly different results.
// »
// » You will also meet it in Apple's stack: Private Relay and OHTTP both use it,
// » and Apple's PIR work builds on the same primitives. Worth an hour even
// » outside this project.
// »
// » Spec: RFC 9180. DAP's usage: draft-ietf-ppm-dap §4.4.

// HPKEConfigID identifies one of an aggregator's HPKE keys.
//
// » An ID rather than the key itself, because keys ROTATE. An aggregator
// » publishes several configs at once; clients cache a config for a while and use
// » whichever they have; the aggregator must keep old private keys until the last
// » cached config expires. This is the same two-key rotation problem as
// » internal/tenant's API keys (EXERCISE 1 there), and the same solution:
// » overlapping validity, never a hard cutover.
// »
// » The operational question to answer in docs/PRIVACY.md: a client caches a
// » config for 24 hours and reports hourly. How long must you keep a retired
// » private key, and what happens to reports encrypted to a key you have already
// » deleted? (They are unrecoverable. Silently. Which is why key retention is a
// » data-loss question, not just a security one.)
type HPKEConfigID uint8

// HPKE algorithm identifiers from the RFC 9180 registries.
//
// » DAP mandates a specific ciphersuite rather than negotiating one, which is the
// » modern approach: negotiation is where protocols acquire downgrade attacks. The
// » client and aggregator either agree or they do not talk. Check the current
// » draft for the mandatory set.
const (
	KEMX25519HKDFSHA256  uint16 = 0x0020
	KDFHKDFSHA256        uint16 = 0x0001
	AEADAES128GCM        uint16 = 0x0001
	AEADChaCha20Poly1305 uint16 = 0x0003
)

// HPKEConfig is an aggregator's published public key and ciphersuite.
type HPKEConfig struct {
	ID        HPKEConfigID
	KEMID     uint16
	KDFID     uint16
	AEADID    uint16
	PublicKey []byte
}

// Encode serialises the config per draft §4.4.
//
// TODO(week3): implement.
func (c *HPKEConfig) Encode() ([]byte, error) { return nil, ErrTODO }

// DecodeHPKEConfig parses a config.
//
// TODO(week3): implement — reject a ciphersuite you do not support, and reject a
// public key whose length does not match the KEM. An X25519 key is 32 bytes; a
// 31-byte one is not "close enough".
func DecodeHPKEConfig(b []byte) (*HPKEConfig, error) { return nil, ErrTODO }

// HPKEKeypair is an aggregator's private key material.
//
// » Never log this, never put it in a config file, and never serialise it into an
// » error message. In deployment it comes from a secret store — see
// » deploy/k8s/secret.example.yaml for the pattern already used for Kafka
// » credentials. A leaked aggregator private key means one party can open both
// » shares, which collapses the entire two-aggregator trust model to zero.
type HPKEKeypair struct {
	Config     HPKEConfig
	PrivateKey []byte
}

// Seal encrypts plaintext to the config's public key, binding aad.
//
// TODO(week3): implement.
func Seal(cfg *HPKEConfig, plaintext, aad []byte) (*HPKECiphertext, error) {
	// EXERCISE-BEGIN
	// ─── EXERCISE 41: HPKE seal/open ────────────────────────────────────────
	// Go's standard library has every piece you need as of Go 1.24, so this is
	// dependency-free — which is worth doing deliberately, because assembling
	// HPKE from primitives once teaches you what the RFC is actually specifying:
	//   crypto/ecdh   — X25519 (ecdh.X25519())
	//   crypto/hkdf   — HKDF-SHA256 (new in Go 1.24)
	//   crypto/aes + crypto/cipher — AES-128-GCM
	//
	// Follow RFC 9180 §5.1.1 (SetupBaseS) and §6.1. The steps:
	//   1. generate an ephemeral X25519 keypair;
	//   2. DH with the recipient's public key → shared secret;
	//   3. ExtractAndExpand with the suite ID and the KEM context (enc ‖ pkR) →
	//      key and nonce;
	//   4. AEAD-seal the plaintext with the given aad;
	//   5. return enc (the ephemeral public key) plus the ciphertext.
	//
	// GET THE LABELS AND THE SUITE ID EXACTLY RIGHT. RFC 9180's labelled
	// extract/expand ("HPKE-v1" ‖ suite_id ‖ label ‖ info) is fussy, and a wrong
	// label produces a perfectly functional encryption scheme that no other
	// implementation can decrypt. Test against the RFC's Appendix A test vectors
	// for DHKEM(X25519, HKDF-SHA256), HKDF-SHA256, AES-128-GCM before wiring it
	// into anything — same discipline as the TurboSHAKE KAT in EXERCISE 17, and
	// for the same reason.
	//
	// If Go's crypto/hkdf generic signature slows you down, take the dependency
	// (github.com/cloudflare/circl/hpke) and come back to it. HPKE is not the
	// learning objective of this project; Prio3 and DAP are. Do not let it eat
	// week 3.
	//
	// THE AAD IS NOT DECORATION. DAP binds the task ID and the report metadata
	// into the AAD, so a ciphertext sealed for task A cannot be replayed into
	// task B, or reattached to a different report's metadata. Work out the attack
	// that becomes possible if you pass nil: a leader could take a client's
	// helper-share ciphertext, pair it with metadata of its choosing, and submit
	// it to a task with a smaller batch size — where the differencing attack
	// works. Write that in THREAT_MODEL.md.
	// ────────────────────────────────────────────────────────────────────────
	// EXERCISE-END
	return nil, ErrTODO
}

// Open decrypts a ciphertext with the keypair, checking aad.
//
// TODO(week3): implement« — see EXERCISE 41».
func Open(kp *HPKEKeypair, ct *HPKECiphertext, aad []byte) ([]byte, error) {
	return nil, ErrTODO
}

// ErrWrongConfig is returned when a ciphertext names a config the aggregator
// does not hold.
//
// » Distinguish this from a decryption failure in the metrics. "Client used a
// » retired key" is a rollout problem you fix by extending key retention;
// » "authentication failed" is either corruption or an attack. Same HTTP status,
// » completely different response, and you cannot tell them apart later if you
// » counted them together.
var ErrWrongConfig = errors.New("dap: no HPKE key for the requested config id")

// InputShareAAD builds the additional authenticated data for an input share.
//
// TODO(week3): implement — per draft §4.4.1: task ID, then the report metadata,
// then the public share. Both the client and the aggregator must build it
// identically or every decryption fails authentication with no clue why.
func InputShareAAD(taskID TaskID, md ReportMetadata, publicShare []byte) ([]byte, error) {
	return nil, ErrTODO
}
