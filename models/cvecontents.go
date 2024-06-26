package models

import (
	"sort"
	"strings"
	"time"

	"github.com/future-architect/vuls/constant"
)

// CveContents has CveContent
type CveContents map[CveContentType][]CveContent

// NewCveContents create CveContents
func NewCveContents(conts ...CveContent) CveContents {
	m := CveContents{}
	for _, cont := range conts {
		if cont.Type == Jvn {
			found := false
			for _, cveCont := range m[cont.Type] {
				if cont.SourceLink == cveCont.SourceLink {
					found = true
					break
				}
			}
			if !found {
				m[cont.Type] = append(m[cont.Type], cont)
			}
		} else {
			m[cont.Type] = []CveContent{cont}
		}
	}
	return m
}

// CveContentStr has CveContentType and Value
type CveContentStr struct {
	Type  CveContentType
	Value string
}

// Except returns CveContents except given keys for enumeration
func (v CveContents) Except(exceptCtypes ...CveContentType) (values CveContents) {
	values = CveContents{}
	for ctype, content := range v {
		found := false
		for _, exceptCtype := range exceptCtypes {
			if ctype == exceptCtype {
				found = true
				break
			}
		}
		if !found {
			values[ctype] = content
		}
	}
	return
}

// PrimarySrcURLs returns link of source
func (v CveContents) PrimarySrcURLs(lang, myFamily, cveID string, confidences Confidences) (values []CveContentStr) {
	if cveID == "" {
		return
	}

	if conts, found := v[Nvd]; found {
		for _, cont := range conts {
			for _, r := range cont.References {
				for _, t := range r.Tags {
					if t == "Vendor Advisory" {
						values = append(values, CveContentStr{Nvd, r.Link})
					}
				}
			}
		}
	}

	order := append(append(CveContentTypes{Nvd}, GetCveContentTypes(myFamily)...), GitHub)
	for _, ctype := range order {
		if conts, found := v[ctype]; found {
			for _, cont := range conts {
				if cont.SourceLink == "" {
					continue
				}
				values = append(values, CveContentStr{ctype, cont.SourceLink})
			}
		}
	}

	jvnMatch := false
	for _, confidence := range confidences {
		if confidence.DetectionMethod == JvnVendorProductMatchStr {
			jvnMatch = true
			break
		}
	}

	if lang == "ja" || jvnMatch {
		if conts, found := v[Jvn]; found {
			for _, cont := range conts {
				if 0 < len(cont.SourceLink) {
					values = append(values, CveContentStr{Jvn, cont.SourceLink})
				}
			}
		}
	}

	if len(values) == 0 && strings.HasPrefix(cveID, "CVE") {
		return []CveContentStr{{
			Type:  Nvd,
			Value: "https://nvd.nist.gov/vuln/detail/" + cveID,
		}}
	}
	return values
}

// PatchURLs returns link of patch
func (v CveContents) PatchURLs() (urls []string) {
	conts, found := v[Nvd]
	if !found {
		return
	}

	for _, cont := range conts {
		for _, r := range cont.References {
			for _, t := range r.Tags {
				if t == "Patch" {
					urls = append(urls, r.Link)
				}
			}
		}
	}
	return
}

// CveContentCpes has CveContentType and Value
type CveContentCpes struct {
	Type  CveContentType
	Value []Cpe
}

// Cpes returns affected CPEs of this Vulnerability
func (v CveContents) Cpes(myFamily string) (values []CveContentCpes) {
	order := GetCveContentTypes(myFamily)
	order = append(order, AllCveContetTypes.Except(order...)...)

	for _, ctype := range order {
		if conts, found := v[ctype]; found {
			for _, cont := range conts {
				if 0 < len(cont.Cpes) {
					values = append(values, CveContentCpes{
						Type:  ctype,
						Value: cont.Cpes,
					})
				}
			}
		}
	}
	return
}

