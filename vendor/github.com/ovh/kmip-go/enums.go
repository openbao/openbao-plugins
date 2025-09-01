package kmip

import "github.com/ovh/kmip-go/ttlv"

//nolint:funlen // There's not cleaner way to do it
func init() {
	ttlv.RegisterEnum(TagResultStatus, map[ResultStatus]string{
		ResultStatusSuccess:          "Success",
		ResultStatusOperationFailed:  "OperationFailed",
		ResultStatusOperationPending: "OperationPending",
		ResultStatusOperationUndone:  "OperationUndone",
	})
	ttlv.RegisterEnum(TagResultReason, map[ResultReason]string{
		// ReasonNone:                             "None",
		ResultReasonItemNotFound:                     "ItemNotFound",
		ResultReasonResponseTooLarge:                 "ResponseTooLarge",
		ResultReasonAuthenticationNotSuccessful:      "AuthenticationNotSuccessful",
		ResultReasonInvalidMessage:                   "InvalidMessage",
		ResultReasonOperationNotSupported:            "OperationNotSupported",
		ResultReasonMissingData:                      "MissingData",
		ResultReasonInvalidField:                     "InvalidField",
		ResultReasonFeatureNotSupported:              "FeatureNotSupported",
		ResultReasonOperationCanceledByRequester:     "OperationCanceledByRequester",
		ResultReasonCryptographicFailure:             "CryptographicFailure",
		ResultReasonIllegalOperation:                 "IllegalOperation",
		ResultReasonPermissionDenied:                 "PermissionDenied",
		ResultReasonObjectarchived:                   "Objectarchived",
		ResultReasonIndexOutofBounds:                 "IndexOutofBounds",
		ResultReasonApplicationNamespaceNotSupported: "ApplicationNamespaceNotSupported",
		ResultReasonKeyFormatTypeNotSupported:        "KeyFormatTypeNotSupported",
		ResultReasonKeyCompressionTypeNotSupported:   "KeyCompressionTypeNotSupported",
		ResultReasonGeneralFailure:                   "GeneralFailure",
		// KMIP 1.1.
		ResultReasonEncodingOptionError: "EncodingOptionError",
		// KMIP 1.2.
		ResultReasonKeyValueNotPresent:  "KeyValueNotPresent",
		ResultReasonAttestationRequired: "AttestationRequired",
		ResultReasonAttestationFailed:   "AttestationFailed",
		// KMIP 1.4.
		ResultReasonSensitive:           "Sensitive",
		ResultReasonNotExtractable:      "NotExtractable",
		ResultReasonObjectAlreadyExists: "ObjectAlreadyExists",
	})
	ttlv.RegisterEnum(TagCredentialType, map[CredentialType]string{
		CredentialTypeUsernameAndPassword: "UsernameAndPassword",
		// KMIP 1.1.
		CredentialTypeDevice: "Device",
		// KMIP 1.2.
		CredentialTypeAttestation: "Attestation",
	})
	ttlv.RegisterEnum(TagRevocationReasonCode, map[RevocationReasonCode]string{
		RevocationReasonCodeUnspecified:          "Unspecified",
		RevocationReasonCodeKeyCompromise:        "KeyCompromise",
		RevocationReasonCodeCACompromise:         "CACompromise",
		RevocationReasonCodeAffiliationChanged:   "AffiliationChanged",
		RevocationReasonCodeSuperseded:           "Superseded",
		RevocationReasonCodeCessationOfOperation: "CessationOfOperation",
		RevocationReasonCodePrivilegeWithdrawn:   "PrivilegeWithdrawn",
	})
	ttlv.RegisterEnum(TagBatchErrorContinuationOption, map[BatchErrorContinuationOption]string{
		BatchErrorContinuationOptionContinue: "Continue",
		BatchErrorContinuationOptionStop:     "Stop",
		BatchErrorContinuationOptionUndo:     "Undo",
	})
	ttlv.RegisterEnum(TagNameType, map[NameType]string{
		NameTypeUninterpretedTextString: "UninterpretedTextString",
		NameTypeUri:                     "Uri",
	})
	ttlv.RegisterEnum(TagObjectType, map[ObjectType]string{
		ObjectTypeCertificate:  "Certificate",
		ObjectTypeSymmetricKey: "SymmetricKey",
		ObjectTypePublicKey:    "PublicKey",
		ObjectTypePrivateKey:   "PrivateKey",
		ObjectTypeSplitKey:     "SplitKey",
		ObjectTypeTemplate:     "Template",
		ObjectTypeSecretData:   "SecretData",
		ObjectTypeOpaqueObject: "OpaqueObject",

		// KMIP 1.2.
		ObjectTypePGPKey: "PGPKey",
	})
	ttlv.RegisterEnum(TagOpaqueDataType, map[OpaqueDataType]string{})
	ttlv.RegisterEnum(TagState, map[State]string{
		StatePreActive:            "PreActive",
		StateActive:               "Active",
		StateDeactivated:          "Deactivated",
		StateCompromised:          "Compromised",
		StateDestroyed:            "Destroyed",
		StateDestroyedCompromised: "DestroyedCompromised",
	})
	ttlv.RegisterEnum(TagCryptographicAlgorithm, map[CryptographicAlgorithm]string{
		CryptographicAlgorithmDES:        "DES",
		CryptographicAlgorithmTDES:       "TDES",
		CryptographicAlgorithmAES:        "AES",
		CryptographicAlgorithmRSA:        "RSA",
		CryptographicAlgorithmDSA:        "DSA",
		CryptographicAlgorithmECDSA:      "ECDSA",
		CryptographicAlgorithmHMACSHA1:   "HMACSHA1",
		CryptographicAlgorithmHMACSHA224: "HMACSHA224",
		CryptographicAlgorithmHMACSHA256: "HMACSHA256",
		CryptographicAlgorithmHMACSHA384: "HMACSHA384",
		CryptographicAlgorithmHMACSHA512: "HMACSHA512",
		CryptographicAlgorithmHMACMD5:    "HMACMD5",
		CryptographicAlgorithmDH:         "DH",
		CryptographicAlgorithmECDH:       "ECDH",
		CryptographicAlgorithmECMQV:      "ECMQV",
		CryptographicAlgorithmBlowfish:   "Blowfish",
		CryptographicAlgorithmCamellia:   "Camellia",
		CryptographicAlgorithmCAST5:      "CAST5",
		CryptographicAlgorithmIDEA:       "IDEA",
		CryptographicAlgorithmMARS:       "MARS",
		CryptographicAlgorithmRC2:        "RC2",
		CryptographicAlgorithmRC4:        "RC4",
		CryptographicAlgorithmRC5:        "RC5",
		CryptographicAlgorithmSKIPJACK:   "SKIPJACK",
		CryptographicAlgorithmTwofish:    "Twofish",

		// KMIP 1.2.
		CryptographicAlgorithmEC: "EC",

		// KMIP 1.3.
		CryptographicAlgorithmOneTimePad: "OneTimePad",

		// KMIP 1.4.
		CryptographicAlgorithmChaCha20:         "ChaCha20",
		CryptographicAlgorithmPoly1305:         "Poly1305",
		CryptographicAlgorithmChaCha20Poly1305: "ChaCha20Poly1305",
		CryptographicAlgorithmSHA3_224:         "SHA3_224",
		CryptographicAlgorithmSHA3_256:         "SHA3_256",
		CryptographicAlgorithmSHA3_384:         "SHA3_384",
		CryptographicAlgorithmSHA3_512:         "SHA3_512",
		CryptographicAlgorithmHMAC_SHA3_224:    "HMAC_SHA3_224",
		CryptographicAlgorithmHMAC_SHA3_256:    "HMAC_SHA3_256",
		CryptographicAlgorithmHMAC_SHA3_384:    "HMAC_SHA3_384",
		CryptographicAlgorithmHMAC_SHA3_512:    "HMAC_SHA3_512",
		CryptographicAlgorithmSHAKE_128:        "SHAKE_128",
		CryptographicAlgorithmSHAKE_256:        "SHAKE_256",
	})
	ttlv.RegisterEnum(TagBlockCipherMode, map[BlockCipherMode]string{
		BlockCipherModeCBC:               "CBC",
		BlockCipherModeECB:               "ECB",
		BlockCipherModePCBC:              "PCBC",
		BlockCipherModeCFB:               "CFB",
		BlockCipherModeOFB:               "OFB",
		BlockCipherModeCTR:               "CTR",
		BlockCipherModeCMAC:              "CMAC",
		BlockCipherModeCCM:               "CCM",
		BlockCipherModeGCM:               "GCM",
		BlockCipherModeCBCMAC:            "CBCMAC",
		BlockCipherModeXTS:               "XTS",
		BlockCipherModeAESKeyWrapPadding: "AESKeyWrapPadding",
		BlockCipherModeNISTKeyWrap:       "NISTKeyWrap",
		BlockCipherModeX9_102AESKW:       "X9_102AESKW",
		BlockCipherModeX9_102TDKW:        "X9_102TDKW",
		BlockCipherModeX9_102AKW1:        "X9_102AKW1",
		BlockCipherModeX9_102AKW2:        "X9_102AKW2",
		// KMIP 1.4
		BlockCipherModeAEAD: "AEAD",
	})
	ttlv.RegisterEnum(TagPaddingMethod, map[PaddingMethod]string{
		PaddingMethodNone:      "None",
		PaddingMethodOAEP:      "OAEP",
		PaddingMethodPKCS5:     "PKCS5",
		PaddingMethodSSL3:      "SSL3",
		PaddingMethodZeros:     "Zeros",
		PaddingMethodANSIX9_23: "ANSIX9_23",
		PaddingMethodISO10126:  "ISO10126",
		PaddingMethodPKCS1V1_5: "PKCS1V1_5",
		PaddingMethodX9_31:     "X9_31",
		PaddingMethodPSS:       "PSS",
	})
	ttlv.RegisterEnum(TagHashingAlgorithm, map[HashingAlgorithm]string{
		HashingAlgorithmMD2:        "MD2",
		HashingAlgorithmMD4:        "MD4",
		HashingAlgorithmMD5:        "MD5",
		HashingAlgorithmSHA_1:      "SHA_1",
		HashingAlgorithmSHA_224:    "SHA_224",
		HashingAlgorithmSHA_256:    "SHA_256",
		HashingAlgorithmSHA_384:    "SHA_384",
		HashingAlgorithmSHA_512:    "SHA_512",
		HashingAlgorithmRIPEMD_160: "RIPEMD_160",
		HashingAlgorithmTiger:      "Tiger",
		HashingAlgorithmWhirlpool:  "Whirlpool",

		// KMIP 1.2.
		HashingAlgorithmSHA_512_224: "SHA_512_224",
		HashingAlgorithmSHA_512_256: "SHA_512_256",

		// KMIP 1.4.
		HashingAlgorithmSHA_3_224: "SHA_3_224",
		HashingAlgorithmSHA_3_256: "SHA_3_256",
		HashingAlgorithmSHA_3_384: "SHA_3_384",
		HashingAlgorithmSHA_3_512: "SHA_3_512",
	})
	ttlv.RegisterEnum(TagKeyRoleType, map[KeyRoleType]string{
		KeyRoleTypeBDK:      "BDK",
		KeyRoleTypeCVK:      "CVK",
		KeyRoleTypeDEK:      "DEK",
		KeyRoleTypeMKAC:     "MKAC",
		KeyRoleTypeMKSMC:    "MKSMC",
		KeyRoleTypeMKSMI:    "MKSMI",
		KeyRoleTypeMKDAC:    "MKDAC",
		KeyRoleTypeMKDN:     "MKDN",
		KeyRoleTypeMKCP:     "MKCP",
		KeyRoleTypeMKOTH:    "MKOTH",
		KeyRoleTypeKEK:      "KEK",
		KeyRoleTypeMAC16609: "MAC16609",
		KeyRoleTypeMAC97971: "MAC97971",
		KeyRoleTypeMAC97972: "MAC97972",
		KeyRoleTypeMAC97973: "MAC97973",
		KeyRoleTypeMAC97974: "MAC97974",
		KeyRoleTypeMAC97975: "MAC97975",
		KeyRoleTypeZPK:      "ZPK",
		KeyRoleTypePVKIBM:   "PVKIBM",
		KeyRoleTypePVKPVV:   "PVKPVV",
		KeyRoleTypePVKOTH:   "PVKOTH",

		// KMIP 1.4
		KeyRoleTypeDUKPT: "DUKPT",
		KeyRoleTypeIV:    "IV",
		KeyRoleTypeTRKBK: "TRKBK",
	})
	ttlv.RegisterEnum(TagRecommendedCurve, map[RecommendedCurve]string{
		RecommendedCurveP_192:            "P_192",
		RecommendedCurveK_163:            "K_163",
		RecommendedCurveB_163:            "B_163",
		RecommendedCurveP_224:            "P_224",
		RecommendedCurveK_233:            "K_233",
		RecommendedCurveB_233:            "B_233",
		RecommendedCurveP_256:            "P_256",
		RecommendedCurveK_283:            "K_283",
		RecommendedCurveB_283:            "B_283",
		RecommendedCurveP_384:            "P_384",
		RecommendedCurveK_409:            "K_409",
		RecommendedCurveB_409:            "B_409",
		RecommendedCurveP_521:            "P_521",
		RecommendedCurveK_571:            "K_571",
		RecommendedCurveB_571:            "B_571",
		RecommendedCurveSECP112R1:        "SECP112R1",
		RecommendedCurveSECP112R2:        "SECP112R2",
		RecommendedCurveSECP128R1:        "SECP128R1",
		RecommendedCurveSECP128R2:        "SECP128R2",
		RecommendedCurveSECP160K1:        "SECP160K1",
		RecommendedCurveSECP160R1:        "SECP160R1",
		RecommendedCurveSECP160R2:        "SECP160R2",
		RecommendedCurveSECP192K1:        "SECP192K1",
		RecommendedCurveSECP224K1:        "SECP224K1",
		RecommendedCurveSECP256K1:        "SECP256K1",
		RecommendedCurveSECT113R1:        "SECT113R1",
		RecommendedCurveSECT113R2:        "SECT113R2",
		RecommendedCurveSECT131R1:        "SECT131R1",
		RecommendedCurveSECT131R2:        "SECT131R2",
		RecommendedCurveSECT163R1:        "SECT163R1",
		RecommendedCurveSECT193R1:        "SECT193R1",
		RecommendedCurveSECT193R2:        "SECT193R2",
		RecommendedCurveSECT239K1:        "SECT239K1",
		RecommendedCurveANSIX9P192V2:     "ANSIX9P192V2",
		RecommendedCurveANSIX9P192V3:     "ANSIX9P192V3",
		RecommendedCurveANSIX9P239V1:     "ANSIX9P239V1",
		RecommendedCurveANSIX9P239V2:     "ANSIX9P239V2",
		RecommendedCurveANSIX9P239V3:     "ANSIX9P239V3",
		RecommendedCurveANSIX9C2PNB163V1: "ANSIX9C2PNB163V1",
		RecommendedCurveANSIX9C2PNB163V2: "ANSIX9C2PNB163V2",
		RecommendedCurveANSIX9C2PNB163V3: "ANSIX9C2PNB163V3",
		RecommendedCurveANSIX9C2PNB176V1: "ANSIX9C2PNB176V1",
		RecommendedCurveANSIX9C2TNB191V1: "ANSIX9C2TNB191V1",
		RecommendedCurveANSIX9C2TNB191V2: "ANSIX9C2TNB191V2",
		RecommendedCurveANSIX9C2TNB191V3: "ANSIX9C2TNB191V3",
		RecommendedCurveANSIX9C2PNB208W1: "ANSIX9C2PNB208W1",
		RecommendedCurveANSIX9C2TNB239V1: "ANSIX9C2TNB239V1",
		RecommendedCurveANSIX9C2TNB239V2: "ANSIX9C2TNB239V2",
		RecommendedCurveANSIX9C2TNB239V3: "ANSIX9C2TNB239V3",
		RecommendedCurveANSIX9C2PNB272W1: "ANSIX9C2PNB272W1",
		RecommendedCurveANSIX9C2PNB304W1: "ANSIX9C2PNB304W1",
		RecommendedCurveANSIX9C2TNB359V1: "ANSIX9C2TNB359V1",
		RecommendedCurveANSIX9C2PNB368W1: "ANSIX9C2PNB368W1",
		RecommendedCurveANSIX9C2TNB431R1: "ANSIX9C2TNB431R1",
		RecommendedCurveBRAINPOOLP160R1:  "BRAINPOOLP160R1",
		RecommendedCurveBRAINPOOLP160T1:  "BRAINPOOLP160T1",
		RecommendedCurveBRAINPOOLP192R1:  "BRAINPOOLP192R1",
		RecommendedCurveBRAINPOOLP192T1:  "BRAINPOOLP192T1",
		RecommendedCurveBRAINPOOLP224R1:  "BRAINPOOLP224R1",
		RecommendedCurveBRAINPOOLP224T1:  "BRAINPOOLP224T1",
		RecommendedCurveBRAINPOOLP256R1:  "BRAINPOOLP256R1",
		RecommendedCurveBRAINPOOLP256T1:  "BRAINPOOLP256T1",
		RecommendedCurveBRAINPOOLP320R1:  "BRAINPOOLP320R1",
		RecommendedCurveBRAINPOOLP320T1:  "BRAINPOOLP320T1",
		RecommendedCurveBRAINPOOLP384R1:  "BRAINPOOLP384R1",
		RecommendedCurveBRAINPOOLP384T1:  "BRAINPOOLP384T1",
		RecommendedCurveBRAINPOOLP512R1:  "BRAINPOOLP512R1",
		RecommendedCurveBRAINPOOLP512T1:  "BRAINPOOLP512T1",
	})
	ttlv.RegisterEnum(TagSecretDataType, map[SecretDataType]string{
		SecretDataTypePassword: "Password",
		SecretDataTypeSeed:     "Seed",
	})
	ttlv.RegisterEnum(TagKeyFormatType, map[KeyFormatType]string{
		KeyFormatTypeRaw:                        "Raw",
		KeyFormatTypeOpaque:                     "Opaque",
		KeyFormatTypePKCS_1:                     "PKCS_1",
		KeyFormatTypePKCS_8:                     "PKCS_8",
		KeyFormatTypeX_509:                      "X_509",
		KeyFormatTypeECPrivateKey:               "ECPrivateKey",
		KeyFormatTypeTransparentSymmetricKey:    "TransparentSymmetricKey",
		KeyFormatTypeTransparentDSAPrivateKey:   "TransparentDSAPrivateKey",
		KeyFormatTypeTransparentDSAPublicKey:    "TransparentDSAPublicKey",
		KeyFormatTypeTransparentRSAPrivateKey:   "TransparentRSAPrivateKey",
		KeyFormatTypeTransparentRSAPublicKey:    "TransparentRSAPublicKey",
		KeyFormatTypeTransparentDHPrivateKey:    "TransparentDHPrivateKey",
		KeyFormatTypeTransparentDHPublicKey:     "TransparentDHPublicKey",
		KeyFormatTypeTransparentECDSAPrivateKey: "TransparentECDSAPrivateKey",
		KeyFormatTypeTransparentECDSAPublicKey:  "TransparentECDSAPublicKey",
		KeyFormatTypeTransparentECDHPrivateKey:  "TransparentECDHPrivateKey",
		KeyFormatTypeTransparentECDHPublicKey:   "TransparentECDHPublicKey",
		KeyFormatTypeTransparentECMQVPrivateKey: "TransparentECMQVPrivateKey",
		KeyFormatTypeTransparentECMQVPublicKey:  "TransparentECMQVPublicKey",

		// KMIP 1.3.
		KeyFormatTypeTransparentECPrivateKey: "TransparentECPrivateKey",
		KeyFormatTypeTransparentECPublicKey:  "TransparentECPublicKey",

		// KMIP 1.4.
		KeyFormatTypePKCS_12: "PKCS_12",
	})
	ttlv.RegisterEnum(TagKeyCompressionType, map[KeyCompressionType]string{
		KeyCompressionTypeECPublicKeyTypeUncompressed:         "ECPublicKeyTypeUncompressed",
		KeyCompressionTypeECPublicKeyTypeX9_62CompressedPrime: "ECPublicKeyTypeX9_62CompressedPrime",
		KeyCompressionTypeECPublicKeyTypeX9_62CompressedChar2: "ECPublicKeyTypeX9_62CompressedChar2",
		KeyCompressionTypeECPublicKeyTypeX9_62Hybrid:          "ECPublicKeyTypeX9_62Hybrid",
	})
	ttlv.RegisterEnum(TagWrappingMethod, map[WrappingMethod]string{
		WrappingMethodEncrypt:            "Encrypt",
		WrappingMethodMACSign:            "MACSign",
		WrappingMethodEncryptThenMACSign: "EncryptThenMACSign",
		WrappingMethodMACSignThenEncrypt: "MACSignThenEncrypt",
		WrappingMethodTR_31:              "TR_31",
	})
	ttlv.RegisterEnum(TagCertificateType, map[CertificateType]string{
		CertificateTypeX_509: "X_509",
		CertificateTypePGP:   "PGP",
	})
	ttlv.RegisterEnum(TagLinkType, map[LinkType]string{
		LinkTypeCertificateLink:          "CertificateLink",
		LinkTypePublicKeyLink:            "PublicKeyLink",
		LinkTypePrivateKeyLink:           "PrivateKeyLink",
		LinkTypeDerivationBaseObjectLink: "DerivationBaseObjectLink",
		LinkTypeDerivedKeyLink:           "DerivedKeyLink",
		LinkTypeReplacementObjectLink:    "ReplacementObjectLink",
		LinkTypeReplacedObjectLink:       "ReplacedObjectLink",

		// KMIP 1.2.
		LinkTypeParentLink:   "ParentLink",
		LinkTypeChildLink:    "ChildLink",
		LinkTypePreviousLink: "PreviousLink",
		LinkTypeNextLink:     "NextLink",

		// KMIP 1.4.
		LinkTypePKCS_12CertificateLink: "PKCS_12CertificateLink",
		LinkTypePKCS_12PasswordLink:    "PKCS_12PasswordLink",

		//FIXME: This is defined in KMIP 2.0+ only.
		LinkTypeWrappingKeyLink: "WrappingKeyLink",
	})
	ttlv.RegisterEnum(TagQueryFunction, map[QueryFunction]string{
		QueryFunctionOperations:            "QueryOperations",
		QueryFunctionObjects:               "QueryObjects",
		QueryFunctionServerInformation:     "QueryServerInformation",
		QueryFunctionApplicationNamespaces: "QueryApplicationNamespaces",
		// KMIP 1.1.
		QueryFunctionExtensionList: "QueryExtensionList",
		QueryFunctionExtensionMap:  "QueryExtensionMap",
		// KMIP 1.2.
		QueryFunctionAttestationTypes: "QueryAttestationTypes",
		// KMIP 1.3.
		QueryFunctionRNGs:                      "QueryRNGs",
		QueryFunctionValidations:               "QueryValidations",
		QueryFunctionProfiles:                  "QueryProfiles",
		QueryFunctionCapabilities:              "QueryCapabilities",
		QueryFunctionClientRegistrationMethods: "QueryClientRegistrationMethods",
	})
	ttlv.RegisterEnum(TagUsageLimitsUnit, map[UsageLimitsUnit]string{
		UsageLimitsUnitByte:   "UnitByte",
		UsageLimitsUnitObject: "UnitObject",
	})
	ttlv.RegisterEnum(TagCancellationResult, map[CancellationResult]string{
		CancellationResultCanceled:       "Canceled",
		CancellationResultUnableToCancel: "UnableToCancel",
		CancellationResultCompleted:      "Completed",
		CancellationResultFailed:         "Failed",
		CancellationResultUnavailable:    "Unavailable",
	})
	ttlv.RegisterEnum(TagPutFunction, map[PutFunction]string{
		PutFunctionNew:     "New",
		PutFunctionReplace: "Replace",
	})
	ttlv.RegisterEnum(TagCertificateRequestType, map[CertificateRequestType]string{
		CertificateRequestTypeCRMF:    "CRMF",
		CertificateRequestTypePKCS_10: "PKCS_10",
		CertificateRequestTypePEM:     "PEM",
		CertificateRequestTypePGP:     "PGP",
	})
	ttlv.RegisterEnum(TagSplitKeyMethod, map[SplitKeyMethod]string{
		SplitKeyMethodXOR:                         "XOR",
		SplitKeyMethodPolynomialSharingGF216:      "PolynomialSharingGF216",
		SplitKeyMethodPolynomialSharingPrimeField: "PolynomialSharingPrimeField",
		// KMIP 1.2.
		SplitKeyMethodPolynomialSharingGF28: "PolynomialSharingGF28",
	})
	ttlv.RegisterEnum(TagObjectGroupMember, map[ObjectGroupMember]string{
		ObjectGroupMemberFresh:   "GroupMemberFresh",
		ObjectGroupMemberDefault: "GroupMemberDefault",
	})
	ttlv.RegisterEnum(TagEncodingOption, map[EncodingOption]string{
		EncodingOptionNoEncoding:   "NoEncoding",
		EncodingOptionTTLVEncoding: "TTLVEncoding",
	})
	ttlv.RegisterEnum(TagDigitalSignatureAlgorithm, map[DigitalSignatureAlgorithm]string{
		DigitalSignatureAlgorithmMD2WithRSAEncryptionPKCS_1v1_5:     "MD2WithRSAEncryptionPKCS_1v1_5",
		DigitalSignatureAlgorithmMD5WithRSAEncryptionPKCS_1v1_5:     "MD5WithRSAEncryptionPKCS_1v1_5",
		DigitalSignatureAlgorithmSHA_1WithRSAEncryptionPKCS_1v1_5:   "SHA_1WithRSAEncryptionPKCS_1v1_5",
		DigitalSignatureAlgorithmSHA_224WithRSAEncryptionPKCS_1v1_5: "SHA_224WithRSAEncryptionPKCS_1v1_5",
		DigitalSignatureAlgorithmSHA_256WithRSAEncryptionPKCS_1v1_5: "SHA_256WithRSAEncryptionPKCS_1v1_5",
		DigitalSignatureAlgorithmSHA_384WithRSAEncryptionPKCS_1v1_5: "SHA_384WithRSAEncryptionPKCS_1v1_5",
		DigitalSignatureAlgorithmSHA_512WithRSAEncryptionPKCS_1v1_5: "SHA_512WithRSAEncryptionPKCS_1v1_5",
		DigitalSignatureAlgorithmRSASSA_PSSPKCS_1v2_1:               "RSASSA_PSSPKCS_1v2_1",
		DigitalSignatureAlgorithmDSAWithSHA_1:                       "DSAWithSHA_1",
		DigitalSignatureAlgorithmDSAWithSHA224:                      "DSAWithSHA224",
		DigitalSignatureAlgorithmDSAWithSHA256:                      "DSAWithSHA256",
		DigitalSignatureAlgorithmECDSAWithSHA_1:                     "ECDSAWithSHA_1",
		DigitalSignatureAlgorithmECDSAWithSHA224:                    "ECDSAWithSHA224",
		DigitalSignatureAlgorithmECDSAWithSHA256:                    "ECDSAWithSHA256",
		DigitalSignatureAlgorithmECDSAWithSHA384:                    "ECDSAWithSHA384",
		DigitalSignatureAlgorithmECDSAWithSHA512:                    "ECDSAWithSHA512",

		// KMIP 1.4.
		DigitalSignatureAlgorithmSHA3_256WithRSAEncryption: "SHA3_256WithRSAEncryption",
		DigitalSignatureAlgorithmSHA3_384WithRSAEncryption: "SHA3_384WithRSAEncryption",
		DigitalSignatureAlgorithmSHA3_512WithRSAEncryption: "SHA3_512WithRSAEncryption",
	})
	ttlv.RegisterEnum(TagAttestationType, map[AttestationType]string{
		AttestationTypeTPMQuote:           "TPMQuote",
		AttestationTypeTCGIntegrityReport: "TCGIntegrityReport",
		AttestationTypeSAMLAssertion:      "SAMLAssertion",
	})
	ttlv.RegisterEnum(TagAlternativeNameType, map[AlternativeNameType]string{
		AlternativeNameTypeUninterpretedTextString: "UninterpretedTextString",
		AlternativeNameTypeURI:                     "URI",
		AlternativeNameTypeObjectSerialNumber:      "ObjectSerialNumber",
		AlternativeNameTypeEmailAddress:            "EmailAddress",
		AlternativeNameTypeDNSName:                 "DNSName",
		AlternativeNameTypeX_500DistinguishedName:  "X_500DistinguishedName",
		AlternativeNameTypeIPAddress:               "IPAddress",
	})
	ttlv.RegisterEnum(TagKeyValueLocationType, map[KeyValueLocationType]string{
		KeyValueLocationTypeUninterpretedTextString: "UninterpretedTextString",
		KeyValueLocationTypeURI:                     "URI",
	})
	ttlv.RegisterEnum(TagValidityIndicator, map[ValidityIndicator]string{
		ValidityIndicatorValid:   "Valid",
		ValidityIndicatorInvalid: "Invalid",
		ValidityIndicatorUnknown: "Unknown",
	})

	ttlv.RegisterEnum(TagRNGAlgorithm, map[RNGAlgorithm]string{
		RNGAlgorithmUnspecified: "Unspecified",
		RNGAlgorithmFIPS186_2:   "FIPS186_2",
		RNGAlgorithmDRBG:        "DRBG",
		RNGAlgorithmNRBG:        "NRBG",
		RNGAlgorithmANSIX9_31:   "ANSIX9_31",
		RNGAlgorithmANSIX9_62:   "ANSIX9_62",
	})
	ttlv.RegisterEnum(TagDRBGAlgorithm, map[DRBGAlgorithm]string{
		DRBGAlgorithmUnspecified: "Unspecified",
		DRBGAlgorithmDual_EC:     "Dual_EC",
		DRBGAlgorithmHash:        "Hash",
		DRBGAlgorithmHMAC:        "HMAC",
		DRBGAlgorithmCTR:         "CTR",
	})
	ttlv.RegisterEnum(TagFIPS186Variation, map[FIPS186Variation]string{
		FIPS186VariationUnspecified:     "Unspecified",
		FIPS186VariationGPXOriginal:     "GPXOriginal",
		FIPS186VariationGPXChangeNotice: "GPXChangeNotice",
		FIPS186VariationXOriginal:       "XOriginal",
		FIPS186VariationXChangeNotice:   "XChangeNotice",
		FIPS186VariationKOriginal:       "KOriginal",
		FIPS186VariationKChangeNotice:   "KChangeNotice",
	})
	ttlv.RegisterEnum(TagProfileName, map[ProfileName]string{
		ProfileNameBaselineServerBasicKMIPV1_2:                       "BaselineServerBasicKMIPV1_2",
		ProfileNameBaselineServerTLSV1_2KMIPV1_2:                     "BaselineServerTLSV1_2KMIPV1_2",
		ProfileNameBaselineClientBasicKMIPV1_2:                       "BaselineClientBasicKMIPV1_2",
		ProfileNameBaselineClientTLSV1_2KMIPV1_2:                     "BaselineClientTLSV1_2KMIPV1_2",
		ProfileNameCompleteServerBasicKMIPV1_2:                       "CompleteServerBasicKMIPV1_2",
		ProfileNameCompleteServerTLSV1_2KMIPV1_2:                     "CompleteServerTLSV1_2KMIPV1_2",
		ProfileNameTapeLibraryClientKMIPV1_0:                         "TapeLibraryClientKMIPV1_0",
		ProfileNameTapeLibraryClientKMIPV1_1:                         "TapeLibraryClientKMIPV1_1",
		ProfileNameTapeLibraryClientKMIPV1_2:                         "TapeLibraryClientKMIPV1_2",
		ProfileNameTapeLibraryServerKMIPV1_0:                         "TapeLibraryServerKMIPV1_0",
		ProfileNameTapeLibraryServerKMIPV1_1:                         "TapeLibraryServerKMIPV1_1",
		ProfileNameTapeLibraryServerKMIPV1_2:                         "TapeLibraryServerKMIPV1_2",
		ProfileNameSymmetricKeyLifecycleClientKMIPV1_0:               "SymmetricKeyLifecycleClientKMIPV1_0",
		ProfileNameSymmetricKeyLifecycleClientKMIPV1_1:               "SymmetricKeyLifecycleClientKMIPV1_1",
		ProfileNameSymmetricKeyLifecycleClientKMIPV1_2:               "SymmetricKeyLifecycleClientKMIPV1_2",
		ProfileNameSymmetricKeyLifecycleServerKMIPV1_0:               "SymmetricKeyLifecycleServerKMIPV1_0",
		ProfileNameSymmetricKeyLifecycleServerKMIPV1_1:               "SymmetricKeyLifecycleServerKMIPV1_1",
		ProfileNameSymmetricKeyLifecycleServerKMIPV1_2:               "SymmetricKeyLifecycleServerKMIPV1_2",
		ProfileNameAsymmetricKeyLifecycleClientKMIPV1_0:              "AsymmetricKeyLifecycleClientKMIPV1_0",
		ProfileNameAsymmetricKeyLifecycleClientKMIPV1_1:              "AsymmetricKeyLifecycleClientKMIPV1_1",
		ProfileNameAsymmetricKeyLifecycleClientKMIPV1_2:              "AsymmetricKeyLifecycleClientKMIPV1_2",
		ProfileNameAsymmetricKeyLifecycleServerKMIPV1_0:              "AsymmetricKeyLifecycleServerKMIPV1_0",
		ProfileNameAsymmetricKeyLifecycleServerKMIPV1_1:              "AsymmetricKeyLifecycleServerKMIPV1_1",
		ProfileNameAsymmetricKeyLifecycleServerKMIPV1_2:              "AsymmetricKeyLifecycleServerKMIPV1_2",
		ProfileNameBasicCryptographicClientKMIPV1_2:                  "BasicCryptographicClientKMIPV1_2",
		ProfileNameBasicCryptographicServerKMIPV1_2:                  "BasicCryptographicServerKMIPV1_2",
		ProfileNameAdvancedCryptographicClientKMIPV1_2:               "AdvancedCryptographicClientKMIPV1_2",
		ProfileNameAdvancedCryptographicServerKMIPV1_2:               "AdvancedCryptographicServerKMIPV1_2",
		ProfileNameRNGCryptographicClientKMIPV1_2:                    "RNGCryptographicClientKMIPV1_2",
		ProfileNameRNGCryptographicServerKMIPV1_2:                    "RNGCryptographicServerKMIPV1_2",
		ProfileNameBasicSymmetricKeyFoundryClientKMIPV1_0:            "BasicSymmetricKeyFoundryClientKMIPV1_0",
		ProfileNameIntermediateSymmetricKeyFoundryClientKMIPV1_0:     "IntermediateSymmetricKeyFoundryClientKMIPV1_0",
		ProfileNameAdvancedSymmetricKeyFoundryClientKMIPV1_0:         "AdvancedSymmetricKeyFoundryClientKMIPV1_0",
		ProfileNameBasicSymmetricKeyFoundryClientKMIPV1_1:            "BasicSymmetricKeyFoundryClientKMIPV1_1",
		ProfileNameIntermediateSymmetricKeyFoundryClientKMIPV1_1:     "IntermediateSymmetricKeyFoundryClientKMIPV1_1",
		ProfileNameAdvancedSymmetricKeyFoundryClientKMIPV1_1:         "AdvancedSymmetricKeyFoundryClientKMIPV1_1",
		ProfileNameBasicSymmetricKeyFoundryClientKMIPV1_2:            "BasicSymmetricKeyFoundryClientKMIPV1_2",
		ProfileNameIntermediateSymmetricKeyFoundryClientKMIPV1_2:     "IntermediateSymmetricKeyFoundryClientKMIPV1_2",
		ProfileNameAdvancedSymmetricKeyFoundryClientKMIPV1_2:         "AdvancedSymmetricKeyFoundryClientKMIPV1_2",
		ProfileNameSymmetricKeyFoundryServerKMIPV1_0:                 "SymmetricKeyFoundryServerKMIPV1_0",
		ProfileNameSymmetricKeyFoundryServerKMIPV1_1:                 "SymmetricKeyFoundryServerKMIPV1_1",
		ProfileNameSymmetricKeyFoundryServerKMIPV1_2:                 "SymmetricKeyFoundryServerKMIPV1_2",
		ProfileNameOpaqueManagedObjectStoreClientKMIPV1_0:            "OpaqueManagedObjectStoreClientKMIPV1_0",
		ProfileNameOpaqueManagedObjectStoreClientKMIPV1_1:            "OpaqueManagedObjectStoreClientKMIPV1_1",
		ProfileNameOpaqueManagedObjectStoreClientKMIPV1_2:            "OpaqueManagedObjectStoreClientKMIPV1_2",
		ProfileNameOpaqueManagedObjectStoreServerKMIPV1_0:            "OpaqueManagedObjectStoreServerKMIPV1_0",
		ProfileNameOpaqueManagedObjectStoreServerKMIPV1_1:            "OpaqueManagedObjectStoreServerKMIPV1_1",
		ProfileNameOpaqueManagedObjectStoreServerKMIPV1_2:            "OpaqueManagedObjectStoreServerKMIPV1_2",
		ProfileNameSuiteBMinLOS_128ClientKMIPV1_0:                    "SuiteBMinLOS_128ClientKMIPV1_0",
		ProfileNameSuiteBMinLOS_128ClientKMIPV1_1:                    "SuiteBMinLOS_128ClientKMIPV1_1",
		ProfileNameSuiteBMinLOS_128ClientKMIPV1_2:                    "SuiteBMinLOS_128ClientKMIPV1_2",
		ProfileNameSuiteBMinLOS_128ServerKMIPV1_0:                    "SuiteBMinLOS_128ServerKMIPV1_0",
		ProfileNameSuiteBMinLOS_128ServerKMIPV1_1:                    "SuiteBMinLOS_128ServerKMIPV1_1",
		ProfileNameSuiteBMinLOS_128ServerKMIPV1_2:                    "SuiteBMinLOS_128ServerKMIPV1_2",
		ProfileNameSuiteBMinLOS_192ClientKMIPV1_0:                    "SuiteBMinLOS_192ClientKMIPV1_0",
		ProfileNameSuiteBMinLOS_192ClientKMIPV1_1:                    "SuiteBMinLOS_192ClientKMIPV1_1",
		ProfileNameSuiteBMinLOS_192ClientKMIPV1_2:                    "SuiteBMinLOS_192ClientKMIPV1_2",
		ProfileNameSuiteBMinLOS_192ServerKMIPV1_0:                    "SuiteBMinLOS_192ServerKMIPV1_0",
		ProfileNameSuiteBMinLOS_192ServerKMIPV1_1:                    "SuiteBMinLOS_192ServerKMIPV1_1",
		ProfileNameSuiteBMinLOS_192ServerKMIPV1_2:                    "SuiteBMinLOS_192ServerKMIPV1_2",
		ProfileNameStorageArrayWithSelfEncryptingDriveClientKMIPV1_0: "StorageArrayWithSelfEncryptingDriveClientKMIPV1_0",
		ProfileNameStorageArrayWithSelfEncryptingDriveClientKMIPV1_1: "StorageArrayWithSelfEncryptingDriveClientKMIPV1_1",
		ProfileNameStorageArrayWithSelfEncryptingDriveClientKMIPV1_2: "StorageArrayWithSelfEncryptingDriveClientKMIPV1_2",
		ProfileNameStorageArrayWithSelfEncryptingDriveServerKMIPV1_0: "StorageArrayWithSelfEncryptingDriveServerKMIPV1_0",
		ProfileNameStorageArrayWithSelfEncryptingDriveServerKMIPV1_1: "StorageArrayWithSelfEncryptingDriveServerKMIPV1_1",
		ProfileNameStorageArrayWithSelfEncryptingDriveServerKMIPV1_2: "StorageArrayWithSelfEncryptingDriveServerKMIPV1_2",
		ProfileNameHTTPSClientKMIPV1_0:                               "HTTPSClientKMIPV1_0",
		ProfileNameHTTPSClientKMIPV1_1:                               "HTTPSClientKMIPV1_1",
		ProfileNameHTTPSClientKMIPV1_2:                               "HTTPSClientKMIPV1_2",
		ProfileNameHTTPSServerKMIPV1_0:                               "HTTPSServerKMIPV1_0",
		ProfileNameHTTPSServerKMIPV1_1:                               "HTTPSServerKMIPV1_1",
		ProfileNameHTTPSServerKMIPV1_2:                               "HTTPSServerKMIPV1_2",
		ProfileNameJSONClientKMIPV1_0:                                "JSONClientKMIPV1_0",
		ProfileNameJSONClientKMIPV1_1:                                "JSONClientKMIPV1_1",
		ProfileNameJSONClientKMIPV1_2:                                "JSONClientKMIPV1_2",
		ProfileNameJSONServerKMIPV1_0:                                "JSONServerKMIPV1_0",
		ProfileNameJSONServerKMIPV1_1:                                "JSONServerKMIPV1_1",
		ProfileNameJSONServerKMIPV1_2:                                "JSONServerKMIPV1_2",
		ProfileNameXMLClientKMIPV1_0:                                 "XMLClientKMIPV1_0",
		ProfileNameXMLClientKMIPV1_1:                                 "XMLClientKMIPV1_1",
		ProfileNameXMLClientKMIPV1_2:                                 "XMLClientKMIPV1_2",
		ProfileNameXMLServerKMIPV1_0:                                 "XMLServerKMIPV1_0",
		ProfileNameXMLServerKMIPV1_1:                                 "XMLServerKMIPV1_1",
		ProfileNameXMLServerKMIPV1_2:                                 "XMLServerKMIPV1_2",
		ProfileNameBaselineServerBasicKMIPV1_3:                       "BaselineServerBasicKMIPV1_3",
		ProfileNameBaselineServerTLSV1_2KMIPV1_3:                     "BaselineServerTLSV1_2KMIPV1_3",
		ProfileNameBaselineClientBasicKMIPV1_3:                       "BaselineClientBasicKMIPV1_3",
		ProfileNameBaselineClientTLSV1_2KMIPV1_3:                     "BaselineClientTLSV1_2KMIPV1_3",
		ProfileNameCompleteServerBasicKMIPV1_3:                       "CompleteServerBasicKMIPV1_3",
		ProfileNameCompleteServerTLSV1_2KMIPV1_3:                     "CompleteServerTLSV1_2KMIPV1_3",
		ProfileNameTapeLibraryClientKMIPV1_3:                         "TapeLibraryClientKMIPV1_3",
		ProfileNameTapeLibraryServerKMIPV1_3:                         "TapeLibraryServerKMIPV1_3",
		ProfileNameSymmetricKeyLifecycleClientKMIPV1_3:               "SymmetricKeyLifecycleClientKMIPV1_3",
		ProfileNameSymmetricKeyLifecycleServerKMIPV1_3:               "SymmetricKeyLifecycleServerKMIPV1_3",
		ProfileNameAsymmetricKeyLifecycleClientKMIPV1_3:              "AsymmetricKeyLifecycleClientKMIPV1_3",
		ProfileNameAsymmetricKeyLifecycleServerKMIPV1_3:              "AsymmetricKeyLifecycleServerKMIPV1_3",
		ProfileNameBasicCryptographicClientKMIPV1_3:                  "BasicCryptographicClientKMIPV1_3",
		ProfileNameBasicCryptographicServerKMIPV1_3:                  "BasicCryptographicServerKMIPV1_3",
		ProfileNameAdvancedCryptographicClientKMIPV1_3:               "AdvancedCryptographicClientKMIPV1_3",
		ProfileNameAdvancedCryptographicServerKMIPV1_3:               "AdvancedCryptographicServerKMIPV1_3",
		ProfileNameRNGCryptographicClientKMIPV1_3:                    "RNGCryptographicClientKMIPV1_3",
		ProfileNameRNGCryptographicServerKMIPV1_3:                    "RNGCryptographicServerKMIPV1_3",
		ProfileNameBasicSymmetricKeyFoundryClientKMIPV1_3:            "BasicSymmetricKeyFoundryClientKMIPV1_3",
		ProfileNameIntermediateSymmetricKeyFoundryClientKMIPV1_3:     "IntermediateSymmetricKeyFoundryClientKMIPV1_3",
		ProfileNameAdvancedSymmetricKeyFoundryClientKMIPV1_3:         "AdvancedSymmetricKeyFoundryClientKMIPV1_3",
		ProfileNameSymmetricKeyFoundryServerKMIPV1_3:                 "SymmetricKeyFoundryServerKMIPV1_3",
		ProfileNameOpaqueManagedObjectStoreClientKMIPV1_3:            "OpaqueManagedObjectStoreClientKMIPV1_3",
		ProfileNameOpaqueManagedObjectStoreServerKMIPV1_3:            "OpaqueManagedObjectStoreServerKMIPV1_3",
		ProfileNameSuiteBMinLOS_128ClientKMIPV1_3:                    "SuiteBMinLOS_128ClientKMIPV1_3",
		ProfileNameSuiteBMinLOS_128ServerKMIPV1_3:                    "SuiteBMinLOS_128ServerKMIPV1_3",
		ProfileNameSuiteBMinLOS_192ClientKMIPV1_3:                    "SuiteBMinLOS_192ClientKMIPV1_3",
		ProfileNameSuiteBMinLOS_192ServerKMIPV1_3:                    "SuiteBMinLOS_192ServerKMIPV1_3",
		ProfileNameStorageArrayWithSelfEncryptingDriveClientKMIPV1_3: "StorageArrayWithSelfEncryptingDriveClientKMIPV1_3",
		ProfileNameStorageArrayWithSelfEncryptingDriveServerKMIPV1_3: "StorageArrayWithSelfEncryptingDriveServerKMIPV1_3",
		ProfileNameHTTPSClientKMIPV1_3:                               "HTTPSClientKMIPV1_3",
		ProfileNameHTTPSServerKMIPV1_3:                               "HTTPSServerKMIPV1_3",
		ProfileNameJSONClientKMIPV1_3:                                "JSONClientKMIPV1_3",
		ProfileNameJSONServerKMIPV1_3:                                "JSONServerKMIPV1_3",
		ProfileNameXMLClientKMIPV1_3:                                 "XMLClientKMIPV1_3",
		ProfileNameXMLServerKMIPV1_3:                                 "XMLServerKMIPV1_3",

		// KMIP 1.4.
		ProfileNameBaselineServerBasicKMIPV1_4:                       "BaselineServerBasicKMIPV1_4",
		ProfileNameBaselineServerTLSV1_2KMIPV1_4:                     "BaselineServerTLSV1_2KMIPV1_4",
		ProfileNameBaselineClientBasicKMIPV1_4:                       "BaselineClientBasicKMIPV1_4",
		ProfileNameBaselineClientTLSV1_2KMIPV1_4:                     "BaselineClientTLSV1_2KMIPV1_4",
		ProfileNameCompleteServerBasicKMIPV1_4:                       "CompleteServerBasicKMIPV1_4",
		ProfileNameCompleteServerTLSV1_2KMIPV1_4:                     "CompleteServerTLSV1_2KMIPV1_4",
		ProfileNameTapeLibraryClientKMIPV1_4:                         "TapeLibraryClientKMIPV1_4",
		ProfileNameTapeLibraryServerKMIPV1_4:                         "TapeLibraryServerKMIPV1_4",
		ProfileNameSymmetricKeyLifecycleClientKMIPV1_4:               "SymmetricKeyLifecycleClientKMIPV1_4",
		ProfileNameSymmetricKeyLifecycleServerKMIPV1_4:               "SymmetricKeyLifecycleServerKMIPV1_4",
		ProfileNameAsymmetricKeyLifecycleClientKMIPV1_4:              "AsymmetricKeyLifecycleClientKMIPV1_4",
		ProfileNameAsymmetricKeyLifecycleServerKMIPV1_4:              "AsymmetricKeyLifecycleServerKMIPV1_4",
		ProfileNameBasicCryptographicClientKMIPV1_4:                  "BasicCryptographicClientKMIPV1_4",
		ProfileNameBasicCryptographicServerKMIPV1_4:                  "BasicCryptographicServerKMIPV1_4",
		ProfileNameAdvancedCryptographicClientKMIPV1_4:               "AdvancedCryptographicClientKMIPV1_4",
		ProfileNameAdvancedCryptographicServerKMIPV1_4:               "AdvancedCryptographicServerKMIPV1_4",
		ProfileNameRNGCryptographicClientKMIPV1_4:                    "RNGCryptographicClientKMIPV1_4",
		ProfileNameRNGCryptographicServerKMIPV1_4:                    "RNGCryptographicServerKMIPV1_4",
		ProfileNameBasicSymmetricKeyFoundryClientKMIPV1_4:            "BasicSymmetricKeyFoundryClientKMIPV1_4",
		ProfileNameIntermediateSymmetricKeyFoundryClientKMIPV1_4:     "IntermediateSymmetricKeyFoundryClientKMIPV1_4",
		ProfileNameAdvancedSymmetricKeyFoundryClientKMIPV1_4:         "AdvancedSymmetricKeyFoundryClientKMIPV1_4",
		ProfileNameSymmetricKeyFoundryServerKMIPV1_4:                 "SymmetricKeyFoundryServerKMIPV1_4",
		ProfileNameOpaqueManagedObjectStoreClientKMIPV1_4:            "OpaqueManagedObjectStoreClientKMIPV1_4",
		ProfileNameOpaqueManagedObjectStoreServerKMIPV1_4:            "OpaqueManagedObjectStoreServerKMIPV1_4",
		ProfileNameSuiteBMinLOS_128ClientKMIPV1_4:                    "SuiteBMinLOS_128ClientKMIPV1_4",
		ProfileNameSuiteBMinLOS_128ServerKMIPV1_4:                    "SuiteBMinLOS_128ServerKMIPV1_4",
		ProfileNameSuiteBMinLOS_192ClientKMIPV1_4:                    "SuiteBMinLOS_192ClientKMIPV1_4",
		ProfileNameSuiteBMinLOS_192ServerKMIPV1_4:                    "SuiteBMinLOS_192ServerKMIPV1_4",
		ProfileNameStorageArrayWithSelfEncryptingDriveClientKMIPV1_4: "StorageArrayWithSelfEncryptingDriveClientKMIPV1_4",
		ProfileNameStorageArrayWithSelfEncryptingDriveServerKMIPV1_4: "StorageArrayWithSelfEncryptingDriveServerKMIPV1_4",
		ProfileNameHTTPSClientKMIPV1_4:                               "HTTPSClientKMIPV1_4",
		ProfileNameHTTPSServerKMIPV1_4:                               "HTTPSServerKMIPV1_4",
		ProfileNameJSONClientKMIPV1_4:                                "JSONClientKMIPV1_4",
		ProfileNameJSONServerKMIPV1_4:                                "JSONServerKMIPV1_4",
		ProfileNameXMLClientKMIPV1_4:                                 "XMLClientKMIPV1_4",
		ProfileNameXMLServerKMIPV1_4:                                 "XMLServerKMIPV1_4",
	})
	ttlv.RegisterEnum(TagValidationAuthorityType, map[ValidationAuthorityType]string{
		ValidationAuthorityTypeUnspecified:    "Unspecified",
		ValidationAuthorityTypeNISTCMVP:       "NISTCMVP",
		ValidationAuthorityTypeCommonCriteria: "CommonCriteria",
	})
	ttlv.RegisterEnum(TagValidationType, map[ValidationType]string{
		ValidationTypeUnspecified: "Unspecified",
		ValidationTypeHardware:    "Hardware",
		ValidationTypeSoftware:    "Software",
		ValidationTypeFirmware:    "Firmware",
		ValidationTypeHybrid:      "Hybrid",
	})
	ttlv.RegisterEnum(TagUnwrapMode, map[UnwrapMode]string{
		UnwrapModeUnspecified:  "Unspecified",
		UnwrapModeProcessed:    "Processed",
		UnwrapModeNotProcessed: "NotProcessed",
	})
	ttlv.RegisterEnum(TagDestroyAction, map[DestroyAction]string{
		DestroyActionUnspecified:         "Unspecified",
		DestroyActionKeyMaterialDeleted:  "KeyMaterialDeleted",
		DestroyActionKeyMaterialShredded: "KeyMaterialShredded",
		DestroyActionMetaDataDeleted:     "MetaDataDeleted",
		DestroyActionMetaDataShredded:    "MetaDataShredded",
		DestroyActionDeleted:             "Deleted",
		DestroyActionShredded:            "Shredded",
	})
	ttlv.RegisterEnum(TagShreddingAlgorithm, map[ShreddingAlgorithm]string{
		ShreddingAlgorithmUnspecified:   "Unspecified",
		ShreddingAlgorithmCryptographic: "Cryptographic",
		ShreddingAlgorithmUnsupported:   "Unsupported",
	})
	ttlv.RegisterEnum(TagRNGMode, map[RNGMode]string{
		RNGModeUnspecified:            "Unspecified",
		RNGModeSharedInstantiation:    "SharedInstantiation",
		RNGModeNonSharedInstantiation: "NonSharedInstantiation",
	})
	ttlv.RegisterEnum(TagClientRegistrationMethod, map[ClientRegistrationMethod]string{
		ClientRegistrationMethodUnspecified:        "Unspecified",
		ClientRegistrationMethodServerPreGenerated: "ServerPreGenerated",
		ClientRegistrationMethodServerOnDemand:     "ServerOnDemand",
		ClientRegistrationMethodClientGenerated:    "ClientGenerated",
		ClientRegistrationMethodClientRegistered:   "ClientRegistered",
	})
	ttlv.RegisterEnum(TagMaskGenerator, map[MaskGenerator]string{
		MaskGeneratorMGF1: "MGF1",
	})
}

