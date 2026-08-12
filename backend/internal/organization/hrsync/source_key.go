package hrsync

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
)

const MaxRawSourceIDLength = 128

type ObjectKind string

const (
	ObjectKindLegalEntity       ObjectKind = "legal_entity"
	ObjectKindManagementCompany ObjectKind = "management_company"
	ObjectKindManagementUnit    ObjectKind = "management_unit"
	ObjectKindLegalUnit         ObjectKind = "legal_unit"
	ObjectKindStructureNode     ObjectKind = "structure_node"
	ObjectKindPosition          ObjectKind = "position"
	ObjectKindEmployee          ObjectKind = "employee"
	ObjectKindAssignment        ObjectKind = "assignment"
)

var objectKinds = map[ObjectKind]struct{}{
	ObjectKindLegalEntity: {}, ObjectKindManagementCompany: {}, ObjectKindManagementUnit: {}, ObjectKindLegalUnit: {},
	ObjectKindStructureNode: {}, ObjectKindPosition: {}, ObjectKindEmployee: {}, ObjectKindAssignment: {},
}

var ErrSourceKeyInvalid = errors.New("org_sync_source_key_invalid")

// SourceKey 将源系统、对象类别和源 ID 一起纳入身份空间；原始 ID 不参与 String 输出。
type SourceKey struct {
	sourceSystemCode string
	objectKind       ObjectKind
	rawSourceID      string
}

func NewSourceKey(sourceSystemCode string, objectKind ObjectKind, rawSourceID string) (SourceKey, error) {
	sourceSystemCode = strings.TrimSpace(sourceSystemCode)
	rawSourceID = strings.TrimSpace(rawSourceID)
	_, kindOK := objectKinds[objectKind]
	if sourceSystemCode == "" || len(sourceSystemCode) > 64 || !kindOK || rawSourceID == "" || len(rawSourceID) > MaxRawSourceIDLength {
		return SourceKey{}, ErrSourceKeyInvalid
	}
	return SourceKey{sourceSystemCode: sourceSystemCode, objectKind: objectKind, rawSourceID: rawSourceID}, nil
}

func (key SourceKey) SourceSystemCode() string { return key.sourceSystemCode }
func (key SourceKey) ObjectKind() ObjectKind   { return key.objectKind }
func (key SourceKey) RawSourceID() string      { return key.rawSourceID }

func (key SourceKey) Digest() string {
	sum := sha256.Sum256([]byte(key.sourceSystemCode + "\x00" + string(key.objectKind) + "\x00" + key.rawSourceID))
	return hex.EncodeToString(sum[:12])
}

func (key SourceKey) String() string {
	if key.objectKind == "" {
		return "source_key[invalid]"
	}
	return "source_key[" + string(key.objectKind) + ":" + key.Digest() + "]"
}

func (key SourceKey) GoString() string { return key.String() }