// CveContentRefs has CveContentType and Cpes
type CveContentRefs struct {
	Type  CveContentType
	Value []Reference
}

// References returns References
func (v CveContents) References(myFamily string) (values []CveContentRefs) {
	order := GetCveContentTypes(myFamily)
	order = append(order, AllCveContetTypes.Except(order...)...)

	for _, ctype := range order {
		if conts, found := v[ctype]; found {
			for _, cont := range conts {
				if 0 < len(cont.References) {
					values = append(values, CveContentRefs{
						Type:  ctype,
						Value: cont.References,
					})
				}
			}
		}
	}

	return
}

// CweIDs returns related CweIDs of the vulnerability
func (v CveContents) CweIDs(myFamily string) (values []CveContentStr) {
	order := GetCveContentTypes(myFamily)
	order = append(order, AllCveContetTypes.Except(order...)...)
	for _, ctype := range order {
		if conts, found := v[ctype]; found {
			for _, cont := range conts {
				if 0 < len(cont.CweIDs) {
					for _, cweID := range cont.CweIDs {
						for _, val := range values {
							if val.Value == cweID {
								continue
							}
						}
						values = append(values, CveContentStr{
							Type:  ctype,
							Value: cweID,
						})
					}
				}
			}
		}
	}
	return
}

// UniqCweIDs returns Uniq CweIDs
func (v CveContents) UniqCweIDs(myFamily string) (values []CveContentStr) {
	uniq := map[string]CveContentStr{}
	for _, cwes := range v.CweIDs(myFamily) {
		uniq[cwes.Value] = cwes
	}
	for _, cwe := range uniq {
		values = append(values, cwe)
	}
	return values
}

// Sort elements for integration-testing
func (v CveContents) Sort() {
	for contType, contents := range v {
		// CVSS3 desc, CVSS2 desc, SourceLink asc
		sort.Slice(contents, func(i, j int) bool {
			if contents[i].Cvss3Score > contents[j].Cvss3Score {
				return true
			} else if contents[i].Cvss3Score == contents[i].Cvss3Score {
				if contents[i].Cvss2Score > contents[j].Cvss2Score {
					return true
				} else if contents[i].Cvss2Score == contents[i].Cvss2Score {
					if contents[i].SourceLink < contents[j].SourceLink {
						return true
					}
				}
			}
			return false
		})
		v[contType] = contents
	}
	for contType, contents := range v {
		for cveID, cont := range contents {
			sort.Slice(cont.References, func(i, j int) bool {
				return cont.References[i].Link < cont.References[j].Link
			})
			sort.Slice(cont.CweIDs, func(i, j int) bool {
				return cont.CweIDs[i] < cont.CweIDs[j]
			})
			for i, ref := range cont.References {
				// sort v.CveContents[].References[].Tags
				sort.Slice(ref.Tags, func(j, k int) bool {
					return ref.Tags[j] < ref.Tags[k]
				})
				cont.References[i] = ref
			}
			contents[cveID] = cont
		}
		v[contType] = contents
	}
}

// CveContent has abstraction of various vulnerability information
type CveContent struct {
	Type          CveContentType    `json:"type"`
	CveID         string            `json:"cveID"`
	Title         string            `json:"title"`
	Summary       string            `json:"summary"`
	Cvss2Score    float64           `json:"cvss2Score"`
	Cvss2Vector   string            `json:"cvss2Vector"`
	Cvss2Severity string            `json:"cvss2Severity"`
	Cvss3Score    float64           `json:"cvss3Score"`
	Cvss3Vector   string            `json:"cvss3Vector"`
	Cvss3Severity string            `json:"cvss3Severity"`
	SourceLink    string            `json:"sourceLink"`
	Cpes          []Cpe             `json:"cpes,omitempty"`
	References    References        `json:"references,omitempty"`
	CweIDs        []string          `json:"cweIDs,omitempty"`
	Published     time.Time         `json:"published"`
	LastModified  time.Time         `json:"lastModified"`
	Optional      map[string]string `json:"optional,omitempty"`
}