type ResultStatus uint32

const (
	ResultStatusSuccess          ResultStatus = 0x00000000
	ResultStatusOperationFailed  ResultStatus = 0x00000001
	ResultStatusOperationPending ResultStatus = 0x00000002
	ResultStatusOperationUndone  ResultStatus = 0x00000003
)

type ResultReason uint32

// See https://docs.oasis-open.org/kmip/spec/v1.4/errata01/os/kmip-spec-v1.4-errata01-os-redlined.html#_Toc490660949
const (
	ResultReasonItemNotFound                     ResultReason = 0x00000001
	ResultReasonResponseTooLarge                 ResultReason = 0x00000002
	ResultReasonAuthenticationNotSuccessful      ResultReason = 0x00000003
	ResultReasonInvalidMessage                   ResultReason = 0x00000004
	ResultReasonOperationNotSupported            ResultReason = 0x00000005
	ResultReasonMissingData                      ResultReason = 0x00000006
	ResultReasonInvalidField                     ResultReason = 0x00000007
	ResultReasonFeatureNotSupported              ResultReason = 0x00000008
	ResultReasonOperationCanceledByRequester     ResultReason = 0x00000009
	ResultReasonCryptographicFailure             ResultReason = 0x0000000A
	ResultReasonIllegalOperation                 ResultReason = 0x0000000B
	ResultReasonPermissionDenied                 ResultReason = 0x0000000C
	ResultReasonObjectarchived                   ResultReason = 0x0000000D
	ResultReasonIndexOutofBounds                 ResultReason = 0x0000000E
	ResultReasonApplicationNamespaceNotSupported ResultReason = 0x0000000F
	ResultReasonKeyFormatTypeNotSupported        ResultReason = 0x00000010
	ResultReasonKeyCompressionTypeNotSupported   ResultReason = 0x00000011
	// KMIP 1.1.
	ResultReasonEncodingOptionError ResultReason = 0x00000012
	// KMIP 1.2.
	ResultReasonKeyValueNotPresent  ResultReason = 0x00000013
	ResultReasonAttestationRequired ResultReason = 0x00000014
	ResultReasonAttestationFailed   ResultReason = 0x00000015

	// KMIP 1.4.
	ResultReasonSensitive           ResultReason = 0x00000016
	ResultReasonNotExtractable      ResultReason = 0x00000017
	ResultReasonObjectAlreadyExists ResultReason = 0x00000018

	ResultReasonGeneralFailure ResultReason = 0x00000100
)

