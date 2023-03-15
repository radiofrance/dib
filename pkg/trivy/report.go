package trivy

import (
	"encoding/json"
	"time"
)

type ScanReport struct {
	SchemaVersion int    `json:"SchemaVersion"`
	ArtifactName  string `json:"ArtifactName"`
	ArtifactType  string `json:"ArtifactType"`
	Metadata      struct {
		Os struct {
			Family string `json:"Family"`
			Name   string `json:"Name"`
		} `json:"OS"`
		ImageID     string   `json:"ImageID"`
		DiffIDs     []string `json:"DiffIDs"`
		RepoTags    []string `json:"RepoTags"`
		RepoDigests []string `json:"RepoDigests"`
		ImageConfig struct {
			Architecture  string    `json:"architecture"`
			Container     string    `json:"container"`
			Created       time.Time `json:"created"`
			DockerVersion string    `json:"docker_version"`
			History       []struct {
				Created    time.Time `json:"created"`
				CreatedBy  string    `json:"created_by"`
				EmptyLayer bool      `json:"empty_layer,omitempty"`
				Author     string    `json:"author,omitempty"`
			} `json:"history"`
			Os     string `json:"os"`
			Rootfs struct {
				Type    string   `json:"type"`
				DiffIds []string `json:"diff_ids"`
			} `json:"rootfs"`
			Config struct {
				Entrypoint []string `json:"Entrypoint"`
				Env        []string `json:"Env"`
				Image      string   `json:"Image"`
				Labels     struct {
					Iii                            string    `json:"iii"`
					Name                           string    `json:"name"`
					OrgOpencontainersImageBaseName string    `json:"org.opencontainers.image.base.name"`
					OrgOpencontainersImageCreated  time.Time `json:"org.opencontainers.image.created"`
					OrgOpencontainersImageRefName  string    `json:"org.opencontainers.image.ref.name"`
					OrgOpencontainersImageRevision string    `json:"org.opencontainers.image.revision"`
					OrgOpencontainersImageSource   string    `json:"org.opencontainers.image.source"`
					OrgOpencontainersImageTitle    string    `json:"org.opencontainers.image.title"`
					OrgOpencontainersImageURL      string    `json:"org.opencontainers.image.url"`
					OrgOpencontainersImageVersion  string    `json:"org.opencontainers.image.version"`
					Version                        string    `json:"version"`
				} `json:"Labels"`
				User  string   `json:"User"`
				Shell []string `json:"Shell"`
			} `json:"config"`
		} `json:"ImageConfig"`
	} `json:"Metadata"`
	Results []struct {
		Target          string `json:"Target"`
		Class           string `json:"Class"`
		Type            string `json:"Type"`
		Vulnerabilities []struct {
			VulnerabilityID  string `json:"VulnerabilityID"`
			PkgID            string `json:"PkgID"`
			PkgName          string `json:"PkgName"`
			InstalledVersion string `json:"InstalledVersion"`
			Layer            struct {
				Digest string `json:"Digest"`
				DiffID string `json:"DiffID"`
			} `json:"Layer"`
			SeveritySource string `json:"SeveritySource,omitempty"`
			PrimaryURL     string `json:"PrimaryURL,omitempty"`
			DataSource     struct {
				ID   string `json:"ID"`
				Name string `json:"Name"`
				URL  string `json:"URL"`
			} `json:"DataSource"`
			Title       string   `json:"Title"`
			Description string   `json:"Description,omitempty"`
			Severity    string   `json:"Severity"`
			CweIDs      []string `json:"CweIDs,omitempty"`
			Cvss        struct {
				Nvd struct {
					V2Vector string  `json:"V2Vector"`
					V3Vector string  `json:"V3Vector"`
					V2Score  float64 `json:"V2Score"`
					V3Score  float64 `json:"V3Score"`
				} `json:"nvd"`
				RedHat struct {
					V2Vector string  `json:"V2Vector"`
					V3Vector string  `json:"V3Vector"`
					V2Score  float64 `json:"V2Score"`
					V3Score  float64 `json:"V3Score"`
				} `json:"redhat"`
			} `json:"CVSS,omitempty"`
			References       []string  `json:"References,omitempty"`
			PublishedDate    time.Time `json:"PublishedDate,omitempty"`
			LastModifiedDate time.Time `json:"LastModifiedDate,omitempty"`
			VendorIDs        []string  `json:"VendorIDs,omitempty"`
			FixedVersion     string    `json:"FixedVersion,omitempty"`
		} `json:"Vulnerabilities"`
	} `json:"Results"`
}

// ParseTrivyReport unmarshals a raw trivy json report into a golang ScanReport structure.
func ParseTrivyReport(raw []byte) (ScanReport, error) {
	var report ScanReport
	err := json.Unmarshal(raw, &report)
	if err != nil {
		return ScanReport{}, err
	}
	return report, nil
}
