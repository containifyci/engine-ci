package build

// https://goreleaser.com/customization/notarize/
func GoreleaserMacOsNotarizeEnvs() []string {
	return []string{
		"MACOS_SIGN_PASSWORD",
		"MACOS_SIGN_P12",
		"MACOS_NOTARY_ISSUER_ID",
		"MACOS_NOTARY_KEY_ID",
		"MACOS_NOTARY_KEY",
	}
}