type CredentialType uint32

const (
	CredentialTypeUsernameAndPassword CredentialType = 0x00000001
	// KMIP 1.1.
	CredentialTypeDevice CredentialType = 0x00000002
	// KMIP 1.2.
	CredentialTypeAttestation CredentialType = 0x00000003
)

type RevocationReasonCode uint32

const (
	RevocationReasonCodeUnspecified          RevocationReasonCode = 0x00000001
	RevocationReasonCodeKeyCompromise        RevocationReasonCode = 0x00000002
	RevocationReasonCodeCACompromise         RevocationReasonCode = 0x00000003
	RevocationReasonCodeAffiliationChanged   RevocationReasonCode = 0x00000004
	RevocationReasonCodeSuperseded           RevocationReasonCode = 0x00000005
	RevocationReasonCodeCessationOfOperation RevocationReasonCode = 0x00000006
	RevocationReasonCodePrivilegeWithdrawn   RevocationReasonCode = 0x00000007
)

type BatchErrorContinuationOption uint32

const (
	BatchErrorContinuationOptionContinue BatchErrorContinuationOption = 1
	BatchErrorContinuationOptionStop     BatchErrorContinuationOption = 2
	BatchErrorContinuationOptionUndo     BatchErrorContinuationOption = 3
)

type NameType uint32

