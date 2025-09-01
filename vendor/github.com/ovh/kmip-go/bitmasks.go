package kmip

import "github.com/ovh/kmip-go/ttlv"

func init() {
	ttlv.RegisterBitmask[CryptographicUsageMask](
		TagCryptographicUsageMask,
		"Sign",
		"Verify",
		"Encrypt",
		"Decrypt",
		"WrapKey",
		"UnwrapKey",
		"Export",
		"MACGenerate",
		"DeriveKey",
		"ContentCommitment",
		"KeyAgreement",
		"CertificateSign",
		"CRLSign",
		"GenerateCryptogram",
		"ValidateCryptogram",
		"TranslateEncrypt",
		"TranslateDecrypt",
		"TranslateWrap",
		"TranslateUnwrap",
	)
	ttlv.RegisterBitmask[StorageStatusMask](
		TagStorageStatusMask,
		"OnlineStorage",
		"ArchivalStorage",
	)
}

type CryptographicUsageMask int32

const (
	CryptographicUsageSign CryptographicUsageMask = 1 << iota
	CryptographicUsageVerify
	CryptographicUsageEncrypt
	CryptographicUsageDecrypt
	CryptographicUsageWrapKey
	CryptographicUsageUnwrapKey
	CryptographicUsageExport
	CryptographicUsageMACGenerate
	CryptographicUsageDeriveKey
	CryptographicUsageContentCommitment
	CryptographicUsageKeyAgreement
	CryptographicUsageCertificateSign
	CryptographicUsageCRLSign
	CryptographicUsageGenerateCryptogram
	CryptographicUsageValidateCryptogram
	CryptographicUsageTranslateEncrypt
	CryptographicUsageTranslateDecrypt
	CryptographicUsageTranslateWrap
	CryptographicUsageTranslateUnwrap
)

func (mask CryptographicUsageMask) MarshalText() ([]byte, error) {
	return []byte(ttlv.BitmaskStr(mask, " | ")), nil
}

type StorageStatusMask int32

const (
	StorageStatusOnlineStorage StorageStatusMask = 1 << iota
	StorageStatusArchivalStorage
)

func (mask StorageStatusMask) MarshalText() ([]byte, error) {
	return []byte(ttlv.BitmaskStr(mask, " | ")), nil
}
