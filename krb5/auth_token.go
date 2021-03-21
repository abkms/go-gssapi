package krb5

/*
 * Derived from github.com/jcmturner/gokrb5/spnego/krb5Token.go
 *
 * The modified version adds functionality to verify an APReq message
 * to be used as part of a mutually-authenticated GSSAPI security
 * context.
 */

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/jcmturner/gofork/encoding/asn1"
	"github.com/jcmturner/gokrb5/v8/asn1tools"
	"github.com/jcmturner/gokrb5/v8/messages"
	"github.com/jcmturner/gokrb5/v8/service"

	"github.com/jake-scott/go-gssapi"
)

// GSSAPI KRB5 MechToken IDs.
const (
	TOK_ID_KRB_AP_REQ = "0100"
	TOK_ID_KRB_AP_REP = "0200"
	TOK_ID_KRB_ERROR  = "0300"
)

// KRB5Token context token implementation for GSSAPI.
type KRB5Token struct {
	OID      asn1.ObjectIdentifier
	tokID    []byte
	APReq    messages.APReq
	APRep    messages.APRep
	KRBError messages.KRBError
	settings *service.Settings
	context  context.Context
}

// Marshal a KRB5Token into a slice of bytes.
func (m *KRB5Token) Marshal() ([]byte, error) {
	// Create the header
	b, _ := asn1.Marshal(m.OID)
	b = append(b, m.tokID...)
	var tb []byte
	var err error
	switch hex.EncodeToString(m.tokID) {
	case TOK_ID_KRB_AP_REQ:
		tb, err = m.APReq.Marshal()
		if err != nil {
			return []byte{}, fmt.Errorf("gssapi: error marshalling AP_REQ for MechToken: %v", err)
		}
	case TOK_ID_KRB_AP_REP:
		return []byte{}, errors.New("gssapi: marshal of AP_REP GSSAPI MechToken not supported by go-gssapi")
	case TOK_ID_KRB_ERROR:
		return []byte{}, errors.New("gssapi: marshal of KRB_ERROR GSSAPI MechToken not supported by go-gssapi")
	}
	if err != nil {
		return []byte{}, fmt.Errorf("gssapi: error mashalling kerberos message within mech token: %v", err)
	}
	b = append(b, tb...)
	return asn1tools.AddASNAppTag(b, 0), nil
}

// Unmarshal a KRB5Token.
func (m *KRB5Token) Unmarshal(b []byte) error {
	var oid asn1.ObjectIdentifier
	r, err := asn1.UnmarshalWithParams(b, &oid, fmt.Sprintf("application,explicit,tag:%v", 0))
	if err != nil {
		return fmt.Errorf("gssapi: error unmarshalling KRB5Token OID: %v", err)
	}
	if !oid.Equal(OID()) {
		return fmt.Errorf("gssapi: error unmarshalling KRB5Token, OID is %s not %s", oid.String(), OID().String())
	}
	m.OID = oid
	if len(r) < 2 {
		return fmt.Errorf("gssapi: krb5token too short")
	}
	m.tokID = r[0:2]
	switch hex.EncodeToString(m.tokID) {
	case TOK_ID_KRB_AP_REQ:
		var a messages.APReq
		err = a.Unmarshal(r[2:])
		if err != nil {
			return fmt.Errorf("gssapi: error unmarshalling KRB5Token AP_REQ: %v", err)
		}
		m.APReq = a
	case TOK_ID_KRB_AP_REP:
		var a messages.APRep
		err = a.Unmarshal(r[2:])
		if err != nil {
			return fmt.Errorf("gssapi: error unmarshalling KRB5Token AP_REP: %v", err)
		}
		m.APRep = a
	case TOK_ID_KRB_ERROR:
		var a messages.KRBError
		err = a.Unmarshal(r[2:])
		if err != nil {
			return fmt.Errorf("gssapi: error unmarshalling KRB5Token KRBError: %v", err)
		}
		m.KRBError = a
	}
	return nil
}

/*

// Verify a KRB5Token.
func (m *KRB5Token) Verify() (bool, gssapi.Status) {
	switch hex.EncodeToString(m.tokID) {
	case TOK_ID_KRB_AP_REQ:
		ok, creds, err := service.VerifyAPREQ(&m.APReq, m.settings)
		if err != nil {
			return false, gssapi.Status{Code: gssapi.StatusDefectiveToken, Message: err.Error()}
		}
		if !ok {
			return false, gssapi.Status{Code: gssapi.StatusDefectiveCredential, Message: "KRB5_AP_REQ token not valid"}
		}
		m.context = context.Background()
		m.context = context.WithValue(m.context, ctxCredentials, creds)
		return true, gssapi.Status{Code: gssapi.StatusComplete}
	case TOK_ID_KRB_AP_REP:
		// Client side
		// TODO how to verify the AP_REP - not yet implemented
		return false, gssapi.Status{Code: gssapi.StatusFailure, Message: "verifying an AP_REP is not currently supported by gokrb5"}
	case TOK_ID_KRB_ERROR:
		if m.KRBError.MsgType != msgtype.KRB_ERROR {
			return false, gssapi.Status{Code: gssapi.StatusDefectiveToken, Message: "KRB5_Error token not valid"}
		}
		return true, gssapi.Status{Code: gssapi.StatusUnavailable}
	}
	return false, gssapi.Status{Code: gssapi.StatusDefectiveToken, Message: "unknown TOK_ID in KRB5 token"}
}

// IsAPReq tests if the MechToken contains an AP_REQ.
func (m *KRB5Token) IsAPReq() bool {
	if hex.EncodeToString(m.tokID) == TOK_ID_KRB_AP_REQ {
		return true
	}
	return false
}

// IsAPRep tests if the MechToken contains an AP_REP.
func (m *KRB5Token) IsAPRep() bool {
	if hex.EncodeToString(m.tokID) == TOK_ID_KRB_AP_REP {
		return true
	}
	return false
}

func (m *KRB5Token) IsKRBError() bool {
	if hex.EncodeToString(m.tokID) == TOK_ID_KRB_ERROR {
		return true
	}
	return false
}

// Context returns the KRB5 token's context which will contain any verify user identity information.
func (m *KRB5Token) Context() context.Context {
	return m.context
}
*/

// Create the GSSAPI checksum for the authenticator.  This isn't really
// a checksum, it is a way to carry GSSAPI level context infromation in
// the Kerberos AP-RREQ message. See RFC 4121 § 4.1.1
func newAuthenticatorChksum(flags gssapi.ContextFlag) []byte {
	// 24 octet minimum length, up to and including context-establishment flags
	a := make([]byte, 24)

	// 4-byte length of "channel binding" info, always 16 bytes
	binary.LittleEndian.PutUint32(a[:4], 16)

	// Octets 4..19: Channel binding info, left zero

	// Context-establishment flags
	binary.LittleEndian.PutUint32(a[20:24], uint32(flags))

	return a
}
