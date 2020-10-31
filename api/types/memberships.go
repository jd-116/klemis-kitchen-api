package types

// Membership is the database struct that contains information
// on the valid users that have access to the app and its dashboard
type Membership struct {
	Username    string `json:"username" bson:"username"`
	AdminAccess bool   `json:"admin_access" bson:"admin_access"`
}

// Permissions extracts the inner struct that is encoded in JWTs
func (m *Membership) Permissions() Permissions {
	return Permissions{
		AdminAccess: m.AdminAccess,
	}
}

// Permissions contains the struct that is encoded in each JWT
type Permissions struct {
	AdminAccess bool `json:"admin_access" bson:"admin_access"`
}
