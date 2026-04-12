package models

func All() []any {
	return []any{
		&UserRecord{},
		&RoleRecord{},
		&UserRoleRecord{},
		&ChannelRecord{},
		&DatabaseSourceRecord{},
		&StorageRecord{},
		&BackupConfigRecord{},
		&BackupRecord{},
		&RestoreRecord{},
		&BuildJobRecord{},
		&ProjectRecord{},
		&EnvironmentRecord{},
		&ResourceRecord{},
		&ResourceRunRecord{},
		&ResourcePortRecord{},
		&ResourceEnvVarRecord{},
		&SourceProviderRecord{},
		&SourceConnectionRecord{},
		&PlatformConfigRecord{},
		&ResourceDomainRecord{},
		&BaseDomainRecord{},
		&ServerRecord{},
		&ClusterRecord{},
		&ClusterNodeRecord{},
		&CloudflareConnectionRecord{},
		&ClusterTunnelRecord{},
		&TunnelExposureRecord{},
	}
}