const (
	NameTypeUninterpretedTextString NameType = 1
	NameTypeUri                     NameType = 2
)

type ObjectType uint32

const (
	ObjectTypeCertificate  ObjectType = 0x00000001
	ObjectTypeSymmetricKey ObjectType = 0x00000002
	ObjectTypePublicKey    ObjectType = 0x00000003
	ObjectTypePrivateKey   ObjectType = 0x00000004
	ObjectTypeSplitKey     ObjectType = 0x00000005
	// Deprecated: deprecated as of kmip 1.3.
	ObjectTypeTemplate     ObjectType = 0x00000006
	ObjectTypeSecretData   ObjectType = 0x00000007
	ObjectTypeOpaqueObject ObjectType = 0x00000008
	// KMIP 1.2.
	ObjectTypePGPKey ObjectType = 0x00000009
)

type OpaqueDataType uint32

type State uint32

const (
	StatePreActive            State = 0x00000001
	StateActive               State = 0x00000002
	StateDeactivated          State = 0x00000003
	StateCompromised          State = 0x00000004
	StateDestroyed            State = 0x00000005
	StateDestroyedCompromised State = 0x00000006
)

type CryptographicAlgorithm uint32

const (
	CryptographicAlgorithmDES        CryptographicAlgorithm = 0x00000001
	CryptographicAlgorithmTDES       CryptographicAlgorithm = 0x00000002
	CryptographicAlgorithmAES        CryptographicAlgorithm = 0x00000003
	CryptographicAlgorithmRSA        CryptographicAlgorithm = 0x00000004
	CryptographicAlgorithmDSA        CryptographicAlgorithm = 0x00000005
	CryptographicAlgorithmECDSA      CryptographicAlgorithm = 0x00000006
	CryptographicAlgorithmHMACSHA1   CryptographicAlgorithm = 0x00000007
	CryptographicAlgorithmHMACSHA224 CryptographicAlgorithm = 0x00000008
	CryptographicAlgorithmHMACSHA256 CryptographicAlgorithm = 0x00000009
	CryptographicAlgorithmHMACSHA384 CryptographicAlgorithm = 0x0000000A
	CryptographicAlgorithmHMACSHA512 CryptographicAlgorithm = 0x0000000B
	CryptographicAlgorithmHMACMD5    CryptographicAlgorithm = 0x0000000C
	CryptographicAlgorithmDH         CryptographicAlgorithm = 0x0000000D
	CryptographicAlgorithmECDH       CryptographicAlgorithm = 0x0000000E
	CryptographicAlgorithmECMQV      CryptographicAlgorithm = 0x0000000F
	CryptographicAlgorithmBlowfish   CryptographicAlgorithm = 0x00000010
	CryptographicAlgorithmCamellia   CryptographicAlgorithm = 0x00000011
	CryptographicAlgorithmCAST5      CryptographicAlgorithm = 0x00000012
	CryptographicAlgorithmIDEA       CryptographicAlgorithm = 0x00000013
	CryptographicAlgorithmMARS       CryptographicAlgorithm = 0x00000014
	CryptographicAlgorithmRC2        CryptographicAlgorithm = 0x00000015
	CryptographicAlgorithmRC4        CryptographicAlgorithm = 0x00000016
	CryptographicAlgorithmRC5        CryptographicAlgorithm = 0x00000017
	CryptographicAlgorithmSKIPJACK   CryptographicAlgorithm = 0x00000018
	CryptographicAlgorithmTwofish    CryptographicAlgorithm = 0x00000019

	// KMIP 1.2.
	CryptographicAlgorithmEC CryptographicAlgorithm = 0x0000001A

	// KMIP 1.3.
	CryptographicAlgorithmOneTimePad CryptographicAlgorithm = 0x0000001B

	// KMIP 1.4.
	CryptographicAlgorithmChaCha20         CryptographicAlgorithm = 0x0000001C
	CryptographicAlgorithmPoly1305         CryptographicAlgorithm = 0x0000001D
	CryptographicAlgorithmChaCha20Poly1305 CryptographicAlgorithm = 0x0000001E
	CryptographicAlgorithmSHA3_224         CryptographicAlgorithm = 0x0000001F
	CryptographicAlgorithmSHA3_256         CryptographicAlgorithm = 0x00000020
	CryptographicAlgorithmSHA3_384         CryptographicAlgorithm = 0x00000021
	CryptographicAlgorithmSHA3_512         CryptographicAlgorithm = 0x00000022
	CryptographicAlgorithmHMAC_SHA3_224    CryptographicAlgorithm = 0x00000023
	CryptographicAlgorithmHMAC_SHA3_256    CryptographicAlgorithm = 0x00000024
	CryptographicAlgorithmHMAC_SHA3_384    CryptographicAlgorithm = 0x00000025
	CryptographicAlgorithmHMAC_SHA3_512    CryptographicAlgorithm = 0x00000026
	CryptographicAlgorithmSHAKE_128        CryptographicAlgorithm = 0x00000027
	CryptographicAlgorithmSHAKE_256        CryptographicAlgorithm = 0x00000028
)