// Empty checks the content is empty
func (c CveContent) Empty() bool {
	return c.Summary == ""
}

// CveContentType is a source of CVE information
type CveContentType string

// NewCveContentType create CveContentType
func NewCveContentType(name string) CveContentType {
	switch name {
	case "nvd":
		return Nvd
	case "jvn":
		return Jvn
	case "redhat", "centos", "alma", "rocky":
		return RedHat
	case "fedora":
		return Fedora
	case "oracle":
		return Oracle
	case "ubuntu":
		return Ubuntu
	case "debian", "debian-oval":
		return Debian
	case "redhat_api":
		return RedHatAPI
	case "debian_security_tracker":
		return DebianSecurityTracker
	case "ubuntu_api":
		return UbuntuAPI
	case constant.OpenSUSE, constant.OpenSUSELeap, constant.SUSEEnterpriseServer, constant.SUSEEnterpriseDesktop:
		return SUSE
	case "microsoft":
		return Microsoft
	case "wordpress":
		return WpScan
	case "amazon":
		return Amazon
	case "trivy":
		return Trivy
	case "trivy:nvd":
		return TrivyNVD
	case "trivy:redhat":
		return TrivyRedHat
	case "trivy:redhat-oval":
		return TrivyRedHatOVAL
	case "trivy:debian":
		return TrivyDebian
	case "trivy:ubuntu":
		return TrivyUbuntu
	case "trivy:centos":
		return TrivyCentOS
	case "trivy:rocky":
		return TrivyRocky
	case "trivy:fedora":
		return TrivyFedora
	case "trivy:amazon":
		return TrivyAmazon
	case "trivy:oracle-oval":
		return TrivyOracleOVAL
	case "trivy:suse-cvrf":
		return TrivySuseCVRF
	case "trivy:alpine":
		return TrivyAlpine
	case "trivy:arch-linux":
		return TrivyArchLinux
	case "trivy:alma":
		return TrivyAlma
	case "trivy:cbl-mariner":
		return TrivyCBLMariner
	case "trivy:photon":
		return TrivyPhoton
	case "trivy:ruby-advisory-db":
		return TrivyRubySec
	case "trivy:php-security-advisories":
		return TrivyPhpSecurityAdvisories
	case "trivy:nodejs-security-wg":
		return TrivyNodejsSecurityWg
	case "trivy:ghsa":
		return TrivyGHSA
	case "trivy:glad":
		return TrivyGLAD
	case "trivy:osv":
		return TrivyOSV
	case "trivy:wolfi":
		return TrivyWolfi
	case "trivy:chainguard":
		return TrivyChainguard
	case "trivy:bitnami":
		return TrivyBitnamiVulndb
	case "trivy:k8s":
		return TrivyK8sVulnDB
	case "trivy:govulndb":
		return TrivyGoVulnDB
	case "GitHub":
		return Trivy
	default:
		return Unknown
	}
}

// GetCveContentTypes return CveContentTypes
func GetCveContentTypes(family string) []CveContentType {
	switch family {
	case constant.RedHat, constant.CentOS, constant.Alma, constant.Rocky:
		return []CveContentType{RedHat, RedHatAPI}
	case constant.Fedora:
		return []CveContentType{Fedora}
	case constant.Oracle:
		return []CveContentType{Oracle}
	case constant.Amazon:
		return []CveContentType{Amazon}
	case constant.Debian, constant.Raspbian:
		return []CveContentType{Debian, DebianSecurityTracker}
	case constant.Ubuntu:
		return []CveContentType{Ubuntu, UbuntuAPI}
	case constant.OpenSUSE, constant.OpenSUSELeap, constant.SUSEEnterpriseServer, constant.SUSEEnterpriseDesktop:
		return []CveContentType{SUSE}
	case constant.Windows:
		return []CveContentType{Microsoft}
	case string(Trivy):
		return []CveContentType{Trivy, TrivyNVD, TrivyRedHat, TrivyRedHatOVAL, TrivyDebian, TrivyUbuntu, TrivyCentOS, TrivyRocky, TrivyFedora, TrivyAmazon, TrivyOracleOVAL, TrivySuseCVRF, TrivyAlpine, TrivyArchLinux, TrivyAlma, TrivyCBLMariner, TrivyPhoton, TrivyRubySec, TrivyPhpSecurityAdvisories, TrivyNodejsSecurityWg, TrivyGHSA, TrivyGLAD, TrivyOSV, TrivyWolfi, TrivyChainguard, TrivyBitnamiVulndb, TrivyK8sVulnDB, TrivyGoVulnDB}
	default:
		return nil
	}
}

