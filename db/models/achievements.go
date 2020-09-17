package models

// Character is the digital representation of a client
type Achievements struct {
	CompanyTour  bool `json:"companyTour" redis:"companyTour"`
	PeerExpo     bool `json:"peerExpo" redis:"peerExpo"`
	Hangouts     bool `json:"hangouts" redis:"hangouts"`
	Workshops    bool `json:"workshops" redis:"workshops"`
	SocialMedia  bool `json:"socialMedia" redis:"socialMedia"`
	MemeLord     bool `json:"memeLord" redis:"memeLord"`
	SponsorQueue bool `json:"sponsorQueue" redis:"sponsorQueue"`
	SendLocation bool `json:"sendLocation" redis:"sendLocation"`
	TrackCounter bool `json:"trackCounter" redis:"trackCounter"`
	MiniEvents   bool `json:"miniEvents" redis:"miniEvents"`
}
