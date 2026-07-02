/**
* This file was @generated using pocketbase-typegen
*/

import type PocketBase from 'pocketbase'
import type { RecordService } from 'pocketbase'

export enum Collections {
	Authorigins = "_authOrigins",
	Externalauths = "_externalAuths",
	Mfas = "_mfas",
	Otps = "_otps",
	Superusers = "_superusers",
	Comands = "comands",
	OperationLogs = "operation_logs",
	Releases = "releases",
	ServiceUptimeView = "service_uptime_view",
	Services = "services",
	ServicesDomains = "services_domains",
	Users = "users",
}

// Alias types for improved usability
export type IsoDateString = string
export type RecordIdString = string
export type HTMLString = string

type ExpandType<T> = unknown extends T
	? T extends unknown
		? { expand?: unknown }
		: { expand: T }
	: { expand: T }

// System fields
export type BaseSystemFields<T = unknown> = {
	id: RecordIdString
	collectionId: string
	collectionName: Collections
} & ExpandType<T>

export type AuthSystemFields<T = unknown> = {
	email: string
	emailVisibility: boolean
	username: string
	verified: boolean
} & BaseSystemFields<T>

// Record types for each collection

export type AuthoriginsRecord = {
	collectionRef: string
	created?: IsoDateString
	fingerprint: string
	id: string
	recordRef: string
	updated?: IsoDateString
}

export type ExternalauthsRecord = {
	collectionRef: string
	created?: IsoDateString
	id: string
	provider: string
	providerId: string
	recordRef: string
	updated?: IsoDateString
}

export type MfasRecord = {
	collectionRef: string
	created?: IsoDateString
	id: string
	method: string
	recordRef: string
	updated?: IsoDateString
}

export type OtpsRecord = {
	collectionRef: string
	created?: IsoDateString
	id: string
	password: string
	recordRef: string
	sentTo?: string
	updated?: IsoDateString
}

export type SuperusersRecord = {
	created?: IsoDateString
	email: string
	emailVisibility?: boolean
	id: string
	password: string
	tokenKey: string
	updated?: IsoDateString
	verified?: boolean
}

export enum ComandsActionOptions {
	"stop" = "stop",
	"start" = "start",
	"restart" = "restart",
	"upgrade" = "upgrade",
}

export enum ComandsStatusOptions {
	"pending" = "pending",
	"success" = "success",
	"error" = "error",
}
export type ComandsRecord = {
	action: ComandsActionOptions
	created?: IsoDateString
	error_message?: string
	executed?: IsoDateString
	id: string
	service: RecordIdString
	status?: ComandsStatusOptions
	target_release?: RecordIdString
}

export enum OperationLogsStatusOptions {
	"success" = "success",
	"error" = "error",
}
export type OperationLogsRecord<Tmetadata = unknown> = {
	created?: IsoDateString
	id: string
	message?: string
	metadata?: null | Tmetadata
	operation: string
	service?: RecordIdString
	status: OperationLogsStatusOptions
}

export type ReleasesRecord = {
	asset_file_name: string
	asset_id: string
	asset_size?: number
	download_url: string
	id: string
	published_at: IsoDateString
	release_name: string
	version: string
}

export type ServiceUptimeViewRecord<Tactive_hours_24h = unknown, Tactive_hours_7d = unknown, Tinactive_hours_24h = unknown, Tinactive_hours_7d = unknown, Tservice_name = unknown, Tservice_status = unknown, Tuptime_24h = unknown, Tuptime_7d = unknown> = {
	active_hours_24h?: null | Tactive_hours_24h
	active_hours_7d?: null | Tactive_hours_7d
	id: string
	inactive_hours_24h?: null | Tinactive_hours_24h
	inactive_hours_7d?: null | Tinactive_hours_7d
	service_name?: null | Tservice_name
	service_status?: null | Tservice_status
	uptime_24h?: null | Tuptime_24h
	uptime_7d?: null | Tuptime_7d
}

export enum ServicesRestartPolicyOptions {
	"no" = "no",
	"on-failure" = "on-failure",
}

export enum ServicesStatusOptions {
	"idle" = "idle",
	"running" = "running",
	"stopped" = "stopped",
	"failure" = "failure",
	"restoring" = "restoring",
	"sleeping" = "sleeping",
}
export type ServicesRecord = {
	_pb_install?: string
	boot_user_email?: string
	boot_user_password?: string
	cpu_quota?: string
	created?: IsoDateString
	deleted?: IsoDateString
	error_message?: string
	id: string
	ip?: string
	last_started?: IsoDateString
	last_vacuum_at?: IsoDateString
	memory_limit?: string
	name: string
	port?: number
	release: RecordIdString
	restart_policy?: ServicesRestartPolicyOptions
	status?: ServicesStatusOptions
}

export enum ServicesDomainsUseHttpsOptions {
	"no" = "no",
	"yes" = "yes",
}