const (
	// Nvd is Nvd JSON
	Nvd CveContentType = "nvd"

	// Jvn is Jvn
	Jvn CveContentType = "jvn"

	// Fortinet is Fortinet
	Fortinet CveContentType = "fortinet"

	// RedHat is RedHat
	RedHat CveContentType = "redhat"

	// RedHatAPI is RedHat
	RedHatAPI CveContentType = "redhat_api"

	// DebianSecurityTracker is Debian Security tracker
	DebianSecurityTracker CveContentType = "debian_security_tracker"

	// Debian is Debian
	Debian CveContentType = "debian"

	// Ubuntu is Ubuntu
	Ubuntu CveContentType = "ubuntu"

	// UbuntuAPI is Ubuntu
	UbuntuAPI CveContentType = "ubuntu_api"

	// Oracle is Oracle Linux
	Oracle CveContentType = "oracle"

	// Amazon is Amazon Linux
	Amazon CveContentType = "amazon"

	// Fedora is Fedora Linux
	Fedora CveContentType = "fedora"

	// SUSE is SUSE Linux
	SUSE CveContentType = "suse"

	// Microsoft is Microsoft
	Microsoft CveContentType = "microsoft"

	// WpScan is WordPress
	WpScan CveContentType = "wpscan"

	// Trivy is Trivy
	Trivy CveContentType = "trivy"

	// TrivyNVD is TrivyNVD
	TrivyNVD CveContentType = "trivy:nvd"

	// TrivyRedHat is TrivyRedHat
	TrivyRedHat CveContentType = "trivy:redhat"

	// TrivyRedHatOVAL is TrivyRedHatOVAL
	TrivyRedHatOVAL CveContentType = "trivy:redhat-oval"

	// TrivyDebian is TrivyDebian
	TrivyDebian CveContentType = "trivy:debian"

	// TrivyUbuntu is TrivyUbuntu
	TrivyUbuntu CveContentType = "trivy:ubuntu"

	// TrivyCentOS is TrivyCentOS
	TrivyCentOS CveContentType = "trivy:centos"

	// TrivyRocky is TrivyRocky
	TrivyRocky CveContentType = "trivy:rocky"

	// TrivyFedora is TrivyFedora
	TrivyFedora CveContentType = "trivy:fedora"

	// TrivyAmazon is TrivyAmazon
	TrivyAmazon CveContentType = "trivy:amazon"

	// TrivyOracleOVAL is TrivyOracle
	TrivyOracleOVAL CveContentType = "trivy:oracle-oval"

	// TrivySuseCVRF is TrivySuseCVRF
	TrivySuseCVRF CveContentType = "trivy:suse-cvrf"

	// TrivyAlpine is TrivyAlpine
	TrivyAlpine CveContentType = "trivy:alpine"

	// TrivyArchLinux is TrivyArchLinux
	TrivyArchLinux CveContentType = "trivy:arch-linux"

	// TrivyAlma is TrivyAlma
	TrivyAlma CveContentType = "trivy:alma"

	// TrivyCBLMariner is TrivyCBLMariner
	TrivyCBLMariner CveContentType = "trivy:cbl-mariner"

	// TrivyPhoton is TrivyPhoton
	TrivyPhoton CveContentType = "trivy:photon"

	// TrivyRubySec is TrivyRubySec
	TrivyRubySec CveContentType = "trivy:ruby-advisory-db"

	// TrivyPhpSecurityAdvisories is TrivyPhpSecurityAdvisories
	TrivyPhpSecurityAdvisories CveContentType = "trivy:php-security-advisories"

	// TrivyNodejsSecurityWg is TrivyNodejsSecurityWg
	TrivyNodejsSecurityWg CveContentType = "trivy:nodejs-security-wg"

	// TrivyGHSA is TrivyGHSA
	TrivyGHSA CveContentType = "trivy:ghsa"

	// TrivyGLAD is TrivyGLAD
	TrivyGLAD CveContentType = "trivy:glad"

	// TrivyOSV is TrivyOSV
	TrivyOSV CveContentType = "trivy:osv"

	// TrivyWolfi is TrivyWolfi
	TrivyWolfi CveContentType = "trivy:wolfi"

	// TrivyChainguard is TrivyChainguard
	TrivyChainguard CveContentType = "trivy:chainguard"

	// TrivyBitnamiVulndb is TrivyBitnamiVulndb
	TrivyBitnamiVulndb CveContentType = "trivy:bitnami"

	// TrivyK8sVulnDB is TrivyK8sVulnDB
	TrivyK8sVulnDB CveContentType = "trivy:k8s"

	// TrivyGoVulnDB is TrivyGoVulnDB
	TrivyGoVulnDB CveContentType = "trivy:govulndb"

	// GitHub is GitHub Security Alerts
	GitHub CveContentType = "github"

	// Unknown is Unknown
	Unknown CveContentType = "unknown"
)