type BlockCipherMode uint32

const (
	BlockCipherModeCBC               BlockCipherMode = 0x00000001
	BlockCipherModeECB               BlockCipherMode = 0x00000002
	BlockCipherModePCBC              BlockCipherMode = 0x00000003
	BlockCipherModeCFB               BlockCipherMode = 0x00000004
	BlockCipherModeOFB               BlockCipherMode = 0x00000005
	BlockCipherModeCTR               BlockCipherMode = 0x00000006
	BlockCipherModeCMAC              BlockCipherMode = 0x00000007
	BlockCipherModeCCM               BlockCipherMode = 0x00000008
	BlockCipherModeGCM               BlockCipherMode = 0x00000009
	BlockCipherModeCBCMAC            BlockCipherMode = 0x0000000A
	BlockCipherModeXTS               BlockCipherMode = 0x0000000B
	BlockCipherModeAESKeyWrapPadding BlockCipherMode = 0x0000000C
	BlockCipherModeNISTKeyWrap       BlockCipherMode = 0x0000000D
	BlockCipherModeX9_102AESKW       BlockCipherMode = 0x0000000E
	BlockCipherModeX9_102TDKW        BlockCipherMode = 0x0000000F
	BlockCipherModeX9_102AKW1        BlockCipherMode = 0x00000010
	BlockCipherModeX9_102AKW2        BlockCipherMode = 0x00000011
	// KMIP 1.4.
	BlockCipherModeAEAD BlockCipherMode = 0x00000012
)

type PaddingMethod uint32

const (
	PaddingMethodNone      PaddingMethod = 0x00000001
	PaddingMethodOAEP      PaddingMethod = 0x00000002
	PaddingMethodPKCS5     PaddingMethod = 0x00000003
	PaddingMethodSSL3      PaddingMethod = 0x00000004
	PaddingMethodZeros     PaddingMethod = 0x00000005
	PaddingMethodANSIX9_23 PaddingMethod = 0x00000006
	PaddingMethodISO10126  PaddingMethod = 0x00000007
	PaddingMethodPKCS1V1_5 PaddingMethod = 0x00000008
	PaddingMethodX9_31     PaddingMethod = 0x00000009
	PaddingMethodPSS       PaddingMethod = 0x0000000A
)

type HashingAlgorithm uint32

const (
	HashingAlgorithmMD2        HashingAlgorithm = 0x00000001
	HashingAlgorithmMD4        HashingAlgorithm = 0x00000002
	HashingAlgorithmMD5        HashingAlgorithm = 0x00000003
	HashingAlgorithmSHA_1      HashingAlgorithm = 0x00000004
	HashingAlgorithmSHA_224    HashingAlgorithm = 0x00000005
	HashingAlgorithmSHA_256    HashingAlgorithm = 0x00000006
	HashingAlgorithmSHA_384    HashingAlgorithm = 0x00000007
	HashingAlgorithmSHA_512    HashingAlgorithm = 0x00000008
	HashingAlgorithmRIPEMD_160 HashingAlgorithm = 0x00000009
	HashingAlgorithmTiger      HashingAlgorithm = 0x0000000A
	HashingAlgorithmWhirlpool  HashingAlgorithm = 0x0000000B

	// KMIP 1.2.
	HashingAlgorithmSHA_512_224 HashingAlgorithm = 0x0000000C
	HashingAlgorithmSHA_512_256 HashingAlgorithm = 0x0000000D

	// KMIP 1.4.
	HashingAlgorithmSHA_3_224 HashingAlgorithm = 0x0000000E
	HashingAlgorithmSHA_3_256 HashingAlgorithm = 0x0000000F
	HashingAlgorithmSHA_3_384 HashingAlgorithm = 0x00000010
	HashingAlgorithmSHA_3_512 HashingAlgorithm = 0x00000011
)

type KeyRoleType uint32

const (
	KeyRoleTypeBDK      KeyRoleType = 0x00000001
	KeyRoleTypeCVK      KeyRoleType = 0x00000002
	KeyRoleTypeDEK      KeyRoleType = 0x00000003
	KeyRoleTypeMKAC     KeyRoleType = 0x00000004
	KeyRoleTypeMKSMC    KeyRoleType = 0x00000005
	KeyRoleTypeMKSMI    KeyRoleType = 0x00000006
	KeyRoleTypeMKDAC    KeyRoleType = 0x00000007
	KeyRoleTypeMKDN     KeyRoleType = 0x00000008
	KeyRoleTypeMKCP     KeyRoleType = 0x00000009
	KeyRoleTypeMKOTH    KeyRoleType = 0x0000000A
	KeyRoleTypeKEK      KeyRoleType = 0x0000000B
	KeyRoleTypeMAC16609 KeyRoleType = 0x0000000C
	KeyRoleTypeMAC97971 KeyRoleType = 0x0000000D
	KeyRoleTypeMAC97972 KeyRoleType = 0x0000000E
	KeyRoleTypeMAC97973 KeyRoleType = 0x0000000F
	KeyRoleTypeMAC97974 KeyRoleType = 0x00000010
	KeyRoleTypeMAC97975 KeyRoleType = 0x00000011
	KeyRoleTypeZPK      KeyRoleType = 0x00000012
	KeyRoleTypePVKIBM   KeyRoleType = 0x00000013
	KeyRoleTypePVKPVV   KeyRoleType = 0x00000014
	KeyRoleTypePVKOTH   KeyRoleType = 0x00000015

	// KMIP 1.4.
	KeyRoleTypeDUKPT KeyRoleType = 0x00000016
	KeyRoleTypeIV    KeyRoleType = 0x00000017
	KeyRoleTypeTRKBK KeyRoleType = 0x00000018
)

type RecommendedCurve uint32

const (
	RecommendedCurveP_192 RecommendedCurve = 0x00000001
	RecommendedCurveK_163 RecommendedCurve = 0x00000002
	RecommendedCurveB_163 RecommendedCurve = 0x00000003
	RecommendedCurveP_224 RecommendedCurve = 0x00000004
	RecommendedCurveK_233 RecommendedCurve = 0x00000005
	RecommendedCurveB_233 RecommendedCurve = 0x00000006
	RecommendedCurveP_256 RecommendedCurve = 0x00000007
	RecommendedCurveK_283 RecommendedCurve = 0x00000008
	RecommendedCurveB_283 RecommendedCurve = 0x00000009
	RecommendedCurveP_384 RecommendedCurve = 0x0000000A
	RecommendedCurveK_409 RecommendedCurve = 0x0000000B
	RecommendedCurveB_409 RecommendedCurve = 0x0000000C
	RecommendedCurveP_521 RecommendedCurve = 0x0000000D
	RecommendedCurveK_571 RecommendedCurve = 0x0000000E
	RecommendedCurveB_571 RecommendedCurve = 0x0000000F

	// KMIP 1.2.
	RecommendedCurveSECP112R1        RecommendedCurve = 0x00000010
	RecommendedCurveSECP112R2        RecommendedCurve = 0x00000011
	RecommendedCurveSECP128R1        RecommendedCurve = 0x00000012
	RecommendedCurveSECP128R2        RecommendedCurve = 0x00000013
	RecommendedCurveSECP160K1        RecommendedCurve = 0x00000014
	RecommendedCurveSECP160R1        RecommendedCurve = 0x00000015
	RecommendedCurveSECP160R2        RecommendedCurve = 0x00000016
	RecommendedCurveSECP192K1        RecommendedCurve = 0x00000017
	RecommendedCurveSECP224K1        RecommendedCurve = 0x00000018
	RecommendedCurveSECP256K1        RecommendedCurve = 0x00000019
	RecommendedCurveSECT113R1        RecommendedCurve = 0x0000001A
	RecommendedCurveSECT113R2        RecommendedCurve = 0x0000001B
	RecommendedCurveSECT131R1        RecommendedCurve = 0x0000001C
	RecommendedCurveSECT131R2        RecommendedCurve = 0x0000001D
	RecommendedCurveSECT163R1        RecommendedCurve = 0x0000001E
	RecommendedCurveSECT193R1        RecommendedCurve = 0x0000001F
	RecommendedCurveSECT193R2        RecommendedCurve = 0x00000020
	RecommendedCurveSECT239K1        RecommendedCurve = 0x00000021
	RecommendedCurveANSIX9P192V2     RecommendedCurve = 0x00000022
	RecommendedCurveANSIX9P192V3     RecommendedCurve = 0x00000023
	RecommendedCurveANSIX9P239V1     RecommendedCurve = 0x00000024
	RecommendedCurveANSIX9P239V2     RecommendedCurve = 0x00000025
	RecommendedCurveANSIX9P239V3     RecommendedCurve = 0x00000026
	RecommendedCurveANSIX9C2PNB163V1 RecommendedCurve = 0x00000027
	RecommendedCurveANSIX9C2PNB163V2 RecommendedCurve = 0x00000028
	RecommendedCurveANSIX9C2PNB163V3 RecommendedCurve = 0x00000029
	RecommendedCurveANSIX9C2PNB176V1 RecommendedCurve = 0x0000002A
	RecommendedCurveANSIX9C2TNB191V1 RecommendedCurve = 0x0000002B
	RecommendedCurveANSIX9C2TNB191V2 RecommendedCurve = 0x0000002C
	RecommendedCurveANSIX9C2TNB191V3 RecommendedCurve = 0x0000002D
	RecommendedCurveANSIX9C2PNB208W1 RecommendedCurve = 0x0000002E
	RecommendedCurveANSIX9C2TNB239V1 RecommendedCurve = 0x0000002F
	RecommendedCurveANSIX9C2TNB239V2 RecommendedCurve = 0x00000030
	RecommendedCurveANSIX9C2TNB239V3 RecommendedCurve = 0x00000031
	RecommendedCurveANSIX9C2PNB272W1 RecommendedCurve = 0x00000032
	RecommendedCurveANSIX9C2PNB304W1 RecommendedCurve = 0x00000033
	RecommendedCurveANSIX9C2TNB359V1 RecommendedCurve = 0x00000034
	RecommendedCurveANSIX9C2PNB368W1 RecommendedCurve = 0x00000035
	RecommendedCurveANSIX9C2TNB431R1 RecommendedCurve = 0x00000036
	RecommendedCurveBRAINPOOLP160R1  RecommendedCurve = 0x00000037
	RecommendedCurveBRAINPOOLP160T1  RecommendedCurve = 0x00000038
	RecommendedCurveBRAINPOOLP192R1  RecommendedCurve = 0x00000039
	RecommendedCurveBRAINPOOLP192T1  RecommendedCurve = 0x0000003A
	RecommendedCurveBRAINPOOLP224R1  RecommendedCurve = 0x0000003B
	RecommendedCurveBRAINPOOLP224T1  RecommendedCurve = 0x0000003C
	RecommendedCurveBRAINPOOLP256R1  RecommendedCurve = 0x0000003D
	RecommendedCurveBRAINPOOLP256T1  RecommendedCurve = 0x0000003E
	RecommendedCurveBRAINPOOLP320R1  RecommendedCurve = 0x0000003F
	RecommendedCurveBRAINPOOLP320T1  RecommendedCurve = 0x00000040
	RecommendedCurveBRAINPOOLP384R1  RecommendedCurve = 0x00000041
	RecommendedCurveBRAINPOOLP384T1  RecommendedCurve = 0x00000042
	RecommendedCurveBRAINPOOLP512R1  RecommendedCurve = 0x00000043
	RecommendedCurveBRAINPOOLP512T1  RecommendedCurve = 0x00000044
)

