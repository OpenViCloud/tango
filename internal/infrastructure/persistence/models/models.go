package models

func All() []any {
	return []any{
		&UserRecord{},
		&RoleRecord{},
		&UserRoleRecord{},
		&ChannelRecord{},
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
	}
}