export enum ServicesDomainsCertStatusOptions {
	"pending" = "pending",
	"approved" = "approved",
	"failed" = "failed",
}
export type ServicesDomainsRecord = {
	cert_attempt?: number
	cert_error?: string
	cert_not_after?: IsoDateString
	cert_not_before?: IsoDateString
	cert_requested?: IsoDateString
	cert_status?: ServicesDomainsCertStatusOptions
	domain: string
	id: string
	service: RecordIdString[]
	use_https: ServicesDomainsUseHttpsOptions
}

export type UsersRecord = {
	avatar?: string
	created?: IsoDateString
	email: string
	emailVisibility?: boolean
	id: string
	name?: string
	password: string
	tokenKey: string
	updated?: IsoDateString
	verified?: boolean
}

// Response types include system fields and match responses from the PocketBase API
export type AuthoriginsResponse<Texpand = unknown> = Required<AuthoriginsRecord> & BaseSystemFields<Texpand>
export type ExternalauthsResponse<Texpand = unknown> = Required<ExternalauthsRecord> & BaseSystemFields<Texpand>
export type MfasResponse<Texpand = unknown> = Required<MfasRecord> & BaseSystemFields<Texpand>
export type OtpsResponse<Texpand = unknown> = Required<OtpsRecord> & BaseSystemFields<Texpand>
export type SuperusersResponse<Texpand = unknown> = Required<SuperusersRecord> & AuthSystemFields<Texpand>
export type ComandsResponse<Texpand = unknown> = Required<ComandsRecord> & BaseSystemFields<Texpand>
export type OperationLogsResponse<Tmetadata = unknown, Texpand = unknown> = Required<OperationLogsRecord<Tmetadata>> & BaseSystemFields<Texpand>
export type ReleasesResponse<Texpand = unknown> = Required<ReleasesRecord> & BaseSystemFields<Texpand>
export type ServiceUptimeViewResponse<Tactive_hours_24h = unknown, Tactive_hours_7d = unknown, Tinactive_hours_24h = unknown, Tinactive_hours_7d = unknown, Tservice_name = unknown, Tservice_status = unknown, Tuptime_24h = unknown, Tuptime_7d = unknown, Texpand = unknown> = Required<ServiceUptimeViewRecord<Tactive_hours_24h, Tactive_hours_7d, Tinactive_hours_24h, Tinactive_hours_7d, Tservice_name, Tservice_status, Tuptime_24h, Tuptime_7d>> & BaseSystemFields<Texpand>
export type ServicesResponse<Texpand = unknown> = Required<ServicesRecord> & BaseSystemFields<Texpand>
export type ServicesDomainsResponse<Texpand = unknown> = Required<ServicesDomainsRecord> & BaseSystemFields<Texpand>
export type UsersResponse<Texpand = unknown> = Required<UsersRecord> & AuthSystemFields<Texpand>

// Types containing all Records and Responses, useful for creating typing helper functions

export type CollectionRecords = {
	_authOrigins: AuthoriginsRecord
	_externalAuths: ExternalauthsRecord
	_mfas: MfasRecord
	_otps: OtpsRecord
	_superusers: SuperusersRecord
	comands: ComandsRecord
	operation_logs: OperationLogsRecord
	releases: ReleasesRecord
	service_uptime_view: ServiceUptimeViewRecord
	services: ServicesRecord
	services_domains: ServicesDomainsRecord
	users: UsersRecord
}

export type CollectionResponses = {
	_authOrigins: AuthoriginsResponse
	_externalAuths: ExternalauthsResponse
	_mfas: MfasResponse
	_otps: OtpsResponse
	_superusers: SuperusersResponse
	comands: ComandsResponse
	operation_logs: OperationLogsResponse
	releases: ReleasesResponse
	service_uptime_view: ServiceUptimeViewResponse
	services: ServicesResponse
	services_domains: ServicesDomainsResponse
	users: UsersResponse
}

// Type for usage with type asserted PocketBase instance
// https://github.com/pocketbase/js-sdk#specify-typescript-definitions

export type TypedPocketBase = PocketBase & {
	collection(idOrName: '_authOrigins'): RecordService<AuthoriginsResponse>
	collection(idOrName: '_externalAuths'): RecordService<ExternalauthsResponse>
	collection(idOrName: '_mfas'): RecordService<MfasResponse>
	collection(idOrName: '_otps'): RecordService<OtpsResponse>
	collection(idOrName: '_superusers'): RecordService<SuperusersResponse>
	collection(idOrName: 'comands'): RecordService<ComandsResponse>
	collection(idOrName: 'operation_logs'): RecordService<OperationLogsResponse>
	collection(idOrName: 'releases'): RecordService<ReleasesResponse>
	collection(idOrName: 'service_uptime_view'): RecordService<ServiceUptimeViewResponse>
	collection(idOrName: 'services'): RecordService<ServicesResponse>
	collection(idOrName: 'services_domains'): RecordService<ServicesDomainsResponse>
	collection(idOrName: 'users'): RecordService<UsersResponse>
}