// Bitlen returns the bit length for the key using the curve.
//
//nolint:gocyclo // We can't do better
func (crv RecommendedCurve) Bitlen() int32 {
	switch crv {
	case RecommendedCurveP_192, RecommendedCurveSECP192K1, RecommendedCurveANSIX9P192V2,
		RecommendedCurveANSIX9P192V3, RecommendedCurveBRAINPOOLP192R1, RecommendedCurveBRAINPOOLP192T1:
		return 192
	case RecommendedCurveK_163, RecommendedCurveB_163, RecommendedCurveSECT163R1,
		RecommendedCurveANSIX9C2PNB163V1, RecommendedCurveANSIX9C2PNB163V2, RecommendedCurveANSIX9C2PNB163V3:
		return 163
	case RecommendedCurveP_256, RecommendedCurveSECP256K1, RecommendedCurveBRAINPOOLP256R1, RecommendedCurveBRAINPOOLP256T1:
		return 256
	case RecommendedCurveP_224, RecommendedCurveSECP224K1:
		return 224
	case RecommendedCurveK_233, RecommendedCurveB_233:
		return 233
	case RecommendedCurveK_283, RecommendedCurveB_283:
		return 283
	case RecommendedCurveP_384, RecommendedCurveBRAINPOOLP384R1, RecommendedCurveBRAINPOOLP384T1:
		return 384
	case RecommendedCurveK_409, RecommendedCurveB_409:
		return 409
	case RecommendedCurveP_521:
		return 521
	case RecommendedCurveK_571, RecommendedCurveB_571:
		return 571
	case RecommendedCurveSECP112R1, RecommendedCurveSECP112R2:
		return 112
	case RecommendedCurveSECP128R1, RecommendedCurveSECP128R2:
		return 128
	case RecommendedCurveSECP160K1, RecommendedCurveSECP160R1,
		RecommendedCurveSECP160R2, RecommendedCurveBRAINPOOLP160R1, RecommendedCurveBRAINPOOLP160T1:
		return 160
	case RecommendedCurveSECT113R1, RecommendedCurveSECT113R2:
		return 113
	case RecommendedCurveSECT131R1, RecommendedCurveSECT131R2:
		return 131
	case RecommendedCurveSECT193R1, RecommendedCurveSECT193R2:
		return 193
	case RecommendedCurveSECT239K1, RecommendedCurveANSIX9P239V1, RecommendedCurveANSIX9P239V2,
		RecommendedCurveANSIX9P239V3, RecommendedCurveANSIX9C2TNB239V1, RecommendedCurveANSIX9C2TNB239V2,
		RecommendedCurveANSIX9C2TNB239V3:
		return 239
	case RecommendedCurveANSIX9C2PNB176V1:
		return 176
	case RecommendedCurveANSIX9C2TNB191V1, RecommendedCurveANSIX9C2TNB191V2, RecommendedCurveANSIX9C2TNB191V3:
		return 191
	case RecommendedCurveANSIX9C2PNB208W1:
		return 208
	case RecommendedCurveANSIX9C2PNB272W1:
		return 272
	case RecommendedCurveANSIX9C2PNB304W1:
		return 304
	case RecommendedCurveANSIX9C2TNB359V1:
		return 359
	case RecommendedCurveANSIX9C2PNB368W1:
		return 368
	case RecommendedCurveANSIX9C2TNB431R1:
		return 431
	case RecommendedCurveBRAINPOOLP224R1, RecommendedCurveBRAINPOOLP224T1:
		return 224
	case RecommendedCurveBRAINPOOLP320R1, RecommendedCurveBRAINPOOLP320T1:
		return 320
	case RecommendedCurveBRAINPOOLP512R1, RecommendedCurveBRAINPOOLP512T1:
		return 512
	default:
		return 0
	}
}

type SecretDataType uint32

const (
	SecretDataTypePassword SecretDataType = 0x00000001
	SecretDataTypeSeed     SecretDataType = 0x00000002
)

type KeyFormatType uint32

const (
	KeyFormatTypeRaw                      KeyFormatType = 0x00000001
	KeyFormatTypeOpaque                   KeyFormatType = 0x00000002
	KeyFormatTypePKCS_1                   KeyFormatType = 0x00000003
	KeyFormatTypePKCS_8                   KeyFormatType = 0x00000004
	KeyFormatTypeX_509                    KeyFormatType = 0x00000005
	KeyFormatTypeECPrivateKey             KeyFormatType = 0x00000006
	KeyFormatTypeTransparentSymmetricKey  KeyFormatType = 0x00000007
	KeyFormatTypeTransparentDSAPrivateKey KeyFormatType = 0x00000008
	KeyFormatTypeTransparentDSAPublicKey  KeyFormatType = 0x00000009
	KeyFormatTypeTransparentRSAPrivateKey KeyFormatType = 0x0000000A
	KeyFormatTypeTransparentRSAPublicKey  KeyFormatType = 0x0000000B
	KeyFormatTypeTransparentDHPrivateKey  KeyFormatType = 0x0000000C
	KeyFormatTypeTransparentDHPublicKey   KeyFormatType = 0x0000000D
	// Deprecated: deprecated as of kmip 1.3.
	KeyFormatTypeTransparentECDSAPrivateKey KeyFormatType = 0x0000000E
	// Deprecated: deprecated as of kmip 1.3.
	KeyFormatTypeTransparentECDSAPublicKey KeyFormatType = 0x0000000F
	// Deprecated: deprecated as of kmip 1.3.
	KeyFormatTypeTransparentECDHPrivateKey KeyFormatType = 0x00000010
	// Deprecated: deprecated as of kmip 1.3.
	KeyFormatTypeTransparentECDHPublicKey KeyFormatType = 0x00000011
	// Deprecated: deprecated as of kmip 1.3.
	KeyFormatTypeTransparentECMQVPrivateKey KeyFormatType = 0x00000012
	// Deprecated: deprecated as of kmip 1.3.
	KeyFormatTypeTransparentECMQVPublicKey KeyFormatType = 0x00000013

	// KMIP 1.3.
	KeyFormatTypeTransparentECPrivateKey KeyFormatType = 0x00000014
	KeyFormatTypeTransparentECPublicKey  KeyFormatType = 0x00000015

	// KMIP 1.4.
	KeyFormatTypePKCS_12 KeyFormatType = 0x00000016
)

type KeyCompressionType uint32

const (
	KeyCompressionTypeECPublicKeyTypeUncompressed         KeyCompressionType = 0x00000001
	KeyCompressionTypeECPublicKeyTypeX9_62CompressedPrime KeyCompressionType = 0x00000002
	KeyCompressionTypeECPublicKeyTypeX9_62CompressedChar2 KeyCompressionType = 0x00000003
	KeyCompressionTypeECPublicKeyTypeX9_62Hybrid          KeyCompressionType = 0x00000004
)

type WrappingMethod uint32

const (
	WrappingMethodEncrypt            WrappingMethod = 0x00000001
	WrappingMethodMACSign            WrappingMethod = 0x00000002
	WrappingMethodEncryptThenMACSign WrappingMethod = 0x00000003
	WrappingMethodMACSignThenEncrypt WrappingMethod = 0x00000004
	WrappingMethodTR_31              WrappingMethod = 0x00000005
)

type CertificateType uint32

const (
	CertificateTypeX_509 CertificateType = 0x00000001
	// Deprecated: deprecated as of version 1.2.
	CertificateTypePGP CertificateType = 0x00000002
)

type LinkType uint32

const (
	LinkTypeCertificateLink          LinkType = 0x00000101
	LinkTypePublicKeyLink            LinkType = 0x00000102
	LinkTypePrivateKeyLink           LinkType = 0x00000103
	LinkTypeDerivationBaseObjectLink LinkType = 0x00000104
	LinkTypeDerivedKeyLink           LinkType = 0x00000105
	LinkTypeReplacementObjectLink    LinkType = 0x00000106
	LinkTypeReplacedObjectLink       LinkType = 0x00000107

	// KMIP 1.2.
	LinkTypeParentLink   LinkType = 0x00000108
	LinkTypeChildLink    LinkType = 0x00000109
	LinkTypePreviousLink LinkType = 0x0000010A
	LinkTypeNextLink     LinkType = 0x0000010B

	// KMPI 1.4.
	LinkTypePKCS_12CertificateLink LinkType = 0x0000010C
	LinkTypePKCS_12PasswordLink    LinkType = 0x0000010D

	//FIXME: This is defined in KMIP 2.0+ only.
	LinkTypeWrappingKeyLink LinkType = 0x0000010E
)

type QueryFunction uint32

const (
	QueryFunctionOperations            QueryFunction = 0x00000001
	QueryFunctionObjects               QueryFunction = 0x00000002
	QueryFunctionServerInformation     QueryFunction = 0x00000003
	QueryFunctionApplicationNamespaces QueryFunction = 0x00000004
	// KMIP 1.1.
	QueryFunctionExtensionList QueryFunction = 0x00000005
	QueryFunctionExtensionMap  QueryFunction = 0x00000006
	// KMIP 1.2.
	QueryFunctionAttestationTypes QueryFunction = 0x00000007

	// KMIP 1.3.
	QueryFunctionRNGs                      QueryFunction = 0x00000008
	QueryFunctionValidations               QueryFunction = 0x00000009
	QueryFunctionProfiles                  QueryFunction = 0x0000000A
	QueryFunctionCapabilities              QueryFunction = 0x0000000B
	QueryFunctionClientRegistrationMethods QueryFunction = 0x0000000C
)

type UsageLimitsUnit uint32

const (
	UsageLimitsUnitByte   UsageLimitsUnit = 0x00000001
	UsageLimitsUnitObject UsageLimitsUnit = 0x00000002
)

type CancellationResult uint32

const (
	CancellationResultCanceled       CancellationResult = 0x00000001
	CancellationResultUnableToCancel CancellationResult = 0x00000002
	CancellationResultCompleted      CancellationResult = 0x00000003
	CancellationResultFailed         CancellationResult = 0x00000004
	CancellationResultUnavailable    CancellationResult = 0x00000005
)

type PutFunction uint32

const (
	PutFunctionNew     PutFunction = 0x00000001
	PutFunctionReplace PutFunction = 0x00000002
)

type CertificateRequestType uint32

const (
	CertificateRequestTypeCRMF    CertificateRequestType = 0x00000001
	CertificateRequestTypePKCS_10 CertificateRequestType = 0x00000002
	CertificateRequestTypePEM     CertificateRequestType = 0x00000003
	CertificateRequestTypePGP     CertificateRequestType = 0x00000004
)

// kmip 1.1.

type SplitKeyMethod uint32

const (
	SplitKeyMethodXOR                         SplitKeyMethod = 0x00000001
	SplitKeyMethodPolynomialSharingGF216      SplitKeyMethod = 0x00000002
	SplitKeyMethodPolynomialSharingPrimeField SplitKeyMethod = 0x00000003

	// KMIP 1.2.
	SplitKeyMethodPolynomialSharingGF28 SplitKeyMethod = 0x00000004
)

type ObjectGroupMember uint32

const (
	ObjectGroupMemberFresh   ObjectGroupMember = 0x00000001
	ObjectGroupMemberDefault ObjectGroupMember = 0x00000002
)

type EncodingOption uint32

const (
	EncodingOptionNoEncoding   EncodingOption = 0x00000001
	EncodingOptionTTLVEncoding EncodingOption = 0x00000002
)

type DigitalSignatureAlgorithm uint32

const (
	DigitalSignatureAlgorithmMD2WithRSAEncryptionPKCS_1v1_5     DigitalSignatureAlgorithm = 0x00000001
	DigitalSignatureAlgorithmMD5WithRSAEncryptionPKCS_1v1_5     DigitalSignatureAlgorithm = 0x00000002
	DigitalSignatureAlgorithmSHA_1WithRSAEncryptionPKCS_1v1_5   DigitalSignatureAlgorithm = 0x00000003
	DigitalSignatureAlgorithmSHA_224WithRSAEncryptionPKCS_1v1_5 DigitalSignatureAlgorithm = 0x00000004
	DigitalSignatureAlgorithmSHA_256WithRSAEncryptionPKCS_1v1_5 DigitalSignatureAlgorithm = 0x00000005
	DigitalSignatureAlgorithmSHA_384WithRSAEncryptionPKCS_1v1_5 DigitalSignatureAlgorithm = 0x00000006
	DigitalSignatureAlgorithmSHA_512WithRSAEncryptionPKCS_1v1_5 DigitalSignatureAlgorithm = 0x00000007
	DigitalSignatureAlgorithmRSASSA_PSSPKCS_1v2_1               DigitalSignatureAlgorithm = 0x00000008
	DigitalSignatureAlgorithmDSAWithSHA_1                       DigitalSignatureAlgorithm = 0x00000009
	DigitalSignatureAlgorithmDSAWithSHA224                      DigitalSignatureAlgorithm = 0x0000000A
	DigitalSignatureAlgorithmDSAWithSHA256                      DigitalSignatureAlgorithm = 0x0000000B
	DigitalSignatureAlgorithmECDSAWithSHA_1                     DigitalSignatureAlgorithm = 0x0000000C
	DigitalSignatureAlgorithmECDSAWithSHA224                    DigitalSignatureAlgorithm = 0x0000000D
	DigitalSignatureAlgorithmECDSAWithSHA256                    DigitalSignatureAlgorithm = 0x0000000E
	DigitalSignatureAlgorithmECDSAWithSHA384                    DigitalSignatureAlgorithm = 0x0000000F
	DigitalSignatureAlgorithmECDSAWithSHA512                    DigitalSignatureAlgorithm = 0x00000010

	// KMIP 1.4.
	DigitalSignatureAlgorithmSHA3_256WithRSAEncryption DigitalSignatureAlgorithm = 0x00000011
	DigitalSignatureAlgorithmSHA3_384WithRSAEncryption DigitalSignatureAlgorithm = 0x00000012
	DigitalSignatureAlgorithmSHA3_512WithRSAEncryption DigitalSignatureAlgorithm = 0x00000013
)

