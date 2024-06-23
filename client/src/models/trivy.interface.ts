export interface TrivyResults {
	SchemaVersion: number;
	CreatedAt: string;
	ArtifactName: string;
	ArtifactType: string;
	Metadata: Metadata;
	Results: Result[];
}

export interface Metadata {
	OS: Os;
	ImageID: string;
	DiffIDs: string[];
	RepoTags: string[];
	RepoDigests: string[];
	ImageConfig: ImageConfig;
}

export interface Os {
	Family: string;
	Name: string;
	EOSL: boolean;
}

export interface ImageConfig {
	architecture: string;
	created: string;
	history: History[];
	os: string;
	rootfs: Rootfs;
	config: Config;
}

export interface History {
	created: string;
	created_by: string;
	empty_layer?: boolean;
	comment?: string;
}

export interface Rootfs {
	type: string;
	diff_ids: string[];
}

export interface Config {
	Cmd: string[];
	Env: string[];
	Labels: Labels;
	User: string;
	Shell: string[];
}

export interface Labels {
	name: string;
	'org.opencontainers.image.base.name': string;
	'org.opencontainers.image.created': string;
	'org.opencontainers.image.ref.name': string;
	'org.opencontainers.image.revision': string;
	'org.opencontainers.image.source': string;
	'org.opencontainers.image.title': string;
	'org.opencontainers.image.url': string;
	'org.opencontainers.image.version': string;
}

export interface Result {
	Target: string;
	Class: string;
	Type: string;
	Vulnerabilities: Vulnerability[];
}

export interface Vulnerability {
	VulnerabilityID: string;
	PkgID: string;
	PkgName: string;
	PkgIdentifier: PkgIdentifier;
	InstalledVersion: string;
	FixedVersion: string;
	Status: string;
	Layer: Layer;
	SeveritySource?: string;
	PrimaryURL: string;
	DataSource: DataSource;
	Title: string;
	Description?: string;
	Severity: string;
	CweIDs?: string[];
	VendorSeverity: VendorSeverity;
	CVSS?: Cvss;
	References?: string[];
	PublishedDate?: string;
	LastModifiedDate?: string;
}

export interface PkgIdentifier {
	PURL: string;
	UID: string;
}

export interface Layer {
	DiffID: string;
}

export interface DataSource {
	ID: string;
	Name: string;
	URL: string;
}

export interface VendorSeverity {
	alma?: number;
	amazon?: number;
	'cbl-mariner'?: number;
	debian?: number;
	ghsa?: number;
	'oracle-oval'?: number;
	nvd?: number;
	photon?: number;
	redhat?: number;
	ubuntu?: number;
	rocky?: number;
}

export interface Cvss {
	ghsa?: Ghsa;
	nvd?: Nvd;
	redhat?: Redhat;
}

export interface Ghsa {
	V3Vector: string;
	V3Score: number;
}

export interface Nvd {
	V3Vector?: string;
	V3Score?: number;
	V2Vector?: string;
	V2Score?: number;
}

export interface Redhat {
	V3Vector?: string;
	V3Score?: number;
	V2Vector?: string;
	V2Score?: number;
}
