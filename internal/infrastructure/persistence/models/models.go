package models

func All() []any {
	return []any{
		&UserRecord{},
		&RoleRecord{},
		&UserRoleRecord{},
		&ChannelRecord{},
		&BuildJobRecord{},
	}
}