// KMIP 1.2.

type AttestationType uint32

const (
	AttestationTypeTPMQuote           AttestationType = 0x00000001
	AttestationTypeTCGIntegrityReport AttestationType = 0x00000002
	AttestationTypeSAMLAssertion      AttestationType = 0x00000003
)

type AlternativeNameType uint32

const (
	AlternativeNameTypeUninterpretedTextString AlternativeNameType = 0x00000001
	AlternativeNameTypeURI                     AlternativeNameType = 0x00000002
	AlternativeNameTypeObjectSerialNumber      AlternativeNameType = 0x00000003
	AlternativeNameTypeEmailAddress            AlternativeNameType = 0x00000004
	AlternativeNameTypeDNSName                 AlternativeNameType = 0x00000005
	AlternativeNameTypeX_500DistinguishedName  AlternativeNameType = 0x00000006
	AlternativeNameTypeIPAddress               AlternativeNameType = 0x00000007
)

type KeyValueLocationType uint32

const (
	KeyValueLocationTypeUninterpretedTextString KeyValueLocationType = 0x00000001
	KeyValueLocationTypeURI                     KeyValueLocationType = 0x00000002
)

type ValidityIndicator uint32

const (
	ValidityIndicatorValid   ValidityIndicator = 0x00000001
	ValidityIndicatorInvalid ValidityIndicator = 0x00000002
	ValidityIndicatorUnknown ValidityIndicator = 0x00000003
)

// KMIP 1.3.

type RNGAlgorithm uint32

const (
	RNGAlgorithmUnspecified RNGAlgorithm = 0x00000001
	RNGAlgorithmFIPS186_2   RNGAlgorithm = 0x00000002
	RNGAlgorithmDRBG        RNGAlgorithm = 0x00000003
	RNGAlgorithmNRBG        RNGAlgorithm = 0x00000004
	RNGAlgorithmANSIX9_31   RNGAlgorithm = 0x00000005
	RNGAlgorithmANSIX9_62   RNGAlgorithm = 0x00000006
)

type DRBGAlgorithm uint32

const (
	DRBGAlgorithmUnspecified DRBGAlgorithm = 0x00000001
	DRBGAlgorithmDual_EC     DRBGAlgorithm = 0x00000002
	DRBGAlgorithmHash        DRBGAlgorithm = 0x00000003
	DRBGAlgorithmHMAC        DRBGAlgorithm = 0x00000004
	DRBGAlgorithmCTR         DRBGAlgorithm = 0x00000005
)

type FIPS186Variation uint32

const (
	FIPS186VariationUnspecified     FIPS186Variation = 0x00000001
	FIPS186VariationGPXOriginal     FIPS186Variation = 0x00000002
	FIPS186VariationGPXChangeNotice FIPS186Variation = 0x00000003
	FIPS186VariationXOriginal       FIPS186Variation = 0x00000004
	FIPS186VariationXChangeNotice   FIPS186Variation = 0x00000005
	FIPS186VariationKOriginal       FIPS186Variation = 0x00000006
	FIPS186VariationKChangeNotice   FIPS186Variation = 0x00000007
)

type ProfileName uint32

const (
	ProfileNameBaselineServerBasicKMIPV1_2                       ProfileName = 0x00000001
	ProfileNameBaselineServerTLSV1_2KMIPV1_2                     ProfileName = 0x00000002
	ProfileNameBaselineClientBasicKMIPV1_2                       ProfileName = 0x00000003
	ProfileNameBaselineClientTLSV1_2KMIPV1_2                     ProfileName = 0x00000004
	ProfileNameCompleteServerBasicKMIPV1_2                       ProfileName = 0x00000005
	ProfileNameCompleteServerTLSV1_2KMIPV1_2                     ProfileName = 0x00000006
	ProfileNameTapeLibraryClientKMIPV1_0                         ProfileName = 0x00000007
	ProfileNameTapeLibraryClientKMIPV1_1                         ProfileName = 0x00000008
	ProfileNameTapeLibraryClientKMIPV1_2                         ProfileName = 0x00000009
	ProfileNameTapeLibraryServerKMIPV1_0                         ProfileName = 0x0000000A
	ProfileNameTapeLibraryServerKMIPV1_1                         ProfileName = 0x0000000B
	ProfileNameTapeLibraryServerKMIPV1_2                         ProfileName = 0x0000000C
	ProfileNameSymmetricKeyLifecycleClientKMIPV1_0               ProfileName = 0x0000000D
	ProfileNameSymmetricKeyLifecycleClientKMIPV1_1               ProfileName = 0x0000000E
	ProfileNameSymmetricKeyLifecycleClientKMIPV1_2               ProfileName = 0x0000000F
	ProfileNameSymmetricKeyLifecycleServerKMIPV1_0               ProfileName = 0x00000010
	ProfileNameSymmetricKeyLifecycleServerKMIPV1_1               ProfileName = 0x00000011
	ProfileNameSymmetricKeyLifecycleServerKMIPV1_2               ProfileName = 0x00000012
	ProfileNameAsymmetricKeyLifecycleClientKMIPV1_0              ProfileName = 0x00000013
	ProfileNameAsymmetricKeyLifecycleClientKMIPV1_1              ProfileName = 0x00000014
	ProfileNameAsymmetricKeyLifecycleClientKMIPV1_2              ProfileName = 0x00000015
	ProfileNameAsymmetricKeyLifecycleServerKMIPV1_0              ProfileName = 0x00000016
	ProfileNameAsymmetricKeyLifecycleServerKMIPV1_1              ProfileName = 0x00000017
	ProfileNameAsymmetricKeyLifecycleServerKMIPV1_2              ProfileName = 0x00000018
	ProfileNameBasicCryptographicClientKMIPV1_2                  ProfileName = 0x00000019
	ProfileNameBasicCryptographicServerKMIPV1_2                  ProfileName = 0x0000001A
	ProfileNameAdvancedCryptographicClientKMIPV1_2               ProfileName = 0x0000001B
	ProfileNameAdvancedCryptographicServerKMIPV1_2               ProfileName = 0x0000001C
	ProfileNameRNGCryptographicClientKMIPV1_2                    ProfileName = 0x0000001D
	ProfileNameRNGCryptographicServerKMIPV1_2                    ProfileName = 0x0000001E
	ProfileNameBasicSymmetricKeyFoundryClientKMIPV1_0            ProfileName = 0x0000001F
	ProfileNameIntermediateSymmetricKeyFoundryClientKMIPV1_0     ProfileName = 0x00000020
	ProfileNameAdvancedSymmetricKeyFoundryClientKMIPV1_0         ProfileName = 0x00000021
	ProfileNameBasicSymmetricKeyFoundryClientKMIPV1_1            ProfileName = 0x00000022
	ProfileNameIntermediateSymmetricKeyFoundryClientKMIPV1_1     ProfileName = 0x00000023
	ProfileNameAdvancedSymmetricKeyFoundryClientKMIPV1_1         ProfileName = 0x00000024
	ProfileNameBasicSymmetricKeyFoundryClientKMIPV1_2            ProfileName = 0x00000025
	ProfileNameIntermediateSymmetricKeyFoundryClientKMIPV1_2     ProfileName = 0x00000026
	ProfileNameAdvancedSymmetricKeyFoundryClientKMIPV1_2         ProfileName = 0x00000027
	ProfileNameSymmetricKeyFoundryServerKMIPV1_0                 ProfileName = 0x00000028
	ProfileNameSymmetricKeyFoundryServerKMIPV1_1                 ProfileName = 0x00000029
	ProfileNameSymmetricKeyFoundryServerKMIPV1_2                 ProfileName = 0x0000002A
	ProfileNameOpaqueManagedObjectStoreClientKMIPV1_0            ProfileName = 0x0000002B
	ProfileNameOpaqueManagedObjectStoreClientKMIPV1_1            ProfileName = 0x0000002C
	ProfileNameOpaqueManagedObjectStoreClientKMIPV1_2            ProfileName = 0x0000002D
	ProfileNameOpaqueManagedObjectStoreServerKMIPV1_0            ProfileName = 0x0000002E
	ProfileNameOpaqueManagedObjectStoreServerKMIPV1_1            ProfileName = 0x0000002F
	ProfileNameOpaqueManagedObjectStoreServerKMIPV1_2            ProfileName = 0x00000030
	ProfileNameSuiteBMinLOS_128ClientKMIPV1_0                    ProfileName = 0x00000031
	ProfileNameSuiteBMinLOS_128ClientKMIPV1_1                    ProfileName = 0x00000032
	ProfileNameSuiteBMinLOS_128ClientKMIPV1_2                    ProfileName = 0x00000033
	ProfileNameSuiteBMinLOS_128ServerKMIPV1_0                    ProfileName = 0x00000034
	ProfileNameSuiteBMinLOS_128ServerKMIPV1_1                    ProfileName = 0x00000035
	ProfileNameSuiteBMinLOS_128ServerKMIPV1_2                    ProfileName = 0x00000036
	ProfileNameSuiteBMinLOS_192ClientKMIPV1_0                    ProfileName = 0x00000037
	ProfileNameSuiteBMinLOS_192ClientKMIPV1_1                    ProfileName = 0x00000038
	ProfileNameSuiteBMinLOS_192ClientKMIPV1_2                    ProfileName = 0x00000039
	ProfileNameSuiteBMinLOS_192ServerKMIPV1_0                    ProfileName = 0x0000003A
	ProfileNameSuiteBMinLOS_192ServerKMIPV1_1                    ProfileName = 0x0000003B
	ProfileNameSuiteBMinLOS_192ServerKMIPV1_2                    ProfileName = 0x0000003C
	ProfileNameStorageArrayWithSelfEncryptingDriveClientKMIPV1_0 ProfileName = 0x0000003D
	ProfileNameStorageArrayWithSelfEncryptingDriveClientKMIPV1_1 ProfileName = 0x0000003E
	ProfileNameStorageArrayWithSelfEncryptingDriveClientKMIPV1_2 ProfileName = 0x0000003F
	ProfileNameStorageArrayWithSelfEncryptingDriveServerKMIPV1_0 ProfileName = 0x00000040
	ProfileNameStorageArrayWithSelfEncryptingDriveServerKMIPV1_1 ProfileName = 0x00000041
	ProfileNameStorageArrayWithSelfEncryptingDriveServerKMIPV1_2 ProfileName = 0x00000042
	ProfileNameHTTPSClientKMIPV1_0                               ProfileName = 0x00000043
	ProfileNameHTTPSClientKMIPV1_1                               ProfileName = 0x00000044
	ProfileNameHTTPSClientKMIPV1_2                               ProfileName = 0x00000045
	ProfileNameHTTPSServerKMIPV1_0                               ProfileName = 0x00000046
	ProfileNameHTTPSServerKMIPV1_1                               ProfileName = 0x00000047
	ProfileNameHTTPSServerKMIPV1_2                               ProfileName = 0x00000048
	ProfileNameJSONClientKMIPV1_0                                ProfileName = 0x00000049
	ProfileNameJSONClientKMIPV1_1                                ProfileName = 0x0000004A
	ProfileNameJSONClientKMIPV1_2                                ProfileName = 0x0000004B
	ProfileNameJSONServerKMIPV1_0                                ProfileName = 0x0000004C
	ProfileNameJSONServerKMIPV1_1                                ProfileName = 0x0000004D
	ProfileNameJSONServerKMIPV1_2                                ProfileName = 0x0000004E
	ProfileNameXMLClientKMIPV1_0                                 ProfileName = 0x0000004F
	ProfileNameXMLClientKMIPV1_1                                 ProfileName = 0x00000050
	ProfileNameXMLClientKMIPV1_2                                 ProfileName = 0x00000051
	ProfileNameXMLServerKMIPV1_0                                 ProfileName = 0x00000052
	ProfileNameXMLServerKMIPV1_1                                 ProfileName = 0x00000053
	ProfileNameXMLServerKMIPV1_2                                 ProfileName = 0x00000054
	ProfileNameBaselineServerBasicKMIPV1_3                       ProfileName = 0x00000055
	ProfileNameBaselineServerTLSV1_2KMIPV1_3                     ProfileName = 0x00000056
	ProfileNameBaselineClientBasicKMIPV1_3                       ProfileName = 0x00000057
	ProfileNameBaselineClientTLSV1_2KMIPV1_3                     ProfileName = 0x00000058
	ProfileNameCompleteServerBasicKMIPV1_3                       ProfileName = 0x00000059
	ProfileNameCompleteServerTLSV1_2KMIPV1_3                     ProfileName = 0x0000005A
	ProfileNameTapeLibraryClientKMIPV1_3                         ProfileName = 0x0000005B
	ProfileNameTapeLibraryServerKMIPV1_3                         ProfileName = 0x0000005C
	ProfileNameSymmetricKeyLifecycleClientKMIPV1_3               ProfileName = 0x0000005D
	ProfileNameSymmetricKeyLifecycleServerKMIPV1_3               ProfileName = 0x0000005E
	ProfileNameAsymmetricKeyLifecycleClientKMIPV1_3              ProfileName = 0x0000005F
	ProfileNameAsymmetricKeyLifecycleServerKMIPV1_3              ProfileName = 0x00000060
	ProfileNameBasicCryptographicClientKMIPV1_3                  ProfileName = 0x00000061
	ProfileNameBasicCryptographicServerKMIPV1_3                  ProfileName = 0x00000062
	ProfileNameAdvancedCryptographicClientKMIPV1_3               ProfileName = 0x00000063
	ProfileNameAdvancedCryptographicServerKMIPV1_3               ProfileName = 0x00000064
	ProfileNameRNGCryptographicClientKMIPV1_3                    ProfileName = 0x00000065
	ProfileNameRNGCryptographicServerKMIPV1_3                    ProfileName = 0x00000066
	ProfileNameBasicSymmetricKeyFoundryClientKMIPV1_3            ProfileName = 0x00000067
	ProfileNameIntermediateSymmetricKeyFoundryClientKMIPV1_3     ProfileName = 0x00000068
	ProfileNameAdvancedSymmetricKeyFoundryClientKMIPV1_3         ProfileName = 0x00000069
	ProfileNameSymmetricKeyFoundryServerKMIPV1_3                 ProfileName = 0x0000006A
	ProfileNameOpaqueManagedObjectStoreClientKMIPV1_3            ProfileName = 0x0000006B
	ProfileNameOpaqueManagedObjectStoreServerKMIPV1_3            ProfileName = 0x0000006C
	ProfileNameSuiteBMinLOS_128ClientKMIPV1_3                    ProfileName = 0x0000006D
	ProfileNameSuiteBMinLOS_128ServerKMIPV1_3                    ProfileName = 0x0000006E
	ProfileNameSuiteBMinLOS_192ClientKMIPV1_3                    ProfileName = 0x0000006F
	ProfileNameSuiteBMinLOS_192ServerKMIPV1_3                    ProfileName = 0x00000070
	ProfileNameStorageArrayWithSelfEncryptingDriveClientKMIPV1_3 ProfileName = 0x00000071
	ProfileNameStorageArrayWithSelfEncryptingDriveServerKMIPV1_3 ProfileName = 0x00000072
	ProfileNameHTTPSClientKMIPV1_3                               ProfileName = 0x00000073
	ProfileNameHTTPSServerKMIPV1_3                               ProfileName = 0x00000074
	ProfileNameJSONClientKMIPV1_3                                ProfileName = 0x00000075
	ProfileNameJSONServerKMIPV1_3                                ProfileName = 0x00000076
	ProfileNameXMLClientKMIPV1_3                                 ProfileName = 0x00000077
	ProfileNameXMLServerKMIPV1_3                                 ProfileName = 0x00000078

	// KMIP 1.4.
	ProfileNameBaselineServerBasicKMIPV1_4                       ProfileName = 0x00000079
	ProfileNameBaselineServerTLSV1_2KMIPV1_4                     ProfileName = 0x0000007A
	ProfileNameBaselineClientBasicKMIPV1_4                       ProfileName = 0x0000007B
	ProfileNameBaselineClientTLSV1_2KMIPV1_4                     ProfileName = 0x0000007C
	ProfileNameCompleteServerBasicKMIPV1_4                       ProfileName = 0x0000007D
	ProfileNameCompleteServerTLSV1_2KMIPV1_4                     ProfileName = 0x0000007E
	ProfileNameTapeLibraryClientKMIPV1_4                         ProfileName = 0x0000007F
	ProfileNameTapeLibraryServerKMIPV1_4                         ProfileName = 0x00000080
	ProfileNameSymmetricKeyLifecycleClientKMIPV1_4               ProfileName = 0x00000081
	ProfileNameSymmetricKeyLifecycleServerKMIPV1_4               ProfileName = 0x00000082
	ProfileNameAsymmetricKeyLifecycleClientKMIPV1_4              ProfileName = 0x00000083
	ProfileNameAsymmetricKeyLifecycleServerKMIPV1_4              ProfileName = 0x00000084
	ProfileNameBasicCryptographicClientKMIPV1_4                  ProfileName = 0x00000085
	ProfileNameBasicCryptographicServerKMIPV1_4                  ProfileName = 0x00000086
	ProfileNameAdvancedCryptographicClientKMIPV1_4               ProfileName = 0x00000087
	ProfileNameAdvancedCryptographicServerKMIPV1_4               ProfileName = 0x00000088
	ProfileNameRNGCryptographicClientKMIPV1_4                    ProfileName = 0x00000089
	ProfileNameRNGCryptographicServerKMIPV1_4                    ProfileName = 0x0000008A
	ProfileNameBasicSymmetricKeyFoundryClientKMIPV1_4            ProfileName = 0x0000008B
	ProfileNameIntermediateSymmetricKeyFoundryClientKMIPV1_4     ProfileName = 0x0000008C
	ProfileNameAdvancedSymmetricKeyFoundryClientKMIPV1_4         ProfileName = 0x0000008D
	ProfileNameSymmetricKeyFoundryServerKMIPV1_4                 ProfileName = 0x0000008E
	ProfileNameOpaqueManagedObjectStoreClientKMIPV1_4            ProfileName = 0x0000008F
	ProfileNameOpaqueManagedObjectStoreServerKMIPV1_4            ProfileName = 0x00000090
	ProfileNameSuiteBMinLOS_128ClientKMIPV1_4                    ProfileName = 0x00000091
	ProfileNameSuiteBMinLOS_128ServerKMIPV1_4                    ProfileName = 0x00000092
	ProfileNameSuiteBMinLOS_192ClientKMIPV1_4                    ProfileName = 0x00000093
	ProfileNameSuiteBMinLOS_192ServerKMIPV1_4                    ProfileName = 0x00000094
	ProfileNameStorageArrayWithSelfEncryptingDriveClientKMIPV1_4 ProfileName = 0x00000095
	ProfileNameStorageArrayWithSelfEncryptingDriveServerKMIPV1_4 ProfileName = 0x00000096
	ProfileNameHTTPSClientKMIPV1_4                               ProfileName = 0x00000097
	ProfileNameHTTPSServerKMIPV1_4                               ProfileName = 0x00000098
	ProfileNameJSONClientKMIPV1_4                                ProfileName = 0x00000099
	ProfileNameJSONServerKMIPV1_4                                ProfileName = 0x0000009A
	ProfileNameXMLClientKMIPV1_4                                 ProfileName = 0x0000009B
	ProfileNameXMLServerKMIPV1_4                                 ProfileName = 0x0000009C
)