// CveContentTypes has slide of CveContentType
type CveContentTypes []CveContentType

// AllCveContetTypes has all of CveContentTypes
var AllCveContetTypes = CveContentTypes{
	Nvd,
	Jvn,
	Fortinet,
	RedHat,
	RedHatAPI,
	Debian,
	DebianSecurityTracker,
	Ubuntu,
	UbuntuAPI,
	Amazon,
	Fedora,
	SUSE,
	WpScan,
	Trivy,
	TrivyNVD,
	TrivyRedHat,
	TrivyRedHatOVAL,
	TrivyDebian,
	TrivyUbuntu,
	TrivyCentOS,
	TrivyRocky,
	TrivyFedora,
	TrivyAmazon,
	TrivyOracleOVAL,
	TrivySuseCVRF,
	TrivyAlpine,
	TrivyArchLinux,
	TrivyAlma,
	TrivyCBLMariner,
	TrivyPhoton,
	TrivyRubySec,
	TrivyPhpSecurityAdvisories,
	TrivyNodejsSecurityWg,
	TrivyGHSA,
	TrivyGLAD,
	TrivyOSV,
	TrivyWolfi,
	TrivyChainguard,
	TrivyBitnamiVulndb,
	TrivyK8sVulnDB,
	TrivyGoVulnDB,
	GitHub,
}

// Except returns CveContentTypes except for given args
func (c CveContentTypes) Except(excepts ...CveContentType) (excepted CveContentTypes) {
	for _, ctype := range c {
		found := false
		for _, except := range excepts {
			if ctype == except {
				found = true
				break
			}
		}
		if !found {
			excepted = append(excepted, ctype)
		}
	}
	return
}

// Cpe is Common Platform Enumeration
type Cpe struct {
	URI             string `json:"uri"`
	FormattedString string `json:"formattedString"`
}

// References is a slice of Reference
type References []Reference

// Reference has a related link of the CVE
type Reference struct {
	Link   string   `json:"link,omitempty"`
	Source string   `json:"source,omitempty"`
	RefID  string   `json:"refID,omitempty"`
	Tags   []string `json:"tags,omitempty"`
}
