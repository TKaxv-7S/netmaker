package ee

// LicenseLimits - struct license limits
type LicenseLimits struct {
	Servers  int `json:"servers"`
	Users    int `json:"users"`
	Hosts    int `json:"hosts"`
	Clients  int `json:"clients"`
	Networks int `json:"networks"`
}

// LicenseLimits.SetDefaults - sets the default values for limits
func (l *LicenseLimits) SetDefaults() {
	l.Clients = 0
	l.Servers = 1
	l.Hosts = 0
	l.Users = 1
	l.Networks = 0
}