type ValidationAuthorityType uint32

const (
	ValidationAuthorityTypeUnspecified    ValidationAuthorityType = 0x00000001
	ValidationAuthorityTypeNISTCMVP       ValidationAuthorityType = 0x00000002
	ValidationAuthorityTypeCommonCriteria ValidationAuthorityType = 0x00000003
)

type ValidationType uint32

const (
	ValidationTypeUnspecified ValidationType = 0x00000001
	ValidationTypeHardware    ValidationType = 0x00000002
	ValidationTypeSoftware    ValidationType = 0x00000003
	ValidationTypeFirmware    ValidationType = 0x00000004
	ValidationTypeHybrid      ValidationType = 0x00000005
)

type UnwrapMode uint32

const (
	UnwrapModeUnspecified  UnwrapMode = 0x00000001
	UnwrapModeProcessed    UnwrapMode = 0x00000002
	UnwrapModeNotProcessed UnwrapMode = 0x00000003
)

type DestroyAction uint32

const (
	DestroyActionUnspecified         DestroyAction = 0x00000001
	DestroyActionKeyMaterialDeleted  DestroyAction = 0x00000002
	DestroyActionKeyMaterialShredded DestroyAction = 0x00000003
	DestroyActionMetaDataDeleted     DestroyAction = 0x00000004
	DestroyActionMetaDataShredded    DestroyAction = 0x00000005
	DestroyActionDeleted             DestroyAction = 0x00000006
	DestroyActionShredded            DestroyAction = 0x00000007
)

type ShreddingAlgorithm uint32

const (
	ShreddingAlgorithmUnspecified   ShreddingAlgorithm = 0x00000001
	ShreddingAlgorithmCryptographic ShreddingAlgorithm = 0x00000002
	ShreddingAlgorithmUnsupported   ShreddingAlgorithm = 0x00000003
)

type RNGMode uint32

const (
	RNGModeUnspecified            RNGMode = 0x00000001
	RNGModeSharedInstantiation    RNGMode = 0x00000002
	RNGModeNonSharedInstantiation RNGMode = 0x00000003
)

type ClientRegistrationMethod uint32

const (
	ClientRegistrationMethodUnspecified        ClientRegistrationMethod = 0x00000001
	ClientRegistrationMethodServerPreGenerated ClientRegistrationMethod = 0x00000002
	ClientRegistrationMethodServerOnDemand     ClientRegistrationMethod = 0x00000003
	ClientRegistrationMethodClientGenerated    ClientRegistrationMethod = 0x00000004
	ClientRegistrationMethodClientRegistered   ClientRegistrationMethod = 0x00000005
)

// KMIP 1.4.

type MaskGenerator uint32

const (
	MaskGeneratorMGF1 MaskGenerator = 0x00000001
)

// Text Marshaling for better display in json outputs.

func (enum ResultStatus) MarshalText() ([]byte, error) {
	return []byte(ttlv.EnumStr(enum)), nil
}
func (enum ResultReason) MarshalText() ([]byte, error) {
	return []byte(ttlv.EnumStr(enum)), nil
}
func (enum CredentialType) MarshalText() ([]byte, error) {
	return []byte(ttlv.EnumStr(enum)), nil
}
func (enum RevocationReasonCode) MarshalText() ([]byte, error) {
	return []byte(ttlv.EnumStr(enum)), nil
}
func (enum BatchErrorContinuationOption) MarshalText() ([]byte, error) {
	return []byte(ttlv.EnumStr(enum)), nil
}
func (enum NameType) MarshalText() ([]byte, error) {
	return []byte(ttlv.EnumStr(enum)), nil
}
func (enum ObjectType) MarshalText() ([]byte, error) {
	return []byte(ttlv.EnumStr(enum)), nil
}
func (enum OpaqueDataType) MarshalText() ([]byte, error) {
	return []byte(ttlv.EnumStr(enum)), nil
}
func (enum State) MarshalText() ([]byte, error) {
	return []byte(ttlv.EnumStr(enum)), nil
}
func (enum CryptographicAlgorithm) MarshalText() ([]byte, error) {
	return []byte(ttlv.EnumStr(enum)), nil
}
func (enum BlockCipherMode) MarshalText() ([]byte, error) {
	return []byte(ttlv.EnumStr(enum)), nil
}
func (enum PaddingMethod) MarshalText() ([]byte, error) {
	return []byte(ttlv.EnumStr(enum)), nil
}
func (enum HashingAlgorithm) MarshalText() ([]byte, error) {
	return []byte(ttlv.EnumStr(enum)), nil
}
func (enum KeyRoleType) MarshalText() ([]byte, error) {
	return []byte(ttlv.EnumStr(enum)), nil
}
func (enum RecommendedCurve) MarshalText() ([]byte, error) {
	return []byte(ttlv.EnumStr(enum)), nil
}
func (enum SecretDataType) MarshalText() ([]byte, error) {
	return []byte(ttlv.EnumStr(enum)), nil
}
func (enum KeyFormatType) MarshalText() ([]byte, error) {
	return []byte(ttlv.EnumStr(enum)), nil
}
func (enum KeyCompressionType) MarshalText() ([]byte, error) {
	return []byte(ttlv.EnumStr(enum)), nil
}
func (enum WrappingMethod) MarshalText() ([]byte, error) {
	return []byte(ttlv.EnumStr(enum)), nil
}
func (enum CertificateType) MarshalText() ([]byte, error) {
	return []byte(ttlv.EnumStr(enum)), nil
}
func (enum LinkType) MarshalText() ([]byte, error) {
	return []byte(ttlv.EnumStr(enum)), nil
}
func (enum QueryFunction) MarshalText() ([]byte, error) {
	return []byte(ttlv.EnumStr(enum)), nil
}
func (enum UsageLimitsUnit) MarshalText() ([]byte, error) {
	return []byte(ttlv.EnumStr(enum)), nil
}
func (enum CancellationResult) MarshalText() ([]byte, error) {
	return []byte(ttlv.EnumStr(enum)), nil
}
func (enum PutFunction) MarshalText() ([]byte, error) {
	return []byte(ttlv.EnumStr(enum)), nil
}
func (enum CertificateRequestType) MarshalText() ([]byte, error) {
	return []byte(ttlv.EnumStr(enum)), nil
}
func (enum SplitKeyMethod) MarshalText() ([]byte, error) {
	return []byte(ttlv.EnumStr(enum)), nil
}
func (enum ObjectGroupMember) MarshalText() ([]byte, error) {
	return []byte(ttlv.EnumStr(enum)), nil
}
func (enum EncodingOption) MarshalText() ([]byte, error) {
	return []byte(ttlv.EnumStr(enum)), nil
}
func (enum DigitalSignatureAlgorithm) MarshalText() ([]byte, error) {
	return []byte(ttlv.EnumStr(enum)), nil
}
func (enum AttestationType) MarshalText() ([]byte, error) {
	return []byte(ttlv.EnumStr(enum)), nil
}
func (enum AlternativeNameType) MarshalText() ([]byte, error) {
	return []byte(ttlv.EnumStr(enum)), nil
}
func (enum KeyValueLocationType) MarshalText() ([]byte, error) {
	return []byte(ttlv.EnumStr(enum)), nil
}
func (enum ValidityIndicator) MarshalText() ([]byte, error) {
	return []byte(ttlv.EnumStr(enum)), nil
}
func (enum RNGAlgorithm) MarshalText() ([]byte, error) {
	return []byte(ttlv.EnumStr(enum)), nil
}
func (enum DRBGAlgorithm) MarshalText() ([]byte, error) {
	return []byte(ttlv.EnumStr(enum)), nil
}
func (enum FIPS186Variation) MarshalText() ([]byte, error) {
	return []byte(ttlv.EnumStr(enum)), nil
}
func (enum ProfileName) MarshalText() ([]byte, error) {
	return []byte(ttlv.EnumStr(enum)), nil
}
func (enum ValidationAuthorityType) MarshalText() ([]byte, error) {
	return []byte(ttlv.EnumStr(enum)), nil
}
func (enum ValidationType) MarshalText() ([]byte, error) {
	return []byte(ttlv.EnumStr(enum)), nil
}
func (enum UnwrapMode) MarshalText() ([]byte, error) {
	return []byte(ttlv.EnumStr(enum)), nil
}
func (enum DestroyAction) MarshalText() ([]byte, error) {
	return []byte(ttlv.EnumStr(enum)), nil
}
func (enum ShreddingAlgorithm) MarshalText() ([]byte, error) {
	return []byte(ttlv.EnumStr(enum)), nil
}
func (enum RNGMode) MarshalText() ([]byte, error) {
	return []byte(ttlv.EnumStr(enum)), nil
}
func (enum ClientRegistrationMethod) MarshalText() ([]byte, error) {
	return []byte(ttlv.EnumStr(enum)), nil
}
func (enum MaskGenerator) MarshalText() ([]byte, error) {
	return []byte(ttlv.EnumStr(enum)), nil
}
